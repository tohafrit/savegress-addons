'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import {
  FileCode2,
  Plus,
  Search,
  Filter,
  MoreVertical,
  ExternalLink,
  Trash2,
  Shield,
  AlertTriangle,
} from 'lucide-react';
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Button,
  Input,
  Badge,
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/components/ui';
import { api } from '@/lib/api';
import { formatAddress, formatDate, getChainColor, getChainName, getScoreColor } from '@/lib/utils';
import type { Contract, Chain, Project } from '@/types';

export default function ContractsPage() {
  const [contracts, setContracts] = useState<Contract[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedChain, setSelectedChain] = useState<string>('all');
  const [selectedProject, setSelectedProject] = useState<string>('all');
  const [showAddModal, setShowAddModal] = useState(false);

  useEffect(() => {
    async function loadData() {
      const [contractsResult, projectsResult] = await Promise.all([
        api.getContracts(),
        api.getProjects(),
      ]);

      if (contractsResult.data) {
        setContracts(contractsResult.data);
      }
      if (projectsResult.data) {
        setProjects(projectsResult.data);
      }
      setIsLoading(false);
    }
    loadData();
  }, []);

  const filteredContracts = contracts.filter((contract) => {
    const matchesSearch =
      contract.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      contract.address.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesChain = selectedChain === 'all' || contract.chain === selectedChain;
    const matchesProject = selectedProject === 'all' || contract.project_id === selectedProject;
    return matchesSearch && matchesChain && matchesProject;
  });

  const handleDelete = async (contractId: string) => {
    if (!confirm('Are you sure you want to delete this contract?')) return;

    const result = await api.deleteContract(contractId);
    if (!result.error) {
      setContracts(contracts.filter((c) => c.id !== contractId));
    }
  };

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
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-white">Contracts</h1>
          <p className="mt-1 text-gray-400">
            Manage and analyze your smart contracts
          </p>
        </div>
        <Button onClick={() => setShowAddModal(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Add Contract
        </Button>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="p-4">
          <div className="flex flex-col sm:flex-row gap-4">
            <div className="flex-1">
              <Input
                placeholder="Search by name or address..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                icon={<Search className="h-4 w-4" />}
              />
            </div>
            <Select value={selectedChain} onValueChange={setSelectedChain}>
              <SelectTrigger className="w-full sm:w-[180px]">
                <SelectValue placeholder="All chains" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All chains</SelectItem>
                <SelectItem value="ethereum">Ethereum</SelectItem>
                <SelectItem value="polygon">Polygon</SelectItem>
                <SelectItem value="arbitrum">Arbitrum</SelectItem>
                <SelectItem value="optimism">Optimism</SelectItem>
                <SelectItem value="base">Base</SelectItem>
              </SelectContent>
            </Select>
            <Select value={selectedProject} onValueChange={setSelectedProject}>
              <SelectTrigger className="w-full sm:w-[180px]">
                <SelectValue placeholder="All projects" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All projects</SelectItem>
                {projects.map((project) => (
                  <SelectItem key={project.id} value={project.id}>
                    {project.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* Contracts grid */}
      {filteredContracts.length > 0 ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {filteredContracts.map((contract) => (
            <ContractCard
              key={contract.id}
              contract={contract}
              onDelete={() => handleDelete(contract.id)}
            />
          ))}
        </div>
      ) : (
        <Card>
          <CardContent className="py-12 text-center">
            <FileCode2 className="mx-auto h-12 w-12 text-gray-600" />
            <h3 className="mt-4 text-lg font-medium text-white">No contracts found</h3>
            <p className="mt-2 text-gray-400">
              {searchQuery || selectedChain !== 'all' || selectedProject !== 'all'
                ? 'Try adjusting your filters'
                : 'Add your first contract to get started'}
            </p>
            {!searchQuery && selectedChain === 'all' && selectedProject === 'all' && (
              <Button className="mt-4" onClick={() => setShowAddModal(true)}>
                <Plus className="mr-2 h-4 w-4" />
                Add Contract
              </Button>
            )}
          </CardContent>
        </Card>
      )}

      {/* Add Contract Modal - simplified for now */}
      {showAddModal && (
        <AddContractModal
          projects={projects}
          onClose={() => setShowAddModal(false)}
          onSuccess={(contract) => {
            setContracts([contract, ...contracts]);
            setShowAddModal(false);
          }}
        />
      )}
    </div>
  );
}

function ContractCard({
  contract,
  onDelete,
}: {
  contract: Contract;
  onDelete: () => void;
}) {
  const score = contract.last_analysis?.score;
  const overallScore = score
    ? Math.round((score.security + score.gas_efficiency + score.code_quality) / 3)
    : null;

  return (
    <Card hover className="relative overflow-hidden">
      <CardContent className="p-5">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/20">
              <FileCode2 className="h-5 w-5 text-accent-cyan" />
            </div>
            <div>
              <h3 className="font-semibold text-white">{contract.name}</h3>
              <p className="text-sm text-gray-400 font-mono">
                {formatAddress(contract.address)}
              </p>
            </div>
          </div>
          <button
            onClick={onDelete}
            className="p-1.5 rounded-lg text-gray-500 hover:text-severity-critical hover:bg-severity-critical/10 transition-colors"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </div>

        <div className="mt-4 flex items-center gap-2">
          <Badge variant={contract.chain as Chain} className="text-xs">
            {getChainName(contract.chain)}
          </Badge>
          {contract.is_verified && (
            <Badge variant="success" className="text-xs">
              Verified
            </Badge>
          )}
        </div>

        {contract.last_analysis && (
          <div className="mt-4 pt-4 border-t border-gray-800">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                {overallScore !== null && (
                  <>
                    <div
                      className="h-2 w-2 rounded-full"
                      style={{ backgroundColor: getScoreColor(overallScore) }}
                    />
                    <span className="text-sm font-medium text-white">
                      Score: {overallScore}
                    </span>
                  </>
                )}
              </div>
              <div className="flex items-center gap-1 text-sm text-gray-400">
                <AlertTriangle className="h-3.5 w-3.5" />
                {contract.last_analysis.issues.length} issues
              </div>
            </div>
          </div>
        )}

        <div className="mt-4 flex gap-2">
          <Button variant="secondary" size="sm" className="flex-1" asChild>
            <Link href={`/contracts/${contract.id}`}>
              <Shield className="mr-1.5 h-3.5 w-3.5" />
              Analyze
            </Link>
          </Button>
          <Button variant="ghost" size="sm" asChild>
            <a
              href={`https://etherscan.io/address/${contract.address}`}
              target="_blank"
              rel="noopener noreferrer"
            >
              <ExternalLink className="h-4 w-4" />
            </a>
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

function AddContractModal({
  projects,
  onClose,
  onSuccess,
}: {
  projects: Project[];
  onClose: () => void;
  onSuccess: (contract: Contract) => void;
}) {
  const [name, setName] = useState('');
  const [address, setAddress] = useState('');
  const [chain, setChain] = useState<Chain>('ethereum');
  const [projectId, setProjectId] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError(null);

    const result = await api.addContract({
      name,
      address,
      chain,
      project_id: projectId || projects[0]?.id || '',
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
              <label className="text-sm font-medium text-gray-300">Address</label>
              <Input
                value={address}
                onChange={(e) => setAddress(e.target.value)}
                placeholder="0x..."
                required
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-300">Chain</label>
              <Select value={chain} onValueChange={(v) => setChain(v as Chain)}>
                <SelectTrigger>
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

            {projects.length > 0 && (
              <div className="space-y-2">
                <label className="text-sm font-medium text-gray-300">Project</label>
                <Select value={projectId} onValueChange={setProjectId}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select project" />
                  </SelectTrigger>
                  <SelectContent>
                    {projects.map((project) => (
                      <SelectItem key={project.id} value={project.id}>
                        {project.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}

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
