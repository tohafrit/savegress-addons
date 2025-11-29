import axios, { AxiosInstance, AxiosError } from 'axios';
import * as vscode from 'vscode';

const API_BASE_URL = 'https://api.chainlens.dev';

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

    constructor(accessToken: string = '') {
        this.accessToken = accessToken;
        this.client = axios.create({
            baseURL: API_BASE_URL,
            timeout: 30000,
            headers: {
                'Content-Type': 'application/json',
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

        // Response interceptor - handle token refresh
        this.client.interceptors.response.use(
            (response) => response,
            async (error: AxiosError) => {
                const originalRequest = error.config;

                if (error.response?.status === 401 && this.refreshToken && originalRequest) {
                    try {
                        const tokens = await this.refreshAccessToken();
                        this.setTokens(tokens.access_token, tokens.refresh_token);
                        originalRequest.headers.Authorization = `Bearer ${tokens.access_token}`;
                        return this.client(originalRequest);
                    } catch {
                        // Refresh failed, clear tokens
                        this.clearTokens();
                    }
                }
                return Promise.reject(error);
            }
        );
    }

    setContext(context: vscode.ExtensionContext): void {
        this.context = context;
        // Load saved tokens
        const savedToken = context.globalState.get<string>('chainlens.accessToken');
        const savedRefresh = context.globalState.get<string>('chainlens.refreshToken');
        if (savedToken) this.accessToken = savedToken;
        if (savedRefresh) this.refreshToken = savedRefresh;
    }

    setTokens(accessToken: string, refreshToken: string): void {
        this.accessToken = accessToken;
        this.refreshToken = refreshToken;
        if (this.context) {
            this.context.globalState.update('chainlens.accessToken', accessToken);
            this.context.globalState.update('chainlens.refreshToken', refreshToken);
        }
    }

    clearTokens(): void {
        this.accessToken = '';
        this.refreshToken = '';
        if (this.context) {
            this.context.globalState.update('chainlens.accessToken', undefined);
            this.context.globalState.update('chainlens.refreshToken', undefined);
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
        this.setTokens(response.data.access_token, response.data.refresh_token);
        return response.data;
    }

    async register(email: string, password: string, name: string): Promise<AuthTokens> {
        const response = await this.client.post<AuthTokens>('/api/v1/auth/register', {
            email,
            password,
            name,
        });
        this.setTokens(response.data.access_token, response.data.refresh_token);
        return response.data;
    }

    async logout(): Promise<void> {
        this.clearTokens();
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

export function initializeApiClient(context: vscode.ExtensionContext): ApiClient {
    const client = getApiClient();
    client.setContext(context);
    return client;
}
