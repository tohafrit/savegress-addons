import Link from 'next/link';
import { ArrowLeft, Key, Globe, FileJson } from 'lucide-react';

export default function ApiPage() {
  return (
    <div className="max-w-4xl mx-auto">
      <Link
        href="/docs"
        className="inline-flex items-center text-gray-400 hover:text-white mb-6 transition-colors"
      >
        <ArrowLeft className="mr-2 h-4 w-4" />
        Back to Documentation
      </Link>

      <h1 className="text-3xl font-bold text-white mb-4">REST API Reference</h1>
      <p className="text-gray-400 text-lg mb-8">
        Programmatic access to GetChainLens features for automation and integrations.
      </p>

      {/* Authentication */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4 flex items-center">
          <Key className="mr-2 h-5 w-5 text-accent-cyan" />
          Authentication
        </h2>
        <p className="text-gray-300 mb-4">
          All API requests require an API key passed in the Authorization header:
        </p>
        <div className="rounded-lg border border-gray-800 bg-dark-bg p-4 mb-4 overflow-x-auto">
          <code className="text-sm text-accent-cyan">
            Authorization: Bearer YOUR_API_KEY
          </code>
        </div>
        <p className="text-gray-400 text-sm">
          Generate API keys in <Link href="/settings" className="text-accent-cyan hover:underline">Settings</Link>.
        </p>
      </section>

      {/* Base URL */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4 flex items-center">
          <Globe className="mr-2 h-5 w-5 text-accent-cyan" />
          Base URL
        </h2>
        <div className="rounded-lg border border-gray-800 bg-dark-bg p-4 overflow-x-auto">
          <code className="text-sm text-white">https://api.getchainlens.com/v1</code>
        </div>
      </section>

      {/* Endpoints */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4 flex items-center">
          <FileJson className="mr-2 h-5 w-5 text-accent-cyan" />
          Endpoints
        </h2>

        {/* Analyze Contract */}
        <div className="mb-6 rounded-lg border border-gray-800 overflow-hidden">
          <div className="bg-dark-bg p-4 border-b border-gray-800">
            <div className="flex items-center gap-3">
              <span className="px-2 py-1 rounded text-xs font-mono bg-accent-green/20 text-accent-green">POST</span>
              <code className="text-white">/contracts/analyze</code>
            </div>
            <p className="text-gray-400 text-sm mt-2">Analyze a Solidity contract for security issues</p>
          </div>
          <div className="p-4">
            <h4 className="text-sm font-medium text-gray-400 mb-2">Request Body</h4>
            <pre className="text-sm text-gray-300 bg-dark-bg rounded p-3 overflow-x-auto">
{`{
  "source_code": "pragma solidity ^0.8.0;...",
  "contract_name": "MyContract",
  "compiler_version": "0.8.20"
}`}
            </pre>
            <h4 className="text-sm font-medium text-gray-400 mt-4 mb-2">Response</h4>
            <pre className="text-sm text-gray-300 bg-dark-bg rounded p-3 overflow-x-auto">
{`{
  "id": "analysis_abc123",
  "status": "completed",
  "issues": [
    {
      "severity": "high",
      "title": "Reentrancy vulnerability",
      "line": 42,
      "description": "..."
    }
  ],
  "gas_estimates": {...}
}`}
            </pre>
          </div>
        </div>

        {/* Get Analysis */}
        <div className="mb-6 rounded-lg border border-gray-800 overflow-hidden">
          <div className="bg-dark-bg p-4 border-b border-gray-800">
            <div className="flex items-center gap-3">
              <span className="px-2 py-1 rounded text-xs font-mono bg-accent-blue/20 text-accent-blue">GET</span>
              <code className="text-white">/contracts/analysis/:id</code>
            </div>
            <p className="text-gray-400 text-sm mt-2">Get analysis results by ID</p>
          </div>
          <div className="p-4">
            <h4 className="text-sm font-medium text-gray-400 mb-2">Response</h4>
            <pre className="text-sm text-gray-300 bg-dark-bg rounded p-3 overflow-x-auto">
{`{
  "id": "analysis_abc123",
  "status": "completed",
  "created_at": "2024-01-15T10:30:00Z",
  "issues": [...],
  "gas_estimates": {...}
}`}
            </pre>
          </div>
        </div>

        {/* Trace Transaction */}
        <div className="mb-6 rounded-lg border border-gray-800 overflow-hidden">
          <div className="bg-dark-bg p-4 border-b border-gray-800">
            <div className="flex items-center gap-3">
              <span className="px-2 py-1 rounded text-xs font-mono bg-accent-green/20 text-accent-green">POST</span>
              <code className="text-white">/traces</code>
            </div>
            <p className="text-gray-400 text-sm mt-2">Trace a transaction</p>
          </div>
          <div className="p-4">
            <h4 className="text-sm font-medium text-gray-400 mb-2">Request Body</h4>
            <pre className="text-sm text-gray-300 bg-dark-bg rounded p-3 overflow-x-auto">
{`{
  "tx_hash": "0xabcd...ef01",
  "network": "ethereum"
}`}
            </pre>
            <h4 className="text-sm font-medium text-gray-400 mt-4 mb-2">Response</h4>
            <pre className="text-sm text-gray-300 bg-dark-bg rounded p-3 overflow-x-auto">
{`{
  "tx_hash": "0xabcd...ef01",
  "status": "success",
  "gas_used": 85000,
  "calls": [...],
  "events": [...],
  "state_changes": [...]
}`}
            </pre>
          </div>
        </div>

        {/* Create Monitor */}
        <div className="mb-6 rounded-lg border border-gray-800 overflow-hidden">
          <div className="bg-dark-bg p-4 border-b border-gray-800">
            <div className="flex items-center gap-3">
              <span className="px-2 py-1 rounded text-xs font-mono bg-accent-green/20 text-accent-green">POST</span>
              <code className="text-white">/monitors</code>
            </div>
            <p className="text-gray-400 text-sm mt-2">Create a new contract monitor</p>
          </div>
          <div className="p-4">
            <h4 className="text-sm font-medium text-gray-400 mb-2">Request Body</h4>
            <pre className="text-sm text-gray-300 bg-dark-bg rounded p-3 overflow-x-auto">
{`{
  "name": "My Token Monitor",
  "contract_address": "0x1234...5678",
  "network": "ethereum",
  "events": ["Transfer", "Approval"],
  "webhook_url": "https://example.com/webhook"
}`}
            </pre>
          </div>
        </div>
      </section>

      {/* Rate Limits */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4">Rate Limits</h2>
        <div className="rounded-lg border border-gray-800 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-dark-bg">
              <tr>
                <th className="text-left text-gray-400 font-medium p-4">Plan</th>
                <th className="text-left text-gray-400 font-medium p-4">Requests/min</th>
                <th className="text-left text-gray-400 font-medium p-4">Requests/day</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-800">
              <tr>
                <td className="p-4 text-gray-300">Free</td>
                <td className="p-4 text-white">10</td>
                <td className="p-4 text-white">100</td>
              </tr>
              <tr>
                <td className="p-4 text-gray-300">Pro</td>
                <td className="p-4 text-white">60</td>
                <td className="p-4 text-white">10,000</td>
              </tr>
              <tr>
                <td className="p-4 text-gray-300">Enterprise</td>
                <td className="p-4 text-white">Custom</td>
                <td className="p-4 text-white">Custom</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Errors */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4">Error Codes</h2>
        <div className="rounded-lg border border-gray-800 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-dark-bg">
              <tr>
                <th className="text-left text-gray-400 font-medium p-4">Code</th>
                <th className="text-left text-gray-400 font-medium p-4">Description</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-800">
              <tr>
                <td className="p-4 font-mono text-severity-high">400</td>
                <td className="p-4 text-gray-300">Bad Request - Invalid parameters</td>
              </tr>
              <tr>
                <td className="p-4 font-mono text-severity-high">401</td>
                <td className="p-4 text-gray-300">Unauthorized - Invalid or missing API key</td>
              </tr>
              <tr>
                <td className="p-4 font-mono text-severity-medium">429</td>
                <td className="p-4 text-gray-300">Too Many Requests - Rate limit exceeded</td>
              </tr>
              <tr>
                <td className="p-4 font-mono text-severity-critical">500</td>
                <td className="p-4 text-gray-300">Internal Server Error</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
