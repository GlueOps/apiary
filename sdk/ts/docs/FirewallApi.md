# FirewallApi

All URIs are relative to *http://127.0.0.1:8080*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**firewallBindingsCreate**](FirewallApi.md#firewallbindingscreate) | **POST** /api/v1/firewall/bindings | Bind a ruleset to a fleet, or to the whole organization |
| [**firewallBindingsDelete**](FirewallApi.md#firewallbindingsdelete) | **DELETE** /api/v1/firewall/bindings/{id} | Remove a binding |
| [**firewallBindingsList**](FirewallApi.md#firewallbindingslist) | **GET** /api/v1/firewall/bindings | List ruleset bindings |
| [**firewallBindingsUpdate**](FirewallApi.md#firewallbindingsupdate) | **PUT** /api/v1/firewall/bindings/{id} | Reprioritise, disable, or re-scope a binding |
| [**firewallPreview**](FirewallApi.md#firewallpreview) | **GET** /api/v1/firewall/preview | Preview the rules a fleet would receive |
| [**firewallPublish**](FirewallApi.md#firewallpublish) | **POST** /api/v1/firewall/hives/{hive_id}/publish | Publish this fleet\&#39;s rules as the next revision |
| [**firewallRevert**](FirewallApi.md#firewallrevert) | **POST** /api/v1/firewall/hives/{hive_id}/revert | Republish an earlier revision\&#39;s ruleset |
| [**firewallRulesCreate**](FirewallApi.md#firewallrulescreate) | **POST** /api/v1/firewall/rulesets/{id}/rules | Add a rule to a ruleset |
| [**firewallRulesDelete**](FirewallApi.md#firewallrulesdelete) | **DELETE** /api/v1/firewall/rules/{id} | Delete a rule |
| [**firewallRulesList**](FirewallApi.md#firewallruleslist) | **GET** /api/v1/firewall/rulesets/{id}/rules | List a ruleset\&#39;s rules |
| [**firewallRulesUpdate**](FirewallApi.md#firewallrulesupdate) | **PUT** /api/v1/firewall/rules/{id} | Replace a rule |
| [**firewallRulesetsCreate**](FirewallApi.md#firewallrulesetscreate) | **POST** /api/v1/firewall/rulesets | Create a firewall ruleset |
| [**firewallRulesetsDelete**](FirewallApi.md#firewallrulesetsdelete) | **DELETE** /api/v1/firewall/rulesets/{id} | Delete a firewall ruleset |
| [**firewallRulesetsList**](FirewallApi.md#firewallrulesetslist) | **GET** /api/v1/firewall/rulesets | List firewall rulesets |
| [**firewallRulesetsUpdate**](FirewallApi.md#firewallrulesetsupdate) | **PUT** /api/v1/firewall/rulesets/{id} | Rename a firewall ruleset |
| [**firewallSnapshotBody**](FirewallApi.md#firewallsnapshotbody) | **GET** /api/v1/firewall/hives/{hive_id}/snapshots/{revision} | Read one published revision\&#39;s ruleset |
| [**firewallSnapshotsList**](FirewallApi.md#firewallsnapshotslist) | **GET** /api/v1/firewall/hives/{hive_id}/snapshots | List this fleet\&#39;s published revisions |



## firewallBindingsCreate

> BindingView firewallBindingsCreate(createBindingRequestBody)

Bind a ruleset to a fleet, or to the whole organization

### Example

```ts
import {
  Configuration,
  FirewallApi,
} from '@glueops/apiary-sdk';
import type { FirewallBindingsCreateRequest } from '@glueops/apiary-sdk';

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
  const api = new FirewallApi(config);

  const body = {
    // CreateBindingRequestBody
    createBindingRequestBody: ...,
  } satisfies FirewallBindingsCreateRequest;

  try {
    const data = await api.firewallBindingsCreate(body);
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
| **createBindingRequestBody** | [CreateBindingRequestBody](CreateBindingRequestBody.md) |  | |

### Return type

[**BindingView**](BindingView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **201** | Created |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## firewallBindingsDelete

> DeletedResponseBody firewallBindingsDelete(id)

Remove a binding

### Example

```ts
import {
  Configuration,
  FirewallApi,
} from '@glueops/apiary-sdk';
import type { FirewallBindingsDeleteRequest } from '@glueops/apiary-sdk';

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
  const api = new FirewallApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
  } satisfies FirewallBindingsDeleteRequest;

  try {
    const data = await api.firewallBindingsDelete(body);
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

[**DeletedResponseBody**](DeletedResponseBody.md)

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


## firewallBindingsList

> ListBindingsResponseBody firewallBindingsList()

List ruleset bindings

Which rulesets apply where, in render order — lowest priority first. Bindings without a fleet are org-wide.

### Example

```ts
import {
  Configuration,
  FirewallApi,
} from '@glueops/apiary-sdk';
import type { FirewallBindingsListRequest } from '@glueops/apiary-sdk';

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
  const api = new FirewallApi(config);

  try {
    const data = await api.firewallBindingsList();
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

[**ListBindingsResponseBody**](ListBindingsResponseBody.md)

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


## firewallBindingsUpdate

> DeletedResponseBody firewallBindingsUpdate(id, updateBindingRequestBody)

Reprioritise, disable, or re-scope a binding

Priority and enabled are always applied. Scope is applied only when the request carries one — omitting it leaves the binding where it is, so a client that knows nothing about scope cannot widen a fleet binding to the whole organization by accident.  Moving a binding onto a scope where the same ruleset is already bound is refused: the ruleset would render twice, duplicating every rule in it.

### Example

```ts
import {
  Configuration,
  FirewallApi,
} from '@glueops/apiary-sdk';
import type { FirewallBindingsUpdateRequest } from '@glueops/apiary-sdk';

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
  const api = new FirewallApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
    // UpdateBindingRequestBody
    updateBindingRequestBody: ...,
  } satisfies FirewallBindingsUpdateRequest;

  try {
    const data = await api.firewallBindingsUpdate(body);
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
| **updateBindingRequestBody** | [UpdateBindingRequestBody](UpdateBindingRequestBody.md) |  | |

### Return type

[**DeletedResponseBody**](DeletedResponseBody.md)

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


## firewallPreview

> PreviewResponseBody1 firewallPreview(hiveId, swarmId)

Preview the rules a fleet would receive

The flattened, ordered result of every binding that applies — org-wide and fleet-specific together. Composition is invisible from any single screen: editing one ruleset cannot show it interleaving with another, and binding priority outranks the positions being edited. Omit hive_id to see what a host belonging to NO fleet receives.

### Example

```ts
import {
  Configuration,
  FirewallApi,
} from '@glueops/apiary-sdk';
import type { FirewallPreviewRequest } from '@glueops/apiary-sdk';

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
  const api = new FirewallApi(config);

  const body = {
    // string | Omit to preview a host that belongs to NO fleet — the state every migration passes through. (optional)
    hiveId: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
    // string | Preview a host in one group inside the fleet. Requires hive_id. Omit to preview a host in no group, which receives fleet and organization scope only. (optional)
    swarmId: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
  } satisfies FirewallPreviewRequest;

  try {
    const data = await api.firewallPreview(body);
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
| **hiveId** | `string` | Omit to preview a host that belongs to NO fleet — the state every migration passes through. | [Optional] [Defaults to `undefined`] |
| **swarmId** | `string` | Preview a host in one group inside the fleet. Requires hive_id. Omit to preview a host in no group, which receives fleet and organization scope only. | [Optional] [Defaults to `undefined`] |

### Return type

[**PreviewResponseBody1**](PreviewResponseBody1.md)

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


## firewallPublish

> PublishResponseBody firewallPublish(hiveId, publishRequestBody)

Publish this fleet\&#39;s rules as the next revision

Renders every binding that applies to this fleet into one nft ruleset, stores it, and points the fleet at it. Bees converge on their next heartbeat.  Publishing an UNCHANGED policy does nothing and says so. Every revision makes the whole fleet re-apply, and re-applying an identical ruleset flushes and reloads every host\&#39;s firewall to arrive at the state it was already in.

### Example

```ts
import {
  Configuration,
  FirewallApi,
} from '@glueops/apiary-sdk';
import type { FirewallPublishRequest } from '@glueops/apiary-sdk';

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
  const api = new FirewallApi(config);

  const body = {
    // string
    hiveId: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
    // PublishRequestBody
    publishRequestBody: ...,
  } satisfies FirewallPublishRequest;

  try {
    const data = await api.firewallPublish(body);
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
| **hiveId** | `string` |  | [Defaults to `undefined`] |
| **publishRequestBody** | [PublishRequestBody](PublishRequestBody.md) |  | |

### Return type

[**PublishResponseBody**](PublishResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **201** | Created |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## firewallRevert

> PublishResponseBody firewallRevert(hiveId, revertRequestBody)

Republish an earlier revision\&#39;s ruleset

Publishes an OLD revision\&#39;s body as a NEW revision. The number goes up, never down: a bee decides whether to converge by comparing what it applied against this number, and a number that moved backwards leaves a host pinned to the very ruleset being backed out.  The old body verbatim, not a re-render of the old rules — the rules have moved on, which is why this is a backout.

### Example

```ts
import {
  Configuration,
  FirewallApi,
} from '@glueops/apiary-sdk';
import type { FirewallRevertRequest } from '@glueops/apiary-sdk';

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
  const api = new FirewallApi(config);

  const body = {
    // string
    hiveId: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
    // RevertRequestBody
    revertRequestBody: ...,
  } satisfies FirewallRevertRequest;

  try {
    const data = await api.firewallRevert(body);
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
| **hiveId** | `string` |  | [Defaults to `undefined`] |
| **revertRequestBody** | [RevertRequestBody](RevertRequestBody.md) |  | |

### Return type

[**PublishResponseBody**](PublishResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **201** | Created |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## firewallRulesCreate

> RuleView firewallRulesCreate(id, ruleBody)

Add a rule to a ruleset

Refused if the rule cannot mean what it says — a port on a protocol that has none is dropped when rendered, so the stored rule would be wider than the one written.

### Example

```ts
import {
  Configuration,
  FirewallApi,
} from '@glueops/apiary-sdk';
import type { FirewallRulesCreateRequest } from '@glueops/apiary-sdk';

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
  const api = new FirewallApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
    // RuleBody
    ruleBody: ...,
  } satisfies FirewallRulesCreateRequest;

  try {
    const data = await api.firewallRulesCreate(body);
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
| **ruleBody** | [RuleBody](RuleBody.md) |  | |

### Return type

[**RuleView**](RuleView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **201** | Created |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## firewallRulesDelete

> DeletedResponseBody firewallRulesDelete(id)

Delete a rule

### Example

```ts
import {
  Configuration,
  FirewallApi,
} from '@glueops/apiary-sdk';
import type { FirewallRulesDeleteRequest } from '@glueops/apiary-sdk';

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
  const api = new FirewallApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
  } satisfies FirewallRulesDeleteRequest;

  try {
    const data = await api.firewallRulesDelete(body);
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

[**DeletedResponseBody**](DeletedResponseBody.md)

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


## firewallRulesList

> ListRulesResponseBody firewallRulesList(id)

List a ruleset\&#39;s rules

Returned in the order they render within this ruleset. Order is the policy — an accept above a drop is a different rule from the reverse.

### Example

```ts
import {
  Configuration,
  FirewallApi,
} from '@glueops/apiary-sdk';
import type { FirewallRulesListRequest } from '@glueops/apiary-sdk';

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
  const api = new FirewallApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
  } satisfies FirewallRulesListRequest;

  try {
    const data = await api.firewallRulesList(body);
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

[**ListRulesResponseBody**](ListRulesResponseBody.md)

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


## firewallRulesUpdate

> RuleView firewallRulesUpdate(id, ruleBody)

Replace a rule

The whole rule, every time. A partial update could apply a new port against the old action, producing a rule nobody intended.

### Example

```ts
import {
  Configuration,
  FirewallApi,
} from '@glueops/apiary-sdk';
import type { FirewallRulesUpdateRequest } from '@glueops/apiary-sdk';

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
  const api = new FirewallApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
    // RuleBody
    ruleBody: ...,
  } satisfies FirewallRulesUpdateRequest;

  try {
    const data = await api.firewallRulesUpdate(body);
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
| **ruleBody** | [RuleBody](RuleBody.md) |  | |

### Return type

[**RuleView**](RuleView.md)

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


## firewallRulesetsCreate

> RulesetView firewallRulesetsCreate(createRulesetRequestBody)

Create a firewall ruleset

### Example

```ts
import {
  Configuration,
  FirewallApi,
} from '@glueops/apiary-sdk';
import type { FirewallRulesetsCreateRequest } from '@glueops/apiary-sdk';

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
  const api = new FirewallApi(config);

  const body = {
    // CreateRulesetRequestBody
    createRulesetRequestBody: ...,
  } satisfies FirewallRulesetsCreateRequest;

  try {
    const data = await api.firewallRulesetsCreate(body);
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
| **createRulesetRequestBody** | [CreateRulesetRequestBody](CreateRulesetRequestBody.md) |  | |

### Return type

[**RulesetView**](RulesetView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **201** | Created |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## firewallRulesetsDelete

> DeletedResponseBody firewallRulesetsDelete(id)

Delete a firewall ruleset

Removes its rules and every binding to it. A binding to a ruleset that no longer exists would render nothing while still listing as though a fleet were covered.

### Example

```ts
import {
  Configuration,
  FirewallApi,
} from '@glueops/apiary-sdk';
import type { FirewallRulesetsDeleteRequest } from '@glueops/apiary-sdk';

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
  const api = new FirewallApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
  } satisfies FirewallRulesetsDeleteRequest;

  try {
    const data = await api.firewallRulesetsDelete(body);
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

[**DeletedResponseBody**](DeletedResponseBody.md)

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


## firewallRulesetsList

> ListRulesetsResponseBody firewallRulesetsList()

List firewall rulesets

Named groups of rules. Fleets are bound to these rather than owning rules directly, so a rule an organization wants everywhere is written once instead of copied per fleet.

### Example

```ts
import {
  Configuration,
  FirewallApi,
} from '@glueops/apiary-sdk';
import type { FirewallRulesetsListRequest } from '@glueops/apiary-sdk';

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
  const api = new FirewallApi(config);

  try {
    const data = await api.firewallRulesetsList();
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

[**ListRulesetsResponseBody**](ListRulesetsResponseBody.md)

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


## firewallRulesetsUpdate

> RulesetView firewallRulesetsUpdate(id, updateRulesetRequestBody)

Rename a firewall ruleset

### Example

```ts
import {
  Configuration,
  FirewallApi,
} from '@glueops/apiary-sdk';
import type { FirewallRulesetsUpdateRequest } from '@glueops/apiary-sdk';

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
  const api = new FirewallApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
    // UpdateRulesetRequestBody
    updateRulesetRequestBody: ...,
  } satisfies FirewallRulesetsUpdateRequest;

  try {
    const data = await api.firewallRulesetsUpdate(body);
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
| **updateRulesetRequestBody** | [UpdateRulesetRequestBody](UpdateRulesetRequestBody.md) |  | |

### Return type

[**RulesetView**](RulesetView.md)

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


## firewallSnapshotBody

> SnapshotBodyResponseBody firewallSnapshotBody(hiveId, revision)

Read one published revision\&#39;s ruleset

The exact text a host loads, so it can be compared against &#x60;nft list ruleset&#x60; on the machine. That comparison is the only way to answer \&#39;is this host running what we think\&#39; from the host itself.

### Example

```ts
import {
  Configuration,
  FirewallApi,
} from '@glueops/apiary-sdk';
import type { FirewallSnapshotBodyRequest } from '@glueops/apiary-sdk';

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
  const api = new FirewallApi(config);

  const body = {
    // string
    hiveId: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
    // number
    revision: 789,
  } satisfies FirewallSnapshotBodyRequest;

  try {
    const data = await api.firewallSnapshotBody(body);
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
| **hiveId** | `string` |  | [Defaults to `undefined`] |
| **revision** | `number` |  | [Defaults to `undefined`] |

### Return type

[**SnapshotBodyResponseBody**](SnapshotBodyResponseBody.md)

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


## firewallSnapshotsList

> SnapshotsResponseBody firewallSnapshotsList(hiveId)

List this fleet\&#39;s published revisions

Newest first. This is what a backout is chosen from, so it carries the reason each revision was published and which revision a backout reproduced.

### Example

```ts
import {
  Configuration,
  FirewallApi,
} from '@glueops/apiary-sdk';
import type { FirewallSnapshotsListRequest } from '@glueops/apiary-sdk';

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
  const api = new FirewallApi(config);

  const body = {
    // string
    hiveId: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
  } satisfies FirewallSnapshotsListRequest;

  try {
    const data = await api.firewallSnapshotsList(body);
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
| **hiveId** | `string` |  | [Defaults to `undefined`] |

### Return type

[**SnapshotsResponseBody**](SnapshotsResponseBody.md)

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

