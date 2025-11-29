'use client';

import React from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { cn } from '@/lib/utils';
import {
  LayoutDashboard,
  FolderKanban,
  FileCode2,
  Search,
  Activity,
  BarChart3,
  CreditCard,
  Settings,
  LogOut,
  Shield,
  Menu,
  X,
} from 'lucide-react';
import { useAuth } from '@/lib/auth-context';
import { Button } from '@/components/ui/button';

interface NavItem {
  name: string;
  href: string;
  icon: React.ElementType;
}

interface NavGroup {
  title: string;
  items: NavItem[];
}

const navigation: NavGroup[] = [
  {
    title: 'Overview',
    items: [
      { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
      { name: 'Projects', href: '/projects', icon: FolderKanban },
      { name: 'Contracts', href: '/contracts', icon: FileCode2 },
    ],
  },
  {
    title: 'Analysis',
    items: [
      { name: 'Traces', href: '/traces', icon: Search },
      { name: 'Monitors', href: '/monitors', icon: Activity },
      { name: 'Analytics', href: '/analytics', icon: BarChart3 },
    ],
  },
  {
    title: 'Account',
    items: [
      { name: 'Billing', href: '/billing', icon: CreditCard },
      { name: 'Settings', href: '/settings', icon: Settings },
    ],
  },
];

export function Sidebar() {
  const pathname = usePathname();
  const { user, logout } = useAuth();
  const [isMobileOpen, setIsMobileOpen] = React.useState(false);

  const isActive = (href: string) => {
    if (href === '/dashboard') return pathname === '/dashboard';
    return pathname.startsWith(href);
  };

  const handleLogout = async () => {
    await logout();
    window.location.href = '/login';
  };

  const SidebarContent = () => (
    <>
      {/* Logo */}
      <div className="flex h-16 items-center gap-3 px-6 border-b border-gray-800">
        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-gradient-to-br from-accent-cyan to-primary">
          <Shield className="h-5 w-5 text-white" />
        </div>
        <div>
          <h1 className="text-lg font-bold text-white">GetChainLens</h1>
          <p className="text-xs text-gray-500">Security Platform</p>
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto px-4 py-6">
        {navigation.map((group) => (
          <div key={group.title} className="mb-6">
            <h2 className="mb-2 px-3 text-xs font-semibold uppercase tracking-wider text-gray-500">
              {group.title}
            </h2>
            <ul className="space-y-1">
              {group.items.map((item) => {
                const Icon = item.icon;
                const active = isActive(item.href);
                return (
                  <li key={item.name}>
                    <Link
                      href={item.href}
                      onClick={() => setIsMobileOpen(false)}
                      className={cn(
                        'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-all duration-200',
                        active
                          ? 'bg-primary/20 text-accent-cyan border-l-2 border-accent-cyan ml-[-2px] pl-[14px]'
                          : 'text-gray-400 hover:bg-dark-card-hover hover:text-white'
                      )}
                    >
                      <Icon className={cn('h-5 w-5', active && 'text-accent-cyan')} />
                      {item.name}
                    </Link>
                  </li>
                );
              })}
            </ul>
          </div>
        ))}
      </nav>

      {/* User section */}
      <div className="border-t border-gray-800 p-4">
        <div className="mb-3 flex items-center gap-3 rounded-lg bg-dark-bg/50 p-3">
          <div className="flex h-9 w-9 items-center justify-center rounded-full bg-primary text-white text-sm font-semibold">
            {user?.name?.charAt(0).toUpperCase() || 'U'}
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium text-white truncate">{user?.name}</p>
            <p className="text-xs text-gray-500 truncate">{user?.email}</p>
          </div>
        </div>
        <Button
          variant="ghost"
          className="w-full justify-start text-gray-400 hover:text-severity-critical"
          onClick={handleLogout}
        >
          <LogOut className="mr-2 h-4 w-4" />
          Sign out
        </Button>
      </div>
    </>
  );

  return (
    <>
      {/* Mobile toggle */}
      <button
        className="fixed top-4 left-4 z-50 lg:hidden p-2 rounded-lg bg-dark-card border border-gray-700"
        onClick={() => setIsMobileOpen(!isMobileOpen)}
      >
        {isMobileOpen ? (
          <X className="h-5 w-5 text-white" />
        ) : (
          <Menu className="h-5 w-5 text-white" />
        )}
      </button>

      {/* Mobile overlay */}
      {isMobileOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 lg:hidden"
          onClick={() => setIsMobileOpen(false)}
        />
      )}

      {/* Mobile sidebar */}
      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-40 w-64 bg-dark-bg-secondary border-r border-gray-800 transform transition-transform duration-300 ease-in-out lg:hidden',
          isMobileOpen ? 'translate-x-0' : '-translate-x-full'
        )}
      >
        <div className="flex h-full flex-col">
          <SidebarContent />
        </div>
      </aside>

      {/* Desktop sidebar */}
      <aside className="hidden lg:fixed lg:inset-y-0 lg:left-0 lg:z-40 lg:block lg:w-64 lg:bg-dark-bg-secondary lg:border-r lg:border-gray-800">
        <div className="flex h-full flex-col">
          <SidebarContent />
        </div>
      </aside>
    </>
  );
}
