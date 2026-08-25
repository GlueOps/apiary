import { Component, type ErrorInfo, type ReactNode } from 'react';

import { AlertTriangle } from 'lucide-react';

import { Button } from '@/components/ui/button.tsx';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card.tsx';

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
}

export class ErrorBoundary extends Component<Props, State> {
  public state: State = {
    hasError: false,
    error: null,
    errorInfo: null,
  };

  public static getDerivedStateFromError(error: Error): State {
    return {
      hasError: true,
      error,
      errorInfo: null,
    };
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('ErrorBoundary caught an error:', error, errorInfo);
    this.setState({
      error,
      errorInfo,
    });
  }

  private handleReset = () => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
    });
    window.location.href = '/';
  };

  public render() {
    if (this.state.hasError) {
      return (
        <div className="flex min-h-screen items-center justify-center bg-background p-6">
          <Card>
            <CardHeader className="items-center text-center">
              <AlertTriangle className="mb-2 h-16 w-16 text-destructive" />
              <CardTitle className="text-3xl font-bold">Oops! Something went wrong</CardTitle>
              <CardDescription className="max-w-md text-base leading-7">
                We&apos;re sorry for the inconvenience. An unexpected error has occurred. Please try
                refreshing the page or contact support if the problem persists.
              </CardDescription>
            </CardHeader>

            <CardContent className="text-center">
              {import.meta.env.DEV && this.state.error && (
                <div className="mt-2 rounded-md border bg-muted p-4 text-left">
                  <pre className="text-xs wrap-break-word whitespace-pre-wrap text-muted-foreground">
                    {this.state.error.toString()}
                    {this.state.errorInfo?.componentStack}
                  </pre>
                </div>
              )}
              <Button onClick={this.handleReset} className="mt-6">
                Return to Dashboard
              </Button>
            </CardContent>
          </Card>
        </div>
      );
    }
    return this.props.children;
  }
}
