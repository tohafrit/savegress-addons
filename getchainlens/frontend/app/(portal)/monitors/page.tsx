'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import {
  Bell,
  Plus,
  Search,
  MoreVertical,
  Pause,
  Play,
  Trash2,
  Settings,
  AlertTriangle,
  CheckCircle2,
  Clock,
  Zap,
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
import { api } from '@/lib/api';
import { formatAddress, formatDate, formatTimeAgo, getChainName } from '@/lib/utils';
import type { Monitor, Chain } from '@/types';

export default function MonitorsPage() {
  const [monitors, setMonitors] = useState<Monitor[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [showCreateModal, setShowCreateModal] = useState(false);

  useEffect(() => {
    async function loadMonitors() {
      const result = await api.getMonitors();
      if (result.data) {
        setMonitors(result.data);
      }
      setIsLoading(false);
    }
    loadMonitors();
  }, []);

  const filteredMonitors = monitors.filter((monitor) => {
    const matchesSearch =
      monitor.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      monitor.contract_address.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesStatus = statusFilter === 'all' || monitor.status === statusFilter;
    return matchesSearch && matchesStatus;
  });

  const handleToggleStatus = async (monitorId: string, currentStatus: string) => {
    const newStatus = currentStatus === 'active' ? 'paused' : 'active';
    const result = await api.updateMonitor(monitorId, { status: newStatus });
    if (result.data) {
      setMonitors(monitors.map((m) => (m.id === monitorId ? result.data! : m)));
    }
  };

  const handleDelete = async (monitorId: string) => {
    if (!confirm('Are you sure you want to delete this monitor?')) return;

    const result = await api.deleteMonitor(monitorId);
    if (!result.error) {
      setMonitors(monitors.filter((m) => m.id !== monitorId));
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
          <h1 className="text-2xl font-bold text-white">Monitors</h1>
          <p className="mt-1 text-gray-400">
            Set up real-time alerts for contract events
          </p>
        </div>
        <Button onClick={() => setShowCreateModal(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Create Monitor
        </Button>
      </div>

      {/* Stats */}
      <div className="grid gap-4 sm:grid-cols-4">
        <StatCard
          title="Total Monitors"
          value={monitors.length}
          icon={<Bell className="h-5 w-5" />}
          color="accent-cyan"
        />
        <StatCard
          title="Active"
          value={monitors.filter((m) => m.status === 'active').length}
          icon={<Play className="h-5 w-5" />}
          color="accent-green"
        />
        <StatCard
          title="Paused"
          value={monitors.filter((m) => m.status === 'paused').length}
          icon={<Pause className="h-5 w-5" />}
          color="accent-yellow"
        />
        <StatCard
          title="Alerts Today"
          value={42}
          icon={<Zap className="h-5 w-5" />}
          color="accent-orange"
        />
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="p-4">
          <div className="flex flex-col sm:flex-row gap-4">
            <div className="flex-1">
              <Input
                placeholder="Search monitors..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                icon={<Search className="h-4 w-4" />}
              />
            </div>
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-full sm:w-[180px]">
                <SelectValue placeholder="All statuses" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All statuses</SelectItem>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="paused">Paused</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* Monitors list */}
      {filteredMonitors.length > 0 ? (
        <div className="space-y-4">
          {filteredMonitors.map((monitor) => (
            <MonitorCard
              key={monitor.id}
              monitor={monitor}
              onToggleStatus={() => handleToggleStatus(monitor.id, monitor.status)}
              onDelete={() => handleDelete(monitor.id)}
            />
          ))}
        </div>
      ) : (
        <Card>
          <CardContent className="py-12 text-center">
            <Bell className="mx-auto h-12 w-12 text-gray-600" />
            <h3 className="mt-4 text-lg font-medium text-white">No monitors found</h3>
            <p className="mt-2 text-gray-400">
              {searchQuery || statusFilter !== 'all'
                ? 'Try adjusting your filters'
                : 'Create your first monitor to get real-time alerts'}
            </p>
            {!searchQuery && statusFilter === 'all' && (
              <Button className="mt-4" onClick={() => setShowCreateModal(true)}>
                <Plus className="mr-2 h-4 w-4" />
                Create Monitor
              </Button>
            )}
          </CardContent>
        </Card>
      )}

      {/* Create Monitor Modal */}
      {showCreateModal && (
        <CreateMonitorModal
          onClose={() => setShowCreateModal(false)}
          onSuccess={(monitor) => {
            setMonitors([monitor, ...monitors]);
            setShowCreateModal(false);
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
  color,
}: {
  title: string;
  value: number;
  icon: React.ReactNode;
  color: string;
}) {
  return (
    <Card>
      <CardContent className="p-5">
        <div className="flex items-center justify-between">
          <div className={`flex h-10 w-10 items-center justify-center rounded-lg bg-${color}/20`}>
            <div className={`text-${color}`}>{icon}</div>
          </div>
          <span className="text-2xl font-bold text-white">{value}</span>
        </div>
        <p className="mt-2 text-sm text-gray-400">{title}</p>
      </CardContent>
    </Card>
  );
}

function MonitorCard({
  monitor,
  onToggleStatus,
  onDelete,
}: {
  monitor: Monitor;
  onToggleStatus: () => void;
  onDelete: () => void;
}) {
  const isActive = monitor.status === 'active';

  return (
    <Card hover>
      <CardContent className="p-5">
        <div className="flex items-start justify-between">
          <div className="flex items-start gap-4">
            <div
              className={`flex h-12 w-12 items-center justify-center rounded-lg ${
                isActive ? 'bg-accent-green/20' : 'bg-gray-700/50'
              }`}
            >
              <Bell className={`h-6 w-6 ${isActive ? 'text-accent-green' : 'text-gray-500'}`} />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h3 className="font-semibold text-white">{monitor.name}</h3>
                <Badge variant={isActive ? 'success' : 'secondary'} className="text-xs">
                  {isActive ? 'Active' : 'Paused'}
                </Badge>
              </div>
              <p className="mt-1 text-sm text-gray-400 font-mono">
                {formatAddress(monitor.contract_address, 10)}
              </p>
              <div className="mt-2 flex items-center gap-4 text-xs text-gray-500">
                <span className="flex items-center gap-1">
                  <Badge variant={monitor.chain as Chain} className="text-xs">
                    {getChainName(monitor.chain)}
                  </Badge>
                </span>
                <span className="flex items-center gap-1">
                  <Clock className="h-3 w-3" />
                  Created {formatTimeAgo(new Date(monitor.created_at))}
                </span>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={onToggleStatus}
              title={isActive ? 'Pause monitor' : 'Activate monitor'}
            >
              {isActive ? (
                <Pause className="h-4 w-4 text-gray-400" />
              ) : (
                <Play className="h-4 w-4 text-gray-400" />
              )}
            </Button>
            <Button variant="ghost" size="icon-sm" asChild>
              <Link href={`/monitors/${monitor.id}`}>
                <Settings className="h-4 w-4 text-gray-400" />
              </Link>
            </Button>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={onDelete}
              className="hover:text-severity-critical"
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        </div>

        {/* Events and conditions preview */}
        <div className="mt-4 pt-4 border-t border-gray-800">
          <div className="flex flex-wrap gap-2">
            {monitor.events.slice(0, 3).map((event, index) => (
              <Badge key={index} variant="secondary" className="text-xs">
                {event}
              </Badge>
            ))}
            {monitor.events.length > 3 && (
              <Badge variant="secondary" className="text-xs">
                +{monitor.events.length - 3} more
              </Badge>
            )}
          </div>
        </div>

        {/* Webhook status */}
        {monitor.webhook_url && (
          <div className="mt-3 flex items-center gap-2 text-xs">
            <div className="h-1.5 w-1.5 rounded-full bg-accent-green" />
            <span className="text-gray-500">Webhook configured</span>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function CreateMonitorModal({
  onClose,
  onSuccess,
}: {
  onClose: () => void;
  onSuccess: (monitor: Monitor) => void;
}) {
  const [name, setName] = useState('');
  const [contractAddress, setContractAddress] = useState('');
  const [chain, setChain] = useState<Chain>('ethereum');
  const [events, setEvents] = useState('');
  const [webhookUrl, setWebhookUrl] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError(null);

    const eventList = events
      .split(',')
      .map((e) => e.trim())
      .filter((e) => e);

    const result = await api.createMonitor({
      name,
      contract_address: contractAddress,
      chain,
      events: eventList,
      conditions: {},
      webhook_url: webhookUrl || undefined,
    });

    if (result.data) {
      onSuccess(result.data);
    } else {
      setError(result.error || 'Failed to create monitor');
    }

    setIsLoading(false);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Create Monitor</CardTitle>
          <CardDescription>
            Set up real-time alerts for contract events
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
                placeholder="My Monitor"
                required
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-300">Contract Address</label>
              <Input
                value={contractAddress}
                onChange={(e) => setContractAddress(e.target.value)}
                placeholder="0x..."
                className="font-mono"
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

            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-300">Events</label>
              <Input
                value={events}
                onChange={(e) => setEvents(e.target.value)}
                placeholder="Transfer, Approval, Swap"
              />
              <p className="text-xs text-gray-500">Comma-separated event names</p>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-300">Webhook URL (optional)</label>
              <Input
                value={webhookUrl}
                onChange={(e) => setWebhookUrl(e.target.value)}
                placeholder="https://..."
                type="url"
              />
            </div>

            <div className="flex gap-3 pt-4">
              <Button type="button" variant="secondary" onClick={onClose} className="flex-1">
                Cancel
              </Button>
              <Button type="submit" isLoading={isLoading} className="flex-1">
                Create Monitor
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
