# OrganizationsApi

All URIs are relative to *http://127.0.0.1:8080*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**orgDelete**](OrganizationsApi.md#orgdelete) | **DELETE** /api/v1/orgs/current | Delete this organization |



## orgDelete

> DeleteOrgResponseBody orgDelete(deleteOrgRequestBody)

Delete this organization

OWNER only, and impersonated sessions are refused. The organization stops resolving immediately: its tenant database becomes unreachable, every session scoped to it is revoked, and its agents can no longer authenticate. Key material is NOT destroyed by this call — the organization row holds the only key that can decrypt the tenant database, so erasing it would destroy the data irreversibly rather than merely retiring it. Purging is a separate operator step, which is what leaves a way back.

### Example

```ts
import {
  Configuration,
  OrganizationsApi,
} from '@glueops/apiary-sdk';
import type { OrgDeleteRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new OrganizationsApi(config);

  const body = {
    // DeleteOrgRequestBody
    deleteOrgRequestBody: ...,
  } satisfies OrgDeleteRequest;

  try {
    const data = await api.orgDelete(body);
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
| **deleteOrgRequestBody** | [DeleteOrgRequestBody](DeleteOrgRequestBody.md) |  | |

### Return type

[**DeleteOrgResponseBody**](DeleteOrgResponseBody.md)

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

