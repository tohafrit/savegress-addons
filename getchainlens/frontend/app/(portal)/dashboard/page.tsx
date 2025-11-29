'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import {
  Shield,
  FileCode2,
  AlertTriangle,
  TrendingUp,
  ArrowRight,
  Plus,
  Search,
  Activity,
} from 'lucide-react';
import { Card, CardHeader, CardTitle, CardContent, Button, Badge } from '@/components/ui';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth-context';
import { formatTimeAgo, formatNumber, getSeverityBadgeClass } from '@/lib/utils';
import type { DashboardStats, SecurityIssue, AnalysisResult, Severity } from '@/types';

interface MetricCardProps {
  title: string;
  value: string | number;
  change?: string;
  changeType?: 'positive' | 'negative' | 'neutral';
  icon: React.ReactNode;
  href?: string;
}

function MetricCard({ title, value, change, changeType, icon, href }: MetricCardProps) {
  const content = (
    <Card hover className="relative overflow-hidden">
      <CardContent className="p-6">
        <div className="flex items-start justify-between">
          <div>
            <p className="text-sm font-medium text-gray-400">{title}</p>
            <p className="mt-2 text-3xl font-bold text-white">{value}</p>
            {change && (
              <p
                className={`mt-1 text-sm ${
                  changeType === 'positive'
                    ? 'text-accent-green'
                    : changeType === 'negative'
                    ? 'text-severity-critical'
                    : 'text-gray-400'
                }`}
              >
                {change}
              </p>
            )}
          </div>
          <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-primary/20 text-accent-cyan">
            {icon}
          </div>
        </div>
      </CardContent>
    </Card>
  );

  if (href) {
    return <Link href={href}>{content}</Link>;
  }

  return content;
}

function SeverityBar({ data }: { data: Record<Severity, number> }) {
  const total = Object.values(data).reduce((a, b) => a + b, 0);
  if (total === 0) return null;

  const colors: Record<Severity, string> = {
    critical: 'bg-severity-critical',
    high: 'bg-severity-high',
    medium: 'bg-severity-medium',
    low: 'bg-severity-low',
    info: 'bg-severity-info',
  };

  return (
    <div className="space-y-2">
      <div className="flex h-2 overflow-hidden rounded-full bg-gray-800">
        {(['critical', 'high', 'medium', 'low', 'info'] as Severity[]).map((severity) => {
          const count = data[severity] || 0;
          const percent = (count / total) * 100;
          if (percent === 0) return null;
          return (
            <div
              key={severity}
              className={`${colors[severity]} transition-all duration-300`}
              style={{ width: `${percent}%` }}
            />
          );
        })}
      </div>
      <div className="flex flex-wrap gap-3 text-xs">
        {(['critical', 'high', 'medium', 'low', 'info'] as Severity[]).map((severity) => {
          const count = data[severity] || 0;
          if (count === 0) return null;
          return (
            <div key={severity} className="flex items-center gap-1.5">
              <div className={`h-2 w-2 rounded-full ${colors[severity]}`} />
              <span className="capitalize text-gray-400">{severity}:</span>
              <span className="font-medium text-white">{count}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export default function DashboardPage() {
  const { user } = useAuth();
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    async function loadStats() {
      const result = await api.getDashboardStats();
      if (result.data) {
        setStats(result.data);
      }
      setIsLoading(false);
    }
    loadStats();
  }, []);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-accent-cyan border-t-transparent" />
      </div>
    );
  }

  return (
    <div className="space-y-8">
      {/* Welcome header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-white">
            Welcome back, {user?.name?.split(' ')[0] || 'there'}
          </h1>
          <p className="mt-1 text-gray-400">
            Here&apos;s an overview of your smart contract security
          </p>
        </div>
        <div className="flex gap-3">
          <Button variant="secondary" asChild>
            <Link href="/traces">
              <Search className="mr-2 h-4 w-4" />
              Trace Transaction
            </Link>
          </Button>
          <Button asChild>
            <Link href="/contracts">
              <Plus className="mr-2 h-4 w-4" />
              Add Contract
            </Link>
          </Button>
        </div>
      </div>

      {/* Metrics */}
      <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Total Contracts"
          value={formatNumber(stats?.total_contracts || 0)}
          icon={<FileCode2 className="h-6 w-6" />}
          href="/contracts"
        />
        <MetricCard
          title="Analyses Run"
          value={formatNumber(stats?.total_analyses || 0)}
          change="+12% this week"
          changeType="positive"
          icon={<Shield className="h-6 w-6" />}
        />
        <MetricCard
          title="Issues Found"
          value={formatNumber(stats?.total_issues || 0)}
          icon={<AlertTriangle className="h-6 w-6" />}
        />
        <MetricCard
          title="Security Score"
          value="85"
          change="Good standing"
          changeType="positive"
          icon={<TrendingUp className="h-6 w-6" />}
        />
      </div>

      {/* Issues by severity */}
      {stats?.issues_by_severity && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-accent-orange" />
              Issues by Severity
            </CardTitle>
          </CardHeader>
          <CardContent>
            <SeverityBar data={stats.issues_by_severity} />
          </CardContent>
        </Card>
      )}

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Recent analyses */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <Activity className="h-5 w-5 text-accent-cyan" />
              Recent Analyses
            </CardTitle>
            <Link
              href="/contracts"
              className="text-sm text-accent-cyan hover:underline flex items-center gap-1"
            >
              View all <ArrowRight className="h-3 w-3" />
            </Link>
          </CardHeader>
          <CardContent>
            {stats?.recent_analyses && stats.recent_analyses.length > 0 ? (
              <div className="space-y-4">
                {stats.recent_analyses.slice(0, 5).map((analysis: AnalysisResult) => (
                  <div
                    key={analysis.id}
                    className="flex items-center justify-between rounded-lg border border-gray-800 p-4 hover:border-gray-700 transition-colors"
                  >
                    <div className="flex items-center gap-3">
                      <div
                        className="flex h-10 w-10 items-center justify-center rounded-lg"
                        style={{
                          backgroundColor:
                            analysis.status === 'completed'
                              ? 'rgba(16, 185, 129, 0.2)'
                              : analysis.status === 'failed'
                              ? 'rgba(220, 38, 38, 0.2)'
                              : 'rgba(251, 191, 36, 0.2)',
                        }}
                      >
                        <Shield
                          className="h-5 w-5"
                          style={{
                            color:
                              analysis.status === 'completed'
                                ? '#10B981'
                                : analysis.status === 'failed'
                                ? '#DC2626'
                                : '#FBBF24',
                          }}
                        />
                      </div>
                      <div>
                        <p className="font-medium text-white">
                          Analysis #{analysis.id.slice(0, 8)}
                        </p>
                        <p className="text-sm text-gray-400">
                          {analysis.issues.length} issues found
                        </p>
                      </div>
                    </div>
                    <div className="text-right">
                      <Badge
                        variant={
                          analysis.status === 'completed'
                            ? 'success'
                            : analysis.status === 'failed'
                            ? 'destructive'
                            : 'warning'
                        }
                      >
                        {analysis.status}
                      </Badge>
                      <p className="mt-1 text-xs text-gray-500">
                        {analysis.created_at && formatTimeAgo(analysis.created_at)}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-8">
                <Shield className="mx-auto h-12 w-12 text-gray-600" />
                <p className="mt-3 text-gray-400">No analyses yet</p>
                <Button variant="secondary" className="mt-4" asChild>
                  <Link href="/contracts">
                    <Plus className="mr-2 h-4 w-4" />
                    Add your first contract
                  </Link>
                </Button>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Top issues */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-severity-high" />
              Critical Issues
            </CardTitle>
          </CardHeader>
          <CardContent>
            {stats?.top_issues && stats.top_issues.length > 0 ? (
              <div className="space-y-3">
                {stats.top_issues.slice(0, 5).map((issue: SecurityIssue, index: number) => (
                  <div
                    key={index}
                    className="rounded-lg border border-gray-800 p-4 hover:border-gray-700 transition-colors"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <Badge className={getSeverityBadgeClass(issue.severity)}>
                            {issue.severity}
                          </Badge>
                          <span className="text-xs text-gray-500 font-mono">
                            {issue.type}
                          </span>
                        </div>
                        <p className="text-sm text-white line-clamp-2">{issue.message}</p>
                        <p className="mt-1 text-xs text-gray-500">
                          Line {issue.line}
                        </p>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-8">
                <AlertTriangle className="mx-auto h-12 w-12 text-gray-600" />
                <p className="mt-3 text-gray-400">No critical issues found</p>
                <p className="mt-1 text-sm text-gray-500">
                  Your contracts are looking secure
                </p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Quick actions */}
      <Card>
        <CardHeader>
          <CardTitle>Quick Actions</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <Link
              href="/contracts"
              className="flex items-center gap-4 rounded-lg border border-gray-800 p-4 hover:border-accent-cyan/50 hover:bg-dark-card-hover transition-all"
            >
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-cyan/20">
                <Plus className="h-5 w-5 text-accent-cyan" />
              </div>
              <div>
                <p className="font-medium text-white">Add Contract</p>
                <p className="text-sm text-gray-400">Import for analysis</p>
              </div>
            </Link>

            <Link
              href="/traces"
              className="flex items-center gap-4 rounded-lg border border-gray-800 p-4 hover:border-accent-cyan/50 hover:bg-dark-card-hover transition-all"
            >
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-blue/20">
                <Search className="h-5 w-5 text-accent-blue" />
              </div>
              <div>
                <p className="font-medium text-white">Trace Transaction</p>
                <p className="text-sm text-gray-400">Debug tx execution</p>
              </div>
            </Link>

            <Link
              href="/monitors"
              className="flex items-center gap-4 rounded-lg border border-gray-800 p-4 hover:border-accent-cyan/50 hover:bg-dark-card-hover transition-all"
            >
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-green/20">
                <Activity className="h-5 w-5 text-accent-green" />
              </div>
              <div>
                <p className="font-medium text-white">Set Up Monitor</p>
                <p className="text-sm text-gray-400">Watch for events</p>
              </div>
            </Link>

            <Link
              href="/analytics/ethereum"
              className="flex items-center gap-4 rounded-lg border border-gray-800 p-4 hover:border-accent-cyan/50 hover:bg-dark-card-hover transition-all"
            >
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-orange/20">
                <TrendingUp className="h-5 w-5 text-accent-orange" />
              </div>
              <div>
                <p className="font-medium text-white">View Analytics</p>
                <p className="text-sm text-gray-400">Network insights</p>
              </div>
            </Link>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
