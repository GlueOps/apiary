# APIKeysApi

All URIs are relative to *http://127.0.0.1:8282*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**apiKeysCreate**](APIKeysApi.md#apikeyscreate) | **POST** /api/v1/api-keys | Mint an API key |
| [**apiKeysList**](APIKeysApi.md#apikeyslist) | **GET** /api/v1/api-keys | List API keys |
| [**apiKeysRevoke**](APIKeysApi.md#apikeysrevoke) | **DELETE** /api/v1/api-keys/{id} | Revoke an API key |



## apiKeysCreate

> MintAPIKeyResponseBody apiKeysCreate(mintAPIKeyRequestBody)

Mint an API key

Returns the plaintext credential ONCE. Only its hash is stored, so a key that is not captured from this response is replaced rather than recovered — no endpoint reissues or reveals one.  A user-scoped key needs nothing beyond membership: it acts as you and can do nothing you cannot. An org-scoped key is standing access to the whole tenant, so it requires admin and refuses impersonated sessions — an operator acting as a user must not be able to leave themselves a way back in. Agent-scoped keys are refused outright; they are issued only by fleet enrolment.

### Example

```ts
import {
  Configuration,
  APIKeysApi,
} from '@glueops/apiary-sdk';
import type { ApiKeysCreateRequest } from '@glueops/apiary-sdk';

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
  const api = new APIKeysApi(config);

  const body = {
    // MintAPIKeyRequestBody
    mintAPIKeyRequestBody: ...,
  } satisfies ApiKeysCreateRequest;

  try {
    const data = await api.apiKeysCreate(body);
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
| **mintAPIKeyRequestBody** | [MintAPIKeyRequestBody](MintAPIKeyRequestBody.md) |  | |

### Return type

[**MintAPIKeyResponseBody**](MintAPIKeyResponseBody.md)

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


## apiKeysList

> ListAPIKeysResponseBody apiKeysList()

List API keys

Your own user-scoped keys, plus the organization\&#39;s org-scoped keys. Never returns key or secret material: only hashes are stored, and a key that is lost is replaced rather than recovered. Agent keys are not listed — they belong to the fleet surface.

### Example

```ts
import {
  Configuration,
  APIKeysApi,
} from '@glueops/apiary-sdk';
import type { ApiKeysListRequest } from '@glueops/apiary-sdk';

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
  const api = new APIKeysApi(config);

  try {
    const data = await api.apiKeysList();
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

[**ListAPIKeysResponseBody**](ListAPIKeysResponseBody.md)

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


## apiKeysRevoke

> RevokeAPIKeyResponseBody apiKeysRevoke(id)

Revoke an API key

Revocation is permanent and takes effect on the next request the key makes. You may revoke your own user-scoped keys. Revoking one of the organization\&#39;s org-scoped keys requires admin and refuses impersonated sessions, for the same reason minting one does: it changes the tenant\&#39;s standing access, and that has to be attributable to a real admin. Nobody may revoke another person\&#39;s user key.

### Example

```ts
import {
  Configuration,
  APIKeysApi,
} from '@glueops/apiary-sdk';
import type { ApiKeysRevokeRequest } from '@glueops/apiary-sdk';

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
  const api = new APIKeysApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
  } satisfies ApiKeysRevokeRequest;

  try {
    const data = await api.apiKeysRevoke(body);
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

[**RevokeAPIKeyResponseBody**](RevokeAPIKeyResponseBody.md)

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

