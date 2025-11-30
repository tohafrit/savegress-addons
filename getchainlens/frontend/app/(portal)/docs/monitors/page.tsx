import Link from 'next/link';
import { ArrowLeft, ArrowRight, Activity, Bell, Webhook, Mail } from 'lucide-react';

export default function MonitorsPage() {
  return (
    <div className="max-w-4xl mx-auto">
      <Link
        href="/docs"
        className="inline-flex items-center text-gray-400 hover:text-white mb-6 transition-colors"
      >
        <ArrowLeft className="mr-2 h-4 w-4" />
        Back to Documentation
      </Link>

      <h1 className="text-3xl font-bold text-white mb-4">Monitors & Alerts</h1>
      <p className="text-gray-400 text-lg mb-8">
        Set up real-time monitoring for your deployed contracts and receive instant alerts.
      </p>

      {/* Creating a Monitor */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4 flex items-center">
          <Activity className="mr-2 h-5 w-5 text-accent-green" />
          Creating a Monitor
        </h2>
        <div className="rounded-lg border border-gray-800 bg-dark-bg p-4 mb-4">
          <ol className="list-decimal list-inside text-gray-300 space-y-2">
            <li>Go to <Link href="/monitors" className="text-accent-cyan hover:underline">Monitors</Link></li>
            <li>Click &quot;Create Monitor&quot;</li>
            <li>Enter your contract address</li>
            <li>Select the network</li>
            <li>Choose events to monitor</li>
            <li>Configure alert destinations</li>
            <li>Click &quot;Create&quot;</li>
          </ol>
        </div>
      </section>

      {/* What Can You Monitor */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4 flex items-center">
          <Bell className="mr-2 h-5 w-5 text-accent-cyan" />
          What Can You Monitor
        </h2>
        <div className="space-y-4">
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">Contract Events</h3>
            <p className="text-gray-400 text-sm">
              Get notified when specific events are emitted (e.g., Transfer, Approval, OwnershipTransferred).
            </p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">Function Calls</h3>
            <p className="text-gray-400 text-sm">
              Track when specific functions are called on your contract.
            </p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">Balance Changes</h3>
            <p className="text-gray-400 text-sm">
              Monitor ETH or token balance changes above a threshold.
            </p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">Failed Transactions</h3>
            <p className="text-gray-400 text-sm">
              Get alerted when transactions to your contract fail or revert.
            </p>
          </div>
        </div>
      </section>

      {/* Alert Destinations */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4">Alert Destinations</h2>
        <div className="grid md:grid-cols-2 gap-4">
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <div className="flex items-center gap-3 mb-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-cyan/20">
                <Webhook className="h-5 w-5 text-accent-cyan" />
              </div>
              <h3 className="font-medium text-white">Webhook</h3>
            </div>
            <p className="text-gray-400 text-sm">
              Send JSON payloads to your own endpoints for custom integrations.
            </p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <div className="flex items-center gap-3 mb-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-blue/20">
                <Mail className="h-5 w-5 text-accent-blue" />
              </div>
              <h3 className="font-medium text-white">Email</h3>
            </div>
            <p className="text-gray-400 text-sm">
              Receive email notifications for important events.
            </p>
          </div>
        </div>
      </section>

      {/* Webhook Payload */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4">Webhook Payload Example</h2>
        <div className="rounded-lg border border-gray-800 bg-dark-bg p-4 overflow-x-auto">
          <pre className="text-sm text-gray-300">
{`{
  "monitor_id": "mon_abc123",
  "event_type": "contract_event",
  "timestamp": "2024-01-15T10:30:00Z",
  "network": "ethereum",
  "contract_address": "0x1234...5678",
  "transaction_hash": "0xabcd...ef01",
  "block_number": 18500000,
  "event": {
    "name": "Transfer",
    "args": {
      "from": "0x1111...1111",
      "to": "0x2222...2222",
      "value": "1000000000000000000"
    }
  }
}`}
          </pre>
        </div>
      </section>

      {/* Best Practices */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4">Best Practices</h2>
        <div className="space-y-3">
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <p className="text-gray-300">
              <strong className="text-white">Use filters:</strong> Set up conditions to avoid alert fatigue (e.g., only transfers above 1 ETH).
            </p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <p className="text-gray-300">
              <strong className="text-white">Multiple destinations:</strong> Send critical alerts to both webhook and email for redundancy.
            </p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <p className="text-gray-300">
              <strong className="text-white">Test webhooks:</strong> Use the &quot;Test&quot; button to verify your endpoint receives payloads correctly.
            </p>
          </div>
        </div>
      </section>

      {/* Next Steps */}
      <section className="rounded-xl border border-gray-800 bg-dark-card p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Next Steps</h2>
        <div className="grid gap-4 md:grid-cols-2">
          <Link
            href="/docs/webhooks"
            className="flex items-center justify-between rounded-lg border border-gray-700 p-4 hover:border-accent-cyan hover:bg-dark-bg transition-all group"
          >
            <span className="text-gray-300 group-hover:text-white">Webhook Reference</span>
            <ArrowRight className="h-4 w-4 text-gray-500 group-hover:text-accent-cyan" />
          </Link>
          <Link
            href="/monitors"
            className="flex items-center justify-between rounded-lg border border-gray-700 p-4 hover:border-accent-cyan hover:bg-dark-bg transition-all group"
          >
            <span className="text-gray-300 group-hover:text-white">Create a Monitor</span>
            <ArrowRight className="h-4 w-4 text-gray-500 group-hover:text-accent-cyan" />
          </Link>
        </div>
      </section>
    </div>
  );
}
