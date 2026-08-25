import { Navigate, Outlet, useLocation } from 'react-router-dom';

import { useAuth } from '@/hooks/use-auth.ts';

export const ProtectedRoute = () => {
  const { authed } = useAuth();
  const loc = useLocation();

  if (!authed) {
    return <Navigate to={`/login?to=${encodeURIComponent(loc.pathname + loc.search)}`} replace />;
  }

  return <Outlet />;
};
