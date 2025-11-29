import axios, { AxiosInstance, AxiosError, AxiosRequestConfig } from 'axios';
import * as vscode from 'vscode';

const API_BASE_URL = 'https://api.getchainlens.com';
const MAX_RETRIES = 3;
const INITIAL_RETRY_DELAY_MS = 1000;
const RETRY_STATUS_CODES = [408, 429, 500, 502, 503, 504];

// =============================================================================
// Types
// =============================================================================

export interface TransactionTrace {
    tx_hash: string;
    chain: string;
    block_number: number;
    from: string;
    to: string;
    value: string;
    gas_used: number;
    gas_price: string;
    status: 'success' | 'failed';
    error?: string;
    calls: InternalCall[];
    logs: EventLog[];
    state_changes: StateChange[];
    gas_breakdown: GasBreakdown;
}

export interface InternalCall {
    type: string;
    from: string;
    to: string;
    value: string;
    gas_used: number;
    input: string;
    output: string;
    error?: string;
    depth: number;
    calls?: InternalCall[];
}

export interface EventLog {
    address: string;
    topics: string[];
    data: string;
    decoded?: {
        name: string;
        params: Record<string, unknown>;
    };
    log_index: number;
}

export interface StateChange {
    address: string;
    slot: string;
    old_value: string;
    new_value: string;
}

export interface GasBreakdown {
    intrinsic: number;
    execution: number;
    storage: number;
    refund: number;
    by_operation: Record<string, number>;
}

export interface AnalysisResult {
    id: string;
    source_hash: string;
    status: 'pending' | 'completed' | 'failed';
    issues: SecurityIssue[];
    gas_estimates: Record<string, GasEstimate>;
    score: {
        security: number;
        gas_efficiency: number;
        code_quality: number;
    };
    duration_ms: number;
}

export interface GasEstimate {
    min: number;
    max: number;
    typical: number;
    level: 'low' | 'medium' | 'high';
}

export interface SecurityIssue {
    id: string;
    type: string;
    severity: 'critical' | 'high' | 'medium' | 'low' | 'info';
    line: number;
    column: number;
    message: string;
    suggestion?: string;
    code?: string;
}

export interface SimulationResult {
    success: boolean;
    gas_used: number;
    gas_estimate: number;
    return_data: string;
    error?: string;
    revert?: {
        message: string;
        selector?: string;
        params?: string;
    };
    logs: EventLog[];
    state_changes: StateChange[];
    trace: InternalCall[];
}

export interface ContractMonitor {
    id: string;
    user_id: string;
    contract_id: string;
    name: string;
    event_filters: string[];
    webhook_url: string;
    is_active: boolean;
    created_at: string;
    updated_at: string;
}

export interface Project {
    id: string;
    user_id: string;
    name: string;
    description: string;
    created_at: string;
    updated_at: string;
}

export interface Contract {
    id: string;
    project_id: string;
    user_id: string;
    address: string;
    chain: string;
    name: string;
    abi?: string;
    source_code?: string;
    is_verified: boolean;
    created_at: string;
    updated_at: string;
}

export interface User {
    id: string;
    email: string;
    name: string;
    plan: 'free' | 'pro' | 'enterprise';
    api_calls_used: number;
    email_verified: boolean;
    created_at: string;
}

export interface AuthTokens {
    access_token: string;
    refresh_token: string;
    expires_in: number;
    user: User;
}

export interface LicenseInfo {
    valid: boolean;
    tier: 'free' | 'pro' | 'enterprise';
    features: string[];
    message?: string;
}

// =============================================================================
// API Client
// =============================================================================

export class ApiClient {
    private client: AxiosInstance;
    private accessToken: string = '';
    private refreshToken: string = '';
    private context: vscode.ExtensionContext | null = null;
    private secretStorage: vscode.SecretStorage | null = null;
    private isRefreshing: boolean = false;
    private refreshPromise: Promise<{ access_token: string; refresh_token: string }> | null = null;

    constructor(accessToken: string = '') {
        this.accessToken = accessToken;
        this.client = axios.create({
            baseURL: API_BASE_URL,
            timeout: 30000,
            headers: {
                'Content-Type': 'application/json',
                'X-Client': 'getchainlens-vscode',
                'X-Client-Version': '1.0.0',
            },
        });

        this.setupInterceptors();
    }

    private setupInterceptors(): void {
        // Request interceptor - add auth header
        this.client.interceptors.request.use((config) => {
            if (this.accessToken) {
                config.headers.Authorization = `Bearer ${this.accessToken}`;
            }
            return config;
        });

        // Response interceptor - handle token refresh and retries
        this.client.interceptors.response.use(
            (response) => response,
            async (error: AxiosError) => {
                const originalRequest = error.config as AxiosRequestConfig & { _retryCount?: number; _isRetry?: boolean };

                if (!originalRequest) {
                    return Promise.reject(error);
                }

                // Handle 401 - token refresh
                if (error.response?.status === 401 && this.refreshToken && !originalRequest._isRetry) {
                    originalRequest._isRetry = true;
                    try {
                        const tokens = await this.refreshAccessTokenWithLock();
                        await this.setTokens(tokens.access_token, tokens.refresh_token);
                        originalRequest.headers = originalRequest.headers || {};
                        originalRequest.headers.Authorization = `Bearer ${tokens.access_token}`;
                        return this.client(originalRequest);
                    } catch {
                        // Refresh failed, clear tokens
                        await this.clearTokens();
                    }
                }

                // Handle retryable errors with exponential backoff
                if (error.response && RETRY_STATUS_CODES.includes(error.response.status)) {
                    const retryCount = originalRequest._retryCount || 0;
                    if (retryCount < MAX_RETRIES) {
                        originalRequest._retryCount = retryCount + 1;
                        const delay = INITIAL_RETRY_DELAY_MS * Math.pow(2, retryCount);

                        // Add jitter to prevent thundering herd
                        const jitter = Math.random() * delay * 0.1;
                        await this.sleep(delay + jitter);

                        return this.client(originalRequest);
                    }
                }

                return Promise.reject(error);
            }
        );
    }

    private sleep(ms: number): Promise<void> {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    // Ensure only one refresh request at a time
    private async refreshAccessTokenWithLock(): Promise<{ access_token: string; refresh_token: string }> {
        if (this.isRefreshing && this.refreshPromise) {
            return this.refreshPromise;
        }

        this.isRefreshing = true;
        this.refreshPromise = this.refreshAccessToken();

        try {
            const result = await this.refreshPromise;
            return result;
        } finally {
            this.isRefreshing = false;
            this.refreshPromise = null;
        }
    }

    async setContext(context: vscode.ExtensionContext): Promise<void> {
        this.context = context;
        this.secretStorage = context.secrets;

        // Load tokens from secure storage
        await this.loadTokensFromStorage();
    }

    private async loadTokensFromStorage(): Promise<void> {
        if (this.secretStorage) {
            // Try secure storage first
            const savedToken = await this.secretStorage.get('getchainlens.accessToken');
            const savedRefresh = await this.secretStorage.get('getchainlens.refreshToken');
            if (savedToken) this.accessToken = savedToken;
            if (savedRefresh) this.refreshToken = savedRefresh;
        } else if (this.context) {
            // Fallback to global state (less secure, for migration)
            const savedToken = this.context.globalState.get<string>('getchainlens.accessToken');
            const savedRefresh = this.context.globalState.get<string>('getchainlens.refreshToken');
            if (savedToken) this.accessToken = savedToken;
            if (savedRefresh) this.refreshToken = savedRefresh;

            // Migrate to secure storage if tokens exist
            if (savedToken || savedRefresh) {
                await this.migrateToSecureStorage();
            }
        }
    }

    private async migrateToSecureStorage(): Promise<void> {
        if (this.secretStorage && this.context) {
            // Save to secure storage
            if (this.accessToken) {
                await this.secretStorage.store('getchainlens.accessToken', this.accessToken);
            }
            if (this.refreshToken) {
                await this.secretStorage.store('getchainlens.refreshToken', this.refreshToken);
            }

            // Remove from global state
            await this.context.globalState.update('getchainlens.accessToken', undefined);
            await this.context.globalState.update('getchainlens.refreshToken', undefined);
        }
    }

    async setTokens(accessToken: string, refreshToken: string): Promise<void> {
        this.accessToken = accessToken;
        this.refreshToken = refreshToken;

        if (this.secretStorage) {
            // Use secure storage (preferred)
            await this.secretStorage.store('getchainlens.accessToken', accessToken);
            await this.secretStorage.store('getchainlens.refreshToken', refreshToken);
        } else if (this.context) {
            // Fallback to global state
            await this.context.globalState.update('getchainlens.accessToken', accessToken);
            await this.context.globalState.update('getchainlens.refreshToken', refreshToken);
        }
    }

    async clearTokens(): Promise<void> {
        this.accessToken = '';
        this.refreshToken = '';

        if (this.secretStorage) {
            await this.secretStorage.delete('getchainlens.accessToken');
            await this.secretStorage.delete('getchainlens.refreshToken');
        }

        if (this.context) {
            await this.context.globalState.update('getchainlens.accessToken', undefined);
            await this.context.globalState.update('getchainlens.refreshToken', undefined);
        }
    }

    isAuthenticated(): boolean {
        return !!this.accessToken;
    }

    // =========================================================================
    // Authentication
    // =========================================================================

    async login(email: string, password: string): Promise<AuthTokens> {
        const response = await this.client.post<AuthTokens>('/api/v1/auth/login', {
            email,
            password,
        });
        await this.setTokens(response.data.access_token, response.data.refresh_token);
        return response.data;
    }

    async register(email: string, password: string, name: string): Promise<AuthTokens> {
        const response = await this.client.post<AuthTokens>('/api/v1/auth/register', {
            email,
            password,
            name,
        });
        await this.setTokens(response.data.access_token, response.data.refresh_token);
        return response.data;
    }

    async logout(): Promise<void> {
        try {
            // Optionally notify the server about logout
            if (this.isAuthenticated()) {
                await this.client.post('/api/v1/auth/logout').catch(() => {
                    // Ignore server errors on logout
                });
            }
        } finally {
            await this.clearTokens();
        }
    }

    private async refreshAccessToken(): Promise<{ access_token: string; refresh_token: string }> {
        const response = await axios.post(`${API_BASE_URL}/api/v1/auth/refresh`, {
            refresh_token: this.refreshToken,
        });
        return response.data;
    }

    async validateLicense(licenseKey: string): Promise<LicenseInfo> {
        const response = await this.client.post<LicenseInfo>('/api/v1/auth/validate-license', {
            license_key: licenseKey,
        });
        return response.data;
    }

    // =========================================================================
    // User
    // =========================================================================

    async getCurrentUser(): Promise<User> {
        const response = await this.client.get<User>('/api/v1/user');
        return response.data;
    }

    async updateUser(name: string): Promise<void> {
        await this.client.put('/api/v1/user', { name });
    }

    async changePassword(currentPassword: string, newPassword: string): Promise<void> {
        await this.client.post('/api/v1/user/password', {
            current_password: currentPassword,
            new_password: newPassword,
        });
    }

    async getUsage(): Promise<{ api_calls_used: number; api_calls_limit: number; period_end: string }> {
        const response = await this.client.get('/api/v1/user/usage');
        return response.data;
    }

    // =========================================================================
    // Analysis
    // =========================================================================

    async analyzeContract(
        source: string,
        compiler: string = '0.8.19',
        options: { security?: boolean; gas?: boolean; patterns?: boolean } = {}
    ): Promise<AnalysisResult> {
        const response = await this.client.post<AnalysisResult>('/api/v1/analyze', {
            source,
            compiler,
            options: {
                security: options.security ?? true,
                gas: options.gas ?? true,
                patterns: options.patterns ?? true,
            },
        });
        return response.data;
    }

    // =========================================================================
    // Transaction Tracing
    // =========================================================================

    async traceTransaction(txHash: string, chain: string = 'ethereum'): Promise<TransactionTrace> {
        const response = await this.client.get<TransactionTrace>(`/api/v1/trace/${txHash}`, {
            params: { chain },
        });
        return response.data;
    }

    // =========================================================================
    // Simulation
    // =========================================================================

    async simulateTransaction(
        chain: string,
        from: string,
        to: string,
        data: string,
        value: string = '0',
        gasLimit: number = 0
    ): Promise<SimulationResult> {
        const response = await this.client.post<SimulationResult>('/api/v1/simulate', {
            chain,
            from,
            to,
            data,
            value,
            gas_limit: gasLimit,
        });
        return response.data;
    }

    // =========================================================================
    // Forks
    // =========================================================================

    async createFork(chain: string, blockNumber?: number): Promise<{
        fork_id: string;
        rpc_url: string;
        chain: string;
        block_number: number;
        expires_at: string;
    }> {
        const response = await this.client.post('/api/v1/forks', {
            chain,
            block_number: blockNumber,
        });
        return response.data;
    }

    async deleteFork(forkId: string): Promise<void> {
        await this.client.delete(`/api/v1/forks/${forkId}`);
    }

    // =========================================================================
    // Projects
    // =========================================================================

    async getProjects(): Promise<Project[]> {
        const response = await this.client.get<Project[]>('/api/v1/projects');
        return response.data;
    }

    async createProject(name: string, description: string = ''): Promise<Project> {
        const response = await this.client.post<Project>('/api/v1/projects', {
            name,
            description,
        });
        return response.data;
    }

    async getProject(projectId: string): Promise<Project> {
        const response = await this.client.get<Project>(`/api/v1/projects/${projectId}`);
        return response.data;
    }

    async updateProject(projectId: string, name: string, description: string): Promise<void> {
        await this.client.put(`/api/v1/projects/${projectId}`, { name, description });
    }

    async deleteProject(projectId: string): Promise<void> {
        await this.client.delete(`/api/v1/projects/${projectId}`);
    }

    // =========================================================================
    // Contracts
    // =========================================================================

    async getContracts(projectId?: string): Promise<Contract[]> {
        const response = await this.client.get<Contract[]>('/api/v1/contracts', {
            params: projectId ? { project_id: projectId } : undefined,
        });
        return response.data;
    }

    async addContract(
        projectId: string,
        address: string,
        chain: string,
        name: string
    ): Promise<Contract> {
        const response = await this.client.post<Contract>('/api/v1/contracts', {
            project_id: projectId,
            address,
            chain,
            name,
        });
        return response.data;
    }

    async deleteContract(contractId: string): Promise<void> {
        await this.client.delete(`/api/v1/contracts/${contractId}`);
    }

    // =========================================================================
    // Monitors
    // =========================================================================

    async getMonitors(): Promise<ContractMonitor[]> {
        const response = await this.client.get<ContractMonitor[]>('/api/v1/monitors');
        return response.data;
    }

    async createMonitor(
        contractId: string,
        name: string,
        webhookUrl: string,
        eventFilters: string[] = []
    ): Promise<ContractMonitor> {
        const response = await this.client.post<ContractMonitor>('/api/v1/monitors', {
            contract_id: contractId,
            name,
            webhook_url: webhookUrl,
            event_filters: eventFilters,
        });
        return response.data;
    }

    async updateMonitor(
        monitorId: string,
        name: string,
        webhookUrl: string,
        eventFilters: string[],
        isActive: boolean
    ): Promise<void> {
        await this.client.put(`/api/v1/monitors/${monitorId}`, {
            name,
            webhook_url: webhookUrl,
            event_filters: eventFilters,
            is_active: isActive,
        });
    }

    async deleteMonitor(monitorId: string): Promise<void> {
        await this.client.delete(`/api/v1/monitors/${monitorId}`);
    }

    // =========================================================================
    // Billing
    // =========================================================================

    async getSubscription(): Promise<{
        plan: string;
        status: string;
        api_calls_used: number;
        api_calls_reset_at: string;
    }> {
        const response = await this.client.get('/api/v1/billing/subscription');
        return response.data;
    }

    async createCheckoutSession(
        priceId: string,
        successUrl: string,
        cancelUrl: string
    ): Promise<{ url: string }> {
        const response = await this.client.post('/api/v1/billing/checkout', {
            price_id: priceId,
            success_url: successUrl,
            cancel_url: cancelUrl,
        });
        return response.data;
    }

    async createBillingPortal(returnUrl: string): Promise<{ url: string }> {
        const response = await this.client.post('/api/v1/billing/portal', {
            return_url: returnUrl,
        });
        return response.data;
    }

    // =========================================================================
    // Health
    // =========================================================================

    async healthCheck(): Promise<{ status: string; version: string }> {
        const response = await this.client.get('/health');
        return response.data;
    }
}

// Singleton instance
let apiClientInstance: ApiClient | null = null;

export function getApiClient(): ApiClient {
    if (!apiClientInstance) {
        apiClientInstance = new ApiClient();
    }
    return apiClientInstance;
}

export async function initializeApiClient(context: vscode.ExtensionContext): Promise<ApiClient> {
    const client = getApiClient();
    await client.setContext(context);
    return client;
}
