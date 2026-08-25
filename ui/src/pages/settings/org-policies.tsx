import {
  orgPoliciesClearMutation,
  orgPoliciesListOptions,
  orgPoliciesListQueryKey,
  orgPoliciesSetMutation,
} from '@/sdk/@tanstack/react-query.gen';

import { PolicyEditor } from './policy-editor';

export const OrgPolicies = () => (
  <PolicyEditor
    title="Security policy"
    description="What this organization requires of its members. These merge with the platform's
      settings toward whichever is stricter — you can raise the floor here, but not lower it, so a
      policy the platform enforces stays enforced whatever this page says."
    listOptions={orgPoliciesListOptions}
    listQueryKey={orgPoliciesListQueryKey}
    setMutation={orgPoliciesSetMutation}
    clearMutation={orgPoliciesClearMutation}
    canEdit
  />
);
