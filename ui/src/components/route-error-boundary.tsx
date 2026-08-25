import { AlertTriangle } from 'lucide-react';
import { isRouteErrorResponse, useNavigate, useRouteError } from 'react-router-dom';

import { Button } from '@/components/ui/button.tsx';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card.tsx';

export function RouteErrorBoundary() {
  const error = useRouteError();
  const navigate = useNavigate();

  let title = 'Something went wrong';
  let description = 'An unexpected error occurred. Please try again or return to the dashboard.';
  let status: number | null = null;

  if (isRouteErrorResponse(error)) {
    status = error.status;

    if (error.status === 404) {
      title = 'Page not found';
      description = 'The page you’re looking for does not exist or has been moved.';
    } else {
      title = `${error.status} ${error.statusText}`;
      description =
        typeof error.data === 'string'
          ? error.data
          : 'A route error occurred while loading this page.';
    }
  } else if (error instanceof Error) {
    title = 'Application error';
    description = error.message || description;
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-6">
      <Card className="w-full max-w-lg">
        <CardHeader className="items-center text-center">
          <AlertTriangle className="mb-2 h-16 w-16 text-destructive" />
          <CardTitle className="text-3xl font-bold">
            {status ? `${status} · ${title}` : title}
          </CardTitle>
          <CardDescription className="max-w-md text-base leading-7">{description}</CardDescription>
        </CardHeader>

        <CardContent className="text-center">
          {import.meta.env.DEV ? (
            <div className="mt-2 rounded-md border bg-muted p-4 text-left">
              <pre className="text-xs wrap-break-word whitespace-pre-wrap text-muted-foreground">
                {formatRouteError(error)}
              </pre>
            </div>
          ) : null}

          <div className="mt-6 flex justify-center gap-2">
            <Button variant="outline" onClick={() => navigate(-1)}>
              Go Back
            </Button>
            <Button onClick={() => navigate('/', { replace: true })}>Go Home</Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function formatRouteError(error: unknown) {
  if (isRouteErrorResponse(error)) {
    return JSON.stringify(
      {
        status: error.status,
        statusText: error.statusText,
        data: error.data,
      },
      null,
      2,
    );
  }

  if (error instanceof Error) {
    return `${error.name}: ${error.message}\n\n${error.stack ?? ''}`;
  }

  try {
    return JSON.stringify(error, null, 2);
  } catch {
    return String(error);
  }
}
