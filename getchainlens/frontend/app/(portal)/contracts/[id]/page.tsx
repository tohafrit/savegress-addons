'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import {
  ArrowLeft,
  Shield,
  Fuel,
  Code2,
  AlertTriangle,
  CheckCircle2,
  ExternalLink,
  RefreshCw,
  Copy,
  ChevronRight,
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
  getChainName,
  getScoreColor,
  getScoreLabel,
  getSeverityBadgeClass,
  copyToClipboard,
} from '@/lib/utils';
import type { Contract, AnalysisResult, SecurityIssue, GasEstimate, Severity } from '@/types';

export default function ContractDetailPage() {
  const params = useParams();
  const contractId = params.id as string;

  const [contract, setContract] = useState<Contract | null>(null);
  const [analysis, setAnalysis] = useState<AnalysisResult | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isAnalyzing, setIsAnalyzing] = useState(false);

  useEffect(() => {
    async function loadContract() {
      const result = await api.getContract(contractId);
      if (result.data) {
        setContract(result.data);
        if (result.data.last_analysis) {
          setAnalysis(result.data.last_analysis);
        }
      }
      setIsLoading(false);
    }
    loadContract();
  }, [contractId]);

  const handleAnalyze = async () => {
    if (!contract?.source_code) return;

    setIsAnalyzing(true);
    const result = await api.analyzeContract(contract.source_code);
    if (result.data) {
      setAnalysis(result.data);
    }
    setIsAnalyzing(false);
  };

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

  const score = analysis?.score;
  const overallScore = score
    ? Math.round((score.security + score.gas_efficiency + score.code_quality) / 3)
    : null;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href="/contracts">
            <ArrowLeft className="h-5 w-5" />
          </Link>
        </Button>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-white">{contract.name}</h1>
            <Badge variant={contract.chain}>{getChainName(contract.chain)}</Badge>
            {contract.is_verified && (
              <Badge variant="success">
                <CheckCircle2 className="mr-1 h-3 w-3" />
                Verified
              </Badge>
            )}
          </div>
          <div className="mt-1 flex items-center gap-2 text-gray-400">
            <code className="font-mono text-sm">{formatAddress(contract.address, 10)}</code>
            <button
              onClick={() => copyToClipboard(contract.address)}
              className="p-1 hover:text-white transition-colors"
            >
              <Copy className="h-3.5 w-3.5" />
            </button>
            <a
              href={`https://etherscan.io/address/${contract.address}`}
              target="_blank"
              rel="noopener noreferrer"
              className="p-1 hover:text-white transition-colors"
            >
              <ExternalLink className="h-3.5 w-3.5" />
            </a>
          </div>
        </div>
        <Button onClick={handleAnalyze} isLoading={isAnalyzing}>
          <RefreshCw className="mr-2 h-4 w-4" />
          Run Analysis
        </Button>
      </div>

      {/* Score cards */}
      {analysis && score && (
        <div className="grid gap-4 sm:grid-cols-4">
          <ScoreCard
            title="Overall Score"
            score={overallScore || 0}
            icon={<Shield className="h-5 w-5" />}
          />
          <ScoreCard
            title="Security"
            score={score.security}
            icon={<Shield className="h-5 w-5" />}
          />
          <ScoreCard
            title="Gas Efficiency"
            score={score.gas_efficiency}
            icon={<Fuel className="h-5 w-5" />}
          />
          <ScoreCard
            title="Code Quality"
            score={score.code_quality}
            icon={<Code2 className="h-5 w-5" />}
          />
        </div>
      )}

      {/* Analysis results */}
      {analysis ? (
        <div className="grid gap-6 lg:grid-cols-2">
          {/* Security Issues */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="flex items-center gap-2">
                  <AlertTriangle className="h-5 w-5 text-accent-orange" />
                  Security Issues
                </CardTitle>
                <Link
                  href={`/contracts/${contractId}/security`}
                  className="text-sm text-accent-cyan hover:underline flex items-center gap-1"
                >
                  View all <ChevronRight className="h-3 w-3" />
                </Link>
              </div>
              <CardDescription>
                {analysis.issues.length} issues found
              </CardDescription>
            </CardHeader>
            <CardContent>
              <IssuesList issues={analysis.issues.slice(0, 5)} />
            </CardContent>
          </Card>

          {/* Gas Estimates */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="flex items-center gap-2">
                  <Fuel className="h-5 w-5 text-accent-yellow" />
                  Gas Estimates
                </CardTitle>
                <Link
                  href={`/contracts/${contractId}/gas`}
                  className="text-sm text-accent-cyan hover:underline flex items-center gap-1"
                >
                  View all <ChevronRight className="h-3 w-3" />
                </Link>
              </div>
              <CardDescription>
                Function-level gas analysis
              </CardDescription>
            </CardHeader>
            <CardContent>
              <GasEstimatesList
                estimates={Object.values(analysis.gas_estimates).slice(0, 5)}
              />
            </CardContent>
          </Card>
        </div>
      ) : (
        <Card>
          <CardContent className="py-12 text-center">
            <Shield className="mx-auto h-12 w-12 text-gray-600" />
            <h3 className="mt-4 text-lg font-medium text-white">No analysis yet</h3>
            <p className="mt-2 text-gray-400">
              Run an analysis to detect vulnerabilities and optimize gas usage
            </p>
            <Button className="mt-4" onClick={handleAnalyze} isLoading={isAnalyzing}>
              <Shield className="mr-2 h-4 w-4" />
              Run Analysis
            </Button>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function ScoreCard({
  title,
  score,
  icon,
}: {
  title: string;
  score: number;
  icon: React.ReactNode;
}) {
  const color = getScoreColor(score);
  const label = getScoreLabel(score);

  return (
    <Card>
      <CardContent className="p-5">
        <div className="flex items-center justify-between">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/20 text-accent-cyan">
            {icon}
          </div>
          <div
            className="flex h-14 w-14 items-center justify-center rounded-full"
            style={{
              background: `conic-gradient(${color} ${score}%, rgba(55, 65, 81, 0.5) ${score}%)`,
            }}
          >
            <div className="flex h-11 w-11 items-center justify-center rounded-full bg-dark-card">
              <span className="text-lg font-bold text-white">{score}</span>
            </div>
          </div>
        </div>
        <div className="mt-3">
          <p className="text-sm text-gray-400">{title}</p>
          <p className="text-sm font-medium" style={{ color }}>
            {label}
          </p>
        </div>
      </CardContent>
    </Card>
  );
}

function IssuesList({ issues }: { issues: SecurityIssue[] }) {
  if (issues.length === 0) {
    return (
      <div className="text-center py-6">
        <CheckCircle2 className="mx-auto h-10 w-10 text-accent-green" />
        <p className="mt-2 text-gray-400">No issues found</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {issues.map((issue, index) => (
        <div
          key={index}
          className="rounded-lg border border-gray-800 p-3 hover:border-gray-700 transition-colors"
        >
          <div className="flex items-start gap-3">
            <Badge className={getSeverityBadgeClass(issue.severity)}>
              {issue.severity}
            </Badge>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-white">{issue.type}</p>
              <p className="text-sm text-gray-400 line-clamp-2 mt-0.5">{issue.message}</p>
              <p className="text-xs text-gray-500 mt-1">Line {issue.line}</p>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

function GasEstimatesList({ estimates }: { estimates: GasEstimate[] }) {
  if (estimates.length === 0) {
    return (
      <div className="text-center py-6">
        <Fuel className="mx-auto h-10 w-10 text-gray-600" />
        <p className="mt-2 text-gray-400">No functions to estimate</p>
      </div>
    );
  }

  const levelColors = {
    low: 'text-accent-green',
    medium: 'text-accent-yellow',
    high: 'text-severity-critical',
  };

  return (
    <div className="space-y-3">
      {estimates.map((estimate, index) => (
        <div
          key={index}
          className="flex items-center justify-between rounded-lg border border-gray-800 p-3 hover:border-gray-700 transition-colors"
        >
          <div className="flex items-center gap-3">
            <div
              className={`flex h-8 w-8 items-center justify-center rounded-lg ${
                estimate.level === 'low'
                  ? 'bg-accent-green/20'
                  : estimate.level === 'medium'
                  ? 'bg-accent-yellow/20'
                  : 'bg-severity-critical/20'
              }`}
            >
              <Fuel
                className={`h-4 w-4 ${levelColors[estimate.level]}`}
              />
            </div>
            <div>
              <p className="text-sm font-medium text-white font-mono">
                {estimate.function_name}()
              </p>
              <p className="text-xs text-gray-500">
                {formatGas(estimate.min)} - {formatGas(estimate.max)} gas
              </p>
            </div>
          </div>
          <div className="text-right">
            <p className={`text-sm font-semibold ${levelColors[estimate.level]}`}>
              ~{formatGas(estimate.typical)}
            </p>
            <p className="text-xs text-gray-500 capitalize">{estimate.level}</p>
          </div>
        </div>
      ))}
    </div>
  );
}
