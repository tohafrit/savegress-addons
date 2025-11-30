import Link from 'next/link';
import { ArrowLeft, ArrowRight, CheckCircle2, Copy } from 'lucide-react';

export default function QuickStartPage() {
  return (
    <div className="max-w-4xl mx-auto">
      <Link
        href="/docs"
        className="inline-flex items-center text-gray-400 hover:text-white mb-6 transition-colors"
      >
        <ArrowLeft className="mr-2 h-4 w-4" />
        Back to Documentation
      </Link>

      <h1 className="text-3xl font-bold text-white mb-4">Quick Start</h1>
      <p className="text-gray-400 text-lg mb-8">
        Get up and running with GetChainLens in just a few minutes.
      </p>

      {/* Step 1 */}
      <section className="mb-10">
        <div className="flex items-center gap-3 mb-4">
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-accent-cyan text-dark-bg font-bold text-sm">
            1
          </div>
          <h2 className="text-xl font-semibold text-white">Create Your First Project</h2>
        </div>
        <div className="pl-11">
          <p className="text-gray-300 mb-4">
            Projects help you organize your smart contracts. Create one from the Dashboard or Projects page.
          </p>
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4 mb-4">
            <ol className="list-decimal list-inside text-gray-300 space-y-2">
              <li>Go to <Link href="/projects" className="text-accent-cyan hover:underline">Projects</Link></li>
              <li>Click &quot;New Project&quot;</li>
              <li>Enter a name and optional description</li>
              <li>Click &quot;Create&quot;</li>
            </ol>
          </div>
        </div>
      </section>

      {/* Step 2 */}
      <section className="mb-10">
        <div className="flex items-center gap-3 mb-4">
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-accent-cyan text-dark-bg font-bold text-sm">
            2
          </div>
          <h2 className="text-xl font-semibold text-white">Add a Contract</h2>
        </div>
        <div className="pl-11">
          <p className="text-gray-300 mb-4">
            Add your Solidity contracts for security analysis and gas optimization.
          </p>
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4 mb-4">
            <ol className="list-decimal list-inside text-gray-300 space-y-2">
              <li>Open your project</li>
              <li>Click &quot;Add Contract&quot;</li>
              <li>Paste your Solidity source code or upload a file</li>
              <li>Click &quot;Analyze&quot;</li>
            </ol>
          </div>
          <p className="text-gray-400 text-sm">
            Tip: You can also import contracts directly from Etherscan by providing the contract address.
          </p>
        </div>
      </section>

      {/* Step 3 */}
      <section className="mb-10">
        <div className="flex items-center gap-3 mb-4">
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-accent-cyan text-dark-bg font-bold text-sm">
            3
          </div>
          <h2 className="text-xl font-semibold text-white">Review Analysis Results</h2>
        </div>
        <div className="pl-11">
          <p className="text-gray-300 mb-4">
            Once analysis is complete, you&apos;ll see a comprehensive report including:
          </p>
          <div className="grid gap-4 md:grid-cols-2 mb-4">
            <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
              <h3 className="font-medium text-white mb-2 flex items-center">
                <CheckCircle2 className="mr-2 h-4 w-4 text-accent-cyan" />
                Security Issues
              </h3>
              <p className="text-gray-400 text-sm">
                Vulnerabilities categorized by severity: Critical, High, Medium, Low
              </p>
            </div>
            <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
              <h3 className="font-medium text-white mb-2 flex items-center">
                <CheckCircle2 className="mr-2 h-4 w-4 text-accent-cyan" />
                Gas Estimates
              </h3>
              <p className="text-gray-400 text-sm">
                Function-level gas costs with optimization suggestions
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Step 4 */}
      <section className="mb-10">
        <div className="flex items-center gap-3 mb-4">
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-accent-cyan text-dark-bg font-bold text-sm">
            4
          </div>
          <h2 className="text-xl font-semibold text-white">Set Up Monitoring (Optional)</h2>
        </div>
        <div className="pl-11">
          <p className="text-gray-300 mb-4">
            After deploying your contract, set up real-time monitoring to track events and receive alerts.
          </p>
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4 mb-4">
            <ol className="list-decimal list-inside text-gray-300 space-y-2">
              <li>Go to <Link href="/monitors" className="text-accent-cyan hover:underline">Monitors</Link></li>
              <li>Click &quot;Create Monitor&quot;</li>
              <li>Enter your deployed contract address</li>
              <li>Select events to monitor</li>
              <li>Configure alert destinations (email, webhook, etc.)</li>
            </ol>
          </div>
        </div>
      </section>

      {/* Next Steps */}
      <section className="rounded-xl border border-gray-800 bg-dark-card p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Next Steps</h2>
        <div className="grid gap-4 md:grid-cols-2">
          <Link
            href="/docs/vscode"
            className="flex items-center justify-between rounded-lg border border-gray-700 p-4 hover:border-accent-cyan hover:bg-dark-bg transition-all group"
          >
            <span className="text-gray-300 group-hover:text-white">Install VS Code Extension</span>
            <ArrowRight className="h-4 w-4 text-gray-500 group-hover:text-accent-cyan" />
          </Link>
          <Link
            href="/docs/security"
            className="flex items-center justify-between rounded-lg border border-gray-700 p-4 hover:border-accent-cyan hover:bg-dark-bg transition-all group"
          >
            <span className="text-gray-300 group-hover:text-white">Learn about Security Analysis</span>
            <ArrowRight className="h-4 w-4 text-gray-500 group-hover:text-accent-cyan" />
          </Link>
        </div>
      </section>
    </div>
  );
}
