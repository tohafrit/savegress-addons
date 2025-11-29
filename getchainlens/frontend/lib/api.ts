import type {
  User,
  AuthTokens,
  Project,
  Contract,
  AnalysisResult,
  TransactionTrace,
  SimulationResult,
  ContractMonitor,
  Monitor,
  MonitorAlert,
  NetworkAnalytics,
  TopContract,
  Subscription,
  Invoice,
  UsageStats,
  DashboardStats,
  Chain,
  ApiResponse,
  PaginatedResponse,
} from '@/types';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'https://api.getchainlens.com';

class ApiClient {
  private accessToken: string = '';
  private refreshToken: string = '';

  constructor() {
    if (typeof window !== 'undefined') {
      this.accessToken = localStorage.getItem('getchainlens_access_token') || '';
      this.refreshToken = localStorage.getItem('getchainlens_refresh_token') || '';
    }
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<ApiResponse<T>> {
    const url = `${API_BASE_URL}${endpoint}`;
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
    };

    if (this.accessToken) {
      (headers as Record<string, string>)['Authorization'] = `Bearer ${this.accessToken}`;
    }

    try {
      const response = await fetch(url, {
        ...options,
        headers,
      });

      // Handle 401 - try to refresh token
      if (response.status === 401 && this.refreshToken) {
        const refreshed = await this.refreshAccessToken();
        if (refreshed) {
          (headers as Record<string, string>)['Authorization'] = `Bearer ${this.accessToken}`;
          const retryResponse = await fetch(url, { ...options, headers });
          if (!retryResponse.ok) {
            const error = await retryResponse.json().catch(() => ({}));
            return { error: error.message || 'Request failed' };
          }
          return { data: await retryResponse.json() };
        }
      }

      if (!response.ok) {
        const error = await response.json().catch(() => ({}));
        return { error: error.message || `HTTP ${response.status}` };
      }

      const data = await response.json();
      return { data };
    } catch (error) {
      return { error: error instanceof Error ? error.message : 'Network error' };
    }
  }

  private setTokens(accessToken: string, refreshToken: string): void {
    this.accessToken = accessToken;
    this.refreshToken = refreshToken;
    if (typeof window !== 'undefined') {
      localStorage.setItem('getchainlens_access_token', accessToken);
      localStorage.setItem('getchainlens_refresh_token', refreshToken);
    }
  }

  clearTokens(): void {
    this.accessToken = '';
    this.refreshToken = '';
    if (typeof window !== 'undefined') {
      localStorage.removeItem('getchainlens_access_token');
      localStorage.removeItem('getchainlens_refresh_token');
    }
  }

  isAuthenticated(): boolean {
    return !!this.accessToken;
  }

  // ============================================================================
  // Authentication
  // ============================================================================

  async login(email: string, password: string): Promise<ApiResponse<AuthTokens>> {
    const result = await this.request<AuthTokens>('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
    if (result.data) {
      this.setTokens(result.data.access_token, result.data.refresh_token);
    }
    return result;
  }

  async register(email: string, password: string, name: string): Promise<ApiResponse<AuthTokens>> {
    const result = await this.request<AuthTokens>('/api/v1/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, password, name }),
    });
    if (result.data) {
      this.setTokens(result.data.access_token, result.data.refresh_token);
    }
    return result;
  }

  async logout(): Promise<void> {
    await this.request('/api/v1/auth/logout', { method: 'POST' }).catch(() => {});
    this.clearTokens();
  }

  private async refreshAccessToken(): Promise<boolean> {
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: this.refreshToken }),
      });

      if (!response.ok) {
        this.clearTokens();
        return false;
      }

      const data = await response.json();
      this.setTokens(data.access_token, data.refresh_token);
      return true;
    } catch {
      this.clearTokens();
      return false;
    }
  }

  async forgotPassword(email: string): Promise<ApiResponse<void>> {
    return this.request('/api/v1/auth/forgot-password', {
      method: 'POST',
      body: JSON.stringify({ email }),
    });
  }

  async resetPassword(token: string, password: string): Promise<ApiResponse<void>> {
    return this.request('/api/v1/auth/reset-password', {
      method: 'POST',
      body: JSON.stringify({ token, password }),
    });
  }

  // ============================================================================
  // User
  // ============================================================================

  async getCurrentUser(): Promise<ApiResponse<User>> {
    return this.request<User>('/api/v1/user');
  }

  async updateUser(data: Partial<User>): Promise<ApiResponse<User>> {
    return this.request<User>('/api/v1/user', {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async changePassword(currentPassword: string, newPassword: string): Promise<ApiResponse<void>> {
    return this.request('/api/v1/user/password', {
      method: 'POST',
      body: JSON.stringify({
        current_password: currentPassword,
        new_password: newPassword,
      }),
    });
  }

  async getUsage(): Promise<ApiResponse<UsageStats>> {
    return this.request<UsageStats>('/api/v1/user/usage');
  }

  // ============================================================================
  // Dashboard
  // ============================================================================

  async getDashboardStats(): Promise<ApiResponse<DashboardStats>> {
    return this.request<DashboardStats>('/api/v1/dashboard/stats');
  }

  // ============================================================================
  // Projects
  // ============================================================================

  async getProjects(): Promise<ApiResponse<Project[]>> {
    return this.request<Project[]>('/api/v1/projects');
  }

  async getProject(id: string): Promise<ApiResponse<Project>> {
    return this.request<Project>(`/api/v1/projects/${id}`);
  }

  async createProject(name: string, description: string = ''): Promise<ApiResponse<Project>> {
    return this.request<Project>('/api/v1/projects', {
      method: 'POST',
      body: JSON.stringify({ name, description }),
    });
  }

  async updateProject(id: string, data: Partial<Project>): Promise<ApiResponse<Project>> {
    return this.request<Project>(`/api/v1/projects/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteProject(id: string): Promise<ApiResponse<void>> {
    return this.request(`/api/v1/projects/${id}`, { method: 'DELETE' });
  }

  // ============================================================================
  // Contracts
  // ============================================================================

  async getContracts(projectId?: string): Promise<ApiResponse<Contract[]>> {
    const query = projectId ? `?project_id=${projectId}` : '';
    return this.request<Contract[]>(`/api/v1/contracts${query}`);
  }

  async getContract(id: string): Promise<ApiResponse<Contract>> {
    return this.request<Contract>(`/api/v1/contracts/${id}`);
  }

  async addContract(data: {
    project_id: string;
    address: string;
    chain: Chain;
    name: string;
  }): Promise<ApiResponse<Contract>> {
    return this.request<Contract>('/api/v1/contracts', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async deleteContract(id: string): Promise<ApiResponse<void>> {
    return this.request(`/api/v1/contracts/${id}`, { method: 'DELETE' });
  }

  // ============================================================================
  // Analysis
  // ============================================================================

  async analyzeContract(
    source: string,
    compiler: string = '0.8.19',
    options: { security?: boolean; gas?: boolean; patterns?: boolean } = {}
  ): Promise<ApiResponse<AnalysisResult>> {
    return this.request<AnalysisResult>('/api/v1/analyze', {
      method: 'POST',
      body: JSON.stringify({
        source,
        compiler,
        options: {
          security: options.security ?? true,
          gas: options.gas ?? true,
          patterns: options.patterns ?? true,
        },
      }),
    });
  }

  async getAnalysis(id: string): Promise<ApiResponse<AnalysisResult>> {
    return this.request<AnalysisResult>(`/api/v1/analyze/${id}`);
  }

  async getContractAnalyses(contractId: string): Promise<ApiResponse<AnalysisResult[]>> {
    return this.request<AnalysisResult[]>(`/api/v1/contracts/${contractId}/analyses`);
  }

  // ============================================================================
  // Transaction Tracing
  // ============================================================================

  async traceTransaction(txHash: string, chain: Chain = 'ethereum'): Promise<ApiResponse<TransactionTrace>> {
    return this.request<TransactionTrace>(`/api/v1/trace/${txHash}?chain=${chain}`);
  }

  async getRecentTraces(): Promise<ApiResponse<TransactionTrace[]>> {
    return this.request<TransactionTrace[]>('/api/v1/traces');
  }

  // ============================================================================
  // Simulation
  // ============================================================================

  async simulateTransaction(params: {
    chain: Chain;
    from: string;
    to: string;
    data: string;
    value?: string;
    gas_limit?: number;
  }): Promise<ApiResponse<SimulationResult>> {
    return this.request<SimulationResult>('/api/v1/simulate', {
      method: 'POST',
      body: JSON.stringify(params),
    });
  }

  // ============================================================================
  // Monitors
  // ============================================================================

  async getMonitors(): Promise<ApiResponse<Monitor[]>> {
    return this.request<Monitor[]>('/api/v1/monitors');
  }

  async getMonitor(id: string): Promise<ApiResponse<Monitor>> {
    return this.request<Monitor>(`/api/v1/monitors/${id}`);
  }

  async createMonitor(data: {
    name: string;
    contract_address: string;
    chain: Chain;
    events: string[];
    conditions?: Record<string, unknown>;
    webhook_url?: string;
  }): Promise<ApiResponse<Monitor>> {
    return this.request<Monitor>('/api/v1/monitors', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateMonitor(id: string, data: Partial<Monitor>): Promise<ApiResponse<Monitor>> {
    return this.request<Monitor>(`/api/v1/monitors/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteMonitor(id: string): Promise<ApiResponse<void>> {
    return this.request(`/api/v1/monitors/${id}`, { method: 'DELETE' });
  }

  async getMonitorAlerts(monitorId: string): Promise<ApiResponse<MonitorAlert[]>> {
    return this.request<MonitorAlert[]>(`/api/v1/monitors/${monitorId}/alerts`);
  }

  // ============================================================================
  // Analytics
  // ============================================================================

  async getNetworkAnalytics(network: Chain): Promise<ApiResponse<NetworkAnalytics>> {
    return this.request<NetworkAnalytics>(`/api/v1/analytics/${network}/overview`);
  }

  async getGasPrice(network: Chain): Promise<ApiResponse<{ fast: number; standard: number; slow: number }>> {
    return this.request(`/api/v1/gas/${network}`);
  }

  async getTopContracts(network: Chain, limit: number = 10): Promise<ApiResponse<TopContract[]>> {
    return this.request<TopContract[]>(`/api/v1/analytics/${network}/top-contracts?limit=${limit}`);
  }

  // ============================================================================
  // Billing
  // ============================================================================

  async getSubscription(): Promise<ApiResponse<Subscription>> {
    return this.request<Subscription>('/api/v1/billing/subscription');
  }

  async createCheckoutSession(
    priceId: string,
    successUrl: string,
    cancelUrl: string
  ): Promise<ApiResponse<{ url: string }>> {
    return this.request('/api/v1/billing/checkout', {
      method: 'POST',
      body: JSON.stringify({
        price_id: priceId,
        success_url: successUrl,
        cancel_url: cancelUrl,
      }),
    });
  }

  async createBillingPortal(returnUrl: string): Promise<ApiResponse<{ url: string }>> {
    return this.request('/api/v1/billing/portal', {
      method: 'POST',
      body: JSON.stringify({ return_url: returnUrl }),
    });
  }

  async getInvoices(): Promise<ApiResponse<Invoice[]>> {
    return this.request<Invoice[]>('/api/v1/billing/invoices');
  }

  // ============================================================================
  // Health
  // ============================================================================

  async healthCheck(): Promise<ApiResponse<{ status: string; version: string }>> {
    return this.request('/health');
  }
}

// Singleton instance
export const api = new ApiClient();
