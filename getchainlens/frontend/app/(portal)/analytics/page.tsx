'use client';

import { useState } from 'react';
import Link from 'next/link';
import {
  BarChart3,
  TrendingUp,
  TrendingDown,
  Fuel,
  Activity,
  Clock,
  ArrowRight,
  Zap,
} from 'lucide-react';
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardDescription,
  Button,
  Badge,
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/components/ui';
import { formatGas, getChainName } from '@/lib/utils';
import type { Chain } from '@/types';

// Mock analytics data
const chainStats = {
  ethereum: {
    avgGasPrice: 35,
    avgBlockTime: 12.1,
    txCount24h: 1250000,
    pendingTxs: 145000,
    trend: 'up' as const,
    trendPercent: 5.2,
  },
  polygon: {
    avgGasPrice: 80,
    avgBlockTime: 2.1,
    txCount24h: 3500000,
    pendingTxs: 25000,
    trend: 'down' as const,
    trendPercent: 2.8,
  },
  arbitrum: {
    avgGasPrice: 0.1,
    avgBlockTime: 0.26,
    txCount24h: 980000,
    pendingTxs: 5000,
    trend: 'up' as const,
    trendPercent: 12.5,
  },
  optimism: {
    avgGasPrice: 0.001,
    avgBlockTime: 2.0,
    txCount24h: 450000,
    pendingTxs: 3000,
    trend: 'up' as const,
    trendPercent: 8.3,
  },
  base: {
    avgGasPrice: 0.005,
    avgBlockTime: 2.0,
    txCount24h: 750000,
    pendingTxs: 8000,
    trend: 'up' as const,
    trendPercent: 25.1,
  },
};

const popularContracts = [
  { name: 'Uniswap V3: Router', address: '0x68b3...B2Fd', chain: 'ethereum', txCount: 125000 },
  { name: 'USDT', address: '0xdAC1...36B0', chain: 'ethereum', txCount: 98000 },
  { name: 'Wrapped ETH', address: '0xC02a...4CC2', chain: 'ethereum', txCount: 87000 },
  { name: 'OpenSea: Seaport', address: '0x0000...0001', chain: 'ethereum', txCount: 65000 },
  { name: 'AAVE: Pool', address: '0x87870...4d3B', chain: 'ethereum', txCount: 45000 },
];

const gasHistory = [
  { time: '00:00', price: 28 },
  { time: '04:00', price: 22 },
  { time: '08:00', price: 45 },
  { time: '12:00', price: 52 },
  { time: '16:00', price: 38 },
  { time: '20:00', price: 35 },
  { time: 'Now', price: 35 },
];

export default function AnalyticsPage() {
  const [selectedChain, setSelectedChain] = useState<Chain>('ethereum');
  const stats = chainStats[selectedChain];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-white">Network Analytics</h1>
          <p className="mt-1 text-gray-400">
            Real-time blockchain statistics and trends
          </p>
        </div>
        <Select value={selectedChain} onValueChange={(v) => setSelectedChain(v as Chain)}>
          <SelectTrigger className="w-[180px]">
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
      </div>

      {/* Chain Stats */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Average Gas Price"
          value={`${stats.avgGasPrice} Gwei`}
          icon={<Fuel className="h-5 w-5" />}
          trend={stats.trend}
          trendPercent={stats.trendPercent}
          color="accent-yellow"
        />
        <StatCard
          title="Block Time"
          value={`${stats.avgBlockTime}s`}
          icon={<Clock className="h-5 w-5" />}
          color="accent-blue"
        />
        <StatCard
          title="24h Transactions"
          value={formatNumber(stats.txCount24h)}
          icon={<Activity className="h-5 w-5" />}
          color="accent-green"
        />
        <StatCard
          title="Pending Txs"
          value={formatNumber(stats.pendingTxs)}
          icon={<Zap className="h-5 w-5" />}
          color="accent-orange"
        />
      </div>

      {/* Chain Selector */}
      <div className="grid gap-4 sm:grid-cols-5">
        {(Object.keys(chainStats) as Chain[]).map((chain) => {
          const isSelected = chain === selectedChain;
          const data = chainStats[chain];
          return (
            <button
              key={chain}
              onClick={() => setSelectedChain(chain)}
              className={`rounded-xl border p-4 text-left transition-all ${
                isSelected
                  ? 'border-accent-cyan bg-accent-cyan/10'
                  : 'border-gray-800 bg-dark-card hover:border-gray-700'
              }`}
            >
              <div className="flex items-center gap-2">
                <Badge variant={chain} className="text-xs">
                  {getChainName(chain)}
                </Badge>
                {data.trend === 'up' ? (
                  <TrendingUp className="h-3 w-3 text-accent-green" />
                ) : (
                  <TrendingDown className="h-3 w-3 text-severity-critical" />
                )}
              </div>
              <p className="mt-2 text-lg font-semibold text-white">
                {data.avgGasPrice} Gwei
              </p>
              <p className="text-xs text-gray-500">
                {formatNumber(data.txCount24h)} txs/24h
              </p>
            </button>
          );
        })}
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Gas Price Chart */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Fuel className="h-5 w-5 text-accent-yellow" />
              Gas Price History
            </CardTitle>
            <CardDescription>
              24-hour gas price trend for {getChainName(selectedChain)}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-64 flex items-end justify-between gap-2">
              {gasHistory.map((point, index) => {
                const height = (point.price / 60) * 100;
                return (
                  <div key={index} className="flex-1 flex flex-col items-center gap-2">
                    <div
                      className="w-full rounded-t-lg bg-gradient-to-t from-accent-yellow/30 to-accent-yellow transition-all hover:from-accent-yellow/50 hover:to-accent-yellow"
                      style={{ height: `${height}%` }}
                    />
                    <span className="text-xs text-gray-500">{point.time}</span>
                  </div>
                );
              })}
            </div>
            <div className="mt-4 flex items-center justify-between text-sm">
              <span className="text-gray-400">Min: 22 Gwei</span>
              <span className="text-gray-400">Max: 52 Gwei</span>
              <span className="text-white font-medium">Current: {stats.avgGasPrice} Gwei</span>
            </div>
          </CardContent>
        </Card>

        {/* Popular Contracts */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2">
                <BarChart3 className="h-5 w-5 text-accent-cyan" />
                Popular Contracts
              </CardTitle>
              <Button variant="ghost" size="sm" asChild>
                <Link href={`/analytics/${selectedChain}/contracts`}>
                  View all <ArrowRight className="ml-1 h-3 w-3" />
                </Link>
              </Button>
            </div>
            <CardDescription>
              Most active contracts in the last 24 hours
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {popularContracts.map((contract, index) => (
                <div
                  key={index}
                  className="flex items-center justify-between rounded-lg border border-gray-800 p-3 hover:border-gray-700 transition-colors"
                >
                  <div className="flex items-center gap-3">
                    <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/20 text-sm font-semibold text-accent-cyan">
                      {index + 1}
                    </span>
                    <div>
                      <p className="text-sm font-medium text-white">{contract.name}</p>
                      <p className="text-xs text-gray-500 font-mono">{contract.address}</p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="text-sm font-semibold text-white">
                      {formatNumber(contract.txCount)}
                    </p>
                    <p className="text-xs text-gray-500">transactions</p>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Network Health */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Activity className="h-5 w-5 text-accent-green" />
            Network Health
          </CardTitle>
          <CardDescription>
            Current network status and performance metrics
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-3">
            <HealthMetric
              label="Network Status"
              value="Healthy"
              status="good"
              description="All systems operational"
            />
            <HealthMetric
              label="RPC Latency"
              value="45ms"
              status="good"
              description="Average response time"
            />
            <HealthMetric
              label="Block Finality"
              value={selectedChain === 'ethereum' ? '~15 min' : '~2 min'}
              status="good"
              description="Time to finality"
            />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function StatCard({
  title,
  value,
  icon,
  trend,
  trendPercent,
  color,
}: {
  title: string;
  value: string;
  icon: React.ReactNode;
  trend?: 'up' | 'down';
  trendPercent?: number;
  color: string;
}) {
  return (
    <Card>
      <CardContent className="p-5">
        <div className="flex items-center justify-between">
          <div className={`flex h-10 w-10 items-center justify-center rounded-lg bg-${color}/20`}>
            <div className={`text-${color}`}>{icon}</div>
          </div>
          {trend && trendPercent && (
            <div
              className={`flex items-center gap-1 text-sm ${
                trend === 'up' ? 'text-accent-green' : 'text-severity-critical'
              }`}
            >
              {trend === 'up' ? (
                <TrendingUp className="h-4 w-4" />
              ) : (
                <TrendingDown className="h-4 w-4" />
              )}
              {trendPercent}%
            </div>
          )}
        </div>
        <p className="mt-3 text-2xl font-bold text-white">{value}</p>
        <p className="mt-1 text-sm text-gray-400">{title}</p>
      </CardContent>
    </Card>
  );
}

function HealthMetric({
  label,
  value,
  status,
  description,
}: {
  label: string;
  value: string;
  status: 'good' | 'warning' | 'critical';
  description: string;
}) {
  const statusColors = {
    good: 'bg-accent-green',
    warning: 'bg-accent-yellow',
    critical: 'bg-severity-critical',
  };

  return (
    <div className="rounded-lg border border-gray-800 p-4">
      <div className="flex items-center gap-2">
        <div className={`h-2 w-2 rounded-full ${statusColors[status]}`} />
        <span className="text-sm text-gray-400">{label}</span>
      </div>
      <p className="mt-2 text-xl font-semibold text-white">{value}</p>
      <p className="mt-1 text-xs text-gray-500">{description}</p>
    </div>
  );
}

function formatNumber(num: number): string {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(1) + 'M';
  }
  if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'K';
  }
  return num.toString();
}
