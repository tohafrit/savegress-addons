'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  Search,
  ArrowRight,
  History,
  ExternalLink,
  CheckCircle2,
  XCircle,
} from 'lucide-react';
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardDescription,
  Button,
  Input,
  Badge,
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/components/ui';
import { formatTxHash, formatTimeAgo, getChainName, formatGas, formatEth } from '@/lib/utils';
import type { Chain, TransactionTrace } from '@/types';

// Mock recent traces for demo
const mockRecentTraces: Partial<TransactionTrace>[] = [
  {
    tx_hash: '0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef',
    chain: 'ethereum',
    status: 'success',
    gas_used: 145000,
    value: '1000000000000000000',
  },
  {
    tx_hash: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
    chain: 'polygon',
    status: 'failed',
    gas_used: 21000,
    value: '0',
  },
];

export default function TracesPage() {
  const router = useRouter();
  const [txHash, setTxHash] = useState('');
  const [chain, setChain] = useState<Chain>('ethereum');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleTrace = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!txHash.match(/^0x[a-fA-F0-9]{64}$/)) {
      setError('Please enter a valid transaction hash (0x followed by 64 hex characters)');
      return;
    }

    setIsLoading(true);
    setError(null);

    // Navigate to trace detail page
    router.push(`/traces/${txHash}?chain=${chain}`);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-white">Transaction Tracer</h1>
        <p className="mt-1 text-gray-400">
          Debug and analyze transaction execution with detailed call traces
        </p>
      </div>

      {/* Search form */}
      <Card variant="gradient">
        <CardContent className="p-6">
          <form onSubmit={handleTrace} className="space-y-4">
            <div className="flex flex-col sm:flex-row gap-4">
              <div className="flex-1">
                <Input
                  value={txHash}
                  onChange={(e) => {
                    setTxHash(e.target.value);
                    setError(null);
                  }}
                  placeholder="Enter transaction hash (0x...)"
                  icon={<Search className="h-4 w-4" />}
                  error={error || undefined}
                  className="font-mono"
                />
              </div>
              <Select value={chain} onValueChange={(v) => setChain(v as Chain)}>
                <SelectTrigger className="w-full sm:w-[180px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="ethereum">Ethereum</SelectItem>
                  <SelectItem value="polygon">Polygon</SelectItem>
                  <SelectItem value="arbitrum">Arbitrum</SelectItem>
                  <SelectItem value="optimism">Optimism</SelectItem>
                  <SelectItem value="base">Base</SelectItem>
                </SelectContent>
              </Select>
              <Button type="submit" isLoading={isLoading} className="sm:w-auto">
                Trace
                <ArrowRight className="ml-2 h-4 w-4" />
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      {/* Features */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Card>
          <CardContent className="p-5">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-cyan/20 mb-3">
              <Search className="h-5 w-5 text-accent-cyan" />
            </div>
            <h3 className="font-semibold text-white">Call Trace</h3>
            <p className="mt-1 text-sm text-gray-400">
              View the complete call tree with all internal transactions
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-5">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-green/20 mb-3">
              <CheckCircle2 className="h-5 w-5 text-accent-green" />
            </div>
            <h3 className="font-semibold text-white">State Changes</h3>
            <p className="mt-1 text-sm text-gray-400">
              See all storage slot changes during execution
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-5">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-orange/20 mb-3">
              <History className="h-5 w-5 text-accent-orange" />
            </div>
            <h3 className="font-semibold text-white">Event Logs</h3>
            <p className="mt-1 text-sm text-gray-400">
              Decoded event logs with parameter values
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Recent traces */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <History className="h-5 w-5 text-gray-400" />
            Recent Traces
          </CardTitle>
          <CardDescription>Your recently traced transactions</CardDescription>
        </CardHeader>
        <CardContent>
          {mockRecentTraces.length > 0 ? (
            <div className="space-y-3">
              {mockRecentTraces.map((trace, index) => (
                <div
                  key={index}
                  className="flex items-center justify-between rounded-lg border border-gray-800 p-4 hover:border-gray-700 transition-colors cursor-pointer"
                  onClick={() =>
                    router.push(`/traces/${trace.tx_hash}?chain=${trace.chain}`)
                  }
                >
                  <div className="flex items-center gap-4">
                    <div
                      className={`flex h-10 w-10 items-center justify-center rounded-lg ${
                        trace.status === 'success'
                          ? 'bg-accent-green/20'
                          : 'bg-severity-critical/20'
                      }`}
                    >
                      {trace.status === 'success' ? (
                        <CheckCircle2 className="h-5 w-5 text-accent-green" />
                      ) : (
                        <XCircle className="h-5 w-5 text-severity-critical" />
                      )}
                    </div>
                    <div>
                      <p className="font-mono text-sm text-white">
                        {formatTxHash(trace.tx_hash || '')}
                      </p>
                      <div className="flex items-center gap-2 mt-1">
                        <Badge variant={trace.chain as Chain} className="text-xs">
                          {getChainName(trace.chain || 'ethereum')}
                        </Badge>
                        <span className="text-xs text-gray-500">
                          {formatGas(trace.gas_used || 0)} gas
                        </span>
                        {trace.value && trace.value !== '0' && (
                          <span className="text-xs text-gray-500">
                            {formatEth(trace.value)}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                  <ArrowRight className="h-4 w-4 text-gray-500" />
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8">
              <History className="mx-auto h-10 w-10 text-gray-600" />
              <p className="mt-3 text-gray-400">No recent traces</p>
              <p className="text-sm text-gray-500">
                Enter a transaction hash above to get started
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Example hashes */}
      <Card>
        <CardHeader>
          <CardTitle>Example Transactions</CardTitle>
          <CardDescription>
            Try these transactions to see the tracer in action
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <ExampleTx
              name="Uniswap V3 Swap"
              hash="0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
              chain="ethereum"
              onSelect={(hash, chain) => {
                setTxHash(hash);
                setChain(chain);
              }}
            />
            <ExampleTx
              name="NFT Mint"
              hash="0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
              chain="ethereum"
              onSelect={(hash, chain) => {
                setTxHash(hash);
                setChain(chain);
              }}
            />
            <ExampleTx
              name="Failed Transaction"
              hash="0x9876543210fedcba9876543210fedcba9876543210fedcba9876543210fedcba"
              chain="polygon"
              onSelect={(hash, chain) => {
                setTxHash(hash);
                setChain(chain);
              }}
            />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function ExampleTx({
  name,
  hash,
  chain,
  onSelect,
}: {
  name: string;
  hash: string;
  chain: Chain;
  onSelect: (hash: string, chain: Chain) => void;
}) {
  return (
    <div className="flex items-center justify-between rounded-lg border border-gray-800 p-3 hover:border-gray-700 transition-colors">
      <div>
        <p className="text-sm font-medium text-white">{name}</p>
        <div className="flex items-center gap-2 mt-1">
          <code className="text-xs text-gray-400 font-mono">{formatTxHash(hash)}</code>
          <Badge variant={chain} className="text-xs">
            {getChainName(chain)}
          </Badge>
        </div>
      </div>
      <Button variant="secondary" size="sm" onClick={() => onSelect(hash, chain)}>
        Use
      </Button>
    </div>
  );
}
