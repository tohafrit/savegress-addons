import axios, { AxiosInstance } from 'axios';

const API_BASE_URL = 'https://api.chainlens.dev';

export interface TransactionTrace {
    txHash: string;
    block: number;
    status: 'success' | 'reverted';
    gasUsed: number;
    error?: string;
    trace: TraceEntry[];
    stateChanges: StateChange[];
}

export interface TraceEntry {
    depth: number;
    type: 'CALL' | 'DELEGATECALL' | 'STATICCALL' | 'CREATE' | 'CREATE2';
    from: string;
    to: string;
    function?: string;
    input: string;
    output?: string;
    gasUsed: number;
    error?: string;
    sourceLocation?: {
        file: string;
        line: number;
        function: string;
    };
}

export interface StateChange {
    address: string;
    slot: string;
    before: string;
    after: string;
}

export interface AnalysisResult {
    id: string;
    status: 'pending' | 'completed' | 'failed';
    issues: SecurityIssue[];
    gasEstimates: Record<string, { min: number; max: number }>;
    score: {
        security: number;
        gasEfficiency: number;
        codeQuality: number;
    };
}

export interface SecurityIssue {
    id: string;
    severity: 'critical' | 'high' | 'medium' | 'low' | 'info';
    type: string;
    line: number;
    column: number;
    message: string;
    suggestion?: string;
    codeSnippet?: string;
}

export interface ContractMonitor {
    id: string;
    contractAddress: string;
    chain: string;
    name: string;
    status: 'active' | 'paused';
    alerts: AlertConfig[];
}

export interface AlertConfig {
    type: 'revert' | 'gas_spike' | 'large_transfer' | 'ownership_change';
    threshold: number;
    window?: string;
    channels: ('email' | 'slack' | 'webhook')[];
}

export class ApiClient {
    private client: AxiosInstance;
    private apiKey: string;

    constructor(apiKey: string) {
        this.apiKey = apiKey;
        this.client = axios.create({
            baseURL: API_BASE_URL,
            timeout: 30000,
            headers: {
                'Content-Type': 'application/json',
            },
        });

        // Add auth header if API key is set
        this.client.interceptors.request.use((config) => {
            if (this.apiKey) {
                config.headers.Authorization = `Bearer ${this.apiKey}`;
            }
            return config;
        });
    }

    setApiKey(apiKey: string): void {
        this.apiKey = apiKey;
    }

    // Transaction tracing
    async traceTransaction(txHash: string, chain: string = 'ethereum'): Promise<TransactionTrace> {
        const response = await this.client.get<TransactionTrace>(`/api/v1/trace/${txHash}`, {
            params: { chain },
        });
        return response.data;
    }

    // Contract analysis
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

    // Contract monitoring
    async createMonitor(
        contractAddress: string,
        chain: string,
        name: string,
        alerts: AlertConfig[]
    ): Promise<ContractMonitor> {
        const response = await this.client.post<ContractMonitor>('/api/v1/monitors', {
            contract_address: contractAddress,
            chain,
            name,
            alerts,
        });
        return response.data;
    }

    async getMonitors(): Promise<ContractMonitor[]> {
        const response = await this.client.get<ContractMonitor[]>('/api/v1/monitors');
        return response.data;
    }

    async deleteMonitor(monitorId: string): Promise<void> {
        await this.client.delete(`/api/v1/monitors/${monitorId}`);
    }

    // Simulation
    async simulateTransaction(
        contractAddress: string,
        chain: string,
        calldata: string,
        from?: string,
        value?: string,
        blockNumber?: number
    ): Promise<{
        success: boolean;
        gasUsed: number;
        returnData: string;
        trace: TraceEntry[];
        stateChanges: StateChange[];
    }> {
        const response = await this.client.post('/api/v1/simulate', {
            contract_address: contractAddress,
            chain,
            calldata,
            from,
            value,
            block_number: blockNumber,
        });
        return response.data;
    }

    // Fork simulation
    async createFork(chain: string, blockNumber?: number): Promise<{
        forkId: string;
        rpcUrl: string;
        blockNumber: number;
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

    // Health check
    async healthCheck(): Promise<{ status: string; version: string }> {
        const response = await this.client.get('/health');
        return response.data;
    }
}
