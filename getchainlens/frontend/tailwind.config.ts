import type { Config } from 'tailwindcss';

const config: Config = {
  darkMode: 'class',
  content: [
    './pages/**/*.{js,ts,jsx,tsx,mdx}',
    './components/**/*.{js,ts,jsx,tsx,mdx}',
    './app/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {
      colors: {
        // Background colors
        dark: {
          bg: '#0A0F1A',
          'bg-secondary': '#0D1321',
          card: '#111827',
          'card-hover': '#1F2937',
          surface: '#0A0F1A',
        },
        // Primary colors
        primary: {
          DEFAULT: '#1E3A5F',
          dark: '#0F2744',
          light: '#2A4A73',
        },
        // Accent colors
        accent: {
          cyan: '#00D4FF',
          'cyan-dark': '#00B4D8',
          orange: '#FF6B35',
          'orange-light': '#FF8A5C',
          yellow: '#FBBF24',
          green: '#10B981',
          blue: '#3B82F6',
        },
        // Severity colors for vulnerabilities
        severity: {
          critical: '#DC2626',
          'critical-bg': 'rgba(220, 38, 38, 0.1)',
          high: '#F97316',
          'high-bg': 'rgba(249, 115, 22, 0.1)',
          medium: '#FBBF24',
          'medium-bg': 'rgba(251, 191, 36, 0.1)',
          low: '#3B82F6',
          'low-bg': 'rgba(59, 130, 246, 0.1)',
          info: '#6B7280',
          'info-bg': 'rgba(107, 114, 128, 0.1)',
        },
        // Neutral colors
        neutral: {
          white: '#FFFFFF',
          'light-gray': '#F5F7FA',
          gray: '#9CA3AF',
          'dark-gray': '#6B7280',
          black: '#0A0A0A',
        },
        // Chain colors
        chain: {
          ethereum: '#627EEA',
          polygon: '#8247E5',
          arbitrum: '#28A0F0',
          optimism: '#FF0420',
          base: '#0052FF',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
      },
      fontSize: {
        'display-hero': ['3.75rem', { lineHeight: '1.1', fontWeight: '800', letterSpacing: '-0.02em' }],
        'display-lg': ['3rem', { lineHeight: '1.1', fontWeight: '700' }],
        'display-md': ['2.25rem', { lineHeight: '1.2', fontWeight: '700' }],
        'heading-lg': ['1.875rem', { lineHeight: '2.25rem', fontWeight: '600' }],
        'heading-md': ['1.5rem', { lineHeight: '2rem', fontWeight: '600' }],
        'heading-sm': ['1.25rem', { lineHeight: '1.75rem', fontWeight: '600' }],
        'body-lg': ['1.125rem', { lineHeight: '1.75rem', fontWeight: '400' }],
        'body-md': ['1rem', { lineHeight: '1.5rem', fontWeight: '400' }],
        'body-sm': ['0.875rem', { lineHeight: '1.25rem', fontWeight: '400' }],
        'caption': ['0.75rem', { lineHeight: '1rem', fontWeight: '500' }],
      },
      borderRadius: {
        '2xl': '16px',
        'xl': '12px',
        'lg': '8px',
        'md': '6px',
      },
      boxShadow: {
        'glow-cyan': '0 0 20px rgba(0, 212, 255, 0.3)',
        'glow-orange': '0 0 20px rgba(255, 107, 53, 0.3)',
        'glow-green': '0 0 20px rgba(16, 185, 129, 0.3)',
        'glow-red': '0 0 20px rgba(220, 38, 38, 0.3)',
        'card': '0 4px 20px rgba(0, 0, 0, 0.25)',
        'card-hover': '0 8px 30px rgba(0, 0, 0, 0.35)',
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary': 'linear-gradient(135deg, #00D4FF 0%, #1E3A5F 100%)',
        'gradient-card': 'linear-gradient(135deg, rgba(17, 24, 39, 0.8) 0%, rgba(30, 58, 95, 0.4) 100%)',
        'gradient-hero': 'linear-gradient(180deg, #0A0F1A 0%, #111827 100%)',
        'gradient-severity-critical': 'linear-gradient(135deg, rgba(220, 38, 38, 0.2) 0%, rgba(220, 38, 38, 0.05) 100%)',
        'gradient-severity-high': 'linear-gradient(135deg, rgba(249, 115, 22, 0.2) 0%, rgba(249, 115, 22, 0.05) 100%)',
        'gradient-severity-medium': 'linear-gradient(135deg, rgba(251, 191, 36, 0.2) 0%, rgba(251, 191, 36, 0.05) 100%)',
      },
      animation: {
        'fade-in': 'fadeIn 0.5s ease-out',
        'fade-in-up': 'fadeInUp 0.6s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'pulse-glow': 'pulseGlow 2s ease-in-out infinite',
        'scan-line': 'scanLine 3s linear infinite',
        'float': 'float 3s ease-in-out infinite',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        fadeInUp: {
          '0%': { opacity: '0', transform: 'translateY(20px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(-20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' },
        },
        pulseGlow: {
          '0%, 100%': { opacity: '0.4' },
          '50%': { opacity: '0.8' },
        },
        scanLine: {
          '0%': { transform: 'translateY(-100%)' },
          '100%': { transform: 'translateY(100%)' },
        },
        float: {
          '0%, 100%': { transform: 'translateY(0)' },
          '50%': { transform: 'translateY(-10px)' },
        },
      },
      spacing: {
        '18': '4.5rem',
        '88': '22rem',
        '128': '32rem',
      },
    },
  },
  plugins: [],
};

export default config;
