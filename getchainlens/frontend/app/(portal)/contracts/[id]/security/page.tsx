'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import {
  ArrowLeft,
  Shield,
  AlertTriangle,
  AlertCircle,
  Info,
  CheckCircle2,
  ChevronDown,
  ChevronUp,
  ExternalLink,
  Code,
  FileText,
  Filter,
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
import { getChainName, getSeverityBadgeClass } from '@/lib/utils';
import type { Contract, AnalysisResult, SecurityIssue, Severity } from '@/types';

const severityInfo: Record<
  Severity,
  { icon: React.ComponentType<{ className?: string }>; label: string; description: string }
> = {
  critical: {
    icon: AlertTriangle,
    label: 'Critical',
    description: 'Severe vulnerabilities that can lead to loss of funds or complete contract compromise',
  },
  high: {
    icon: AlertCircle,
    label: 'High',
    description: 'Significant issues that could lead to unexpected behavior or partial exploits',
  },
  medium: {
    icon: AlertCircle,
    label: 'Medium',
    description: 'Moderate issues that may affect contract functionality under certain conditions',
  },
  low: {
    icon: Info,
    label: 'Low',
    description: 'Minor issues or deviations from best practices',
  },
  info: {
    icon: Info,
    label: 'Info',
    description: 'Informational findings and suggestions for improvement',
  },
};

export default function SecurityAnalysisPage() {
  const params = useParams();
  const contractId = params.id as string;

  const [contract, setContract] = useState<Contract | null>(null);
  const [analysis, setAnalysis] = useState<AnalysisResult | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [expandedIssue, setExpandedIssue] = useState<string | null>(null);
  const [filterSeverity, setFilterSeverity] = useState<Severity | 'all'>('all');
  const [filterType, setFilterType] = useState<string>('all');

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

  const issues = analysis?.issues || [];

  // Get unique issue types
  const issueTypes = Array.from(new Set(issues.map((i) => i.type)));

  // Filter issues
  const filteredIssues = issues.filter((issue) => {
    if (filterSeverity !== 'all' && issue.severity !== filterSeverity) return false;
    if (filterType !== 'all' && issue.type !== filterType) return false;
    return true;
  });

  // Count by severity
  const countBySeverity: Record<Severity, number> = {
    critical: issues.filter((i) => i.severity === 'critical').length,
    high: issues.filter((i) => i.severity === 'high').length,
    medium: issues.filter((i) => i.severity === 'medium').length,
    low: issues.filter((i) => i.severity === 'low').length,
    info: issues.filter((i) => i.severity === 'info').length,
  };

  const score = analysis?.score?.security || 0;

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
            <h1 className="text-2xl font-bold text-white">Security Analysis</h1>
            <Badge variant={contract.chain}>{getChainName(contract.chain)}</Badge>
          </div>
          <p className="mt-1 text-gray-400">{contract.name}</p>
        </div>
      </div>

      {/* Overview Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-6">
        {/* Security Score */}
        <Card className="sm:col-span-2 lg:col-span-2">
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-400">Security Score</p>
                <p className="mt-1 text-4xl font-bold text-white">{score}</p>
                <p
                  className={`mt-1 text-sm ${
                    score >= 80
                      ? 'text-accent-green'
                      : score >= 60
                      ? 'text-accent-yellow'
                      : 'text-severity-critical'
                  }`}
                >
                  {score >= 80 ? 'Good' : score >= 60 ? 'Moderate' : 'Needs Attention'}
                </p>
              </div>
              <div
                className="flex h-20 w-20 items-center justify-center rounded-full"
                style={{
                  background: `conic-gradient(${
                    score >= 80 ? '#10B981' : score >= 60 ? '#FBBF24' : '#DC2626'
                  } ${score}%, rgba(55, 65, 81, 0.5) ${score}%)`,
                }}
              >
                <div className="flex h-16 w-16 items-center justify-center rounded-full bg-dark-card">
                  <Shield className="h-8 w-8 text-accent-cyan" />
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Severity counts */}
        {(['critical', 'high', 'medium', 'low'] as Severity[]).map((severity) => {
          const info = severityInfo[severity];
          const count = countBySeverity[severity];
          const Icon = info.icon;
          return (
            <Card key={severity}>
              <CardContent className="p-4">
                <div className="flex items-center gap-2">
                  <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${getSeverityBadgeClass(severity).replace('text-', 'bg-').replace('/20', '/20')}`}>
                    <Icon className={`h-4 w-4 ${getSeverityBadgeClass(severity).split(' ')[1]}`} />
                  </div>
                  <div>
                    <p className="text-2xl font-bold text-white">{count}</p>
                    <p className="text-xs text-gray-500 capitalize">{severity}</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="p-4">
          <div className="flex flex-wrap items-center gap-4">
            <div className="flex items-center gap-2">
              <Filter className="h-4 w-4 text-gray-400" />
              <span className="text-sm text-gray-400">Filter:</span>
            </div>

            <div className="flex items-center gap-2">
              <span className="text-sm text-gray-500">Severity:</span>
              <select
                value={filterSeverity}
                onChange={(e) => setFilterSeverity(e.target.value as Severity | 'all')}
                className="rounded-lg border border-gray-700 bg-dark-bg px-3 py-1.5 text-sm text-white focus:border-accent-cyan focus:outline-none"
              >
                <option value="all">All</option>
                <option value="critical">Critical</option>
                <option value="high">High</option>
                <option value="medium">Medium</option>
                <option value="low">Low</option>
                <option value="info">Info</option>
              </select>
            </div>

            <div className="flex items-center gap-2">
              <span className="text-sm text-gray-500">Type:</span>
              <select
                value={filterType}
                onChange={(e) => setFilterType(e.target.value)}
                className="rounded-lg border border-gray-700 bg-dark-bg px-3 py-1.5 text-sm text-white focus:border-accent-cyan focus:outline-none"
              >
                <option value="all">All Types</option>
                {issueTypes.map((type) => (
                  <option key={type} value={type}>
                    {type}
                  </option>
                ))}
              </select>
            </div>

            <span className="text-sm text-gray-500 ml-auto">
              Showing {filteredIssues.length} of {issues.length} issues
            </span>
          </div>
        </CardContent>
      </Card>

      {/* Issues List */}
      {filteredIssues.length > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-accent-orange" />
              Security Issues
            </CardTitle>
            <CardDescription>
              Detected vulnerabilities and security concerns
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {filteredIssues.map((issue) => (
                <IssueCard
                  key={issue.id}
                  issue={issue}
                  isExpanded={expandedIssue === issue.id}
                  onToggle={() =>
                    setExpandedIssue(expandedIssue === issue.id ? null : issue.id)
                  }
                />
              ))}
            </div>
          </CardContent>
        </Card>
      ) : issues.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <CheckCircle2 className="mx-auto h-12 w-12 text-accent-green" />
            <h3 className="mt-4 text-lg font-medium text-white">No security issues found</h3>
            <p className="mt-2 text-gray-400">
              Your contract passed all security checks
            </p>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="py-12 text-center">
            <Filter className="mx-auto h-12 w-12 text-gray-600" />
            <h3 className="mt-4 text-lg font-medium text-white">No issues match your filters</h3>
            <p className="mt-2 text-gray-400">
              Try adjusting your filter criteria
            </p>
            <Button
              variant="secondary"
              className="mt-4"
              onClick={() => {
                setFilterSeverity('all');
                setFilterType('all');
              }}
            >
              Clear Filters
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Vulnerability Reference */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileText className="h-5 w-5 text-accent-cyan" />
            Vulnerability Reference
          </CardTitle>
          <CardDescription>
            Common smart contract vulnerabilities detected by ChainLens
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <VulnerabilityCard
              title="Reentrancy"
              description="Recursive calls that can drain funds before state updates"
              link="https://swcregistry.io/docs/SWC-107"
            />
            <VulnerabilityCard
              title="Integer Overflow"
              description="Arithmetic operations that exceed type bounds"
              link="https://swcregistry.io/docs/SWC-101"
            />
            <VulnerabilityCard
              title="Access Control"
              description="Missing or incorrect access restrictions on functions"
              link="https://swcregistry.io/docs/SWC-105"
            />
            <VulnerabilityCard
              title="Unchecked Call"
              description="External calls without return value validation"
              link="https://swcregistry.io/docs/SWC-104"
            />
            <VulnerabilityCard
              title="tx.origin Auth"
              description="Using tx.origin for authentication instead of msg.sender"
              link="https://swcregistry.io/docs/SWC-115"
            />
            <VulnerabilityCard
              title="Timestamp Dependency"
              description="Using block.timestamp for critical logic"
              link="https://swcregistry.io/docs/SWC-116"
            />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function IssueCard({
  issue,
  isExpanded,
  onToggle,
}: {
  issue: SecurityIssue;
  isExpanded: boolean;
  onToggle: () => void;
}) {
  const info = severityInfo[issue.severity];
  const Icon = info.icon;

  return (
    <div className={`rounded-lg border ${getSeverityBadgeClass(issue.severity).replace('text-', 'border-').replace('bg-', 'border-').split(' ')[0]}/30 overflow-hidden`}>
      <button
        onClick={onToggle}
        className="w-full flex items-start justify-between p-4 hover:bg-dark-card-hover transition-colors text-left"
      >
        <div className="flex items-start gap-4">
          <div className={`flex h-10 w-10 items-center justify-center rounded-lg shrink-0 ${getSeverityBadgeClass(issue.severity).split(' ')[0]}`}>
            <Icon className={`h-5 w-5 ${getSeverityBadgeClass(issue.severity).split(' ')[1]}`} />
          </div>
          <div>
            <div className="flex items-center gap-2 mb-1">
              <Badge className={getSeverityBadgeClass(issue.severity)}>
                {issue.severity}
              </Badge>
              <span className="text-sm text-gray-500 font-mono">{issue.type}</span>
            </div>
            <p className="text-white">{issue.message}</p>
            <p className="text-sm text-gray-500 mt-1">
              Line {issue.line}
              {issue.end_line && issue.end_line !== issue.line && ` - ${issue.end_line}`}
            </p>
          </div>
        </div>
        <div className="ml-4 shrink-0">
          {isExpanded ? (
            <ChevronUp className="h-5 w-5 text-gray-400" />
          ) : (
            <ChevronDown className="h-5 w-5 text-gray-400" />
          )}
        </div>
      </button>

      {isExpanded && (
        <div className="border-t border-gray-800 p-4 bg-dark-bg/50 space-y-4">
          {issue.suggestion && (
            <div>
              <p className="text-sm font-medium text-gray-300 mb-1">Suggestion:</p>
              <p className="text-sm text-gray-400">{issue.suggestion}</p>
            </div>
          )}

          {issue.code && (
            <div>
              <p className="text-sm font-medium text-gray-300 mb-1">Code:</p>
              <pre className="rounded-lg bg-dark-bg p-3 text-sm text-gray-300 font-mono overflow-x-auto">
                {issue.code}
              </pre>
            </div>
          )}

          {issue.references && issue.references.length > 0 && (
            <div>
              <p className="text-sm font-medium text-gray-300 mb-2">References:</p>
              <div className="flex flex-wrap gap-2">
                {issue.references.map((ref, index) => (
                  <a
                    key={index}
                    href={ref}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center gap-1 text-sm text-accent-cyan hover:underline"
                  >
                    <ExternalLink className="h-3 w-3" />
                    {new URL(ref).hostname}
                  </a>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function VulnerabilityCard({
  title,
  description,
  link,
}: {
  title: string;
  description: string;
  link: string;
}) {
  return (
    <a
      href={link}
      target="_blank"
      rel="noopener noreferrer"
      className="block rounded-lg border border-gray-800 p-4 hover:border-gray-700 transition-colors group"
    >
      <div className="flex items-start justify-between">
        <h4 className="font-medium text-white group-hover:text-accent-cyan transition-colors">
          {title}
        </h4>
        <ExternalLink className="h-4 w-4 text-gray-500 group-hover:text-accent-cyan transition-colors" />
      </div>
      <p className="mt-1 text-sm text-gray-400">{description}</p>
    </a>
  );
}
