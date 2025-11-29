'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import {
  ArrowLeft,
  Bell,
  Settings,
  Pause,
  Play,
  Trash2,
  Save,
  Clock,
  Zap,
  CheckCircle2,
  AlertTriangle,
  ExternalLink,
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
import type { Monitor, MonitorAlert, Chain } from '@/types';

// Mock alerts for demo
const mockAlerts: MonitorAlert[] = [
  {
    id: '1',
    monitor_id: '1',
    event_name: 'Transfer',
    transaction_hash: '0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef',
    block_number: 18500000,
    data: { from: '0xabc...', to: '0xdef...', value: '1000000000000000000' },
    triggered_at: new Date(Date.now() - 1000 * 60 * 5).toISOString(),
    delivered: true,
  },
  {
    id: '2',
    monitor_id: '1',
    event_name: 'Approval',
    transaction_hash: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
    block_number: 18499999,
    data: { owner: '0x123...', spender: '0x456...', value: '115792089237316195423570985008687907853269984665640564039457584007913129639935' },
    triggered_at: new Date(Date.now() - 1000 * 60 * 30).toISOString(),
    delivered: true,
  },
  {
    id: '3',
    monitor_id: '1',
    event_name: 'Transfer',
    transaction_hash: '0x9876543210fedcba9876543210fedcba9876543210fedcba9876543210fedcba',
    block_number: 18499950,
    data: { from: '0xfed...', to: '0xcba...', value: '500000000000000000' },
    triggered_at: new Date(Date.now() - 1000 * 60 * 60 * 2).toISOString(),
    delivered: false,
  },
];

export default function MonitorDetailPage() {
  const params = useParams();
  const router = useRouter();
  const monitorId = params.id as string;

  const [monitor, setMonitor] = useState<Monitor | null>(null);
  const [alerts, setAlerts] = useState<MonitorAlert[]>(mockAlerts);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [isEditing, setIsEditing] = useState(false);

  // Edit form state
  const [name, setName] = useState('');
  const [events, setEvents] = useState('');
  const [webhookUrl, setWebhookUrl] = useState('');

  useEffect(() => {
    async function loadMonitor() {
      const result = await api.getMonitor(monitorId);
      if (result.data) {
        setMonitor(result.data);
        setName(result.data.name);
        setEvents(result.data.events.join(', '));
        setWebhookUrl(result.data.webhook_url || '');
      }
      setIsLoading(false);
    }
    loadMonitor();
  }, [monitorId]);

  const handleSave = async () => {
    if (!monitor) return;

    setIsSaving(true);
    const eventList = events
      .split(',')
      .map((e) => e.trim())
      .filter((e) => e);

    const result = await api.updateMonitor(monitorId, {
      name,
      events: eventList,
      webhook_url: webhookUrl || undefined,
    });

    if (result.data) {
      setMonitor(result.data);
      setIsEditing(false);
    }
    setIsSaving(false);
  };

  const handleToggleStatus = async () => {
    if (!monitor) return;

    const newStatus = monitor.status === 'active' ? 'paused' : 'active';
    const result = await api.updateMonitor(monitorId, { status: newStatus });
    if (result.data) {
      setMonitor(result.data);
    }
  };

  const handleDelete = async () => {
    if (!confirm('Are you sure you want to delete this monitor? This action cannot be undone.')) return;

    const result = await api.deleteMonitor(monitorId);
    if (!result.error) {
      router.push('/monitors');
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-accent-cyan border-t-transparent" />
      </div>
    );
  }

  if (!monitor) {
    return (
      <div className="text-center py-12">
        <h2 className="text-xl font-semibold text-white">Monitor not found</h2>
        <Button variant="secondary" className="mt-4" asChild>
          <Link href="/monitors">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Monitors
          </Link>
        </Button>
      </div>
    );
  }

  const isActive = monitor.status === 'active';

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href="/monitors">
            <ArrowLeft className="h-5 w-5" />
          </Link>
        </Button>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-white">{monitor.name}</h1>
            <Badge variant={isActive ? 'success' : 'secondary'}>
              {isActive ? 'Active' : 'Paused'}
            </Badge>
          </div>
          <p className="mt-1 text-gray-400 font-mono text-sm">
            {formatAddress(monitor.contract_address, 12)}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="secondary"
            onClick={handleToggleStatus}
          >
            {isActive ? (
              <>
                <Pause className="mr-2 h-4 w-4" />
                Pause
              </>
            ) : (
              <>
                <Play className="mr-2 h-4 w-4" />
                Activate
              </>
            )}
          </Button>
          <Button variant="destructive" onClick={handleDelete}>
            <Trash2 className="mr-2 h-4 w-4" />
            Delete
          </Button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid gap-4 sm:grid-cols-4">
        <Card>
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-cyan/20">
                <Zap className="h-5 w-5 text-accent-cyan" />
              </div>
              <span className="text-2xl font-bold text-white">{alerts.length}</span>
            </div>
            <p className="mt-2 text-sm text-gray-400">Total Alerts</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-green/20">
                <CheckCircle2 className="h-5 w-5 text-accent-green" />
              </div>
              <span className="text-2xl font-bold text-white">
                {alerts.filter((a) => a.delivered).length}
              </span>
            </div>
            <p className="mt-2 text-sm text-gray-400">Delivered</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-severity-critical/20">
                <AlertTriangle className="h-5 w-5 text-severity-critical" />
              </div>
              <span className="text-2xl font-bold text-white">
                {alerts.filter((a) => !a.delivered).length}
              </span>
            </div>
            <p className="mt-2 text-sm text-gray-400">Failed</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-blue/20">
                <Clock className="h-5 w-5 text-accent-blue" />
              </div>
              <span className="text-sm font-medium text-white">
                {formatTimeAgo(new Date(monitor.created_at))}
              </span>
            </div>
            <p className="mt-2 text-sm text-gray-400">Created</p>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Configuration */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2">
                <Settings className="h-5 w-5 text-gray-400" />
                Configuration
              </CardTitle>
              {!isEditing ? (
                <Button variant="secondary" size="sm" onClick={() => setIsEditing(true)}>
                  Edit
                </Button>
              ) : (
                <div className="flex gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setIsEditing(false);
                      setName(monitor.name);
                      setEvents(monitor.events.join(', '));
                      setWebhookUrl(monitor.webhook_url || '');
                    }}
                  >
                    Cancel
                  </Button>
                  <Button size="sm" onClick={handleSave} isLoading={isSaving}>
                    <Save className="mr-1.5 h-3.5 w-3.5" />
                    Save
                  </Button>
                </div>
              )}
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-300">Name</label>
              {isEditing ? (
                <Input value={name} onChange={(e) => setName(e.target.value)} />
              ) : (
                <p className="text-white">{monitor.name}</p>
              )}
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-300">Contract Address</label>
              <p className="text-white font-mono text-sm">
                {monitor.contract_address}
              </p>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-300">Chain</label>
              <Badge variant={monitor.chain as Chain}>
                {getChainName(monitor.chain)}
              </Badge>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-300">Events</label>
              {isEditing ? (
                <>
                  <Input value={events} onChange={(e) => setEvents(e.target.value)} />
                  <p className="text-xs text-gray-500">Comma-separated event names</p>
                </>
              ) : (
                <div className="flex flex-wrap gap-2">
                  {monitor.events.map((event, index) => (
                    <Badge key={index} variant="secondary">
                      {event}
                    </Badge>
                  ))}
                </div>
              )}
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-300">Webhook URL</label>
              {isEditing ? (
                <Input
                  value={webhookUrl}
                  onChange={(e) => setWebhookUrl(e.target.value)}
                  placeholder="https://..."
                  type="url"
                />
              ) : monitor.webhook_url ? (
                <p className="text-white text-sm truncate">{monitor.webhook_url}</p>
              ) : (
                <p className="text-gray-500 text-sm">Not configured</p>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Recent Alerts */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Bell className="h-5 w-5 text-gray-400" />
              Recent Alerts
            </CardTitle>
            <CardDescription>
              Latest events triggered by this monitor
            </CardDescription>
          </CardHeader>
          <CardContent>
            {alerts.length > 0 ? (
              <div className="space-y-3">
                {alerts.map((alert) => (
                  <div
                    key={alert.id}
                    className="rounded-lg border border-gray-800 p-3 hover:border-gray-700 transition-colors"
                  >
                    <div className="flex items-start justify-between">
                      <div>
                        <div className="flex items-center gap-2">
                          <Badge variant="secondary" className="text-xs">
                            {alert.event_name}
                          </Badge>
                          {alert.delivered ? (
                            <CheckCircle2 className="h-3.5 w-3.5 text-accent-green" />
                          ) : (
                            <AlertTriangle className="h-3.5 w-3.5 text-severity-critical" />
                          )}
                        </div>
                        <p className="mt-1 text-xs text-gray-500 font-mono">
                          {formatAddress(alert.transaction_hash, 12)}
                        </p>
                        <p className="text-xs text-gray-500 mt-1">
                          Block {alert.block_number.toLocaleString()} • {formatTimeAgo(new Date(alert.triggered_at))}
                        </p>
                      </div>
                      <Button variant="ghost" size="icon-sm" asChild>
                        <Link href={`/traces/${alert.transaction_hash}?chain=${monitor.chain}`}>
                          <ExternalLink className="h-3.5 w-3.5" />
                        </Link>
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-8">
                <Bell className="mx-auto h-10 w-10 text-gray-600" />
                <p className="mt-3 text-gray-400">No alerts yet</p>
                <p className="text-sm text-gray-500">
                  Alerts will appear here when events are detected
                </p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
