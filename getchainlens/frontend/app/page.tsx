import Link from 'next/link';
import {
  Shield,
  Fuel,
  Search,
  Activity,
  Code2,
  Zap,
  Lock,
  BarChart3,
  ArrowRight,
  Check,
} from 'lucide-react';
import { Logo } from '@/components/ui/logo';

export default function LandingPage() {
  return (
    <div className="min-h-screen bg-dark-bg">
      {/* Navbar */}
      <nav className="fixed top-0 left-0 right-0 z-50 border-b border-gray-800 bg-dark-bg/80 backdrop-blur-xl">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex h-16 items-center justify-between">
            <div className="flex items-center gap-3">
              <Logo size={36} />
              <span className="text-xl font-bold text-white">GetChainLens</span>
            </div>
            <div className="hidden md:flex items-center gap-8">
              <a href="#features" className="text-gray-400 hover:text-white transition-colors">
                Features
              </a>
              <a href="#pricing" className="text-gray-400 hover:text-white transition-colors">
                Pricing
              </a>
              <a href="/login" className="text-gray-400 hover:text-white transition-colors">
                Docs
              </a>
            </div>
            <div className="flex items-center gap-4">
              <Link
                href="/login"
                className="text-gray-400 hover:text-white transition-colors"
              >
                Sign in
              </Link>
              <Link
                href="/register"
                className="inline-flex items-center justify-center rounded-lg bg-gradient-to-r from-accent-cyan to-primary px-4 py-2 text-sm font-medium text-white hover:shadow-glow-cyan transition-all"
              >
                Get Started
              </Link>
            </div>
          </div>
        </div>
      </nav>

      {/* Hero */}
      <section className="relative pt-32 pb-20 overflow-hidden">
        {/* Background effects */}
        <div className="absolute inset-0 overflow-hidden pointer-events-none">
          <div className="absolute -top-40 -right-40 h-96 w-96 rounded-full bg-accent-cyan/10 blur-3xl" />
          <div className="absolute -bottom-40 -left-40 h-96 w-96 rounded-full bg-primary/20 blur-3xl" />
        </div>

        <div className="relative mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 text-center">
          <div className="inline-flex items-center gap-2 rounded-full border border-accent-cyan/30 bg-accent-cyan/10 px-4 py-1.5 text-sm text-accent-cyan mb-8">
            <Zap className="h-4 w-4" />
            Production-grade smart contract security
          </div>

          <h1 className="text-4xl sm:text-5xl lg:text-6xl font-bold text-white leading-tight">
            Secure Your Smart Contracts
            <br />
            <span className="text-gradient">Before Deployment</span>
          </h1>

          <p className="mt-6 text-lg text-gray-400 max-w-2xl mx-auto">
            Detect vulnerabilities, optimize gas costs, and trace transactions with
            GetChainLens. The complete security platform for Solidity developers.
          </p>

          <div className="mt-10 flex flex-col sm:flex-row items-center justify-center gap-4">
            <Link
              href="/register"
              className="inline-flex items-center justify-center rounded-lg bg-gradient-to-r from-accent-cyan to-primary px-8 py-3 text-base font-medium text-white hover:shadow-glow-cyan transition-all"
            >
              Start Free Trial
              <ArrowRight className="ml-2 h-4 w-4" />
            </Link>
            <a
              href="https://marketplace.visualstudio.com/items?itemName=getchainlens.getchainlens"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center justify-center rounded-lg border border-gray-600 bg-dark-card px-8 py-3 text-base font-medium text-white hover:bg-dark-card-hover transition-all"
            >
              <Code2 className="mr-2 h-4 w-4" />
              VS Code Extension
            </a>
          </div>

          {/* Stats */}
          <div className="mt-16 grid grid-cols-2 sm:grid-cols-4 gap-8">
            {[
              { value: '50K+', label: 'Contracts Analyzed' },
              { value: '10M+', label: 'Issues Detected' },
              { value: '5', label: 'Chains Supported' },
              { value: '99.9%', label: 'Uptime' },
            ].map((stat) => (
              <div key={stat.label}>
                <p className="text-3xl font-bold text-white">{stat.value}</p>
                <p className="mt-1 text-sm text-gray-400">{stat.label}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Features */}
      <section id="features" className="py-20 bg-dark-bg-secondary">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-3xl font-bold text-white">
              Everything You Need for Smart Contract Security
            </h2>
            <p className="mt-4 text-gray-400 max-w-2xl mx-auto">
              From vulnerability detection to gas optimization, GetChainLens provides
              comprehensive tools for secure smart contract development.
            </p>
          </div>

          <div className="grid gap-8 md:grid-cols-2 lg:grid-cols-3">
            {[
              {
                icon: Shield,
                title: 'Security Analysis',
                description:
                  'Detect reentrancy, integer overflow, access control issues, and more with our advanced static analysis engine.',
                color: 'accent-cyan',
              },
              {
                icon: Fuel,
                title: 'Gas Optimization',
                description:
                  'Get function-level gas estimates and optimization suggestions to reduce deployment and execution costs.',
                color: 'accent-yellow',
              },
              {
                icon: Search,
                title: 'Transaction Tracing',
                description:
                  'Debug failed transactions with detailed call traces, state changes, and decoded event logs.',
                color: 'accent-blue',
              },
              {
                icon: Activity,
                title: 'Real-time Monitoring',
                description:
                  'Set up alerts for contract events and get notified instantly via webhooks.',
                color: 'accent-green',
              },
              {
                icon: BarChart3,
                title: 'Network Analytics',
                description:
                  'Track gas prices, popular contracts, and network statistics across multiple chains.',
                color: 'accent-orange',
              },
              {
                icon: Lock,
                title: 'Enterprise Security',
                description:
                  'SOC 2 compliant infrastructure with encrypted data storage and role-based access control.',
                color: 'chain-polygon',
              },
            ].map((feature) => (
              <div
                key={feature.title}
                className="rounded-xl border border-gray-800 bg-dark-card p-6 hover:border-gray-700 transition-colors"
              >
                <div
                  className={`flex h-12 w-12 items-center justify-center rounded-lg bg-${feature.color}/20`}
                >
                  <feature.icon className={`h-6 w-6 text-${feature.color}`} />
                </div>
                <h3 className="mt-4 text-lg font-semibold text-white">{feature.title}</h3>
                <p className="mt-2 text-gray-400">{feature.description}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Chains */}
      <section className="py-20">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 text-center">
          <h2 className="text-2xl font-bold text-white mb-8">
            Multi-Chain Support
          </h2>
          <div className="flex flex-wrap items-center justify-center gap-8">
            {[
              { name: 'Ethereum', color: '#627EEA' },
              { name: 'Polygon', color: '#8247E5' },
              { name: 'Arbitrum', color: '#28A0F0' },
              { name: 'Optimism', color: '#FF0420' },
              { name: 'Base', color: '#0052FF' },
            ].map((chain) => (
              <div
                key={chain.name}
                className="flex items-center gap-2 rounded-lg border border-gray-800 bg-dark-card px-4 py-2"
              >
                <div
                  className="h-3 w-3 rounded-full"
                  style={{ backgroundColor: chain.color }}
                />
                <span className="text-gray-300">{chain.name}</span>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Pricing */}
      <section id="pricing" className="py-20 bg-dark-bg-secondary">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-3xl font-bold text-white">
              Simple, Transparent Pricing
            </h2>
            <p className="mt-4 text-gray-400 max-w-2xl mx-auto">
              Start free and scale as your needs grow. No hidden fees.
            </p>
          </div>

          <div className="grid gap-8 lg:grid-cols-3 max-w-5xl mx-auto">
            {/* Free */}
            <div className="rounded-xl border border-gray-800 bg-dark-card p-8">
              <h3 className="text-lg font-semibold text-white">Free</h3>
              <p className="mt-2 text-gray-400 text-sm">For individual developers</p>
              <div className="mt-6">
                <span className="text-4xl font-bold text-white">$0</span>
                <span className="text-gray-400">/month</span>
              </div>
              <ul className="mt-8 space-y-4">
                {[
                  '5 contracts per month',
                  'Basic security analysis',
                  'Gas estimates',
                  'Community support',
                ].map((feature) => (
                  <li key={feature} className="flex items-start gap-3">
                    <Check className="h-5 w-5 text-accent-cyan shrink-0" />
                    <span className="text-gray-300 text-sm">{feature}</span>
                  </li>
                ))}
              </ul>
              <Link
                href="/register"
                className="mt-8 block w-full rounded-lg border border-gray-600 py-3 text-center text-sm font-medium text-white hover:bg-dark-card-hover transition-all"
              >
                Get Started
              </Link>
            </div>

            {/* Pro */}
            <div className="rounded-xl border-2 border-accent-cyan bg-dark-card p-8 relative">
              <div className="absolute -top-3 left-1/2 -translate-x-1/2 px-3 py-1 bg-accent-cyan text-dark-bg text-xs font-semibold rounded-full">
                Popular
              </div>
              <h3 className="text-lg font-semibold text-white">Pro</h3>
              <p className="mt-2 text-gray-400 text-sm">For professional teams</p>
              <div className="mt-6">
                <span className="text-4xl font-bold text-white">$49</span>
                <span className="text-gray-400">/month</span>
              </div>
              <ul className="mt-8 space-y-4">
                {[
                  'Unlimited contracts',
                  'Advanced security analysis',
                  'Transaction tracing',
                  'Real-time monitoring',
                  '10 monitors included',
                  'Priority support',
                ].map((feature) => (
                  <li key={feature} className="flex items-start gap-3">
                    <Check className="h-5 w-5 text-accent-cyan shrink-0" />
                    <span className="text-gray-300 text-sm">{feature}</span>
                  </li>
                ))}
              </ul>
              <Link
                href="/register"
                className="mt-8 block w-full rounded-lg bg-gradient-to-r from-accent-cyan to-primary py-3 text-center text-sm font-medium text-white hover:shadow-glow-cyan transition-all"
              >
                Start Free Trial
              </Link>
            </div>

            {/* Enterprise */}
            <div className="rounded-xl border border-gray-800 bg-dark-card p-8">
              <h3 className="text-lg font-semibold text-white">Enterprise</h3>
              <p className="mt-2 text-gray-400 text-sm">For large organizations</p>
              <div className="mt-6">
                <span className="text-4xl font-bold text-white">Custom</span>
              </div>
              <ul className="mt-8 space-y-4">
                {[
                  'Everything in Pro',
                  'Unlimited monitors',
                  'Custom integrations',
                  'SLA guarantee',
                  'Dedicated support',
                  'On-premise option',
                ].map((feature) => (
                  <li key={feature} className="flex items-start gap-3">
                    <Check className="h-5 w-5 text-accent-cyan shrink-0" />
                    <span className="text-gray-300 text-sm">{feature}</span>
                  </li>
                ))}
              </ul>
              <a
                href="mailto:contact@getchainlens.com"
                className="mt-8 block w-full rounded-lg border border-gray-600 py-3 text-center text-sm font-medium text-white hover:bg-dark-card-hover transition-all"
              >
                Contact Sales
              </a>
            </div>
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="py-20 bg-gradient-to-b from-dark-bg-secondary to-dark-bg">
        <div className="mx-auto max-w-4xl px-4 sm:px-6 lg:px-8 text-center">
          <h2 className="text-3xl font-bold text-white">
            Ready to Secure Your Smart Contracts?
          </h2>
          <p className="mt-4 text-gray-400">
            Start with our free tier and upgrade as you grow. No credit card required.
          </p>
          <div className="mt-8 flex flex-col sm:flex-row items-center justify-center gap-4">
            <Link
              href="/register"
              className="inline-flex items-center justify-center rounded-lg bg-gradient-to-r from-accent-cyan to-primary px-8 py-3 text-base font-medium text-white hover:shadow-glow-cyan transition-all"
            >
              Get Started Free
              <ArrowRight className="ml-2 h-4 w-4" />
            </Link>
            <a
              href="mailto:contact@getchainlens.com"
              className="inline-flex items-center justify-center rounded-lg border border-gray-600 px-8 py-3 text-base font-medium text-white hover:bg-dark-card transition-all"
            >
              Contact Sales
            </a>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-gray-800 py-12">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col md:flex-row items-center justify-between gap-6">
            <div className="flex items-center gap-3">
              <Logo size={32} />
              <span className="text-lg font-bold text-white">GetChainLens</span>
            </div>
            <div className="flex items-center gap-6 text-sm text-gray-400">
              <a href="/privacy" className="hover:text-white">Privacy</a>
              <a href="/terms" className="hover:text-white">Terms</a>
              <a href="/login" className="hover:text-white">Docs</a>
            </div>
          </div>
          <div className="mt-8 text-center text-sm text-gray-500">
            &copy; {new Date().getFullYear()} GetChainLens. All rights reserved.
          </div>
        </div>
      </footer>
    </div>
  );
}
