import { IconBuilding } from '@tabler/icons-react';
import { useMatches } from 'react-router-dom';

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb';
import { Separator } from '@/components/ui/separator';
import { SidebarTrigger } from '@/components/ui/sidebar';
import { useAuth } from '@/hooks/use-auth';

export function SiteHeader() {
  const { session } = useAuth();
  const matches = useMatches();

  const currentMatch = [...matches].reverse().find((m) => (m.handle as { title?: string })?.title);
  const pageTitle = (currentMatch?.handle as { title?: string })?.title ?? 'Dashboard';

  return (
    <header className="flex h-12 shrink-0 items-center gap-2 border-b px-4">
      <SidebarTrigger className="-ml-1" />
      <Separator orientation="vertical" className="mr-2 h-4" />
      <Breadcrumb>
        <BreadcrumbList>
          {session?.organization && (
            <>
              <BreadcrumbItem className="hidden md:block">
                <div className="flex items-center gap-1.5 text-muted-foreground">
                  <IconBuilding className="size-3.5" />
                  <span className="text-sm">{session.organization}</span>
                </div>
              </BreadcrumbItem>
              <BreadcrumbSeparator className="hidden md:block" />
            </>
          )}
          <BreadcrumbItem>
            <BreadcrumbPage>{pageTitle}</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>
    </header>
  );
}
