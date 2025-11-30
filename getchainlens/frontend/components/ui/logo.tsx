'use client';

import React, { useId } from 'react';

interface LogoProps {
  className?: string;
  size?: number;
}

export function Logo({ className = '', size = 36 }: LogoProps) {
  const uniqueId = useId();
  const bgId = `logo-bg-${uniqueId}`;
  const accentId = `logo-accent-${uniqueId}`;

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 512 512"
      width={size}
      height={size}
      className={className}
      aria-label="ChainLens Logo"
      role="img"
    >
      <defs>
        <linearGradient id={bgId} x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#667eea" />
          <stop offset="100%" stopColor="#764ba2" />
        </linearGradient>
        <linearGradient id={accentId} x1="0%" y1="0%" x2="100%" y2="0%">
          <stop offset="0%" stopColor="#00d4aa" />
          <stop offset="100%" stopColor="#00b894" />
        </linearGradient>
      </defs>

      {/* Background circle */}
      <circle cx="256" cy="256" r="240" fill={`url(#${bgId})`} />

      {/* Inner glow ring */}
      <circle cx="256" cy="256" r="220" fill="none" stroke="#fff" strokeWidth="2" opacity="0.1" />

      {/* Magnifying glass */}
      <circle cx="230" cy="210" r="100" fill="none" stroke="#fff" strokeWidth="28" strokeLinecap="round" />
      <line x1="305" y1="285" x2="380" y2="360" stroke="#fff" strokeWidth="32" strokeLinecap="round" />

      {/* Blockchain links inside lens */}
      <g stroke={`url(#${accentId})`} strokeWidth="16" fill="none" strokeLinecap="round">
        <rect x="160" y="185" width="50" height="28" rx="14" />
        <line x1="210" y1="199" x2="230" y2="199" />
        <rect x="230" y="205" width="50" height="28" rx="14" />
      </g>

      {/* Center focus point */}
      <circle cx="230" cy="210" r="20" fill="#fff" opacity="0.9" />
      <circle cx="230" cy="210" r="10" fill={`url(#${accentId})`} />
    </svg>
  );
}
