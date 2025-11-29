import type { Metadata } from 'next';
import { Inter, JetBrains_Mono } from 'next/font/google';
import './globals.css';
import { AuthProvider } from '@/lib/auth-context';

const inter = Inter({
  subsets: ['latin'],
  variable: '--font-inter',
});

const jetbrainsMono = JetBrains_Mono({
  subsets: ['latin'],
  variable: '--font-jetbrains-mono',
});

export const metadata: Metadata = {
  title: 'GetChainLens - Smart Contract Security Platform',
  description: 'Production-grade security analysis, gas optimization, and debugging for Solidity smart contracts. Detect vulnerabilities, estimate gas costs, and trace transactions.',
  keywords: ['solidity', 'smart contracts', 'security', 'ethereum', 'blockchain', 'audit', 'gas optimization'],
  authors: [{ name: 'GetChainLens' }],
  openGraph: {
    title: 'GetChainLens - Smart Contract Security Platform',
    description: 'Production-grade security analysis for Solidity smart contracts',
    url: 'https://getchainlens.com',
    siteName: 'GetChainLens',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'GetChainLens',
    description: 'Production-grade security analysis for Solidity smart contracts',
  },
  robots: {
    index: true,
    follow: true,
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="dark">
      <body className={`${inter.variable} ${jetbrainsMono.variable} font-sans antialiased`}>
        <AuthProvider>
          {children}
        </AuthProvider>
      </body>
    </html>
  );
}
