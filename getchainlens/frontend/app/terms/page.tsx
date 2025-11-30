import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';
import { Logo } from '@/components/ui/logo';

export const metadata = {
  title: 'Terms of Service - GetChainLens',
  description: 'Terms of Service for GetChainLens smart contract security platform',
};

export default function TermsPage() {
  return (
    <div className="min-h-screen bg-dark-bg">
      {/* Navbar */}
      <nav className="border-b border-gray-800 bg-dark-bg/80 backdrop-blur-xl">
        <div className="mx-auto max-w-4xl px-4 sm:px-6 lg:px-8">
          <div className="flex h-16 items-center justify-between">
            <Link href="/" className="flex items-center gap-3">
              <Logo size={36} />
              <span className="text-xl font-bold text-white">GetChainLens</span>
            </Link>
            <Link
              href="/"
              className="inline-flex items-center text-gray-400 hover:text-white transition-colors"
            >
              <ArrowLeft className="mr-2 h-4 w-4" />
              Back to Home
            </Link>
          </div>
        </div>
      </nav>

      {/* Content */}
      <main className="mx-auto max-w-4xl px-4 sm:px-6 lg:px-8 py-16">
        <h1 className="text-4xl font-bold text-white mb-8">Terms of Service</h1>
        <p className="text-gray-400 mb-8">Last updated: November 2024</p>

        <div className="prose prose-invert prose-gray max-w-none">
          <section className="mb-12">
            <h2 className="text-2xl font-semibold text-white mb-4">1. Acceptance of Terms</h2>
            <p className="text-gray-300 leading-relaxed">
              By accessing or using GetChainLens (&quot;Service&quot;), you agree to be bound by these
              Terms of Service. If you do not agree to these terms, please do not use our Service.
            </p>
          </section>

          <section className="mb-12">
            <h2 className="text-2xl font-semibold text-white mb-4">2. Description of Service</h2>
            <p className="text-gray-300 leading-relaxed">
              GetChainLens provides smart contract security analysis, gas optimization tools,
              transaction tracing, and related blockchain development services. Our Service is
              designed to assist developers in identifying potential vulnerabilities and
              optimizing their smart contracts.
            </p>
          </section>

          <section className="mb-12">
            <h2 className="text-2xl font-semibold text-white mb-4">3. User Accounts</h2>
            <ul className="list-disc list-inside text-gray-300 space-y-2">
              <li>You must provide accurate and complete registration information</li>
              <li>You are responsible for maintaining the security of your account</li>
              <li>You must notify us immediately of any unauthorized access</li>
              <li>You may not share your account credentials with others</li>
            </ul>
          </section>

          <section className="mb-12">
            <h2 className="text-2xl font-semibold text-white mb-4">4. Acceptable Use</h2>
            <p className="text-gray-300 leading-relaxed mb-4">You agree not to:</p>
            <ul className="list-disc list-inside text-gray-300 space-y-2">
              <li>Use the Service for any illegal purposes</li>
              <li>Submit malicious code designed to harm our systems</li>
              <li>Attempt to gain unauthorized access to our infrastructure</li>
              <li>Interfere with or disrupt the Service</li>
              <li>Reverse engineer or decompile our software</li>
              <li>Resell or redistribute the Service without authorization</li>
            </ul>
          </section>

          <section className="mb-12">
            <h2 className="text-2xl font-semibold text-white mb-4">5. Intellectual Property</h2>
            <p className="text-gray-300 leading-relaxed">
              The Service and its original content, features, and functionality are owned by
              GetChainLens and are protected by international copyright, trademark, and other
              intellectual property laws. Your submitted code remains your property.
            </p>
          </section>

          <section className="mb-12">
            <h2 className="text-2xl font-semibold text-white mb-4">6. Disclaimer of Warranties</h2>
            <p className="text-gray-300 leading-relaxed">
              THE SERVICE IS PROVIDED &quot;AS IS&quot; WITHOUT WARRANTIES OF ANY KIND. While we strive
              to provide accurate security analysis, we do not guarantee that our Service will
              identify all vulnerabilities or that your smart contracts will be secure after
              using our tools. You are solely responsible for the security of your deployed
              smart contracts.
            </p>
          </section>

          <section className="mb-12">
            <h2 className="text-2xl font-semibold text-white mb-4">7. Limitation of Liability</h2>
            <p className="text-gray-300 leading-relaxed">
              IN NO EVENT SHALL GETCHAINLENS BE LIABLE FOR ANY INDIRECT, INCIDENTAL, SPECIAL,
              CONSEQUENTIAL, OR PUNITIVE DAMAGES, INCLUDING LOSS OF PROFITS, DATA, OR OTHER
              INTANGIBLE LOSSES, RESULTING FROM YOUR USE OF THE SERVICE. Our total liability
              shall not exceed the amount paid by you for the Service in the past 12 months.
            </p>
          </section>

          <section className="mb-12">
            <h2 className="text-2xl font-semibold text-white mb-4">8. Subscription and Payment</h2>
            <ul className="list-disc list-inside text-gray-300 space-y-2">
              <li>Paid subscriptions are billed in advance on a monthly or annual basis</li>
              <li>All fees are non-refundable unless otherwise stated</li>
              <li>We reserve the right to modify pricing with 30 days notice</li>
              <li>Failure to pay may result in suspension of your account</li>
            </ul>
          </section>

          <section className="mb-12">
            <h2 className="text-2xl font-semibold text-white mb-4">9. Termination</h2>
            <p className="text-gray-300 leading-relaxed">
              We may terminate or suspend your account at any time for violations of these
              Terms. You may cancel your account at any time. Upon termination, your right
              to use the Service will immediately cease.
            </p>
          </section>

          <section className="mb-12">
            <h2 className="text-2xl font-semibold text-white mb-4">10. Changes to Terms</h2>
            <p className="text-gray-300 leading-relaxed">
              We reserve the right to modify these Terms at any time. We will notify you of
              significant changes via email or through the Service. Continued use after changes
              constitutes acceptance of the new Terms.
            </p>
          </section>

          <section className="mb-12">
            <h2 className="text-2xl font-semibold text-white mb-4">11. Governing Law</h2>
            <p className="text-gray-300 leading-relaxed">
              These Terms shall be governed by and construed in accordance with the laws of
              the jurisdiction in which GetChainLens is incorporated, without regard to its
              conflict of law provisions.
            </p>
          </section>

          <section className="mb-12">
            <h2 className="text-2xl font-semibold text-white mb-4">12. Contact Us</h2>
            <p className="text-gray-300 leading-relaxed">
              If you have questions about these Terms, please contact us at:{' '}
              <a href="mailto:contact@getchainlens.com" className="text-accent-cyan hover:underline">
                contact@getchainlens.com
              </a>
            </p>
          </section>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t border-gray-800 py-8">
        <div className="mx-auto max-w-4xl px-4 sm:px-6 lg:px-8 text-center text-sm text-gray-500">
          &copy; {new Date().getFullYear()} GetChainLens. All rights reserved.
        </div>
      </footer>
    </div>
  );
}
