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
  CheckCircle2,
  Github,
  Twitter,
} from 'lucide-react';

export default function LandingPage() {
  return (
    <div className="min-h-screen bg-dark-bg">
      {/* Navbar */}
      <nav className="fixed top-0 left-0 right-0 z-50 border-b border-gray-800 bg-dark-bg/80 backdrop-blur-xl">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex h-16 items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-gradient-to-br from-accent-cyan to-primary">
                <Shield className="h-5 w-5 text-white" />
              </div>
              <span className="text-xl font-bold text-white">GetChainLens</span>
            </div>
            <div className="hidden md:flex items-center gap-8">
              <a href="#features" className="text-gray-400 hover:text-white transition-colors">
                Features
              </a>
              <a href="#pricing" className="text-gray-400 hover:text-white transition-colors">
                Pricing
              </a>
              <a href="https://docs.getchainlens.com" className="text-gray-400 hover:text-white transition-colors">
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
              href="mailto:sales@getchainlens.com"
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
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-accent-cyan to-primary">
                <Shield className="h-4 w-4 text-white" />
              </div>
              <span className="text-lg font-bold text-white">GetChainLens</span>
            </div>
            <div className="flex items-center gap-6 text-sm text-gray-400">
              <a href="/privacy" className="hover:text-white">Privacy</a>
              <a href="/terms" className="hover:text-white">Terms</a>
              <a href="https://docs.getchainlens.com" className="hover:text-white">Docs</a>
            </div>
            <div className="flex items-center gap-4">
              <a
                href="https://github.com/getchainlens"
                target="_blank"
                rel="noopener noreferrer"
                className="text-gray-400 hover:text-white"
              >
                <Github className="h-5 w-5" />
              </a>
              <a
                href="https://twitter.com/getchainlens"
                target="_blank"
                rel="noopener noreferrer"
                className="text-gray-400 hover:text-white"
              >
                <Twitter className="h-5 w-5" />
              </a>
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
