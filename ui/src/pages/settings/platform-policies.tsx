import {
  platformPoliciesClearMutation,
  platformPoliciesListOptions,
  platformPoliciesListQueryKey,
  platformPoliciesSetMutation,
} from '@/sdk/@tanstack/react-query.gen';

import { PolicyEditor } from './policy-editor';

export const PlatformPolicies = () => (
  <PolicyEditor
    title="Platform policy"
    description="Set here, enforced everywhere. This is the origin of the floor — nothing sits
      above it, so an unset policy is simply not enforced rather than inherited from somewhere.
      An organization can require more than what you set, never less, so enabling something here
      applies it to every tenant including those who have switched it off for themselves."
    inherits={false}
    listOptions={platformPoliciesListOptions}
    listQueryKey={platformPoliciesListQueryKey}
    setMutation={platformPoliciesSetMutation}
    clearMutation={platformPoliciesClearMutation}
    canEdit
  />
);
