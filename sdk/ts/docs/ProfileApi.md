# ProfileApi

All URIs are relative to *http://127.0.0.1:8080*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**meChangePassword**](ProfileApi.md#mechangepassword) | **PUT** /api/v1/me/password | Change password |
| [**meGet**](ProfileApi.md#meget) | **GET** /api/v1/me | Get current user profile |
| [**meOrgsList**](ProfileApi.md#meorgslist) | **GET** /api/v1/me/orgs | List my organizations |
| [**meUpdate**](ProfileApi.md#meupdate) | **PATCH** /api/v1/me | Update current user profile |



## meChangePassword

> ChangePasswordResponseBody meChangePassword(changePasswordRequestBody)

Change password

Change the authenticated user\&#39;s password. Requires the current password.

### Example

```ts
import {
  Configuration,
  ProfileApi,
} from '@glueops/apiary-sdk';
import type { MeChangePasswordRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new ProfileApi(config);

  const body = {
    // ChangePasswordRequestBody
    changePasswordRequestBody: ...,
  } satisfies MeChangePasswordRequest;

  try {
    const data = await api.meChangePassword(body);
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
| **changePasswordRequestBody** | [ChangePasswordRequestBody](ChangePasswordRequestBody.md) |  | |

### Return type

[**ChangePasswordResponseBody**](ChangePasswordResponseBody.md)

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


## meGet

> GetMeResponseBody meGet()

Get current user profile

Returns the authenticated user\&#39;s profile, MFA status, and organization.

### Example

```ts
import {
  Configuration,
  ProfileApi,
} from '@glueops/apiary-sdk';
import type { MeGetRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new ProfileApi(config);

  try {
    const data = await api.meGet();
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

[**GetMeResponseBody**](GetMeResponseBody.md)

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


## meOrgsList

> GetMyOrganizationsResponseBody meOrgsList()

List my organizations

Returns the organizations the authenticated user belongs to.

### Example

```ts
import {
  Configuration,
  ProfileApi,
} from '@glueops/apiary-sdk';
import type { MeOrgsListRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new ProfileApi(config);

  try {
    const data = await api.meOrgsList();
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

[**GetMyOrganizationsResponseBody**](GetMyOrganizationsResponseBody.md)

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


## meUpdate

> UpdateMeResponseBody meUpdate(updateMeRequestBody)

Update current user profile

Update the authenticated user\&#39;s first name, last name, or email.

### Example

```ts
import {
  Configuration,
  ProfileApi,
} from '@glueops/apiary-sdk';
import type { MeUpdateRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new ProfileApi(config);

  const body = {
    // UpdateMeRequestBody
    updateMeRequestBody: ...,
  } satisfies MeUpdateRequest;

  try {
    const data = await api.meUpdate(body);
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
| **updateMeRequestBody** | [UpdateMeRequestBody](UpdateMeRequestBody.md) |  | |

### Return type

[**UpdateMeResponseBody**](UpdateMeResponseBody.md)

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

