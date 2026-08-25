# AdminApi

All URIs are relative to *http://127.0.0.1:8282*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**adminImpersonateEnd**](AdminApi.md#adminimpersonateend) | **POST** /api/v1/admin/impersonate/end | End the current impersonated session |
| [**adminImpersonateStart**](AdminApi.md#adminimpersonatestart) | **POST** /api/v1/admin/impersonate | Begin impersonating a user |



## adminImpersonateEnd

> ImpersonateEndResponseBody adminImpersonateEnd(impersonateEndRequestBody, xRealIp, userAgent)

End the current impersonated session

Called BY the impersonated session, which is the only caller that holds the JTI being revoked. It takes no parameters for that reason: everything it needs — which session, which target, which actor — comes from the token, so it cannot be pointed at somebody else\&#39;s session.

### Example

```ts
import {
  Configuration,
  AdminApi,
} from '@glueops/apiary-sdk';
import type { AdminImpersonateEndRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new AdminApi(config);

  const body = {
    // ImpersonateEndRequestBody
    impersonateEndRequestBody: ...,
    // string (optional)
    xRealIp: xRealIp_example,
    // string (optional)
    userAgent: userAgent_example,
  } satisfies AdminImpersonateEndRequest;

  try {
    const data = await api.adminImpersonateEnd(body);
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
| **impersonateEndRequestBody** | [ImpersonateEndRequestBody](ImpersonateEndRequestBody.md) |  | |
| **xRealIp** | `string` |  | [Optional] [Defaults to `undefined`] |
| **userAgent** | `string` |  | [Optional] [Defaults to `undefined`] |

### Return type

[**ImpersonateEndResponseBody**](ImpersonateEndResponseBody.md)

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


## adminImpersonateStart

> ImpersonateStartResponseBody adminImpersonateStart(impersonateStartRequestBody, xRealIp, userAgent)

Begin impersonating a user

Platform staff only. The target must have enabled support access or be covered by an active grant — the service refuses otherwise, and this endpoint cannot override that. Returns an access token whose subject is the target user and whose actor is the caller; every action taken with it is attributed to both.

### Example

```ts
import {
  Configuration,
  AdminApi,
} from '@glueops/apiary-sdk';
import type { AdminImpersonateStartRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new AdminApi(config);

  const body = {
    // ImpersonateStartRequestBody
    impersonateStartRequestBody: ...,
    // string (optional)
    xRealIp: xRealIp_example,
    // string (optional)
    userAgent: userAgent_example,
  } satisfies AdminImpersonateStartRequest;

  try {
    const data = await api.adminImpersonateStart(body);
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
| **impersonateStartRequestBody** | [ImpersonateStartRequestBody](ImpersonateStartRequestBody.md) |  | |
| **xRealIp** | `string` |  | [Optional] [Defaults to `undefined`] |
| **userAgent** | `string` |  | [Optional] [Defaults to `undefined`] |

### Return type

[**ImpersonateStartResponseBody**](ImpersonateStartResponseBody.md)

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

