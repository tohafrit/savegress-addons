import Link from 'next/link';
import { ArrowLeft, ArrowRight, Fuel, TrendingDown, Zap } from 'lucide-react';

export default function GasPage() {
  return (
    <div className="max-w-4xl mx-auto">
      <Link
        href="/docs"
        className="inline-flex items-center text-gray-400 hover:text-white mb-6 transition-colors"
      >
        <ArrowLeft className="mr-2 h-4 w-4" />
        Back to Documentation
      </Link>

      <h1 className="text-3xl font-bold text-white mb-4">Gas Optimization</h1>
      <p className="text-gray-400 text-lg mb-8">
        Understand gas costs and optimize your smart contracts for efficiency.
      </p>

      {/* How It Works */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4 flex items-center">
          <Fuel className="mr-2 h-5 w-5 text-accent-yellow" />
          How Gas Estimation Works
        </h2>
        <div className="rounded-lg border border-gray-800 bg-dark-bg p-4 mb-4">
          <p className="text-gray-300 mb-4">
            GetChainLens analyzes your contract bytecode and provides estimates for:
          </p>
          <ul className="list-disc list-inside text-gray-300 space-y-2">
            <li><strong className="text-white">Deployment cost</strong> - Gas required to deploy the contract</li>
            <li><strong className="text-white">Function costs</strong> - Estimated gas for each public/external function</li>
            <li><strong className="text-white">Storage operations</strong> - SSTORE and SLOAD costs</li>
          </ul>
        </div>
        <p className="text-gray-400 text-sm">
          Note: Actual gas costs may vary based on input data and contract state.
        </p>
      </section>

      {/* Optimization Tips */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4 flex items-center">
          <TrendingDown className="mr-2 h-5 w-5 text-accent-green" />
          Optimization Tips
        </h2>
        <div className="space-y-4">
          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">Use uint256 Instead of Smaller Types</h3>
            <p className="text-gray-400 text-sm mb-3">
              The EVM operates on 256-bit words. Using uint8, uint16, etc. requires additional operations to convert.
            </p>
            <div className="grid md:grid-cols-2 gap-4">
              <div className="rounded bg-severity-high/10 p-3">
                <p className="text-xs text-severity-high mb-1">Avoid</p>
                <code className="text-sm text-gray-300">uint8 count;</code>
              </div>
              <div className="rounded bg-accent-green/10 p-3">
                <p className="text-xs text-accent-green mb-1">Prefer</p>
                <code className="text-sm text-gray-300">uint256 count;</code>
              </div>
            </div>
          </div>

          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">Pack Storage Variables</h3>
            <p className="text-gray-400 text-sm mb-3">
              Multiple variables smaller than 32 bytes can be packed into a single storage slot.
            </p>
            <div className="grid md:grid-cols-2 gap-4">
              <div className="rounded bg-severity-high/10 p-3">
                <p className="text-xs text-severity-high mb-1">3 storage slots</p>
                <code className="text-sm text-gray-300 block">uint128 a;<br/>uint256 b;<br/>uint128 c;</code>
              </div>
              <div className="rounded bg-accent-green/10 p-3">
                <p className="text-xs text-accent-green mb-1">2 storage slots</p>
                <code className="text-sm text-gray-300 block">uint128 a;<br/>uint128 c;<br/>uint256 b;</code>
              </div>
            </div>
          </div>

          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">Use calldata for Read-Only Arrays</h3>
            <p className="text-gray-400 text-sm mb-3">
              For external functions that don&apos;t modify array arguments, use calldata instead of memory.
            </p>
            <div className="grid md:grid-cols-2 gap-4">
              <div className="rounded bg-severity-high/10 p-3">
                <p className="text-xs text-severity-high mb-1">Copies data to memory</p>
                <code className="text-sm text-gray-300">function f(uint[] memory arr)</code>
              </div>
              <div className="rounded bg-accent-green/10 p-3">
                <p className="text-xs text-accent-green mb-1">Reads directly</p>
                <code className="text-sm text-gray-300">function f(uint[] calldata arr)</code>
              </div>
            </div>
          </div>

          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">Cache Storage Variables</h3>
            <p className="text-gray-400 text-sm mb-3">
              Reading from storage costs ~100 gas. Cache frequently accessed values in memory.
            </p>
            <div className="grid md:grid-cols-2 gap-4">
              <div className="rounded bg-severity-high/10 p-3">
                <p className="text-xs text-severity-high mb-1">Multiple SLOAD</p>
                <code className="text-sm text-gray-300 block">for(i; i &lt; arr.length; i++)</code>
              </div>
              <div className="rounded bg-accent-green/10 p-3">
                <p className="text-xs text-accent-green mb-1">Single SLOAD</p>
                <code className="text-sm text-gray-300 block">uint len = arr.length;<br/>for(i; i &lt; len; i++)</code>
              </div>
            </div>
          </div>

          <div className="rounded-lg border border-gray-800 bg-dark-bg p-4">
            <h3 className="font-medium text-white mb-2">Use Custom Errors</h3>
            <p className="text-gray-400 text-sm mb-3">
              Custom errors are cheaper than require strings (especially long ones).
            </p>
            <div className="grid md:grid-cols-2 gap-4">
              <div className="rounded bg-severity-high/10 p-3">
                <p className="text-xs text-severity-high mb-1">Stores string in bytecode</p>
                <code className="text-sm text-gray-300">require(x, &quot;Error message&quot;);</code>
              </div>
              <div className="rounded bg-accent-green/10 p-3">
                <p className="text-xs text-accent-green mb-1">Just 4 bytes selector</p>
                <code className="text-sm text-gray-300">if(!x) revert MyError();</code>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Gas Costs Reference */}
      <section className="mb-10">
        <h2 className="text-xl font-semibold text-white mb-4 flex items-center">
          <Zap className="mr-2 h-5 w-5 text-accent-cyan" />
          Common Gas Costs
        </h2>
        <div className="rounded-lg border border-gray-800 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-dark-bg">
              <tr>
                <th className="text-left text-gray-400 font-medium p-4">Operation</th>
                <th className="text-right text-gray-400 font-medium p-4">Gas Cost</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-800">
              <tr>
                <td className="p-4 text-gray-300">Transaction base cost</td>
                <td className="p-4 text-right font-mono text-white">21,000</td>
              </tr>
              <tr>
                <td className="p-4 text-gray-300">SSTORE (new value)</td>
                <td className="p-4 text-right font-mono text-white">20,000</td>
              </tr>
              <tr>
                <td className="p-4 text-gray-300">SSTORE (update)</td>
                <td className="p-4 text-right font-mono text-white">5,000</td>
              </tr>
              <tr>
                <td className="p-4 text-gray-300">SLOAD</td>
                <td className="p-4 text-right font-mono text-white">100</td>
              </tr>
              <tr>
                <td className="p-4 text-gray-300">Memory expansion (per word)</td>
                <td className="p-4 text-right font-mono text-white">3</td>
              </tr>
              <tr>
                <td className="p-4 text-gray-300">External call base</td>
                <td className="p-4 text-right font-mono text-white">100</td>
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
            href="/docs/tracing"
            className="flex items-center justify-between rounded-lg border border-gray-700 p-4 hover:border-accent-cyan hover:bg-dark-bg transition-all group"
          >
            <span className="text-gray-300 group-hover:text-white">Transaction Tracing</span>
            <ArrowRight className="h-4 w-4 text-gray-500 group-hover:text-accent-cyan" />
          </Link>
          <Link
            href="/docs/api"
            className="flex items-center justify-between rounded-lg border border-gray-700 p-4 hover:border-accent-cyan hover:bg-dark-bg transition-all group"
          >
            <span className="text-gray-300 group-hover:text-white">API Reference</span>
            <ArrowRight className="h-4 w-4 text-gray-500 group-hover:text-accent-cyan" />
          </Link>
        </div>
      </section>
    </div>
  );
}
