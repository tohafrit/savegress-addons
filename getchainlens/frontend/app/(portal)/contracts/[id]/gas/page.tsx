'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import {
  ArrowLeft,
  Fuel,
  TrendingUp,
  TrendingDown,
  AlertCircle,
  Lightbulb,
  ChevronDown,
  ChevronUp,
  DollarSign,
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
} from '@/components/ui';
import { api } from '@/lib/api';
import { formatGas, formatAddress, getChainName } from '@/lib/utils';
import type { Contract, AnalysisResult, GasEstimate } from '@/types';

export default function GasAnalysisPage() {
  const params = useParams();
  const contractId = params.id as string;

  const [contract, setContract] = useState<Contract | null>(null);
  const [analysis, setAnalysis] = useState<AnalysisResult | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [sortBy, setSortBy] = useState<'name' | 'gas' | 'level'>('gas');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');
  const [expandedFunction, setExpandedFunction] = useState<string | null>(null);

  useEffect(() => {
    async function loadData() {
      const result = await api.getContract(contractId);
      if (result.data) {
        setContract(result.data);
        if (result.data.last_analysis) {
          setAnalysis(result.data.last_analysis);
        }
      }
      setIsLoading(false);
    }
    loadData();
  }, [contractId]);

  const gasEstimates = analysis?.gas_estimates
    ? Object.values(analysis.gas_estimates)
    : [];

  const sortedEstimates = [...gasEstimates].sort((a, b) => {
    let comparison = 0;
    if (sortBy === 'name') {
      comparison = a.function_name.localeCompare(b.function_name);
    } else if (sortBy === 'gas') {
      comparison = a.typical - b.typical;
    } else if (sortBy === 'level') {
      const levelOrder = { high: 3, medium: 2, low: 1 };
      comparison = levelOrder[a.level] - levelOrder[b.level];
    }
    return sortOrder === 'asc' ? comparison : -comparison;
  });

  const totalGas = gasEstimates.reduce((sum, e) => sum + e.typical, 0);
  const avgGas = gasEstimates.length > 0 ? Math.round(totalGas / gasEstimates.length) : 0;
  const highGasCount = gasEstimates.filter((e) => e.level === 'high').length;
  const optimizableCount = gasEstimates.filter(
    (e) => e.suggestions && e.suggestions.length > 0
  ).length;

  // Mock gas price for cost calculation
  const gasPrice = 35; // Gwei
  const ethPrice = 2000; // USD

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-accent-cyan border-t-transparent" />
      </div>
    );
  }

  if (!contract) {
    return (
      <div className="text-center py-12">
        <h2 className="text-xl font-semibold text-white">Contract not found</h2>
        <Button variant="secondary" className="mt-4" asChild>
          <Link href="/contracts">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Contracts
          </Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href={`/contracts/${contractId}`}>
            <ArrowLeft className="h-5 w-5" />
          </Link>
        </Button>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-white">Gas Analysis</h1>
            <Badge variant={contract.chain}>{getChainName(contract.chain)}</Badge>
          </div>
          <p className="mt-1 text-gray-400">{contract.name}</p>
        </div>
      </div>

      {/* Overview Cards */}
      <div className="grid gap-4 sm:grid-cols-4">
        <Card>
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-yellow/20 text-accent-yellow">
                <Fuel className="h-5 w-5" />
              </div>
              <Badge variant="secondary">{gasEstimates.length} functions</Badge>
            </div>
            <p className="mt-3 text-2xl font-bold text-white">{formatGas(avgGas)}</p>
            <p className="mt-1 text-sm text-gray-400">Average Gas</p>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-severity-critical/20 text-severity-critical">
                <TrendingUp className="h-5 w-5" />
              </div>
            </div>
            <p className="mt-3 text-2xl font-bold text-white">{highGasCount}</p>
            <p className="mt-1 text-sm text-gray-400">High Gas Functions</p>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-green/20 text-accent-green">
                <Lightbulb className="h-5 w-5" />
              </div>
            </div>
            <p className="mt-3 text-2xl font-bold text-white">{optimizableCount}</p>
            <p className="mt-1 text-sm text-gray-400">Optimization Tips</p>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-blue/20 text-accent-blue">
                <DollarSign className="h-5 w-5" />
              </div>
            </div>
            <p className="mt-3 text-2xl font-bold text-white">
              ${((avgGas * gasPrice * ethPrice) / 1e9).toFixed(2)}
            </p>
            <p className="mt-1 text-sm text-gray-400">Avg Cost @ {gasPrice} Gwei</p>
          </CardContent>
        </Card>
      </div>

      {/* Gas Price Info */}
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Zap className="h-5 w-5 text-accent-yellow" />
              <span className="text-gray-400">Current Gas Price:</span>
              <span className="font-semibold text-white">{gasPrice} Gwei</span>
              <span className="text-gray-500">•</span>
              <span className="text-gray-400">ETH Price:</span>
              <span className="font-semibold text-white">${ethPrice}</span>
            </div>
            <Link href="/analytics/ethereum" className="text-sm text-accent-cyan hover:underline">
              View network stats
            </Link>
          </div>
        </CardContent>
      </Card>

      {/* Function Gas Estimates */}
      {gasEstimates.length > 0 ? (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2">
                <Fuel className="h-5 w-5 text-accent-yellow" />
                Function Gas Estimates
              </CardTitle>
              <div className="flex items-center gap-2">
                <span className="text-sm text-gray-400">Sort by:</span>
                <select
                  value={sortBy}
                  onChange={(e) => setSortBy(e.target.value as 'name' | 'gas' | 'level')}
                  className="rounded-lg border border-gray-700 bg-dark-bg px-3 py-1.5 text-sm text-white focus:border-accent-cyan focus:outline-none"
                >
                  <option value="gas">Gas Usage</option>
                  <option value="name">Function Name</option>
                  <option value="level">Level</option>
                </select>
                <button
                  onClick={() => setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')}
                  className="p-1.5 rounded-lg border border-gray-700 hover:border-gray-600 transition-colors"
                >
                  {sortOrder === 'asc' ? (
                    <ChevronUp className="h-4 w-4 text-gray-400" />
                  ) : (
                    <ChevronDown className="h-4 w-4 text-gray-400" />
                  )}
                </button>
              </div>
            </div>
            <CardDescription>
              Detailed gas consumption for each function
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {sortedEstimates.map((estimate) => (
                <GasEstimateRow
                  key={estimate.function_name}
                  estimate={estimate}
                  gasPrice={gasPrice}
                  ethPrice={ethPrice}
                  isExpanded={expandedFunction === estimate.function_name}
                  onToggle={() =>
                    setExpandedFunction(
                      expandedFunction === estimate.function_name
                        ? null
                        : estimate.function_name
                    )
                  }
                />
              ))}
            </div>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="py-12 text-center">
            <Fuel className="mx-auto h-12 w-12 text-gray-600" />
            <h3 className="mt-4 text-lg font-medium text-white">No gas analysis available</h3>
            <p className="mt-2 text-gray-400">
              Run an analysis on the contract to see gas estimates
            </p>
            <Button variant="secondary" className="mt-4" asChild>
              <Link href={`/contracts/${contractId}`}>
                Go to Contract
              </Link>
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Gas Optimization Tips */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Lightbulb className="h-5 w-5 text-accent-green" />
            General Optimization Tips
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2">
            <TipCard
              title="Use calldata instead of memory"
              description="For external functions with array parameters, use calldata to save gas on copying data."
            />
            <TipCard
              title="Pack storage variables"
              description="Group smaller data types together to fit multiple variables in a single storage slot."
            />
            <TipCard
              title="Use unchecked for safe math"
              description="When overflow is impossible, use unchecked blocks to skip safety checks (Solidity 0.8+)."
            />
            <TipCard
              title="Cache storage reads"
              description="Read storage variables into memory once and reuse them to avoid multiple SLOAD operations."
            />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function GasEstimateRow({
  estimate,
  gasPrice,
  ethPrice,
  isExpanded,
  onToggle,
}: {
  estimate: GasEstimate;
  gasPrice: number;
  ethPrice: number;
  isExpanded: boolean;
  onToggle: () => void;
}) {
  const levelColors = {
    low: { bg: 'bg-accent-green/20', text: 'text-accent-green', border: 'border-accent-green/30' },
    medium: { bg: 'bg-accent-yellow/20', text: 'text-accent-yellow', border: 'border-accent-yellow/30' },
    high: { bg: 'bg-severity-critical/20', text: 'text-severity-critical', border: 'border-severity-critical/30' },
  };

  const colors = levelColors[estimate.level];
  const cost = ((estimate.typical * gasPrice * ethPrice) / 1e9).toFixed(4);

  return (
    <div className={`rounded-lg border ${colors.border} overflow-hidden`}>
      <button
        onClick={onToggle}
        className="w-full flex items-center justify-between p-4 hover:bg-dark-card-hover transition-colors"
      >
        <div className="flex items-center gap-4">
          <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${colors.bg}`}>
            <Fuel className={`h-5 w-5 ${colors.text}`} />
          </div>
          <div className="text-left">
            <p className="font-medium text-white font-mono">{estimate.function_name}()</p>
            <div className="flex items-center gap-2 mt-0.5">
              <Badge className={`${colors.bg} ${colors.text} border-none text-xs`}>
                {estimate.level}
              </Badge>
              <span className="text-xs text-gray-500">
                {formatGas(estimate.min)} - {formatGas(estimate.max)} gas
              </span>
            </div>
          </div>
        </div>
        <div className="flex items-center gap-6">
          <div className="text-right">
            <p className={`text-lg font-semibold ${colors.text}`}>
              ~{formatGas(estimate.typical)}
            </p>
            <p className="text-xs text-gray-500">${cost} USD</p>
          </div>
          {estimate.suggestions && estimate.suggestions.length > 0 && (
            <Badge variant="secondary" className="text-xs">
              {estimate.suggestions.length} tips
            </Badge>
          )}
          {isExpanded ? (
            <ChevronUp className="h-5 w-5 text-gray-400" />
          ) : (
            <ChevronDown className="h-5 w-5 text-gray-400" />
          )}
        </div>
      </button>

      {isExpanded && estimate.suggestions && estimate.suggestions.length > 0 && (
        <div className="border-t border-gray-800 p-4 bg-dark-bg/50">
          <p className="text-sm font-medium text-gray-300 mb-3">Optimization Suggestions:</p>
          <ul className="space-y-2">
            {estimate.suggestions.map((suggestion, index) => (
              <li key={index} className="flex items-start gap-2 text-sm">
                <Lightbulb className="h-4 w-4 text-accent-green mt-0.5 shrink-0" />
                <span className="text-gray-400">{suggestion}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

function TipCard({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div className="rounded-lg border border-gray-800 p-4">
      <h4 className="font-medium text-white">{title}</h4>
      <p className="mt-1 text-sm text-gray-400">{description}</p>
    </div>
  );
}
