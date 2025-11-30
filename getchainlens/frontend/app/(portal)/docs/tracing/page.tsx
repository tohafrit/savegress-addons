import Link from 'next/link';
import { ArrowLeft, ArrowRight, Search, GitBranch, Database } from 'lucide-react';

export default function TracingPage() {
  return (
    <div className="max-w-4xl mx-auto">
      <Link
        href="/docs"
        className="inline-flex items-center text-gray-400 hover:text-white mb-6 transition-colors"
      >
        <ArrowLeft className="mr-2 h-4 w-4" />
        Back to Documentation
      </Link>

      <h1 className="text-3xl font-bold text-white mb-4">Transaction Tracing</h1>
      <p className="text-gray-400 text-lg mb-8">
        Debug failed transactions and understand execution flow with detailed traces.
      </p>

      {/* How to Trace */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4 flex items-center">
          <Search className="mr-2 h-5 w-5 text-accent-blue" />
          How to Trace a Transaction
        </h2>
        <div className="rounded-lg border border-gray-800 bg-dark-bg p-4 mb-4">
          <ol className="list-decimal list-inside text-gray-300 space-y-2">
            <li>Go to <Link href="/traces" className="text-accent-cyan hover:underline">Traces</Link></li>
            <li>Enter a transaction hash</li>
            <li>Select the network (Ethereum, Polygon, etc.)</li>
            <li>Click &quot;Trace&quot;</li>
          </ol>
        </div>
      </section>

      {/* What You Get */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4 flex items-center">
          <GitBranch className="mr-2 h-5 w-5 text-accent-cyan" />
          Trace Information
        </h2>
        <div className="space-y-4">
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">Call Tree</h3>
            <p className="text-gray-400 text-sm">
              Visualize all internal and external calls made during transaction execution, including:
            </p>
            <ul className="list-disc list-inside text-gray-400 text-sm mt-2 space-y-1">
              <li>Function signatures (decoded if ABI available)</li>
              <li>Input parameters</li>
              <li>Return values</li>
              <li>Gas used per call</li>
            </ul>
          </div>

          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">State Changes</h3>
            <p className="text-gray-400 text-sm">
              See all storage modifications made by the transaction:
            </p>
            <ul className="list-disc list-inside text-gray-400 text-sm mt-2 space-y-1">
              <li>Storage slot addresses</li>
              <li>Previous values</li>
              <li>New values</li>
              <li>Variable names (if source code available)</li>
            </ul>
          </div>

          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">Event Logs</h3>
            <p className="text-gray-400 text-sm">
              All events emitted during execution, decoded with parameter names and values.
            </p>
          </div>

          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">Error Analysis</h3>
            <p className="text-gray-400 text-sm">
              For failed transactions, see exactly where and why the transaction reverted:
            </p>
            <ul className="list-disc list-inside text-gray-400 text-sm mt-2 space-y-1">
              <li>Revert reason (decoded)</li>
              <li>Exact call that failed</li>
              <li>Stack trace to the failure point</li>
            </ul>
          </div>
        </div>
      </section>

      {/* Supported Networks */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4 flex items-center">
          <Database className="mr-2 h-5 w-5 text-accent-cyan" />
          Supported Networks
        </h2>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
          {[
            { name: 'Ethereum', color: '#627EEA' },
            { name: 'Polygon', color: '#8247E5' },
            { name: 'Arbitrum', color: '#28A0F0' },
            { name: 'Optimism', color: '#FF0420' },
            { name: 'Base', color: '#0052FF' },
            { name: 'Goerli', color: '#627EEA' },
          ].map((network) => (
            <div
              key={network.name}
              className="flex items-center gap-2 rounded-lg border border-gray-800 bg-dark-bg px-4 py-3"
            >
              <div
                className="h-3 w-3 rounded-full"
                style={{ backgroundColor: network.color }}
              />
              <span className="text-gray-300">{network.name}</span>
            </div>
          ))}
        </div>
      </section>

      {/* Tips */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4">Tips</h2>
        <div className="space-y-3">
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <p className="text-gray-300">
              <strong className="text-white">Add verified contracts:</strong> Upload your contract source code to get decoded function names and parameters in traces.
            </p>
          </div>
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <p className="text-gray-300">
              <strong className="text-white">Use source maps:</strong> If you have compilation artifacts, traces can show exact Solidity line numbers.
            </p>
          </div>
        </div>
      </section>

      {/* Next Steps */}
      <section className="rounded-xl border border-gray-800 bg-dark-card p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Next Steps</h2>
        <div className="grid gap-4 md:grid-cols-2">
          <Link
            href="/docs/monitors"
            className="flex items-center justify-between rounded-lg border border-gray-700 p-4 hover:border-accent-cyan hover:bg-dark-bg transition-all group"
          >
            <span className="text-gray-300 group-hover:text-white">Set Up Monitors</span>
            <ArrowRight className="h-4 w-4 text-gray-500 group-hover:text-accent-cyan" />
          </Link>
          <Link
            href="/traces"
            className="flex items-center justify-between rounded-lg border border-gray-700 p-4 hover:border-accent-cyan hover:bg-dark-bg transition-all group"
          >
            <span className="text-gray-300 group-hover:text-white">Try Transaction Tracing</span>
            <ArrowRight className="h-4 w-4 text-gray-500 group-hover:text-accent-cyan" />
          </Link>
        </div>
      </section>
    </div>
  );
}
