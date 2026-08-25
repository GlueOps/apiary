# PlatformApi

All URIs are relative to *http://127.0.0.1:8080*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**platformAccessRulesCreate**](PlatformApi.md#platformaccessrulescreate) | **POST** /api/v1/platform/access-rules | Create an auth access rule |
| [**platformAccessRulesDelete**](PlatformApi.md#platformaccessrulesdelete) | **DELETE** /api/v1/platform/access-rules/{id} | Delete an auth access rule |
| [**platformAccessRulesList**](PlatformApi.md#platformaccessruleslist) | **GET** /api/v1/platform/access-rules | List auth access rules |
| [**platformAccessRulesUpdate**](PlatformApi.md#platformaccessrulesupdate) | **PUT** /api/v1/platform/access-rules/{id} | Change a rule\&#39;s policy |
| [**platformPoliciesClear**](PlatformApi.md#platformpoliciesclear) | **DELETE** /api/v1/platform/policies/{kind} | Return a platform policy to inherit |
| [**platformPoliciesImpact**](PlatformApi.md#platformpoliciesimpact) | **GET** /api/v1/platform/policies/{kind}/impact | How many users this policy would lock out |
| [**platformPoliciesList**](PlatformApi.md#platformpolicieslist) | **GET** /api/v1/platform/policies | List platform policies |
| [**platformPoliciesSet**](PlatformApi.md#platformpoliciesset) | **PUT** /api/v1/platform/policies/{kind} | Set a platform policy |



## platformAccessRulesCreate

> AccessRuleView platformAccessRulesCreate(createRequestBody2)

Create an auth access rule

Purpose is required and has no default. Email rules beat domain rules, and among domain rules the most specific wins; anything that is not exactly \&#39;allow\&#39; denies.

### Example

```ts
import {
  Configuration,
  PlatformApi,
} from '@glueops/apiary-sdk';
import type { PlatformAccessRulesCreateRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new PlatformApi(config);

  const body = {
    // CreateRequestBody2
    createRequestBody2: ...,
  } satisfies PlatformAccessRulesCreateRequest;

  try {
    const data = await api.platformAccessRulesCreate(body);
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
| **createRequestBody2** | [CreateRequestBody2](CreateRequestBody2.md) |  | |

### Return type

[**AccessRuleView**](AccessRuleView.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **201** | Created |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## platformAccessRulesDelete

> DeleteResponseBody2 platformAccessRulesDelete(id)

Delete an auth access rule

Answers with a body rather than 204 because deleting a SEEDED rule does not last: the next boot re-creates it as a create_org deny. The response says so explicitly. Re-creating is fail-safe and stays that way — being silent about it was the problem.

### Example

```ts
import {
  Configuration,
  PlatformApi,
} from '@glueops/apiary-sdk';
import type { PlatformAccessRulesDeleteRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new PlatformApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
  } satisfies PlatformAccessRulesDeleteRequest;

  try {
    const data = await api.platformAccessRulesDelete(body);
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

[**DeleteResponseBody2**](DeleteResponseBody2.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## platformAccessRulesList

> ListResponseBody4 platformAccessRulesList()

List auth access rules

Every rule that gates authentication, organization creation and invitation acceptance, across every tenant. Rows written by the boot-time public-domain seed are marked, as are the ones an operator has overridden to allow — that divergence exists only in this table and nothing else reports it.

### Example

```ts
import {
  Configuration,
  PlatformApi,
} from '@glueops/apiary-sdk';
import type { PlatformAccessRulesListRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new PlatformApi(config);

  try {
    const data = await api.platformAccessRulesList();
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

[**ListResponseBody4**](ListResponseBody4.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## platformAccessRulesUpdate

> AccessRuleView platformAccessRulesUpdate(id, updateRequestBody)

Change a rule\&#39;s policy

Policy and reason only. Kind, value and purpose are the rule\&#39;s identity — retargeting one is a delete and a create, so that both facts stay visible. This is also the durable way to permit a seeded public domain: an update survives every restart, whereas a delete is undone by the next one.

### Example

```ts
import {
  Configuration,
  PlatformApi,
} from '@glueops/apiary-sdk';
import type { PlatformAccessRulesUpdateRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new PlatformApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
    // UpdateRequestBody
    updateRequestBody: ...,
  } satisfies PlatformAccessRulesUpdateRequest;

  try {
    const data = await api.platformAccessRulesUpdate(body);
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
| **updateRequestBody** | [UpdateRequestBody](UpdateRequestBody.md) |  | |

### Return type

[**AccessRuleView**](AccessRuleView.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## platformPoliciesClear

> DeleteResponseBody platformPoliciesClear(kind)

Return a platform policy to inherit

Deleting is how the platform withdraws its opinion. That is NOT the same as disabling: a disabled policy contributes nothing to the merge either, but it is a stated position, and the distinction is what an organization sees when deciding whether it needs its own.

### Example

```ts
import {
  Configuration,
  PlatformApi,
} from '@glueops/apiary-sdk';
import type { PlatformPoliciesClearRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new PlatformApi(config);

  const body = {
    // 'require_mfa' | 'require_passkey' | 'password_min_length' | 'password_complexity' | 'session_max_lifetime'
    kind: kind_example,
  } satisfies PlatformPoliciesClearRequest;

  try {
    const data = await api.platformPoliciesClear(body);
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

[**DeleteResponseBody**](DeleteResponseBody.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## platformPoliciesImpact

> ImpactResponseBody platformPoliciesImpact(kind)

How many users this policy would lock out

Ask BEFORE enabling. Turning on require_mfa across a platform where half the users have never enrolled a factor locks them out at their next request, and the only signal is a support queue.

### Example

```ts
import {
  Configuration,
  PlatformApi,
} from '@glueops/apiary-sdk';
import type { PlatformPoliciesImpactRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new PlatformApi(config);

  const body = {
    // 'require_mfa' | 'require_passkey'
    kind: kind_example,
  } satisfies PlatformPoliciesImpactRequest;

  try {
    const data = await api.platformPoliciesImpact(body);
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

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## platformPoliciesList

> ListResponseBody platformPoliciesList()

List platform policies

Every policy kind, whether or not the platform has an opinion on it. A kind the platform has never set reads as \&#39;inherit\&#39;, which is the identity element of the merge.

### Example

```ts
import {
  Configuration,
  PlatformApi,
} from '@glueops/apiary-sdk';
import type { PlatformPoliciesListRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new PlatformApi(config);

  try {
    const data = await api.platformPoliciesList();
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

[**ListResponseBody**](ListResponseBody.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## platformPoliciesSet

> PolicyView platformPoliciesSet(kind, setRequestBody)

Set a platform policy

Impersonated sessions are refused: a platform policy applies to every tenant, and an operator acting as somebody else should not be able to change it under their name.

### Example

```ts
import {
  Configuration,
  PlatformApi,
} from '@glueops/apiary-sdk';
import type { PlatformPoliciesSetRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new PlatformApi(config);

  const body = {
    // 'require_mfa' | 'require_passkey' | 'password_min_length' | 'password_complexity' | 'session_max_lifetime'
    kind: kind_example,
    // SetRequestBody
    setRequestBody: ...,
  } satisfies PlatformPoliciesSetRequest;

  try {
    const data = await api.platformPoliciesSet(body);
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

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

