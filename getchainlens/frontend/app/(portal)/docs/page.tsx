'use client';

import Link from 'next/link';
import {
  BookOpen,
  Rocket,
  Shield,
  Fuel,
  Search,
  Activity,
  Code2,
  Webhook,
  ArrowRight,
  ExternalLink,
} from 'lucide-react';

interface DocCard {
  title: string;
  description: string;
  href: string;
  icon: React.ElementType;
  color: string;
}

const gettingStarted: DocCard[] = [
  {
    title: 'Quick Start',
    description: 'Get up and running with GetChainLens in 5 minutes',
    href: '/docs/quickstart',
    icon: Rocket,
    color: 'accent-cyan',
  },
  {
    title: 'VS Code Extension',
    description: 'Install and configure the VS Code extension for real-time analysis',
    href: '/docs/vscode',
    icon: Code2,
    color: 'accent-blue',
  },
];

const features: DocCard[] = [
  {
    title: 'Security Analysis',
    description: 'Learn about vulnerability detection and security best practices',
    href: '/docs/security',
    icon: Shield,
    color: 'accent-cyan',
  },
  {
    title: 'Gas Optimization',
    description: 'Understand gas estimates and optimization recommendations',
    href: '/docs/gas',
    icon: Fuel,
    color: 'accent-yellow',
  },
  {
    title: 'Transaction Tracing',
    description: 'Debug transactions with detailed call traces and state changes',
    href: '/docs/tracing',
    icon: Search,
    color: 'accent-blue',
  },
  {
    title: 'Monitors & Alerts',
    description: 'Set up real-time monitoring and notifications for your contracts',
    href: '/docs/monitors',
    icon: Activity,
    color: 'accent-green',
  },
];

const api: DocCard[] = [
  {
    title: 'REST API',
    description: 'Complete API reference for programmatic access',
    href: '/docs/api',
    icon: BookOpen,
    color: 'chain-polygon',
  },
  {
    title: 'Webhooks',
    description: 'Configure webhooks for event notifications',
    href: '/docs/webhooks',
    icon: Webhook,
    color: 'accent-orange',
  },
];

function DocCardComponent({ card }: { card: DocCard }) {
  return (
    <Link
      href={card.href}
      className="group flex flex-col rounded-xl border border-gray-800 bg-dark-card p-6 hover:border-gray-700 hover:bg-dark-card-hover transition-all"
    >
      <div className={`flex h-12 w-12 items-center justify-center rounded-lg bg-${card.color}/20 mb-4`}>
        <card.icon className={`h-6 w-6 text-${card.color}`} />
      </div>
      <h3 className="text-lg font-semibold text-white mb-2 group-hover:text-accent-cyan transition-colors">
        {card.title}
      </h3>
      <p className="text-gray-400 text-sm flex-1">{card.description}</p>
      <div className="mt-4 flex items-center text-sm text-accent-cyan opacity-0 group-hover:opacity-100 transition-opacity">
        Read more <ArrowRight className="ml-1 h-4 w-4" />
      </div>
    </Link>
  );
}

export default function DocsPage() {
  return (
    <div className="max-w-5xl mx-auto">
      <div className="mb-12">
        <h1 className="text-3xl font-bold text-white mb-4">Documentation</h1>
        <p className="text-gray-400 text-lg">
          Learn how to use GetChainLens to secure and optimize your smart contracts.
        </p>
      </div>

      {/* Getting Started */}
      <section className="mb-12">
        <h2 className="text-xl font-semibold text-white mb-6 flex items-center">
          <Rocket className="mr-2 h-5 w-5 text-accent-cyan" />
          Getting Started
        </h2>
        <div className="grid gap-6 md:grid-cols-2">
          {gettingStarted.map((card) => (
            <DocCardComponent key={card.title} card={card} />
          ))}
        </div>
      </section>

      {/* Features */}
      <section className="mb-12">
        <h2 className="text-xl font-semibold text-white mb-6 flex items-center">
          <Shield className="mr-2 h-5 w-5 text-accent-cyan" />
          Features
        </h2>
        <div className="grid gap-6 md:grid-cols-2">
          {features.map((card) => (
            <DocCardComponent key={card.title} card={card} />
          ))}
        </div>
      </section>

      {/* API Reference */}
      <section className="mb-12">
        <h2 className="text-xl font-semibold text-white mb-6 flex items-center">
          <BookOpen className="mr-2 h-5 w-5 text-accent-cyan" />
          API Reference
        </h2>
        <div className="grid gap-6 md:grid-cols-2">
          {api.map((card) => (
            <DocCardComponent key={card.title} card={card} />
          ))}
        </div>
      </section>

      {/* Help */}
      <section className="rounded-xl border border-gray-800 bg-dark-card p-6">
        <h2 className="text-lg font-semibold text-white mb-2">Need Help?</h2>
        <p className="text-gray-400 mb-4">
          Can&apos;t find what you&apos;re looking for? We&apos;re here to help.
        </p>
        <div className="flex flex-wrap gap-4">
          <a
            href="mailto:contact@getchainlens.com"
            className="inline-flex items-center text-accent-cyan hover:underline"
          >
            Contact Support <ExternalLink className="ml-1 h-4 w-4" />
          </a>
          <a
            href="https://github.com/getchainlens/getchainlens/issues"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center text-accent-cyan hover:underline"
          >
            GitHub Issues <ExternalLink className="ml-1 h-4 w-4" />
          </a>
        </div>
      </section>
    </div>
  );
}
