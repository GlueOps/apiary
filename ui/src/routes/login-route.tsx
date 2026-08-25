import { Navigate, useLocation } from 'react-router-dom';

import { useAuth } from '@/hooks/use-auth.ts';
import { Login } from '@/pages/auth/login.tsx';

export const LoginRoute = () => {
  const { authed } = useAuth();
  const loc = useLocation();

  if (authed) {
    const to = new URLSearchParams(loc.search).get('to') || '/';
    return <Navigate to={to} replace />;
  }

  return <Login />;
};
