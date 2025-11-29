import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatAddress(address: string, chars = 6): string {
  if (!address) return '';
  return `${address.slice(0, chars + 2)}...${address.slice(-chars)}`;
}

export function formatTxHash(hash: string, chars = 8): string {
  if (!hash) return '';
  return `${hash.slice(0, chars + 2)}...${hash.slice(-chars)}`;
}

export function formatNumber(num: number, decimals = 2): string {
  if (num >= 1_000_000_000) {
    return (num / 1_000_000_000).toFixed(decimals) + 'B';
  }
  if (num >= 1_000_000) {
    return (num / 1_000_000).toFixed(decimals) + 'M';
  }
  if (num >= 1_000) {
    return (num / 1_000).toFixed(decimals) + 'K';
  }
  return num.toLocaleString();
}

export function formatGas(gas: number): string {
  if (gas >= 1_000_000) {
    return (gas / 1_000_000).toFixed(2) + 'M';
  }
  if (gas >= 1_000) {
    return (gas / 1_000).toFixed(1) + 'K';
  }
  return gas.toLocaleString();
}

export function formatGwei(wei: number): string {
  const gwei = wei / 1e9;
  return gwei.toFixed(2) + ' Gwei';
}

export function formatEth(wei: string | number): string {
  const eth = Number(wei) / 1e18;
  if (eth < 0.0001) {
    return '< 0.0001 ETH';
  }
  return eth.toFixed(4) + ' ETH';
}

export function formatDate(date: string | Date): string {
  return new Date(date).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

export function formatDateTime(date: string | Date): string {
  return new Date(date).toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function formatTimeAgo(date: string | Date): string {
  const now = new Date();
  const past = new Date(date);
  const diffMs = now.getTime() - past.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHour = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHour / 24);

  if (diffSec < 60) return 'just now';
  if (diffMin < 60) return `${diffMin}m ago`;
  if (diffHour < 24) return `${diffHour}h ago`;
  if (diffDay < 7) return `${diffDay}d ago`;
  return formatDate(date);
}

export function getSeverityColor(severity: string): string {
  const colors: Record<string, string> = {
    critical: 'severity-critical',
    high: 'severity-high',
    medium: 'severity-medium',
    low: 'severity-low',
    info: 'severity-info',
  };
  return colors[severity.toLowerCase()] || 'severity-info';
}

export function getSeverityBadgeClass(severity: string): string {
  const classes: Record<string, string> = {
    critical: 'badge-critical',
    high: 'badge-high',
    medium: 'badge-medium',
    low: 'badge-low',
    info: 'badge-info',
  };
  return classes[severity.toLowerCase()] || 'badge-info';
}

export function getChainColor(chain: string): string {
  const colors: Record<string, string> = {
    ethereum: 'chain-ethereum',
    polygon: 'chain-polygon',
    arbitrum: 'chain-arbitrum',
    optimism: 'chain-optimism',
    base: 'chain-base',
  };
  return colors[chain.toLowerCase()] || 'chain-ethereum';
}

export function getChainName(chain: string): string {
  const names: Record<string, string> = {
    ethereum: 'Ethereum',
    polygon: 'Polygon',
    arbitrum: 'Arbitrum',
    optimism: 'Optimism',
    base: 'Base',
  };
  return names[chain.toLowerCase()] || chain;
}

export function getScoreColor(score: number): string {
  if (score >= 80) return '#10B981'; // green
  if (score >= 60) return '#FBBF24'; // yellow
  if (score >= 40) return '#F97316'; // orange
  return '#DC2626'; // red
}

export function getScoreLabel(score: number): string {
  if (score >= 80) return 'Excellent';
  if (score >= 60) return 'Good';
  if (score >= 40) return 'Fair';
  return 'Poor';
}

export function copyToClipboard(text: string): Promise<void> {
  return navigator.clipboard.writeText(text);
}

export function debounce<T extends (...args: unknown[]) => unknown>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: NodeJS.Timeout;
  return (...args: Parameters<T>) => {
    clearTimeout(timeout);
    timeout = setTimeout(() => func(...args), wait);
  };
}

export function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}
