# OrganizationApi

All URIs are relative to *http://127.0.0.1:8282*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**orgMembersList**](OrganizationApi.md#orgmemberslist) | **GET** /api/v1/organization/members | List the members of this organization |
| [**orgMembersRemove**](OrganizationApi.md#orgmembersremove) | **DELETE** /api/v1/organization/members/{id} | Remove a member from this organization |
| [**orgMembersUpdateRole**](OrganizationApi.md#orgmembersupdaterole) | **PATCH** /api/v1/organization/members/{id} | Change a member\&#39;s role |
| [**orgPoliciesClear**](OrganizationApi.md#orgpoliciesclear) | **DELETE** /api/v1/organization/policies/{kind} | Return a policy to inherit |
| [**orgPoliciesImpact**](OrganizationApi.md#orgpoliciesimpact) | **GET** /api/v1/organization/policies/{kind}/impact | How many members this policy would lock out |
| [**orgPoliciesList**](OrganizationApi.md#orgpolicieslist) | **GET** /api/v1/organization/policies | List this organization\&#39;s policies |
| [**orgPoliciesSet**](OrganizationApi.md#orgpoliciesset) | **PUT** /api/v1/organization/policies/{kind} | Set a policy for this organization |



## orgMembersList

> ListMembersResponseBody orgMembersList()

List the members of this organization

Any member may see who else is in their organization. Roles are included because they determine who to ask for access.

### Example

```ts
import {
  Configuration,
  OrganizationApi,
} from '@glueops/apiary-sdk';
import type { OrgMembersListRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new OrganizationApi(config);

  try {
    const data = await api.orgMembersList();
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**ListMembersResponseBody**](ListMembersResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## orgMembersRemove

> RemoveMemberResponseBody orgMembersRemove(id)

Remove a member from this organization

Requires admin, and an owner to remove an owner. The last owner cannot be removed.

### Example

```ts
import {
  Configuration,
  OrganizationApi,
} from '@glueops/apiary-sdk';
import type { OrgMembersRemoveRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new OrganizationApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
  } satisfies OrgMembersRemoveRequest;

  try {
    const data = await api.orgMembersRemove(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **id** | `string` |  | [Defaults to `undefined`] |

### Return type

[**RemoveMemberResponseBody**](RemoveMemberResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## orgMembersUpdateRole

> UpdateRoleResponseBody orgMembersUpdateRole(id, updateRoleRequestBody)

Change a member\&#39;s role

Requires admin, and an owner for anything crossing the owner line in either direction. The last owner cannot be demoted. Impersonated sessions are refused: an operator acting as a user must not be able to grant themselves standing access.

### Example

```ts
import {
  Configuration,
  OrganizationApi,
} from '@glueops/apiary-sdk';
import type { OrgMembersUpdateRoleRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new OrganizationApi(config);

  const body = {
    // string | The membership id, not the user id.
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
    // UpdateRoleRequestBody
    updateRoleRequestBody: ...,
  } satisfies OrgMembersUpdateRoleRequest;

  try {
    const data = await api.orgMembersUpdateRole(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **id** | `string` | The membership id, not the user id. | [Defaults to `undefined`] |
| **updateRoleRequestBody** | [UpdateRoleRequestBody](UpdateRoleRequestBody.md) |  | |

### Return type

[**UpdateRoleResponseBody**](UpdateRoleResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## orgPoliciesClear

> DeleteResponseBody1 orgPoliciesClear(kind)

Return a policy to inherit

### Example

```ts
import {
  Configuration,
  OrganizationApi,
} from '@glueops/apiary-sdk';
import type { OrgPoliciesClearRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new OrganizationApi(config);

  const body = {
    // 'require_mfa' | 'require_passkey' | 'password_min_length' | 'password_complexity' | 'session_max_lifetime'
    kind: kind_example,
  } satisfies OrgPoliciesClearRequest;

  try {
    const data = await api.orgPoliciesClear(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **kind** | `require_mfa`, `require_passkey`, `password_min_length`, `password_complexity`, `session_max_lifetime` |  | [Defaults to `undefined`] [Enum: require_mfa, require_passkey, password_min_length, password_complexity, session_max_lifetime] |

### Return type

[**DeleteResponseBody1**](DeleteResponseBody1.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## orgPoliciesImpact

> ImpactResponseBody orgPoliciesImpact(kind)

How many members this policy would lock out

Scoped to this organization. Ask before enabling — the alternative is finding out from the people who can no longer sign in.

### Example

```ts
import {
  Configuration,
  OrganizationApi,
} from '@glueops/apiary-sdk';
import type { OrgPoliciesImpactRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new OrganizationApi(config);

  const body = {
    // 'require_mfa' | 'require_passkey'
    kind: kind_example,
  } satisfies OrgPoliciesImpactRequest;

  try {
    const data = await api.orgPoliciesImpact(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **kind** | `require_mfa`, `require_passkey` |  | [Defaults to `undefined`] [Enum: require_mfa, require_passkey] |

### Return type

[**ImpactResponseBody**](ImpactResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## orgPoliciesList

> ListResponseBody2 orgPoliciesList()

List this organization\&#39;s policies

Each kind shows this organization\&#39;s own position AND the effective result, because \&#39;what applies to us\&#39; is the question being asked and neither level answers it alone. A restriction merges toward strictness: the platform\&#39;s floor cannot be lowered here, only raised.

### Example

```ts
import {
  Configuration,
  OrganizationApi,
} from '@glueops/apiary-sdk';
import type { OrgPoliciesListRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new OrganizationApi(config);

  try {
    const data = await api.orgPoliciesList();
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**ListResponseBody2**](ListResponseBody2.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## orgPoliciesSet

> PolicyView orgPoliciesSet(kind, setRequestBody)

Set a policy for this organization

Requires admin. Impersonated sessions are refused: an operator acting as a user must not be able to change that organization\&#39;s security posture under their name. Note that disabling here does NOT switch off a platform policy — a restriction merges toward strictness, so an organization can raise the floor but never lower it.

### Example

```ts
import {
  Configuration,
  OrganizationApi,
} from '@glueops/apiary-sdk';
import type { OrgPoliciesSetRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new OrganizationApi(config);

  const body = {
    // 'require_mfa' | 'require_passkey' | 'password_min_length' | 'password_complexity' | 'session_max_lifetime'
    kind: kind_example,
    // SetRequestBody
    setRequestBody: ...,
  } satisfies OrgPoliciesSetRequest;

  try {
    const data = await api.orgPoliciesSet(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **kind** | `require_mfa`, `require_passkey`, `password_min_length`, `password_complexity`, `session_max_lifetime` |  | [Defaults to `undefined`] [Enum: require_mfa, require_passkey, password_min_length, password_complexity, session_max_lifetime] |
| **setRequestBody** | [SetRequestBody](SetRequestBody.md) |  | |

### Return type

[**PolicyView**](PolicyView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

