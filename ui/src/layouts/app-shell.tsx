import {
  IconAlertTriangle,
  IconBuildingWarehouse,
  IconChevronRight,
  IconCpu,
  IconEye,
  IconKey,
  IconLayoutDashboard,
  IconLock,
  IconLockAccess,
  IconLogout,
  IconRoute,
  IconServer,
  IconServerBolt,
  IconShield,
  IconShieldLock,
  IconUser,
  IconUsers,
  IconWorldCog,
} from '@tabler/icons-react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Link, Outlet, useLocation } from 'react-router-dom';
import { toast } from 'sonner';

import { PLATFORM_ADMIN_ROLE } from '@/components/platform-route.tsx';
import { SiteHeader } from '@/components/site-header.tsx';
import { Button } from '@/components/ui/button.tsx';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible.tsx';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarProvider,
} from '@/components/ui/sidebar.tsx';
import { useAuth } from '@/hooks/use-auth.ts';
import { cn } from '@/lib/utils.ts';
import { adminImpersonateEndMutation, meGetOptions } from '@/sdk/@tanstack/react-query.gen';

import { type NavEntry, activeHref, allHrefs, isGroup } from './nav.ts';

function parseJwtPayload(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    return JSON.parse(atob(parts[1].replace(/-/g, '+').replace(/_/g, '/')));
  } catch {
    return null;
  }
}

function ImpersonationBanner() {
  const { session, clear } = useAuth();
  const { data: meData } = useQuery({ ...meGetOptions() });

  const endImpersonation = useMutation({
    ...adminImpersonateEndMutation(),
    onSuccess: () => {
      clear();
      toast.success('Impersonation ended');
    },
    onError: () => {
      // Even on error, clear the session since the impersonation token is one-time-use
      clear();
    },
  });

  if (!session?.accessToken) return null;

  const claims = parseJwtPayload(session.accessToken);
  if (!claims?.imp) return null;

  const email = meData?.email ?? 'user';

  return (
    <div className="flex items-center justify-between gap-3 bg-amber-500 px-4 py-2 text-amber-950">
      <div className="flex items-center gap-2 text-sm font-medium">
        <IconEye className="size-4" />
        Viewing as <strong>{email}</strong> in{' '}
        <strong>{session.organization || session.organizationId}</strong>
      </div>
      <Button
        variant="outline"
        size="sm"
        className="h-7 border-amber-700 bg-amber-600 text-amber-950 hover:bg-amber-700"
        disabled={endImpersonation.isPending}
        onClick={() => endImpersonation.mutate({ body: {} })}
      >
        {endImpersonation.isPending ? 'Ending...' : 'End impersonation'}
      </Button>
    </div>
  );
}

// Grouped to match the routes. Agents, Hypervisors and Hives all live under
// /fleet and are three views of one subject, so listing them as siblings of
// Dashboard said nothing about how they relate — and it made the two firewall
// entries look like two unrelated features rather than one.
const navItems: NavEntry[] = [
  { title: 'Dashboard', href: '/', icon: IconLayoutDashboard },
  { title: 'Credentials', href: '/estates', icon: IconServer },
  {
    title: 'Proxmox Datacenter',
    icon: IconCpu,
    children: [
      { title: 'Clusters', href: '/fleet/hives', icon: IconBuildingWarehouse },
      { title: 'Hypervisors', href: '/fleet/hypervisors', icon: IconServerBolt },
      { title: 'Agents', href: '/fleet', icon: IconCpu },
    ],
  },
  {
    title: 'Firewall',
    icon: IconShield,
    children: [
      { title: 'Rules', href: '/firewall/rulesets', icon: IconShield },
      { title: 'Bindings', href: '/firewall/bindings', icon: IconRoute },
    ],
  },
];

const settingsItems = [
  { title: 'Profile', href: '/settings/profile', icon: IconUser },
  { title: 'Security', href: '/settings/security', icon: IconShieldLock },
  { title: 'Team', href: '/settings/team', icon: IconUsers },
  { title: 'Security policy', href: '/settings/policies', icon: IconLock },
  { title: 'API keys', href: '/settings/api-keys', icon: IconKey },
  { title: 'Fleet SSH keys', href: '/settings/ssh-keys', icon: IconKey },
  { title: 'Danger zone', href: '/settings/danger-zone', icon: IconAlertTriangle },
];

// Mirrors RequireAdminAuth on the server, which admits only this exact role
// (internal/apimiddleware/admin_auth.go). Anything the platform group links to
// 403s for every other role, so the check has to be the same one.
// Imported rather than redeclared: the nav entry and the route guard deciding
// 'platform admin' differently is how a link appears that its own page rejects.

// Access rules bind every tenant at once, so the entry sits in the platform
// group rather than under Settings even though its path is /settings/*.
const platformItems = [
  { title: 'Platform policy', href: '/platform/policies', icon: IconWorldCog },
  { title: 'Access rules', href: '/settings/access-rules', icon: IconLockAccess },
];

export const AppShell = () => {
  const { pathname } = useLocation();
  const active = activeHref(pathname, allHrefs(navItems));
  const { logout } = useAuth();
  const { data: me } = useQuery({ ...meGetOptions() });

  // Undefined while /me is in flight, so the group stays hidden until the role
  // is actually known — a customer must never see the link flash in first.
  const isPlatformAdmin = me?.role === PLATFORM_ADMIN_ROLE;

  return (
    <div className="border-grid flex flex-1 flex-col">
      <SidebarProvider defaultOpen>
        <Sidebar collapsible="icon" variant="inset">
          <SidebarHeader>
            <SidebarMenu>
              <SidebarMenuItem>APIary</SidebarMenuItem>
            </SidebarMenu>
          </SidebarHeader>
          <SidebarContent>
            <SidebarGroup>
              <SidebarGroupLabel>Platform</SidebarGroupLabel>
              <SidebarGroupContent>
                <SidebarMenu>
                  {navItems.map((item) =>
                    isGroup(item) ? (
                      <Collapsible
                        key={item.title}
                        // Open when the current page is inside it. An operator who
                        // navigates straight to /fleet/hives should not have to
                        // expand the group to see where they are — and it stays
                        // open afterwards because the user can close it, which
                        // defaultOpen respects and `open` would fight.
                        defaultOpen={item.children.some((c) => c.href === active)}
                        render={<SidebarMenuItem />}
                      >
                        <CollapsibleTrigger
                          render={
                            <SidebarMenuButton
                              // The group itself is not a destination, so it is
                              // never marked active: the child that matches is.
                              // Highlighting both would say the operator is in two
                              // places.
                              className="group/collapsible w-full"
                            />
                          }
                        >
                          <item.icon className="size-4" />
                          <span>{item.title}</span>
                          <IconChevronRight className="ml-auto size-4 transition-transform group-data-[panel-open]/collapsible:rotate-90" />
                        </CollapsibleTrigger>
                        <CollapsibleContent>
                          <SidebarMenuSub>
                            {item.children.map((child) => (
                              <SidebarMenuSubItem key={child.href}>
                                <SidebarMenuSubButton
                                  render={<Link to={child.href} />}
                                  isActive={child.href === active}
                                >
                                  <span>{child.title}</span>
                                </SidebarMenuSubButton>
                              </SidebarMenuSubItem>
                            ))}
                          </SidebarMenuSub>
                        </CollapsibleContent>
                      </Collapsible>
                    ) : (
                      <SidebarMenuItem key={item.href}>
                        <SidebarMenuButton
                          render={<Link to={item.href} />}
                          isActive={item.href === active}
                        >
                          <item.icon className="size-4" />
                          <span>{item.title}</span>
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    ),
                  )}
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>

            <SidebarGroup>
              <SidebarGroupLabel>Settings</SidebarGroupLabel>
              <SidebarGroupContent>
                <SidebarMenu>
                  {settingsItems.map((item) => (
                    <SidebarMenuItem key={item.href}>
                      <SidebarMenuButton
                        render={<Link to={item.href} />}
                        isActive={pathname.startsWith(item.href)}
                      >
                        <item.icon className="size-4" />
                        <span>{item.title}</span>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  ))}
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>

            {isPlatformAdmin && (
              <SidebarGroup>
                <SidebarGroupLabel>Platform administration</SidebarGroupLabel>
                <SidebarGroupContent>
                  <SidebarMenu>
                    {platformItems.map((item) => (
                      <SidebarMenuItem key={item.href}>
                        <SidebarMenuButton
                          render={<Link to={item.href} />}
                          isActive={pathname.startsWith(item.href)}
                        >
                          <item.icon className="size-4" />
                          <span>{item.title}</span>
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    ))}
                  </SidebarMenu>
                </SidebarGroupContent>
              </SidebarGroup>
            )}
          </SidebarContent>
          <SidebarFooter>
            <div className="px-2 py-2">
              <Button variant="ghost" size="sm" className="w-full" onClick={() => logout()}>
                <IconLogout />
                Sign out
              </Button>
            </div>
          </SidebarFooter>
        </Sidebar>

        <SidebarInset>
          <ImpersonationBanner />
          <SiteHeader />
          <main
            id="content"
            className={cn(
              'flex h-full w-full flex-col',
              'has-[div[data-layout=fixed]]:h-svh',
              'group-data-[scroll-locked=1]/body:h-full',
              'has-[data-layout=fixed]:group-data-[scroll-locked=1]/body:h-svh',
            )}
          >
            <Outlet />
          </main>
        </SidebarInset>
      </SidebarProvider>
    </div>
  );
};
