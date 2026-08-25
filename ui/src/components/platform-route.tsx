import { useQuery } from '@tanstack/react-query';
import { Navigate, Outlet } from 'react-router-dom';

import { meGetOptions } from '@/sdk/@tanstack/react-query.gen';

/**
 * The platform role that may reach these routes.
 *
 * A named constant because the generated type for `role` is a bare `string` —
 * the OpenAPI field carries no enum — so a typo in a comparison is not a compile
 * error. One definition means one place to be wrong, rather than one per route.
 */
export const PLATFORM_ADMIN_ROLE = 'platform_admin';

/**
 * Gates routes that edit platform-wide controls.
 *
 * Hiding the sidebar entry is not protection: it removes the link, not the URL,
 * and anyone who has seen the address once can type it. The endpoints behind
 * these pages are mounted on RegisterAdminProtected and refuse a non-admin, so
 * this is not what stops the data leaking — but without it a customer who lands
 * here gets a fully rendered platform-administration screen whose every request
 * fails, which reads as a broken product rather than as somewhere they should
 * not be.
 *
 * Redirects rather than rendering a refusal, and does so only once the role is
 * KNOWN. Treating "still loading" as "not an admin" would bounce a legitimate
 * platform admin off their own page on every hard refresh.
 */
export const PlatformRoute = () => {
  // The same query the shell uses, so it is served from cache rather than
  // refetched — and so the guard and the nav entry can never disagree about who
  // the caller is.
  const { data: me, isPending } = useQuery({ ...meGetOptions() });

  if (isPending) return null;
  if (me?.role !== PLATFORM_ADMIN_ROLE) return <Navigate to="/" replace />;

  return <Outlet />;
};
