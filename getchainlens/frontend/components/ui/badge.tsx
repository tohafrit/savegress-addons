import * as React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/lib/utils';

const badgeVariants = cva(
  'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold transition-colors',
  {
    variants: {
      variant: {
        default: 'bg-primary/20 text-accent-cyan border border-accent-cyan/30',
        secondary: 'bg-gray-700/50 text-gray-300 border border-gray-600',
        outline: 'border border-gray-600 text-gray-400 bg-transparent',
        success: 'bg-accent-green/20 text-accent-green border border-accent-green/30',
        warning: 'bg-accent-orange/20 text-accent-orange border border-accent-orange/30',
        destructive: 'bg-severity-critical/20 text-severity-critical border border-severity-critical/30',
        // Severity variants
        critical: 'bg-severity-critical/20 text-severity-critical border border-severity-critical/30',
        high: 'bg-severity-high/20 text-severity-high border border-severity-high/30',
        medium: 'bg-severity-medium/20 text-severity-medium border border-severity-medium/30',
        low: 'bg-severity-low/20 text-severity-low border border-severity-low/30',
        info: 'bg-severity-info/20 text-severity-info border border-severity-info/30',
        // Chain variants
        ethereum: 'bg-chain-ethereum/20 text-chain-ethereum border border-chain-ethereum/30',
        polygon: 'bg-chain-polygon/20 text-chain-polygon border border-chain-polygon/30',
        arbitrum: 'bg-chain-arbitrum/20 text-chain-arbitrum border border-chain-arbitrum/30',
        optimism: 'bg-chain-optimism/20 text-chain-optimism border border-chain-optimism/30',
        base: 'bg-chain-base/20 text-chain-base border border-chain-base/30',
        // Status variants
        active: 'bg-accent-green/20 text-accent-green border border-accent-green/30',
        inactive: 'bg-gray-600/20 text-gray-400 border border-gray-600/30',
        pending: 'bg-accent-yellow/20 text-accent-yellow border border-accent-yellow/30',
        error: 'bg-severity-critical/20 text-severity-critical border border-severity-critical/30',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  }
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return (
    <div className={cn(badgeVariants({ variant }), className)} {...props} />
  );
}

export { Badge, badgeVariants };
