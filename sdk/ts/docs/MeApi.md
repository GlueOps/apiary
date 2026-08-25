# MeApi

All URIs are relative to *http://127.0.0.1:8080*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**meDeleteAccount**](MeApi.md#medeleteaccount) | **DELETE** /api/v1/me | Delete your account |
| [**meSoleOwnerships**](MeApi.md#mesoleownerships) | **GET** /api/v1/me/sole-ownerships | Organizations this account solely owns |



## meDeleteAccount

> DeleteAccountResponseBody meDeleteAccount(deleteAccountRequestBody)

Delete your account

Refused while this account is the only owner of an organization — see /me/sole-ownerships. Every session is revoked and every authentication factor removed. The record itself is retained: audit events, fleet tasks and invitations name the actor, and an audit trail that vanishes with the account is one the audited person can erase.

### Example

```ts
import {
  Configuration,
  MeApi,
} from '@glueops/apiary-sdk';
import type { MeDeleteAccountRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new MeApi(config);

  const body = {
    // DeleteAccountRequestBody
    deleteAccountRequestBody: ...,
  } satisfies MeDeleteAccountRequest;

  try {
    const data = await api.meDeleteAccount(body);
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
| **deleteAccountRequestBody** | [DeleteAccountRequestBody](DeleteAccountRequestBody.md) |  | |

### Return type

[**DeleteAccountResponseBody**](DeleteAccountResponseBody.md)

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


## meSoleOwnerships

> SoleOwnershipsResponseBody meSoleOwnerships()

Organizations this account solely owns

Deleting an account is refused while it is the only owner of an organization, because that would leave a tenant nobody can administer. This names them, so the answer is actionable rather than just a refusal.

### Example

```ts
import {
  Configuration,
  MeApi,
} from '@glueops/apiary-sdk';
import type { MeSoleOwnershipsRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new MeApi(config);

  try {
    const data = await api.meSoleOwnerships();
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

[**SoleOwnershipsResponseBody**](SoleOwnershipsResponseBody.md)

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

