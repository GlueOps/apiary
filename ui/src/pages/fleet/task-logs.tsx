import { useEffect, useRef, useState } from 'react';

import { useQuery } from '@tanstack/react-query';

import { Button } from '@/components/ui/button';
import { fleetTasksLogsOptions } from '@/sdk/@tanstack/react-query.gen';

const STREAM_TONE: Record<string, string> = {
  stderr: 'text-destructive',
  // The agent's own narration — what it decided and why — is dimmed so it reads
  // as commentary rather than as command output it is describing.
  agent: 'text-muted-foreground italic',
  stdout: '',
};

/**
 * Tails a task's output.
 *
 * Polls with the highest sequence already held rather than re-reading the whole
 * stream: an `apt upgrade` produces thousands of lines, and refetching all of
 * them every two seconds would make the viewer heavier than the work it is
 * watching. The server's `since` parameter exists for this.
 */
export function TaskLogs({ taskId, onClose }: { taskId: string; onClose: () => void }) {
  const [since, setSince] = useState(0);
  const [lines, setLines] = useState<
    { seq: number; stream: string; message: string; logged_at: string }[]
  >([]);
  const [follow, setFollow] = useState(true);
  const bottom = useRef<HTMLDivElement | null>(null);

  const { data } = useQuery({
    ...fleetTasksLogsOptions({ path: { id: taskId }, query: { since } }),
    refetchInterval: 2000,
  });

  useEffect(() => {
    const incoming = data?.lines ?? [];
    if (incoming.length === 0) return;
    setLines((prev) => [...prev, ...incoming]);
    setSince(incoming[incoming.length - 1].seq);
  }, [data]);

  useEffect(() => {
    // Only when following. Yanking the viewport to the bottom while someone is
    // reading the error that started the failure is the classic log-viewer sin.
    if (follow) bottom.current?.scrollIntoView({ block: 'end' });
  }, [lines, follow]);

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <Button
          size="xs"
          variant={follow ? 'default' : 'outline'}
          onClick={() => setFollow((f) => !f)}
        >
          {follow ? 'Following' : 'Follow'}
        </Button>
        <Button size="xs" variant="ghost" onClick={onClose}>
          Close
        </Button>
        <span className="ml-auto text-xs text-muted-foreground tabular-nums">
          {lines.length} line{lines.length === 1 ? '' : 's'}
        </span>
      </div>

      <div
        className="max-h-96 overflow-auto rounded-lg bg-muted/40 p-3 font-mono text-xs"
        onScroll={(e) => {
          const el = e.currentTarget;
          // Following is a consequence of being at the bottom, not a mode the
          // user has to remember to leave: scrolling up stops it.
          const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 24;
          if (!atBottom && follow) setFollow(false);
        }}
      >
        {lines.length === 0 ? (
          <p className="text-muted-foreground">Waiting for output…</p>
        ) : (
          lines.map((l) => (
            <div key={l.seq} className={`whitespace-pre-wrap ${STREAM_TONE[l.stream] ?? ''}`}>
              {l.message}
            </div>
          ))
        )}
        <div ref={bottom} />
      </div>
    </div>
  );
}
