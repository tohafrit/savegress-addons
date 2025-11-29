'use client';

import { useState } from 'react';
import {
  User,
  Bell,
  Shield,
  Key,
  Palette,
  Globe,
  Mail,
  Smartphone,
  Save,
  Eye,
  EyeOff,
  Copy,
  RefreshCw,
  LogOut,
  Trash2,
} from 'lucide-react';
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardDescription,
  Button,
  Input,
  Badge,
  Label,
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/components/ui';
import { useAuth } from '@/lib/auth-context';
import { copyToClipboard } from '@/lib/utils';

export default function SettingsPage() {
  const { user, logout } = useAuth();
  const [activeTab, setActiveTab] = useState('profile');

  const tabs = [
    { id: 'profile', label: 'Profile', icon: User },
    { id: 'notifications', label: 'Notifications', icon: Bell },
    { id: 'security', label: 'Security', icon: Shield },
    { id: 'api', label: 'API Keys', icon: Key },
    { id: 'preferences', label: 'Preferences', icon: Palette },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-white">Settings</h1>
        <p className="mt-1 text-gray-400">
          Manage your account settings and preferences
        </p>
      </div>

      <div className="flex flex-col lg:flex-row gap-6">
        {/* Sidebar */}
        <div className="lg:w-64 shrink-0">
          <Card>
            <CardContent className="p-2">
              <nav className="space-y-1">
                {tabs.map((tab) => {
                  const Icon = tab.icon;
                  return (
                    <button
                      key={tab.id}
                      onClick={() => setActiveTab(tab.id)}
                      className={`flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors ${
                        activeTab === tab.id
                          ? 'bg-accent-cyan/10 text-accent-cyan'
                          : 'text-gray-400 hover:bg-dark-card-hover hover:text-white'
                      }`}
                    >
                      <Icon className="h-4 w-4" />
                      {tab.label}
                    </button>
                  );
                })}
              </nav>
            </CardContent>
          </Card>
        </div>

        {/* Content */}
        <div className="flex-1 space-y-6">
          {activeTab === 'profile' && <ProfileSettings user={user} />}
          {activeTab === 'notifications' && <NotificationSettings />}
          {activeTab === 'security' && <SecuritySettings onLogout={logout} />}
          {activeTab === 'api' && <ApiSettings />}
          {activeTab === 'preferences' && <PreferencesSettings />}
        </div>
      </div>
    </div>
  );
}

function ProfileSettings({ user }: { user: any }) {
  const [name, setName] = useState(user?.name || '');
  const [email, setEmail] = useState(user?.email || '');
  const [isSaving, setIsSaving] = useState(false);

  const handleSave = async () => {
    setIsSaving(true);
    // Simulate API call
    await new Promise((resolve) => setTimeout(resolve, 1000));
    setIsSaving(false);
  };

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Profile Information</CardTitle>
          <CardDescription>
            Update your personal information
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-4">
            <div className="flex h-16 w-16 items-center justify-center rounded-full bg-gradient-to-br from-accent-cyan to-primary">
              <span className="text-2xl font-bold text-white">
                {name?.charAt(0)?.toUpperCase() || 'U'}
              </span>
            </div>
            <Button variant="secondary">Change Avatar</Button>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>Full Name</Label>
              <Input value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label>Email Address</Label>
              <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
            </div>
          </div>

          <div className="flex justify-end">
            <Button onClick={handleSave} isLoading={isSaving}>
              <Save className="mr-2 h-4 w-4" />
              Save Changes
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card className="border-severity-critical/30">
        <CardHeader>
          <CardTitle className="text-severity-critical">Danger Zone</CardTitle>
          <CardDescription>
            Irreversible and destructive actions
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between rounded-lg border border-severity-critical/30 p-4">
            <div>
              <p className="font-medium text-white">Delete Account</p>
              <p className="text-sm text-gray-400">
                Permanently delete your account and all data
              </p>
            </div>
            <Button variant="destructive">
              <Trash2 className="mr-2 h-4 w-4" />
              Delete Account
            </Button>
          </div>
        </CardContent>
      </Card>
    </>
  );
}

function NotificationSettings() {
  const [emailNotifications, setEmailNotifications] = useState(true);
  const [securityAlerts, setSecurityAlerts] = useState(true);
  const [monitorAlerts, setMonitorAlerts] = useState(true);
  const [weeklyReport, setWeeklyReport] = useState(false);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Notification Preferences</CardTitle>
        <CardDescription>
          Choose how you want to be notified
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <NotificationToggle
          icon={Mail}
          title="Email Notifications"
          description="Receive notifications via email"
          checked={emailNotifications}
          onChange={setEmailNotifications}
        />
        <NotificationToggle
          icon={Shield}
          title="Security Alerts"
          description="Get notified about security issues in your contracts"
          checked={securityAlerts}
          onChange={setSecurityAlerts}
        />
        <NotificationToggle
          icon={Bell}
          title="Monitor Alerts"
          description="Receive alerts from your contract monitors"
          checked={monitorAlerts}
          onChange={setMonitorAlerts}
        />
        <NotificationToggle
          icon={Mail}
          title="Weekly Report"
          description="Receive a weekly summary of your activity"
          checked={weeklyReport}
          onChange={setWeeklyReport}
        />
      </CardContent>
    </Card>
  );
}

function NotificationToggle({
  icon: Icon,
  title,
  description,
  checked,
  onChange,
}: {
  icon: any;
  title: string;
  description: string;
  checked: boolean;
  onChange: (value: boolean) => void;
}) {
  return (
    <div className="flex items-center justify-between rounded-lg border border-gray-800 p-4">
      <div className="flex items-start gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/20">
          <Icon className="h-5 w-5 text-accent-cyan" />
        </div>
        <div>
          <p className="font-medium text-white">{title}</p>
          <p className="text-sm text-gray-400">{description}</p>
        </div>
      </div>
      <button
        onClick={() => onChange(!checked)}
        className={`relative h-6 w-11 rounded-full transition-colors ${
          checked ? 'bg-accent-cyan' : 'bg-gray-700'
        }`}
      >
        <span
          className={`absolute top-0.5 left-0.5 h-5 w-5 rounded-full bg-white transition-transform ${
            checked ? 'translate-x-5' : 'translate-x-0'
          }`}
        />
      </button>
    </div>
  );
}

function SecuritySettings({ onLogout }: { onLogout: () => void }) {
  const [showPassword, setShowPassword] = useState(false);
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  const handleChangePassword = async () => {
    setIsSaving(true);
    // Simulate API call
    await new Promise((resolve) => setTimeout(resolve, 1000));
    setCurrentPassword('');
    setNewPassword('');
    setConfirmPassword('');
    setIsSaving(false);
  };

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Change Password</CardTitle>
          <CardDescription>
            Update your password to keep your account secure
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>Current Password</Label>
            <div className="relative">
              <Input
                type={showPassword ? 'text' : 'password'}
                value={currentPassword}
                onChange={(e) => setCurrentPassword(e.target.value)}
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white"
              >
                {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
          </div>
          <div className="space-y-2">
            <Label>New Password</Label>
            <Input
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label>Confirm New Password</Label>
            <Input
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
            />
          </div>
          <Button onClick={handleChangePassword} isLoading={isSaving}>
            Update Password
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Active Sessions</CardTitle>
          <CardDescription>
            Manage your active sessions across devices
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center justify-between rounded-lg border border-gray-800 p-4">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-green/20">
                <Globe className="h-5 w-5 text-accent-green" />
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <p className="font-medium text-white">Current Session</p>
                  <Badge variant="success" className="text-xs">Active</Badge>
                </div>
                <p className="text-sm text-gray-400">Chrome on macOS • San Francisco, US</p>
              </div>
            </div>
          </div>
          <div className="flex items-center justify-between rounded-lg border border-gray-800 p-4">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-700">
                <Smartphone className="h-5 w-5 text-gray-400" />
              </div>
              <div>
                <p className="font-medium text-white">iPhone 15</p>
                <p className="text-sm text-gray-400">Safari on iOS • 2 hours ago</p>
              </div>
            </div>
            <Button variant="ghost" size="sm" className="text-severity-critical">
              Revoke
            </Button>
          </div>
        </CardContent>
      </Card>

      <Button variant="outline" onClick={onLogout} className="w-full">
        <LogOut className="mr-2 h-4 w-4" />
        Sign Out of All Devices
      </Button>
    </>
  );
}

function ApiSettings() {
  const [showKey, setShowKey] = useState(false);
  const apiKey = 'gck_live_1234567890abcdef1234567890abcdef';

  const handleRegenerateKey = () => {
    if (confirm('Are you sure? This will invalidate your current API key.')) {
      // Regenerate key
    }
  };

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>API Key</CardTitle>
          <CardDescription>
            Use this key to authenticate API requests
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-2">
            <div className="flex-1 rounded-lg border border-gray-800 bg-dark-bg p-3 font-mono text-sm">
              {showKey ? apiKey : '•'.repeat(40)}
            </div>
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setShowKey(!showKey)}
            >
              {showKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </Button>
            <Button
              variant="ghost"
              size="icon"
              onClick={() => copyToClipboard(apiKey)}
            >
              <Copy className="h-4 w-4" />
            </Button>
          </div>

          <div className="flex items-center justify-between rounded-lg border border-accent-yellow/30 bg-accent-yellow/5 p-4">
            <div>
              <p className="font-medium text-white">Keep your API key secret</p>
              <p className="text-sm text-gray-400">
                Never share your API key in public repositories or client-side code
              </p>
            </div>
          </div>

          <Button variant="outline" onClick={handleRegenerateKey}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Regenerate API Key
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>API Usage</CardTitle>
          <CardDescription>
            Monitor your API usage this month
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <UsageBar label="Requests" used={2456} limit={10000} />
            <UsageBar label="Analyses" used={156} limit={1000} />
            <UsageBar label="Traces" used={89} limit={500} />
          </div>
        </CardContent>
      </Card>
    </>
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
  const percentage = Math.min((used / limit) * 100, 100);

  return (
    <div>
      <div className="flex items-center justify-between text-sm">
        <span className="text-gray-400">{label}</span>
        <span className="text-white">
          {used.toLocaleString()} / {limit.toLocaleString()}
        </span>
      </div>
      <div className="mt-2 h-2 rounded-full bg-gray-800">
        <div
          className="h-2 rounded-full bg-accent-cyan transition-all"
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  );
}

function PreferencesSettings() {
  const [theme, setTheme] = useState('dark');
  const [language, setLanguage] = useState('en');
  const [defaultChain, setDefaultChain] = useState('ethereum');

  return (
    <Card>
      <CardHeader>
        <CardTitle>Preferences</CardTitle>
        <CardDescription>
          Customize your experience
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label>Theme</Label>
            <Select value={theme} onValueChange={setTheme}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="dark">Dark</SelectItem>
                <SelectItem value="light">Light (Coming soon)</SelectItem>
                <SelectItem value="system">System</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label>Language</Label>
            <Select value={language} onValueChange={setLanguage}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="en">English</SelectItem>
                <SelectItem value="es">Spanish (Coming soon)</SelectItem>
                <SelectItem value="zh">Chinese (Coming soon)</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div className="space-y-2">
          <Label>Default Chain</Label>
          <Select value={defaultChain} onValueChange={setDefaultChain}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="ethereum">Ethereum</SelectItem>
              <SelectItem value="polygon">Polygon</SelectItem>
              <SelectItem value="arbitrum">Arbitrum</SelectItem>
              <SelectItem value="optimism">Optimism</SelectItem>
              <SelectItem value="base">Base</SelectItem>
            </SelectContent>
          </Select>
          <p className="text-xs text-gray-500">
            This chain will be pre-selected when adding contracts or tracing transactions
          </p>
        </div>
      </CardContent>
    </Card>
  );
}
