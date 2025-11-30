import Link from 'next/link';
import { ArrowLeft, Webhook, Shield, RefreshCw } from 'lucide-react';

export default function WebhooksPage() {
  return (
    <div className="max-w-4xl mx-auto">
      <Link
        href="/docs"
        className="inline-flex items-center text-gray-400 hover:text-white mb-6 transition-colors"
      >
        <ArrowLeft className="mr-2 h-4 w-4" />
        Back to Documentation
      </Link>

      <h1 className="text-3xl font-bold text-white mb-4">Webhooks</h1>
      <p className="text-gray-400 text-lg mb-8">
        Receive real-time notifications when events occur on your monitored contracts.
      </p>

      {/* Setup */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4 flex items-center">
          <Webhook className="mr-2 h-5 w-5 text-accent-orange" />
          Setting Up Webhooks
        </h2>
        <div className="rounded-lg border border-gray-800 bg-dark-bg p-4 mb-4">
          <ol className="list-decimal list-inside text-gray-300 space-y-2">
            <li>Create an endpoint on your server to receive POST requests</li>
            <li>Add the URL when creating a monitor</li>
            <li>Verify the webhook is working using the Test button</li>
            <li>Respond with 2xx status to acknowledge receipt</li>
          </ol>
        </div>
      </section>

      {/* Payload Format */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4">Payload Format</h2>
        <p className="text-gray-300 mb-4">All webhooks are sent as POST requests with JSON body:</p>

        <h3 className="text-lg font-medium text-white mt-6 mb-3">Contract Event</h3>
        <div className="rounded-lg border border-gray-800 bg-dark-bg p-4 overflow-x-auto mb-4">
          <pre className="text-sm text-gray-300">
{`{
  "id": "evt_abc123",
  "type": "contract.event",
  "created_at": "2024-01-15T10:30:00Z",
  "monitor_id": "mon_xyz789",
  "data": {
    "network": "ethereum",
    "contract_address": "0x1234...5678",
    "transaction_hash": "0xabcd...ef01",
    "block_number": 18500000,
    "block_timestamp": "2024-01-15T10:29:55Z",
    "event_name": "Transfer",
    "event_signature": "Transfer(address,address,uint256)",
    "args": {
      "from": "0x1111...1111",
      "to": "0x2222...2222",
      "value": "1000000000000000000"
    },
    "log_index": 5
  }
}`}
          </pre>
        </div>

        <h3 className="text-lg font-medium text-white mt-6 mb-3">Transaction Failed</h3>
        <div className="rounded-lg border border-gray-800 bg-dark-bg p-4 overflow-x-auto mb-4">
          <pre className="text-sm text-gray-300">
{`{
  "id": "evt_def456",
  "type": "transaction.failed",
  "created_at": "2024-01-15T10:30:00Z",
  "monitor_id": "mon_xyz789",
  "data": {
    "network": "ethereum",
    "contract_address": "0x1234...5678",
    "transaction_hash": "0xffff...0001",
    "block_number": 18500001,
    "from": "0x3333...3333",
    "revert_reason": "ERC20: insufficient balance",
    "gas_used": 45000
  }
}`}
          </pre>
        </div>
      </section>

      {/* Verification */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4 flex items-center">
          <Shield className="mr-2 h-5 w-5 text-accent-cyan" />
          Verifying Webhooks
        </h2>
        <p className="text-gray-300 mb-4">
          Each webhook includes a signature header to verify authenticity:
        </p>
        <div className="rounded-lg border border-gray-800 bg-dark-bg p-4 overflow-x-auto mb-4">
          <code className="text-sm text-accent-cyan">
            X-ChainLens-Signature: sha256=abc123...
          </code>
        </div>
        <p className="text-gray-300 mb-4">
          To verify, compute HMAC-SHA256 of the raw request body using your webhook secret:
        </p>
        <div className="rounded-lg border border-gray-800 bg-dark-bg p-4 overflow-x-auto">
          <pre className="text-sm text-gray-300">
{`const crypto = require('crypto');

function verifyWebhook(body, signature, secret) {
  const expected = 'sha256=' + crypto
    .createHmac('sha256', secret)
    .update(body, 'utf8')
    .digest('hex');

  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(expected)
  );
}`}
          </pre>
        </div>
      </section>

      {/* Retry Policy */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4 flex items-center">
          <RefreshCw className="mr-2 h-5 w-5 text-accent-cyan" />
          Retry Policy
        </h2>
        <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
          <p className="text-gray-300 mb-4">
            If your endpoint returns a non-2xx status or times out, we&apos;ll retry with exponential backoff:
          </p>
          <ul className="list-disc list-inside text-gray-300 space-y-1">
            <li>1st retry: 1 minute</li>
            <li>2nd retry: 5 minutes</li>
            <li>3rd retry: 30 minutes</li>
            <li>4th retry: 2 hours</li>
            <li>5th retry: 24 hours</li>
          </ul>
          <p className="text-gray-400 text-sm mt-4">
            After 5 failed attempts, the webhook is marked as failed and we send an email notification.
          </p>
        </div>
      </section>

      {/* Best Practices */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4">Best Practices</h2>
        <div className="space-y-3">
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <p className="text-gray-300">
              <strong className="text-white">Respond quickly:</strong> Return 200 immediately and process asynchronously. Webhooks timeout after 30 seconds.
            </p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <p className="text-gray-300">
              <strong className="text-white">Handle duplicates:</strong> Use the event ID to deduplicate. The same event may be delivered multiple times.
            </p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <p className="text-gray-300">
              <strong className="text-white">Always verify signatures:</strong> Ensure webhooks are from GetChainLens, not malicious actors.
            </p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <p className="text-gray-300">
              <strong className="text-white">Use HTTPS:</strong> Webhook URLs must use HTTPS for security.
            </p>
          </div>
        </div>
      </section>

      {/* Event Types */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4">Event Types</h2>
        <div className="rounded-lg border border-gray-800 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-dark-bg">
              <tr>
                <th className="text-left text-gray-400 font-medium p-4">Type</th>
                <th className="text-left text-gray-400 font-medium p-4">Description</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-800">
              <tr>
                <td className="p-4 font-mono text-accent-cyan">contract.event</td>
                <td className="p-4 text-gray-300">A monitored event was emitted</td>
              </tr>
              <tr>
                <td className="p-4 font-mono text-accent-cyan">transaction.failed</td>
                <td className="p-4 text-gray-300">A transaction to the contract failed</td>
              </tr>
              <tr>
                <td className="p-4 font-mono text-accent-cyan">balance.changed</td>
                <td className="p-4 text-gray-300">Contract balance crossed threshold</td>
              </tr>
              <tr>
                <td className="p-4 font-mono text-accent-cyan">function.called</td>
                <td className="p-4 text-gray-300">A monitored function was called</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
