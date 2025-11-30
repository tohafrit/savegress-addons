import Link from 'next/link';
import { ArrowLeft, ArrowRight, Terminal, Settings, Zap } from 'lucide-react';

export default function VSCodePage() {
  return (
    <div className="max-w-4xl mx-auto">
      <Link
        href="/docs"
        className="inline-flex items-center text-gray-400 hover:text-white mb-6 transition-colors"
      >
        <ArrowLeft className="mr-2 h-4 w-4" />
        Back to Documentation
      </Link>

      <h1 className="text-3xl font-bold text-white mb-4">VS Code Extension</h1>
      <p className="text-gray-400 text-lg mb-8">
        Get real-time security analysis and gas optimization directly in your editor.
      </p>

      {/* Installation */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4 flex items-center">
          <Terminal className="mr-2 h-5 w-5 text-accent-cyan" />
          Installation
        </h2>
        <div className="rounded-lg border border-gray-800 bg-dark-bg p-4 mb-4">
          <p className="text-gray-300 mb-4">Install from VS Code Marketplace:</p>
          <ol className="list-decimal list-inside text-gray-300 space-y-2">
            <li>Open VS Code</li>
            <li>Go to Extensions (Cmd+Shift+X / Ctrl+Shift+X)</li>
            <li>Search for &quot;GetChainLens&quot;</li>
            <li>Click Install</li>
          </ol>
        </div>
        <p className="text-gray-400 text-sm">
          Or install directly from the{' '}
          <a
            href="https://marketplace.visualstudio.com/items?itemName=getchainlens.getchainlens"
            target="_blank"
            rel="noopener noreferrer"
            className="text-accent-cyan hover:underline"
          >
            VS Code Marketplace
          </a>
        </p>
      </section>

      {/* Authentication */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4 flex items-center">
          <Settings className="mr-2 h-5 w-5 text-accent-cyan" />
          Authentication
        </h2>
        <div className="rounded-lg border border-gray-800 bg-dark-bg p-4 mb-4">
          <ol className="list-decimal list-inside text-gray-300 space-y-2">
            <li>Open Command Palette (Cmd+Shift+P / Ctrl+Shift+P)</li>
            <li>Type &quot;ChainLens: Login&quot;</li>
            <li>Enter your GetChainLens credentials</li>
            <li>You&apos;ll see a confirmation in the status bar</li>
          </ol>
        </div>
        <p className="text-gray-400 text-sm">
          Your API token is stored securely in VS Code&apos;s credential storage.
        </p>
      </section>

      {/* Features */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4 flex items-center">
          <Zap className="mr-2 h-5 w-5 text-accent-cyan" />
          Features
        </h2>
        <div className="space-y-4">
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">Real-time Diagnostics</h3>
            <p className="text-gray-400 text-sm">
              Security issues and warnings appear as you type, with inline suggestions and quick fixes.
            </p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">Gas Estimates on Hover</h3>
            <p className="text-gray-400 text-sm">
              Hover over any function to see estimated gas costs and optimization tips.
            </p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">Contract Explorer</h3>
            <p className="text-gray-400 text-sm">
              View all contracts in your workspace with their security status in the sidebar.
            </p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">One-Click Analysis</h3>
            <p className="text-gray-400 text-sm">
              Run full security analysis from the context menu or command palette.
            </p>
          </div>
        </div>
      </section>

      {/* Commands */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4">Available Commands</h2>
        <div className="rounded-lg border border-gray-800 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-dark-bg">
              <tr>
                <th className="text-left text-gray-400 font-medium p-4">Command</th>
                <th className="text-left text-gray-400 font-medium p-4">Description</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-800">
              <tr>
                <td className="p-4 font-mono text-accent-cyan">ChainLens: Analyze Contract</td>
                <td className="p-4 text-gray-300">Run security analysis on current file</td>
              </tr>
              <tr>
                <td className="p-4 font-mono text-accent-cyan">ChainLens: Show Gas Report</td>
                <td className="p-4 text-gray-300">Display gas estimates for all functions</td>
              </tr>
              <tr>
                <td className="p-4 font-mono text-accent-cyan">ChainLens: Login</td>
                <td className="p-4 text-gray-300">Authenticate with GetChainLens</td>
              </tr>
              <tr>
                <td className="p-4 font-mono text-accent-cyan">ChainLens: Logout</td>
                <td className="p-4 text-gray-300">Sign out of GetChainLens</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Next Steps */}
      <section className="rounded-xl border border-gray-800 bg-dark-card p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Next Steps</h2>
        <div className="grid gap-4 md:grid-cols-2">
          <Link
            href="/docs/security"
            className="flex items-center justify-between rounded-lg border border-gray-700 p-4 hover:border-accent-cyan hover:bg-dark-bg transition-all group"
          >
            <span className="text-gray-300 group-hover:text-white">Security Analysis</span>
            <ArrowRight className="h-4 w-4 text-gray-500 group-hover:text-accent-cyan" />
          </Link>
          <Link
            href="/docs/gas"
            className="flex items-center justify-between rounded-lg border border-gray-700 p-4 hover:border-accent-cyan hover:bg-dark-bg transition-all group"
          >
            <span className="text-gray-300 group-hover:text-white">Gas Optimization</span>
            <ArrowRight className="h-4 w-4 text-gray-500 group-hover:text-accent-cyan" />
          </Link>
        </div>
      </section>
    </div>
  );
}
