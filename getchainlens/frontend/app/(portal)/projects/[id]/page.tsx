'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import {
  ArrowLeft,
  FileCode2,
  Plus,
  Search,
  Shield,
  AlertTriangle,
  Settings,
  Trash2,
  ExternalLink,
  Activity,
  CheckCircle2,
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
} from '@/components/ui';
import { api } from '@/lib/api';
import {
  formatAddress,
  formatTimeAgo,
  getChainName,
  getSeverityBadgeClass,
  copyToClipboard,
} from '@/lib/utils';
import type { Project, Contract, Chain, Severity } from '@/types';

export default function ProjectDetailPage() {
  const params = useParams();
  const projectId = params.id as string;

  const [project, setProject] = useState<Project | null>(null);
  const [contracts, setContracts] = useState<Contract[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [showAddContract, setShowAddContract] = useState(false);

  useEffect(() => {
    async function loadData() {
      const [projectResult, contractsResult] = await Promise.all([
        api.getProject(projectId),
        api.getContracts(projectId),
      ]);

      if (projectResult.data) {
        setProject(projectResult.data);
      }
      if (contractsResult.data) {
        setContracts(contractsResult.data);
      }
      setIsLoading(false);
    }
    loadData();
  }, [projectId]);

  const filteredContracts = contracts.filter(
    (contract) =>
      contract.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      contract.address.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const handleDeleteContract = async (contractId: string) => {
    if (!confirm('Are you sure you want to remove this contract from the project?')) return;

    const result = await api.deleteContract(contractId);
    if (!result.error) {
      setContracts(contracts.filter((c) => c.id !== contractId));
    }
  };

  // Calculate project stats
  const totalIssues = contracts.reduce((sum, c) => sum + (c.last_analysis?.issues.length || 0), 0);
  const avgScore = contracts.length > 0
    ? Math.round(
        contracts.reduce((sum, c) => {
          const score = c.last_analysis?.score;
          if (!score) return sum;
          return sum + (score.security + score.gas_efficiency + score.code_quality) / 3;
        }, 0) / contracts.filter((c) => c.last_analysis?.score).length || 0
      )
    : 0;

  const issuesBySeverity: Record<Severity, number> = {
    critical: 0,
    high: 0,
    medium: 0,
    low: 0,
    info: 0,
  };

  contracts.forEach((contract) => {
    contract.last_analysis?.issues.forEach((issue) => {
      issuesBySeverity[issue.severity]++;
    });
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-accent-cyan border-t-transparent" />
      </div>
    );
  }

  if (!project) {
    return (
      <div className="text-center py-12">
        <h2 className="text-xl font-semibold text-white">Project not found</h2>
        <Button variant="secondary" className="mt-4" asChild>
          <Link href="/projects">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Projects
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
          <Link href="/projects">
            <ArrowLeft className="h-5 w-5" />
          </Link>
        </Button>
        <div className="flex-1">
          <h1 className="text-2xl font-bold text-white">{project.name}</h1>
          {project.description && (
            <p className="mt-1 text-gray-400">{project.description}</p>
          )}
        </div>
        <div className="flex gap-2">
          <Button variant="secondary" asChild>
            <Link href={`/projects/${projectId}/settings`}>
              <Settings className="mr-2 h-4 w-4" />
              Settings
            </Link>
          </Button>
          <Button onClick={() => setShowAddContract(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Add Contract
          </Button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid gap-4 sm:grid-cols-4">
        <StatCard
          title="Contracts"
          value={contracts.length}
          icon={<FileCode2 className="h-5 w-5" />}
        />
        <StatCard
          title="Total Issues"
          value={totalIssues}
          icon={<AlertTriangle className="h-5 w-5" />}
        />
        <StatCard
          title="Avg Score"
          value={avgScore || '-'}
          icon={<Shield className="h-5 w-5" />}
        />
        <StatCard
          title="Analyzed"
          value={contracts.filter((c) => c.last_analysis).length}
          icon={<Activity className="h-5 w-5" />}
        />
      </div>

      {/* Issues Summary */}
      {totalIssues > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-accent-orange" />
              Issues Summary
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-4">
              {(['critical', 'high', 'medium', 'low', 'info'] as Severity[]).map((severity) => {
                const count = issuesBySeverity[severity];
                if (count === 0) return null;
                return (
                  <div key={severity} className="flex items-center gap-2">
                    <Badge className={getSeverityBadgeClass(severity)}>
                      {severity}
                    </Badge>
                    <span className="text-white font-medium">{count}</span>
                  </div>
                );
              })}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Contracts */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <FileCode2 className="h-5 w-5 text-accent-cyan" />
              Contracts
            </CardTitle>
            <span className="text-sm text-gray-400">
              {contracts.length} contract{contracts.length !== 1 ? 's' : ''}
            </span>
          </div>
        </CardHeader>
        <CardContent>
          {/* Search */}
          <div className="mb-4">
            <Input
              placeholder="Search contracts..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              icon={<Search className="h-4 w-4" />}
            />
          </div>

          {filteredContracts.length > 0 ? (
            <div className="space-y-3">
              {filteredContracts.map((contract) => (
                <ContractRow
                  key={contract.id}
                  contract={contract}
                  onDelete={() => handleDeleteContract(contract.id)}
                />
              ))}
            </div>
          ) : (
            <div className="text-center py-8">
              <FileCode2 className="mx-auto h-12 w-12 text-gray-600" />
              <p className="mt-3 text-gray-400">
                {searchQuery ? 'No contracts match your search' : 'No contracts in this project yet'}
              </p>
              {!searchQuery && (
                <Button className="mt-4" onClick={() => setShowAddContract(true)}>
                  <Plus className="mr-2 h-4 w-4" />
                  Add Contract
                </Button>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Add Contract Modal */}
      {showAddContract && (
        <AddContractModal
          projectId={projectId}
          onClose={() => setShowAddContract(false)}
          onSuccess={(contract) => {
            setContracts([contract, ...contracts]);
            setShowAddContract(false);
          }}
        />
      )}
    </div>
  );
}

function StatCard({
  title,
  value,
  icon,
}: {
  title: string;
  value: string | number;
  icon: React.ReactNode;
}) {
  return (
    <Card>
      <CardContent className="p-5">
        <div className="flex items-center justify-between">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/20 text-accent-cyan">
            {icon}
          </div>
          <p className="text-2xl font-bold text-white">{value}</p>
        </div>
        <p className="mt-2 text-sm text-gray-400">{title}</p>
      </CardContent>
    </Card>
  );
}

function ContractRow({
  contract,
  onDelete,
}: {
  contract: Contract;
  onDelete: () => void;
}) {
  const analysis = contract.last_analysis;
  const score = analysis?.score;
  const overallScore = score
    ? Math.round((score.security + score.gas_efficiency + score.code_quality) / 3)
    : null;

  return (
    <div className="flex items-center justify-between rounded-lg border border-gray-800 p-4 hover:border-gray-700 transition-colors">
      <div className="flex items-center gap-4 flex-1 min-w-0">
        <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-primary/20">
          <FileCode2 className="h-6 w-6 text-accent-cyan" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <Link
              href={`/contracts/${contract.id}`}
              className="font-medium text-white hover:text-accent-cyan transition-colors"
            >
              {contract.name}
            </Link>
            <Badge variant={contract.chain} className="text-xs">
              {getChainName(contract.chain)}
            </Badge>
            {contract.is_verified && (
              <CheckCircle2 className="h-4 w-4 text-accent-green" />
            )}
          </div>
          <p className="text-sm text-gray-500 font-mono truncate">
            {formatAddress(contract.address, 8)}
          </p>
        </div>
      </div>

      <div className="flex items-center gap-6">
        {/* Analysis Status */}
        {analysis ? (
          <div className="flex items-center gap-4">
            <div className="text-center">
              <p className="text-lg font-semibold text-white">{overallScore}</p>
              <p className="text-xs text-gray-500">Score</p>
            </div>
            <div className="text-center">
              <p className="text-lg font-semibold text-white">{analysis.issues.length}</p>
              <p className="text-xs text-gray-500">Issues</p>
            </div>
          </div>
        ) : (
          <span className="text-sm text-gray-500">Not analyzed</span>
        )}

        {/* Actions */}
        <div className="flex items-center gap-2">
          <Button variant="secondary" size="sm" asChild>
            <Link href={`/contracts/${contract.id}`}>
              View
            </Link>
          </Button>
          <button
            onClick={onDelete}
            className="p-2 rounded-lg text-gray-500 hover:text-severity-critical hover:bg-severity-critical/10 transition-colors"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  );
}

function AddContractModal({
  projectId,
  onClose,
  onSuccess,
}: {
  projectId: string;
  onClose: () => void;
  onSuccess: (contract: Contract) => void;
}) {
  const [name, setName] = useState('');
  const [address, setAddress] = useState('');
  const [chain, setChain] = useState<Chain>('ethereum');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError(null);

    const result = await api.addContract({
      project_id: projectId,
      name,
      address,
      chain,
    });

    if (result.data) {
      onSuccess(result.data);
    } else {
      setError(result.error || 'Failed to add contract');
    }

    setIsLoading(false);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Add Contract</CardTitle>
          <CardDescription>
            Import a deployed contract to analyze
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
              <div className="rounded-lg bg-severity-critical/10 border border-severity-critical/30 p-3 text-sm text-severity-critical">
                {error}
              </div>
            )}

            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-300">Name</label>
              <Input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="My Contract"
                required
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-300">Contract Address</label>
              <Input
                value={address}
                onChange={(e) => setAddress(e.target.value)}
                placeholder="0x..."
                required
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-300">Chain</label>
              <div className="grid grid-cols-3 gap-2">
                {(['ethereum', 'polygon', 'arbitrum', 'optimism', 'base'] as Chain[]).map((c) => (
                  <button
                    key={c}
                    type="button"
                    onClick={() => setChain(c)}
                    className={`rounded-lg border p-2 text-sm transition-all ${
                      chain === c
                        ? 'border-accent-cyan bg-accent-cyan/10 text-white'
                        : 'border-gray-700 text-gray-400 hover:border-gray-600'
                    }`}
                  >
                    {getChainName(c)}
                  </button>
                ))}
              </div>
            </div>

            <div className="flex gap-3 pt-4">
              <Button type="button" variant="secondary" onClick={onClose} className="flex-1">
                Cancel
              </Button>
              <Button type="submit" isLoading={isLoading} className="flex-1">
                Add Contract
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
