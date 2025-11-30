import Link from 'next/link';
import { ArrowLeft, ArrowRight, Shield, AlertTriangle, AlertCircle, Info } from 'lucide-react';

export default function SecurityPage() {
  return (
    <div className="max-w-4xl mx-auto">
      <Link
        href="/docs"
        className="inline-flex items-center text-gray-400 hover:text-white mb-6 transition-colors"
      >
        <ArrowLeft className="mr-2 h-4 w-4" />
        Back to Documentation
      </Link>

      <h1 className="text-3xl font-bold text-white mb-4">Security Analysis</h1>
      <p className="text-gray-400 text-lg mb-8">
        GetChainLens detects common smart contract vulnerabilities using static analysis.
      </p>

      {/* Severity Levels */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4">Severity Levels</h2>
        <div className="space-y-3">
          <div className="flex items-start gap-3 rounded-lg border border-severity-critical/30 bg-severity-critical/10 p-4">
            <AlertCircle className="h-5 w-5 text-severity-critical mt-0.5" />
            <div>
              <h3 className="font-medium text-severity-critical">Critical</h3>
              <p className="text-gray-400 text-sm">Exploitable vulnerabilities that can lead to loss of funds or complete contract compromise.</p>
            </div>
          </div>
          <div className="flex items-start gap-3 rounded-lg border border-severity-high/30 bg-severity-high/10 p-4">
            <AlertTriangle className="h-5 w-5 text-severity-high mt-0.5" />
            <div>
              <h3 className="font-medium text-severity-high">High</h3>
              <p className="text-gray-400 text-sm">Serious issues that could be exploited under certain conditions.</p>
            </div>
          </div>
          <div className="flex items-start gap-3 rounded-lg border border-severity-medium/30 bg-severity-medium/10 p-4">
            <AlertTriangle className="h-5 w-5 text-severity-medium mt-0.5" />
            <div>
              <h3 className="font-medium text-severity-medium">Medium</h3>
              <p className="text-gray-400 text-sm">Issues that could cause problems but are harder to exploit.</p>
            </div>
          </div>
          <div className="flex items-start gap-3 rounded-lg border border-severity-low/30 bg-severity-low/10 p-4">
            <Info className="h-5 w-5 text-severity-low mt-0.5" />
            <div>
              <h3 className="font-medium text-severity-low">Low / Informational</h3>
              <p className="text-gray-400 text-sm">Best practice violations and code quality issues.</p>
            </div>
          </div>
        </div>
      </section>

      {/* Vulnerability Types */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4 flex items-center">
          <Shield className="mr-2 h-5 w-5 text-accent-cyan" />
          Detected Vulnerabilities
        </h2>
        <div className="rounded-lg border border-gray-800 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-dark-bg">
              <tr>
                <th className="text-left text-gray-400 font-medium p-4">Vulnerability</th>
                <th className="text-left text-gray-400 font-medium p-4">Severity</th>
                <th className="text-left text-gray-400 font-medium p-4">Description</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-800">
              <tr>
                <td className="p-4 text-white font-medium">Reentrancy</td>
                <td className="p-4"><span className="px-2 py-1 rounded text-xs bg-severity-critical/20 text-severity-critical">Critical</span></td>
                <td className="p-4 text-gray-300">External calls before state updates allow attackers to re-enter the function</td>
              </tr>
              <tr>
                <td className="p-4 text-white font-medium">Integer Overflow/Underflow</td>
                <td className="p-4"><span className="px-2 py-1 rounded text-xs bg-severity-high/20 text-severity-high">High</span></td>
                <td className="p-4 text-gray-300">Arithmetic operations that can wrap around (pre-Solidity 0.8)</td>
              </tr>
              <tr>
                <td className="p-4 text-white font-medium">Access Control</td>
                <td className="p-4"><span className="px-2 py-1 rounded text-xs bg-severity-high/20 text-severity-high">High</span></td>
                <td className="p-4 text-gray-300">Missing or incorrect access modifiers on sensitive functions</td>
              </tr>
              <tr>
                <td className="p-4 text-white font-medium">Unchecked External Calls</td>
                <td className="p-4"><span className="px-2 py-1 rounded text-xs bg-severity-medium/20 text-severity-medium">Medium</span></td>
                <td className="p-4 text-gray-300">Return values from external calls not checked</td>
              </tr>
              <tr>
                <td className="p-4 text-white font-medium">Tx.origin Authentication</td>
                <td className="p-4"><span className="px-2 py-1 rounded text-xs bg-severity-medium/20 text-severity-medium">Medium</span></td>
                <td className="p-4 text-gray-300">Using tx.origin for authorization instead of msg.sender</td>
              </tr>
              <tr>
                <td className="p-4 text-white font-medium">Floating Pragma</td>
                <td className="p-4"><span className="px-2 py-1 rounded text-xs bg-severity-low/20 text-severity-low">Low</span></td>
                <td className="p-4 text-gray-300">Unlocked compiler version can lead to unexpected behavior</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Best Practices */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4">Best Practices</h2>
        <div className="space-y-4">
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">Use Checks-Effects-Interactions Pattern</h3>
            <p className="text-gray-400 text-sm">
              Always perform checks first, then update state, and finally interact with external contracts.
            </p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">Use ReentrancyGuard</h3>
            <p className="text-gray-400 text-sm">
              Import OpenZeppelin&apos;s ReentrancyGuard for functions that make external calls.
            </p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">Lock Pragma Version</h3>
            <p className="text-gray-400 text-sm">
              Use a specific compiler version (e.g., pragma solidity 0.8.20;) instead of ^0.8.0.
            </p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">Implement Access Control</h3>
            <p className="text-gray-400 text-sm">
              Use OpenZeppelin&apos;s Ownable or AccessControl for managing permissions.
            </p>
          </div>
        </div>
      </section>

      {/* Next Steps */}
      <section className="rounded-xl border border-gray-800 bg-dark-card p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Next Steps</h2>
        <div className="grid gap-4 md:grid-cols-2">
          <Link
            href="/docs/gas"
            className="flex items-center justify-between rounded-lg border border-gray-700 p-4 hover:border-accent-cyan hover:bg-dark-bg transition-all group"
          >
            <span className="text-gray-300 group-hover:text-white">Gas Optimization</span>
            <ArrowRight className="h-4 w-4 text-gray-500 group-hover:text-accent-cyan" />
          </Link>
          <Link
            href="/docs/monitors"
            className="flex items-center justify-between rounded-lg border border-gray-700 p-4 hover:border-accent-cyan hover:bg-dark-bg transition-all group"
          >
            <span className="text-gray-300 group-hover:text-white">Set Up Monitoring</span>
            <ArrowRight className="h-4 w-4 text-gray-500 group-hover:text-accent-cyan" />
          </Link>
        </div>
      </section>
    </div>
  );
}
