import { useMemo } from 'react';

import { useQuery } from '@tanstack/react-query';

import { beeLabel, hiveIdOf } from '@/pages/fleet/bee-table';
import { fleetBeesListOptions } from '@/sdk/@tanstack/react-query.gen';

/**
 * Sentinel for "no fleet" in the scope selects.
 *
 * The API expresses org-wide as an ABSENT hive_id, but a select needs a value
 * for every option and '' is indistinguishable from "nothing chosen yet". This
 * is translated back to an omitted hive_id at the call site.
 */
export const ORG_WIDE = '__org_wide__';

export type Fleet = { id: string; label: string; hostnames: string[] };

/**
 * The set of fleets this org has, assembled from what we can actually see.
 *
 * There is no hives LIST endpoint, so fleets are derived from two places: the
 * hive_id on each enrolled agent, and the hive_id on each existing binding. The
 * second source matters — a fleet whose agents have not enrolled yet would
 * otherwise vanish, and its binding would render as pointing at nothing.
 *
 * The consequence to be honest about on screen: a fleet with neither an agent
 * nor a binding is invisible here, so a blast-radius count is a floor, not a
 * total.
 */
export function useFleets(bindings: ReadonlyArray<{ hive_id?: string }>) {
  const { data, isLoading } = useQuery(fleetBeesListOptions());

  const fleets = useMemo<Fleet[]>(() => {
    const hostnames = new Map<string, string[]>();
    for (const bee of data?.bees ?? []) {
      // hiveIdOf rather than a local truthiness check. bee-table.tsx already
      // owns the definition of "no fleet" and treats the nil uuid as one too;
      // two modules disagreeing about that means a host grouped under a fleet
      // that does not exist here while correctly shown as fleetless there.
      const id = hiveIdOf(bee);
      if (!id) continue;
      hostnames.set(id, [...(hostnames.get(id) ?? []), beeLabel(bee)]);
    }
    for (const binding of bindings) {
      if (binding.hive_id && !hostnames.has(binding.hive_id)) hostnames.set(binding.hive_id, []);
    }
    return [...hostnames.entries()]
      .map(([id, hosts]) => ({
        id,
        // Hive names are not on any payload the UI can read, so the id prefix is
        // the only stable handle. Hostnames carry the actual recognition.
        label: `Fleet ${id.slice(0, 8)}`,
        hostnames: [...hosts].sort(),
      }))
      .sort((a, b) => a.label.localeCompare(b.label));
  }, [data, bindings]);

  return { fleets, isLoading };
}

export function describeScope(hiveId: string | undefined, fleets: Fleet[]) {
  if (!hiveId) return 'Org-wide';
  return fleets.find((f) => f.id === hiveId)?.label ?? `Fleet ${hiveId.slice(0, 8)}`;
}
