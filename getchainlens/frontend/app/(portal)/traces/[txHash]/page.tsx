'use client';

import { useEffect, useState } from 'react';
import { useParams, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import {
  ArrowLeft,
  CheckCircle2,
  XCircle,
  Fuel,
  Clock,
  ExternalLink,
  Copy,
  ChevronRight,
  ChevronDown,
  Code2,
  Activity,
  Layers,
} from 'lucide-react';
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardDescription,
  Button,
  Badge,
} from '@/components/ui';
import { api } from '@/lib/api';
import {
  formatAddress,
  formatGas,
  formatEth,
  getChainName,
  copyToClipboard,
} from '@/lib/utils';
import type { TransactionTrace, CallTrace, Chain } from '@/types';

// Mock trace data for demo
const mockTrace: TransactionTrace = {
  tx_hash: '0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef',
  chain: 'ethereum',
  block_number: 18500000,
  from: '0x742d35Cc6634C0532925a3b844Bc9e7595f8bE72',
  to: '0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45',
  value: '1000000000000000000',
  gas_limit: 300000,
  gas_used: 145000,
  gas_price: '35000000000',
  status: 'success',
  input: '0x5ae401dc...',
  calls: [
    {
      type: 'CALL',
      from: '0x742d35Cc6634C0532925a3b844Bc9e7595f8bE72',
      to: '0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45',
      value: '1000000000000000000',
      gas: 280000,
      gas_used: 140000,
      input: '0x5ae401dc...',
      output: '0x...',
      calls: [
        {
          type: 'DELEGATECALL',
          from: '0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45',
          to: '0x1111111254EEB25477B68fb85Ed929f73A960582',
          value: '0',
          gas: 250000,
          gas_used: 120000,
          input: '0x12aa3caf...',
          output: '0x...',
          calls: [
            {
              type: 'CALL',
              from: '0x1111111254EEB25477B68fb85Ed929f73A960582',
              to: '0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2',
              value: '1000000000000000000',
              gas: 50000,
              gas_used: 25000,
              input: '0xd0e30db0',
              output: '0x',
            },
            {
              type: 'CALL',
              from: '0x1111111254EEB25477B68fb85Ed929f73A960582',
              to: '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48',
              value: '0',
              gas: 80000,
              gas_used: 45000,
              input: '0xa9059cbb...',
              output: '0x0000000000000000000000000000000000000000000000000000000000000001',
            },
          ],
        },
      ],
    },
  ],
  logs: [
    {
      address: '0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2',
      topics: ['0xddf252ad...', '0x000...742d35', '0x000...1111111'],
      data: '0x0000000000000000000000000000000000000000000000000de0b6b3a7640000',
      decoded: {
        name: 'Transfer',
        args: {
          from: '0x742d35Cc6634C0532925a3b844Bc9e7595f8bE72',
          to: '0x1111111254EEB25477B68fb85Ed929f73A960582',
          value: '1000000000000000000',
        },
      },
    },
    {
      address: '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48',
      topics: ['0xddf252ad...', '0x000...1111111', '0x000...742d35'],
      data: '0x0000000000000000000000000000000000000000000000000000000005f5e100',
      decoded: {
        name: 'Transfer',
        args: {
          from: '0x1111111254EEB25477B68fb85Ed929f73A960582',
          to: '0x742d35Cc6634C0532925a3b844Bc9e7595f8bE72',
          value: '100000000',
        },
      },
    },
  ],
  state_changes: [
    {
      address: '0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2',
      slot: '0x...1',
      before: '0x0000...1000',
      after: '0x0000...0000',
    },
    {
      address: '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48',
      slot: '0x...2',
      before: '0x0000...0000',
      after: '0x0000...5f5e100',
    },
  ],
};

export default function TraceDetailPage() {
  const params = useParams();
  const searchParams = useSearchParams();
  const txHash = params.txHash as string;
  const chain = (searchParams.get('chain') || 'ethereum') as Chain;

  const [trace, setTrace] = useState<TransactionTrace | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<'calls' | 'logs' | 'state'>('calls');

  useEffect(() => {
    async function loadTrace() {
      const result = await api.traceTransaction(txHash, chain);
      if (result.data) {
        setTrace(result.data);
      } else {
        // Use mock data for demo
        setTrace({ ...mockTrace, tx_hash: txHash, chain });
      }
      setIsLoading(false);
    }
    loadTrace();
  }, [txHash, chain]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-accent-cyan border-t-transparent" />
      </div>
    );
  }

  if (!trace) {
    return (
      <div className="text-center py-12">
        <h2 className="text-xl font-semibold text-white">Transaction not found</h2>
        <Button variant="secondary" className="mt-4" asChild>
          <Link href="/traces">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Tracer
          </Link>
        </Button>
      </div>
    );
  }

  const isSuccess = trace.status === 'success';

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href="/traces">
            <ArrowLeft className="h-5 w-5" />
          </Link>
        </Button>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-bold text-white">Transaction Trace</h1>
            <Badge variant={isSuccess ? 'success' : 'destructive'}>
              {isSuccess ? (
                <>
                  <CheckCircle2 className="mr-1 h-3 w-3" />
                  Success
                </>
              ) : (
                <>
                  <XCircle className="mr-1 h-3 w-3" />
                  Failed
                </>
              )}
            </Badge>
            <Badge variant={chain}>{getChainName(chain)}</Badge>
          </div>
          <div className="mt-1 flex items-center gap-2 text-gray-400">
            <code className="font-mono text-sm">{formatAddress(txHash, 16)}</code>
            <button
              onClick={() => copyToClipboard(txHash)}
              className="p-1 hover:text-white transition-colors"
            >
              <Copy className="h-3.5 w-3.5" />
            </button>
            <a
              href={`https://etherscan.io/tx/${txHash}`}
              target="_blank"
              rel="noopener noreferrer"
              className="p-1 hover:text-white transition-colors"
            >
              <ExternalLink className="h-3.5 w-3.5" />
            </a>
          </div>
        </div>
      </div>

      {/* Overview */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardContent className="p-5">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/20">
                <Layers className="h-5 w-5 text-accent-cyan" />
              </div>
              <div>
                <p className="text-sm text-gray-400">Block</p>
                <p className="font-semibold text-white">{trace.block_number.toLocaleString()}</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-5">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-yellow/20">
                <Fuel className="h-5 w-5 text-accent-yellow" />
              </div>
              <div>
                <p className="text-sm text-gray-400">Gas Used</p>
                <p className="font-semibold text-white">
                  {formatGas(trace.gas_used)} / {formatGas(trace.gas_limit)}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-5">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-green/20">
                <Activity className="h-5 w-5 text-accent-green" />
              </div>
              <div>
                <p className="text-sm text-gray-400">Value</p>
                <p className="font-semibold text-white">{formatEth(trace.value)}</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-5">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-blue/20">
                <Clock className="h-5 w-5 text-accent-blue" />
              </div>
              <div>
                <p className="text-sm text-gray-400">Gas Price</p>
                <p className="font-semibold text-white">
                  {(parseInt(trace.gas_price) / 1e9).toFixed(2)} Gwei
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* From / To */}
      <Card>
        <CardContent className="p-5">
          <div className="flex flex-col sm:flex-row sm:items-center gap-4">
            <div className="flex-1">
              <p className="text-sm text-gray-400 mb-1">From</p>
              <div className="flex items-center gap-2">
                <code className="font-mono text-white">{formatAddress(trace.from, 12)}</code>
                <button
                  onClick={() => copyToClipboard(trace.from)}
                  className="text-gray-400 hover:text-white transition-colors"
                >
                  <Copy className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
            <ChevronRight className="h-5 w-5 text-gray-600 hidden sm:block" />
            <div className="flex-1">
              <p className="text-sm text-gray-400 mb-1">To</p>
              <div className="flex items-center gap-2">
                <code className="font-mono text-white">{formatAddress(trace.to, 12)}</code>
                <button
                  onClick={() => copyToClipboard(trace.to)}
                  className="text-gray-400 hover:text-white transition-colors"
                >
                  <Copy className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Tabs */}
      <div className="flex gap-2 border-b border-gray-800">
        <button
          onClick={() => setActiveTab('calls')}
          className={`px-4 py-2 text-sm font-medium transition-colors ${
            activeTab === 'calls'
              ? 'text-accent-cyan border-b-2 border-accent-cyan'
              : 'text-gray-400 hover:text-white'
          }`}
        >
          Call Trace ({trace.calls?.length || 0})
        </button>
        <button
          onClick={() => setActiveTab('logs')}
          className={`px-4 py-2 text-sm font-medium transition-colors ${
            activeTab === 'logs'
              ? 'text-accent-cyan border-b-2 border-accent-cyan'
              : 'text-gray-400 hover:text-white'
          }`}
        >
          Event Logs ({trace.logs?.length || 0})
        </button>
        <button
          onClick={() => setActiveTab('state')}
          className={`px-4 py-2 text-sm font-medium transition-colors ${
            activeTab === 'state'
              ? 'text-accent-cyan border-b-2 border-accent-cyan'
              : 'text-gray-400 hover:text-white'
          }`}
        >
          State Changes ({trace.state_changes?.length || 0})
        </button>
      </div>

      {/* Tab Content */}
      {activeTab === 'calls' && (
        <Card>
          <CardContent className="p-5">
            {trace.calls && trace.calls.length > 0 ? (
              <div className="space-y-2">
                {trace.calls.map((call, index) => (
                  <CallTraceItem key={index} call={call} depth={0} />
                ))}
              </div>
            ) : (
              <p className="text-center text-gray-400 py-8">No internal calls</p>
            )}
          </CardContent>
        </Card>
      )}

      {activeTab === 'logs' && (
        <Card>
          <CardContent className="p-5">
            {trace.logs && trace.logs.length > 0 ? (
              <div className="space-y-4">
                {trace.logs.map((log, index) => (
                  <div
                    key={index}
                    className="rounded-lg border border-gray-800 p-4"
                  >
                    <div className="flex items-start justify-between">
                      <div>
                        <Badge variant="secondary" className="mb-2">
                          {log.decoded?.name || 'Unknown Event'}
                        </Badge>
                        <p className="font-mono text-sm text-gray-400">
                          {formatAddress(log.address, 12)}
                        </p>
                      </div>
                      <span className="text-xs text-gray-500">#{index}</span>
                    </div>
                    {log.decoded?.args && (
                      <div className="mt-3 space-y-1">
                        {Object.entries(log.decoded.args).map(([key, value]) => (
                          <div key={key} className="flex gap-2 text-sm">
                            <span className="text-gray-500">{key}:</span>
                            <span className="font-mono text-white truncate">
                              {typeof value === 'string' && value.startsWith('0x')
                                ? formatAddress(value, 10)
                                : String(value)}
                            </span>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-center text-gray-400 py-8">No event logs</p>
            )}
          </CardContent>
        </Card>
      )}

      {activeTab === 'state' && (
        <Card>
          <CardContent className="p-5">
            {trace.state_changes && trace.state_changes.length > 0 ? (
              <div className="space-y-4">
                {trace.state_changes.map((change, index) => (
                  <div
                    key={index}
                    className="rounded-lg border border-gray-800 p-4"
                  >
                    <p className="font-mono text-sm text-gray-400 mb-2">
                      {formatAddress(change.address, 12)}
                    </p>
                    <div className="space-y-2 text-sm">
                      <div className="flex gap-2">
                        <span className="text-gray-500 w-12">Slot:</span>
                        <span className="font-mono text-white">{change.slot}</span>
                      </div>
                      <div className="flex gap-2">
                        <span className="text-gray-500 w-12">Before:</span>
                        <span className="font-mono text-severity-critical">{change.before}</span>
                      </div>
                      <div className="flex gap-2">
                        <span className="text-gray-500 w-12">After:</span>
                        <span className="font-mono text-accent-green">{change.after}</span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-center text-gray-400 py-8">No state changes</p>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function CallTraceItem({ call, depth }: { call: CallTrace; depth: number }) {
  const [isExpanded, setIsExpanded] = useState(true);
  const hasChildren = call.calls && call.calls.length > 0;

  const typeColors: Record<string, string> = {
    CALL: 'text-accent-cyan',
    DELEGATECALL: 'text-accent-yellow',
    STATICCALL: 'text-accent-blue',
    CREATE: 'text-accent-green',
    CREATE2: 'text-accent-green',
  };

  return (
    <div style={{ marginLeft: depth * 20 }}>
      <div
        className={`flex items-start gap-2 rounded-lg border border-gray-800 p-3 ${
          hasChildren ? 'cursor-pointer hover:border-gray-700' : ''
        }`}
        onClick={() => hasChildren && setIsExpanded(!isExpanded)}
      >
        {hasChildren ? (
          isExpanded ? (
            <ChevronDown className="h-4 w-4 text-gray-500 mt-0.5 shrink-0" />
          ) : (
            <ChevronRight className="h-4 w-4 text-gray-500 mt-0.5 shrink-0" />
          )
        ) : (
          <div className="w-4" />
        )}

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <Badge variant="outline" className={`text-xs ${typeColors[call.type] || 'text-gray-400'}`}>
              {call.type}
            </Badge>
            <span className="font-mono text-sm text-white">{formatAddress(call.to, 10)}</span>
            {call.value && call.value !== '0' && (
              <Badge variant="secondary" className="text-xs">
                {formatEth(call.value)}
              </Badge>
            )}
          </div>
          <div className="mt-1 flex items-center gap-3 text-xs text-gray-500">
            <span>Gas: {formatGas(call.gas_used)}</span>
            {call.input && (
              <span className="font-mono truncate max-w-[200px]">
                Input: {call.input.slice(0, 10)}...
              </span>
            )}
          </div>
        </div>
      </div>

      {hasChildren && isExpanded && (
        <div className="mt-2 space-y-2">
          {call.calls!.map((childCall, index) => (
            <CallTraceItem key={index} call={childCall} depth={depth + 1} />
          ))}
        </div>
      )}
    </div>
  );
}
