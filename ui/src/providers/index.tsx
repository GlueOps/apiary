import type { ReactNode } from 'react';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

import { ErrorBoundary } from '@/components/error-boundary.tsx';
import { Toaster } from '@/components/ui/sonner.tsx';
import { TooltipProvider } from '@/components/ui/tooltip.tsx';
import { AuthProvider } from '@/providers/auth-provider.tsx';
import { ThemeProvider } from '@/providers/theme-provider.tsx';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 60 * 1000,
      retry: false,
      refetchOnWindowFocus: true,
      refetchOnReconnect: true,
    },
  },
});

export const Providers = ({ children }: { children: ReactNode }) => {
  return (
    <ThemeProvider defaultTheme="system" storageKey="apiary-theme">
      <ErrorBoundary>
        <QueryClientProvider client={queryClient}>
          <AuthProvider>
            <TooltipProvider>{children}</TooltipProvider>
          </AuthProvider>
          <Toaster richColors expand position="top-center" />
        </QueryClientProvider>
      </ErrorBoundary>
    </ThemeProvider>
  );
};
