import * as React from 'react';
import { cn } from '@/lib/utils';

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  error?: string;
  icon?: React.ReactNode;
}

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, error, icon, ...props }, ref) => {
    return (
      <div className="relative w-full">
        {icon && (
          <div className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">
            {icon}
          </div>
        )}
        <input
          type={type}
          className={cn(
            'flex h-11 w-full rounded-lg border bg-dark-bg px-4 py-2 text-sm text-white transition-all duration-200',
            'placeholder:text-gray-500',
            'focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-dark-bg',
            'disabled:cursor-not-allowed disabled:opacity-50',
            icon && 'pl-10',
            error
              ? 'border-severity-critical focus:border-severity-critical focus:ring-severity-critical/50'
              : 'border-gray-600 focus:border-accent-cyan focus:ring-accent-cyan/50',
            className
          )}
          ref={ref}
          {...props}
        />
        {error && (
          <p className="mt-1.5 text-xs text-severity-critical">{error}</p>
        )}
      </div>
    );
  }
);
Input.displayName = 'Input';

export { Input };
