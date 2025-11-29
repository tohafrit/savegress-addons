// User & Auth Types
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

// Project & Contract Types
export interface Project {
  id: string;
  user_id: string;
  name: string;
  description: string;
  contracts_count?: number;
  created_at: string;
  updated_at: string;
}

export interface Contract {
  id: string;
  project_id: string;
  user_id: string;
  address: string;
  chain: Chain;
  name: string;
  abi?: string;
  source_code?: string;
  is_verified: boolean;
  last_analysis?: AnalysisResult;
  created_at: string;
  updated_at: string;
}

// Analysis Types
export interface AnalysisResult {
  id: string;
  source_hash: string;
  status: 'pending' | 'completed' | 'failed';
  issues: SecurityIssue[];
  gas_estimates: Record<string, GasEstimate>;
  score: AnalysisScore;
  duration_ms: number;
  created_at?: string;
}

export interface AnalysisScore {
  security: number;
  gas_efficiency: number;
  code_quality: number;
  overall?: number;
}

export interface SecurityIssue {
  id: string;
  type: string;
  severity: Severity;
  line: number;
  column: number;
  end_line?: number;
  end_column?: number;
  message: string;
  suggestion?: string;
  code?: string;
  references?: string[];
}

export interface GasEstimate {
  function_name: string;
  min: number;
  max: number;
  typical: number;
  level: 'low' | 'medium' | 'high';
  suggestions?: string[];
}

export type Severity = 'critical' | 'high' | 'medium' | 'low' | 'info';

// Transaction Tracing Types
export interface TransactionTrace {
  tx_hash: string;
  chain: Chain;
  block_number: number;
  from: string;
  to: string;
  value: string;
  gas_limit: number;
  gas_used: number;
  gas_price: string;
  status: 'success' | 'failed';
  input: string;
  error?: string;
  calls?: CallTrace[];
  logs?: EventLog[];
  state_changes?: StateChange[];
  gas_breakdown?: GasBreakdown;
}

export interface CallTrace {
  type: 'CALL' | 'DELEGATECALL' | 'STATICCALL' | 'CREATE' | 'CREATE2' | 'SELFDESTRUCT';
  from: string;
  to: string;
  value: string;
  gas: number;
  gas_used: number;
  input: string;
  output?: string;
  error?: string;
  calls?: CallTrace[];
}

export interface InternalCall {
  type: 'CALL' | 'DELEGATECALL' | 'STATICCALL' | 'CREATE' | 'CREATE2' | 'SELFDESTRUCT';
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
    args?: Record<string, unknown>;
    params?: Record<string, unknown>;
  };
  log_index?: number;
}

export interface StateChange {
  address: string;
  slot: string;
  before: string;
  after: string;
  old_value?: string;
  new_value?: string;
}

export interface GasBreakdown {
  intrinsic: number;
  execution: number;
  storage: number;
  refund: number;
  by_operation: Record<string, number>;
}

// Simulation Types
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

// Monitor Types
export interface ContractMonitor {
  id: string;
  user_id: string;
  contract_id: string;
  contract?: Contract;
  name: string;
  event_filters: string[];
  webhook_url: string;
  is_active: boolean;
  last_triggered_at?: string;
  trigger_count: number;
  created_at: string;
  updated_at: string;
}

export interface Monitor {
  id: string;
  user_id: string;
  name: string;
  contract_address: string;
  chain: Chain;
  events: string[];
  conditions: Record<string, unknown>;
  webhook_url?: string;
  status: 'active' | 'paused';
  created_at: string;
  updated_at: string;
}

export interface MonitorAlert {
  id: string;
  monitor_id: string;
  event_name: string;
  transaction_hash: string;
  block_number: number;
  data: Record<string, unknown>;
  triggered_at: string;
  delivered: boolean;
}

// Analytics Types
export interface NetworkAnalytics {
  network: Chain;
  overview: NetworkOverview;
  charts: NetworkCharts;
}

export interface NetworkOverview {
  total_transactions: number;
  total_contracts: number;
  total_value_transferred: string;
  avg_gas_price: number;
  active_addresses: number;
}

export interface NetworkCharts {
  transactions_per_day: TimeSeriesData[];
  gas_price_history: TimeSeriesData[];
  contract_deployments: TimeSeriesData[];
}

export interface TimeSeriesData {
  timestamp: string;
  value: number;
}

export interface TopContract {
  address: string;
  name?: string;
  transactions: number;
  gas_used: number;
  chain: Chain;
}

// Billing Types
export interface Subscription {
  id: string;
  status: 'active' | 'canceled' | 'past_due' | 'trialing';
  plan: 'free' | 'pro' | 'enterprise';
  current_period_start: string;
  current_period_end: string;
  cancel_at_period_end: boolean;
}

export interface Invoice {
  id: string;
  amount: number;
  currency: string;
  status: 'paid' | 'open' | 'void' | 'uncollectible';
  invoice_url: string;
  created_at: string;
}

export interface UsageStats {
  api_calls_used: number;
  api_calls_limit: number;
  analyses_count: number;
  traces_count: number;
  contracts_count: number;
  period_end: string;
}

// Chain Types
export type Chain = 'ethereum' | 'polygon' | 'arbitrum' | 'optimism' | 'base';

export const CHAINS: { id: Chain; name: string; color: string }[] = [
  { id: 'ethereum', name: 'Ethereum', color: '#627EEA' },
  { id: 'polygon', name: 'Polygon', color: '#8247E5' },
  { id: 'arbitrum', name: 'Arbitrum', color: '#28A0F0' },
  { id: 'optimism', name: 'Optimism', color: '#FF0420' },
  { id: 'base', name: 'Base', color: '#0052FF' },
];

// API Response Types
export interface ApiResponse<T> {
  data?: T;
  error?: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

// Dashboard Types
export interface DashboardStats {
  total_contracts: number;
  total_analyses: number;
  total_issues: number;
  issues_by_severity: Record<Severity, number>;
  recent_analyses: AnalysisResult[];
  top_issues: SecurityIssue[];
}

// Form Types
export interface LoginFormData {
  email: string;
  password: string;
  remember?: boolean;
}

export interface RegisterFormData {
  email: string;
  password: string;
  name: string;
}

export interface ContractFormData {
  name: string;
  address: string;
  chain: Chain;
  project_id: string;
}

export interface MonitorFormData {
  name: string;
  contract_id: string;
  event_filters: string[];
  webhook_url: string;
}
