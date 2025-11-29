'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import {
  ArrowLeft,
  BarChart3,
  TrendingUp,
  TrendingDown,
  Fuel,
  Activity,
  Clock,
  Zap,
  FileCode2,
  RefreshCw,
  ExternalLink,
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
import { getChainName, formatAddress } from '@/lib/utils';
import type { Chain, NetworkAnalytics, TopContract, TimeSeriesData } from '@/types';

// Mock data for demonstration
const mockOverview = {
  ethereum: {
    total_transactions: 2150000000,
    total_contracts: 45000000,
    total_value_transferred: '125000000',
    avg_gas_price: 35,
    active_addresses: 850000,
  },
  polygon: {
    total_transactions: 3500000000,
    total_contracts: 12000000,
    total_value_transferred: '8500000',
    avg_gas_price: 80,
    active_addresses: 450000,
  },
  arbitrum: {
    total_transactions: 980000000,
    total_contracts: 5500000,
    total_value_transferred: '35000000',
    avg_gas_price: 0.1,
    active_addresses: 320000,
  },
  optimism: {
    total_transactions: 650000000,
    total_contracts: 3200000,
    total_value_transferred: '18000000',
    avg_gas_price: 0.001,
    active_addresses: 180000,
  },
  base: {
    total_transactions: 450000000,
    total_contracts: 1800000,
    total_value_transferred: '12000000',
    avg_gas_price: 0.005,
    active_addresses: 250000,
  },
};

const mockTopContracts: Record<Chain, TopContract[]> = {
  ethereum: [
    { address: '0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D', name: 'Uniswap V2: Router', transactions: 2500000, gas_used: 450000000000, chain: 'ethereum' },
    { address: '0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45', name: 'Uniswap V3: Router 2', transactions: 1800000, gas_used: 380000000000, chain: 'ethereum' },
    { address: '0xdAC17F958D2ee523a2206206994597C13D831ec7', name: 'USDT', transactions: 1500000, gas_used: 95000000000, chain: 'ethereum' },
    { address: '0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2', name: 'Wrapped Ether', transactions: 1200000, gas_used: 75000000000, chain: 'ethereum' },
    { address: '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48', name: 'USDC', transactions: 1100000, gas_used: 68000000000, chain: 'ethereum' },
    { address: '0x00000000006c3852cbEf3e08E8dF289169EdE581', name: 'Seaport 1.1', transactions: 980000, gas_used: 285000000000, chain: 'ethereum' },
    { address: '0x1111111254EEB25477B68fb85Ed929f73A960582', name: '1inch V5', transactions: 750000, gas_used: 195000000000, chain: 'ethereum' },
    { address: '0x6B175474E89094C44Da98b954EescdeCB5B5B5B5B', name: 'DAI', transactions: 680000, gas_used: 42000000000, chain: 'ethereum' },
    { address: '0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2', name: 'Aave V3 Pool', transactions: 520000, gas_used: 165000000000, chain: 'ethereum' },
    { address: '0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD', name: 'Uniswap Universal Router', transactions: 480000, gas_used: 145000000000, chain: 'ethereum' },
  ],
  polygon: [
    { address: '0xa5E0829CaCEd8fFDD4De3c43696c57F7D7A678ff', name: 'QuickSwap Router', transactions: 3200000, gas_used: 520000000000, chain: 'polygon' },
    { address: '0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174', name: 'USDC (PoS)', transactions: 2800000, gas_used: 175000000000, chain: 'polygon' },
    { address: '0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270', name: 'WMATIC', transactions: 2100000, gas_used: 130000000000, chain: 'polygon' },
    { address: '0xc2132D05D31c914a87C6611C10748AEb04B58e8F', name: 'USDT (PoS)', transactions: 1900000, gas_used: 118000000000, chain: 'polygon' },
    { address: '0x1BFD67037B42Cf73acF2047067bd4F2C47D9BfD6', name: 'WBTC (PoS)', transactions: 850000, gas_used: 52000000000, chain: 'polygon' },
  ],
  arbitrum: [
    { address: '0x82aF49447D8a07e3bd95BD0d56f35241523fBab1', name: 'WETH', transactions: 1800000, gas_used: 95000000000, chain: 'arbitrum' },
    { address: '0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9', name: 'USDT', transactions: 1500000, gas_used: 85000000000, chain: 'arbitrum' },
    { address: '0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8', name: 'USDC', transactions: 1400000, gas_used: 78000000000, chain: 'arbitrum' },
    { address: '0xfc5A1A6EB076a2C7aD06eD22C90d7E710E35ad0a', name: 'GMX', transactions: 980000, gas_used: 245000000000, chain: 'arbitrum' },
    { address: '0x912CE59144191C1204E64559FE8253a0e49E6548', name: 'ARB', transactions: 850000, gas_used: 52000000000, chain: 'arbitrum' },
  ],
  optimism: [
    { address: '0x4200000000000000000000000000000000000006', name: 'WETH', transactions: 1200000, gas_used: 65000000000, chain: 'optimism' },
    { address: '0x7F5c764cBc14f9669B88837ca1490cCa17c31607', name: 'USDC', transactions: 980000, gas_used: 55000000000, chain: 'optimism' },
    { address: '0x4200000000000000000000000000000000000042', name: 'OP', transactions: 850000, gas_used: 48000000000, chain: 'optimism' },
    { address: '0x68f180fcCe6836688e9084f035309E29Bf0A2095', name: 'WBTC', transactions: 520000, gas_used: 28000000000, chain: 'optimism' },
    { address: '0x94b008aA00579c1307B0EF2c499aD98a8ce58e58', name: 'USDT', transactions: 480000, gas_used: 26000000000, chain: 'optimism' },
  ],
  base: [
    { address: '0x4200000000000000000000000000000000000006', name: 'WETH', transactions: 980000, gas_used: 52000000000, chain: 'base' },
    { address: '0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913', name: 'USDC', transactions: 850000, gas_used: 45000000000, chain: 'base' },
    { address: '0x2Ae3F1Ec7F1F5012CFEab0185bfc7aa3cf0DEc22', name: 'cbETH', transactions: 520000, gas_used: 28000000000, chain: 'base' },
    { address: '0xd9aAEc86B65D86f6A7B5B1b0c42FFA531710b6CA', name: 'USDbC', transactions: 480000, gas_used: 25000000000, chain: 'base' },
    { address: '0x50c5725949A6F0c72E6C4a641F24049A917DB0Cb', name: 'DAI', transactions: 320000, gas_used: 17000000000, chain: 'base' },
  ],
};

const mockGasHistory: TimeSeriesData[] = Array.from({ length: 24 }, (_, i) => ({
  timestamp: new Date(Date.now() - (23 - i) * 3600000).toISOString(),
  value: Math.floor(Math.random() * 40) + 20,
}));

const mockTxHistory: TimeSeriesData[] = Array.from({ length: 7 }, (_, i) => ({
  timestamp: new Date(Date.now() - (6 - i) * 86400000).toISOString(),
  value: Math.floor(Math.random() * 500000) + 1000000,
}));

export default function NetworkAnalyticsPage() {
  const params = useParams();
  const network = params.network as Chain;

  const [isLoading, setIsLoading] = useState(true);
  const [overview, setOverview] = useState(mockOverview[network] || mockOverview.ethereum);
  const [topContracts, setTopContracts] = useState<TopContract[]>(mockTopContracts[network] || mockTopContracts.ethereum);

  const validNetworks: Chain[] = ['ethereum', 'polygon', 'arbitrum', 'optimism', 'base'];
  const isValidNetwork = validNetworks.includes(network);

  useEffect(() => {
    async function loadData() {
      // In production, this would call the API
      // const [analyticsResult, contractsResult] = await Promise.all([
      //   api.getNetworkAnalytics(network),
      //   api.getTopContracts(network, 10),
      // ]);

      setOverview(mockOverview[network] || mockOverview.ethereum);
      setTopContracts(mockTopContracts[network] || mockTopContracts.ethereum);
      setIsLoading(false);
    }
    loadData();
  }, [network]);

  if (!isValidNetwork) {
    return (
      <div className="text-center py-12">
        <h2 className="text-xl font-semibold text-white">Network not supported</h2>
        <p className="mt-2 text-gray-400">Please select a valid network</p>
        <Button variant="secondary" className="mt-4" asChild>
          <Link href="/analytics">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Analytics
          </Link>
        </Button>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-accent-cyan border-t-transparent" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href="/analytics">
            <ArrowLeft className="h-5 w-5" />
          </Link>
        </Button>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-white">{getChainName(network)} Analytics</h1>
            <Badge variant={network}>{network}</Badge>
          </div>
          <p className="mt-1 text-gray-400">
            Detailed network statistics and metrics
          </p>
        </div>
        <Button variant="secondary" onClick={() => window.location.reload()}>
          <RefreshCw className="mr-2 h-4 w-4" />
          Refresh
        </Button>
      </div>

      {/* Network Selector */}
      <div className="flex gap-2 overflow-x-auto pb-2">
        {validNetworks.map((n) => (
          <Button
            key={n}
            variant={n === network ? 'default' : 'secondary'}
            size="sm"
            asChild
          >
            <Link href={`/analytics/${n}`}>
              {getChainName(n)}
            </Link>
          </Button>
        ))}
      </div>

      {/* Overview Stats */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
        <StatCard
          title="Total Transactions"
          value={formatLargeNumber(overview.total_transactions)}
          icon={<Activity className="h-5 w-5" />}
          color="accent-cyan"
        />
        <StatCard
          title="Total Contracts"
          value={formatLargeNumber(overview.total_contracts)}
          icon={<FileCode2 className="h-5 w-5" />}
          color="accent-blue"
        />
        <StatCard
          title="Avg Gas Price"
          value={`${overview.avg_gas_price} Gwei`}
          icon={<Fuel className="h-5 w-5" />}
          color="accent-yellow"
        />
        <StatCard
          title="Active Addresses"
          value={formatLargeNumber(overview.active_addresses)}
          icon={<Zap className="h-5 w-5" />}
          color="accent-green"
        />
        <StatCard
          title="Value Transferred"
          value={`${formatLargeNumber(parseInt(overview.total_value_transferred))} ETH`}
          icon={<TrendingUp className="h-5 w-5" />}
          color="accent-orange"
        />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Gas Price Chart */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Fuel className="h-5 w-5 text-accent-yellow" />
              Gas Price (24h)
            </CardTitle>
            <CardDescription>
              Hourly average gas price in Gwei
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-64 flex items-end justify-between gap-1">
              {mockGasHistory.map((point, index) => {
                const height = (point.value / 70) * 100;
                const isLatest = index === mockGasHistory.length - 1;
                return (
                  <div key={index} className="flex-1 flex flex-col items-center gap-1">
                    <span className="text-xs text-gray-500 opacity-0 group-hover:opacity-100">
                      {point.value}
                    </span>
                    <div
                      className={`w-full rounded-t transition-all ${
                        isLatest
                          ? 'bg-accent-yellow'
                          : 'bg-accent-yellow/30 hover:bg-accent-yellow/50'
                      }`}
                      style={{ height: `${height}%` }}
                    />
                  </div>
                );
              })}
            </div>
            <div className="mt-4 flex items-center justify-between text-sm">
              <span className="text-gray-400">24 hours ago</span>
              <span className="text-white font-medium">Current: {overview.avg_gas_price} Gwei</span>
              <span className="text-gray-400">Now</span>
            </div>
          </CardContent>
        </Card>

        {/* Transaction Volume Chart */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <BarChart3 className="h-5 w-5 text-accent-cyan" />
              Transaction Volume (7d)
            </CardTitle>
            <CardDescription>
              Daily transaction count
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-64 flex items-end justify-between gap-2">
              {mockTxHistory.map((point, index) => {
                const height = (point.value / 1800000) * 100;
                const isLatest = index === mockTxHistory.length - 1;
                const date = new Date(point.timestamp);
                const dayName = date.toLocaleDateString('en', { weekday: 'short' });
                return (
                  <div key={index} className="flex-1 flex flex-col items-center gap-2">
                    <div
                      className={`w-full rounded-t transition-all ${
                        isLatest
                          ? 'bg-accent-cyan'
                          : 'bg-accent-cyan/30 hover:bg-accent-cyan/50'
                      }`}
                      style={{ height: `${height}%` }}
                    />
                    <span className="text-xs text-gray-500">{dayName}</span>
                  </div>
                );
              })}
            </div>
            <div className="mt-4 flex items-center justify-center text-sm">
              <span className="text-white font-medium">
                {formatLargeNumber(mockTxHistory.reduce((sum, d) => sum + d.value, 0))} total transactions this week
              </span>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Top Contracts */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileCode2 className="h-5 w-5 text-accent-cyan" />
            Top Contracts by Transaction Volume
          </CardTitle>
          <CardDescription>
            Most active contracts on {getChainName(network)}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-800">
                  <th className="text-left text-sm font-medium text-gray-400 pb-3 pr-4">#</th>
                  <th className="text-left text-sm font-medium text-gray-400 pb-3 pr-4">Contract</th>
                  <th className="text-left text-sm font-medium text-gray-400 pb-3 pr-4">Address</th>
                  <th className="text-right text-sm font-medium text-gray-400 pb-3 pr-4">Transactions</th>
                  <th className="text-right text-sm font-medium text-gray-400 pb-3">Gas Used</th>
                  <th className="text-right text-sm font-medium text-gray-400 pb-3"></th>
                </tr>
              </thead>
              <tbody>
                {topContracts.map((contract, index) => (
                  <tr key={contract.address} className="border-b border-gray-800/50 hover:bg-dark-card-hover transition-colors">
                    <td className="py-4 pr-4">
                      <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/20 text-sm font-semibold text-accent-cyan">
                        {index + 1}
                      </span>
                    </td>
                    <td className="py-4 pr-4">
                      <p className="font-medium text-white">{contract.name || 'Unknown'}</p>
                    </td>
                    <td className="py-4 pr-4">
                      <code className="text-sm text-gray-400 font-mono">
                        {formatAddress(contract.address, 6)}
                      </code>
                    </td>
                    <td className="py-4 pr-4 text-right">
                      <span className="font-medium text-white">
                        {formatLargeNumber(contract.transactions)}
                      </span>
                    </td>
                    <td className="py-4 text-right">
                      <span className="text-gray-400">
                        {formatLargeNumber(contract.gas_used)}
                      </span>
                    </td>
                    <td className="py-4 text-right pl-4">
                      <a
                        href={getExplorerUrl(network, contract.address)}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-accent-cyan hover:underline"
                      >
                        <ExternalLink className="h-4 w-4" />
                      </a>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      {/* Network Info */}
      <div className="grid gap-6 lg:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle>Network Type</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold text-white">
              {network === 'ethereum' ? 'L1 Mainnet' : 'L2 Rollup'}
            </p>
            <p className="mt-1 text-gray-400">
              {network === 'ethereum'
                ? 'Ethereum Mainnet - Proof of Stake'
                : network === 'polygon'
                ? 'Polygon PoS Sidechain'
                : network === 'arbitrum'
                ? 'Optimistic Rollup on Ethereum'
                : network === 'optimism'
                ? 'Optimistic Rollup on Ethereum'
                : 'Optimistic Rollup on Ethereum'}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Block Time</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold text-white">
              {network === 'ethereum'
                ? '~12 sec'
                : network === 'polygon'
                ? '~2 sec'
                : network === 'arbitrum'
                ? '~0.25 sec'
                : '~2 sec'}
            </p>
            <p className="mt-1 text-gray-400">Average block production time</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Finality</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold text-white">
              {network === 'ethereum'
                ? '~15 min'
                : network === 'polygon'
                ? '~30 min'
                : '~7 days*'}
            </p>
            <p className="mt-1 text-gray-400">
              {network === 'ethereum' || network === 'polygon'
                ? 'Time to finality'
                : '*Challenge period for L2'}
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function StatCard({
  title,
  value,
  icon,
  color,
}: {
  title: string;
  value: string;
  icon: React.ReactNode;
  color: string;
}) {
  return (
    <Card>
      <CardContent className="p-5">
        <div className="flex items-center justify-between">
          <div className={`flex h-10 w-10 items-center justify-center rounded-lg bg-${color}/20 text-${color}`}>
            {icon}
          </div>
        </div>
        <p className="mt-3 text-2xl font-bold text-white">{value}</p>
        <p className="mt-1 text-sm text-gray-400">{title}</p>
      </CardContent>
    </Card>
  );
}

function formatLargeNumber(num: number): string {
  if (num >= 1e12) {
    return (num / 1e12).toFixed(2) + 'T';
  }
  if (num >= 1e9) {
    return (num / 1e9).toFixed(2) + 'B';
  }
  if (num >= 1e6) {
    return (num / 1e6).toFixed(2) + 'M';
  }
  if (num >= 1e3) {
    return (num / 1e3).toFixed(1) + 'K';
  }
  return num.toString();
}

function getExplorerUrl(chain: Chain, address: string): string {
  const explorers: Record<Chain, string> = {
    ethereum: 'https://etherscan.io/address/',
    polygon: 'https://polygonscan.com/address/',
    arbitrum: 'https://arbiscan.io/address/',
    optimism: 'https://optimistic.etherscan.io/address/',
    base: 'https://basescan.org/address/',
  };
  return explorers[chain] + address;
}
