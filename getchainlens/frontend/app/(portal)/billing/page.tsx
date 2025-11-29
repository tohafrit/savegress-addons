'use client';

import { useState } from 'react';
import {
  CreditCard,
  Check,
  Zap,
  Shield,
  BarChart3,
  Bell,
  Users,
  Building2,
  ArrowRight,
  Download,
  Calendar,
} from 'lucide-react';
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardDescription,
  Button,
  Badge,
} from '@/components/ui';

const plans = [
  {
    id: 'free',
    name: 'Free',
    price: 0,
    description: 'Perfect for trying out GetChainLens',
    features: [
      '5 contracts',
      '100 analyses/month',
      '10 transaction traces/day',
      'Community support',
      'Basic security checks',
    ],
    limits: {
      contracts: 5,
      analyses: 100,
      traces: 10,
    },
  },
  {
    id: 'pro',
    name: 'Pro',
    price: 49,
    description: 'For professional developers and small teams',
    features: [
      '50 contracts',
      '1,000 analyses/month',
      '100 transaction traces/day',
      'Priority support',
      'Advanced security analysis',
      'Gas optimization suggestions',
      '5 monitors',
      'Webhook notifications',
    ],
    limits: {
      contracts: 50,
      analyses: 1000,
      traces: 100,
    },
    popular: true,
  },
  {
    id: 'team',
    name: 'Team',
    price: 199,
    description: 'For growing teams with advanced needs',
    features: [
      'Unlimited contracts',
      '10,000 analyses/month',
      'Unlimited transaction traces',
      'Dedicated support',
      'Full security suite',
      'Custom gas profiles',
      'Unlimited monitors',
      'API access',
      'Team collaboration',
      'Audit reports',
    ],
    limits: {
      contracts: -1,
      analyses: 10000,
      traces: -1,
    },
  },
  {
    id: 'enterprise',
    name: 'Enterprise',
    price: -1,
    description: 'For large organizations with custom requirements',
    features: [
      'Everything in Team',
      'Custom integrations',
      'On-premise deployment',
      'SLA guarantee',
      'Dedicated account manager',
      'Custom training',
      'SOC 2 compliance',
      'RBAC & SSO',
    ],
    limits: {
      contracts: -1,
      analyses: -1,
      traces: -1,
    },
  },
];

const invoices = [
  { id: 'INV-001', date: '2024-01-01', amount: 49, status: 'paid' },
  { id: 'INV-002', date: '2024-02-01', amount: 49, status: 'paid' },
  { id: 'INV-003', date: '2024-03-01', amount: 49, status: 'paid' },
];

export default function BillingPage() {
  const [currentPlan] = useState('pro');
  const [billingCycle, setBillingCycle] = useState<'monthly' | 'yearly'>('monthly');

  const currentPlanData = plans.find((p) => p.id === currentPlan);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-white">Billing & Plans</h1>
        <p className="mt-1 text-gray-400">
          Manage your subscription and billing information
        </p>
      </div>

      {/* Current Plan */}
      <Card variant="gradient">
        <CardContent className="p-6">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-lg font-semibold text-white">Current Plan</h2>
                <Badge variant="default">{currentPlanData?.name}</Badge>
              </div>
              <p className="mt-1 text-gray-400">{currentPlanData?.description}</p>
            </div>
            <div className="text-right">
              <p className="text-3xl font-bold text-white">
                ${currentPlanData?.price}
                <span className="text-sm font-normal text-gray-400">/month</span>
              </p>
              <p className="text-sm text-gray-500">Next billing: March 1, 2024</p>
            </div>
          </div>

          {/* Usage */}
          <div className="mt-6 grid gap-4 sm:grid-cols-3">
            <UsageBar
              label="Contracts"
              used={12}
              limit={currentPlanData?.limits.contracts || 50}
            />
            <UsageBar
              label="Analyses"
              used={456}
              limit={currentPlanData?.limits.analyses || 1000}
            />
            <UsageBar
              label="Traces/day"
              used={23}
              limit={currentPlanData?.limits.traces || 100}
            />
          </div>
        </CardContent>
      </Card>

      {/* Billing Cycle Toggle */}
      <div className="flex justify-center">
        <div className="inline-flex items-center rounded-lg border border-gray-800 bg-dark-card p-1">
          <button
            onClick={() => setBillingCycle('monthly')}
            className={`rounded-md px-4 py-2 text-sm font-medium transition-colors ${
              billingCycle === 'monthly'
                ? 'bg-accent-cyan text-white'
                : 'text-gray-400 hover:text-white'
            }`}
          >
            Monthly
          </button>
          <button
            onClick={() => setBillingCycle('yearly')}
            className={`rounded-md px-4 py-2 text-sm font-medium transition-colors ${
              billingCycle === 'yearly'
                ? 'bg-accent-cyan text-white'
                : 'text-gray-400 hover:text-white'
            }`}
          >
            Yearly
            <Badge variant="success" className="ml-2 text-xs">
              Save 20%
            </Badge>
          </button>
        </div>
      </div>

      {/* Plans Grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {plans.map((plan) => {
          const isCurrentPlan = plan.id === currentPlan;
          const yearlyPrice = plan.price > 0 ? Math.round(plan.price * 0.8) : plan.price;
          const displayPrice = billingCycle === 'yearly' ? yearlyPrice : plan.price;

          return (
            <Card
              key={plan.id}
              className={`relative overflow-hidden ${
                plan.popular ? 'border-accent-cyan' : ''
              }`}
            >
              {plan.popular && (
                <div className="absolute top-0 right-0 rounded-bl-lg bg-accent-cyan px-3 py-1 text-xs font-medium text-white">
                  Popular
                </div>
              )}
              <CardContent className="p-5">
                <h3 className="text-lg font-semibold text-white">{plan.name}</h3>
                <div className="mt-2">
                  {displayPrice === -1 ? (
                    <p className="text-2xl font-bold text-white">Custom</p>
                  ) : (
                    <p className="text-2xl font-bold text-white">
                      ${displayPrice}
                      <span className="text-sm font-normal text-gray-400">/month</span>
                    </p>
                  )}
                </div>
                <p className="mt-2 text-sm text-gray-400">{plan.description}</p>

                <ul className="mt-4 space-y-2">
                  {plan.features.slice(0, 5).map((feature, index) => (
                    <li key={index} className="flex items-start gap-2 text-sm">
                      <Check className="h-4 w-4 text-accent-green shrink-0 mt-0.5" />
                      <span className="text-gray-300">{feature}</span>
                    </li>
                  ))}
                  {plan.features.length > 5 && (
                    <li className="text-sm text-gray-500">
                      +{plan.features.length - 5} more features
                    </li>
                  )}
                </ul>

                <Button
                  className="mt-4 w-full"
                  variant={isCurrentPlan ? 'secondary' : plan.popular ? 'default' : 'outline'}
                  disabled={isCurrentPlan}
                >
                  {isCurrentPlan ? 'Current Plan' : plan.price === -1 ? 'Contact Sales' : 'Upgrade'}
                </Button>
              </CardContent>
            </Card>
          );
        })}
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Payment Method */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <CreditCard className="h-5 w-5 text-gray-400" />
              Payment Method
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between rounded-lg border border-gray-800 p-4">
              <div className="flex items-center gap-4">
                <div className="flex h-10 w-14 items-center justify-center rounded-md bg-gradient-to-r from-blue-500 to-blue-600">
                  <span className="text-sm font-bold text-white">VISA</span>
                </div>
                <div>
                  <p className="font-medium text-white">•••• •••• •••• 4242</p>
                  <p className="text-sm text-gray-500">Expires 12/25</p>
                </div>
              </div>
              <Button variant="ghost" size="sm">
                Update
              </Button>
            </div>
            <Button variant="outline" className="mt-4 w-full">
              <CreditCard className="mr-2 h-4 w-4" />
              Add Payment Method
            </Button>
          </CardContent>
        </Card>

        {/* Billing History */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2">
                <Calendar className="h-5 w-5 text-gray-400" />
                Billing History
              </CardTitle>
              <Button variant="ghost" size="sm">
                View all
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {invoices.map((invoice) => (
                <div
                  key={invoice.id}
                  className="flex items-center justify-between rounded-lg border border-gray-800 p-3"
                >
                  <div>
                    <p className="font-medium text-white">{invoice.id}</p>
                    <p className="text-sm text-gray-500">{invoice.date}</p>
                  </div>
                  <div className="flex items-center gap-4">
                    <div className="text-right">
                      <p className="font-medium text-white">${invoice.amount}</p>
                      <Badge variant="success" className="text-xs">
                        {invoice.status}
                      </Badge>
                    </div>
                    <Button variant="ghost" size="icon-sm">
                      <Download className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Enterprise CTA */}
      <Card className="border-accent-cyan/30 bg-gradient-to-r from-accent-cyan/5 to-primary/5">
        <CardContent className="p-6">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <div className="flex items-start gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-accent-cyan/20">
                <Building2 className="h-6 w-6 text-accent-cyan" />
              </div>
              <div>
                <h3 className="text-lg font-semibold text-white">Need Enterprise Features?</h3>
                <p className="text-gray-400">
                  Get custom integrations, dedicated support, and enterprise-grade security.
                </p>
              </div>
            </div>
            <Button>
              Contact Sales
              <ArrowRight className="ml-2 h-4 w-4" />
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function UsageBar({
  label,
  used,
  limit,
}: {
  label: string;
  used: number;
  limit: number;
}) {
  const percentage = limit === -1 ? 0 : Math.min((used / limit) * 100, 100);
  const isUnlimited = limit === -1;

  const getColor = () => {
    if (isUnlimited) return 'bg-accent-cyan';
    if (percentage >= 90) return 'bg-severity-critical';
    if (percentage >= 70) return 'bg-accent-yellow';
    return 'bg-accent-green';
  };

  return (
    <div>
      <div className="flex items-center justify-between text-sm">
        <span className="text-gray-400">{label}</span>
        <span className="text-white">
          {used.toLocaleString()} / {isUnlimited ? 'Unlimited' : limit.toLocaleString()}
        </span>
      </div>
      <div className="mt-2 h-2 rounded-full bg-gray-800">
        <div
          className={`h-2 rounded-full transition-all ${getColor()}`}
          style={{ width: isUnlimited ? '20%' : `${percentage}%` }}
        />
      </div>
    </div>
  );
}
