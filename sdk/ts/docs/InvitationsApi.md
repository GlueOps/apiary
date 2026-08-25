# InvitationsApi

All URIs are relative to *http://127.0.0.1:8282*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**invitationsAccept**](InvitationsApi.md#invitationsaccept) | **POST** /api/v1/invitations/accept | Accept an invitation |
| [**invitationsPreview**](InvitationsApi.md#invitationspreview) | **GET** /api/v1/invitations/preview | Describe an invitation |
| [**orgInvitationsCreate**](InvitationsApi.md#orginvitationscreate) | **POST** /api/v1/invitations | Invite someone into this organization |
| [**orgInvitationsList**](InvitationsApi.md#orginvitationslist) | **GET** /api/v1/invitations | List this organization\&#39;s invitations |
| [**orgInvitationsRevoke**](InvitationsApi.md#orginvitationsrevoke) | **DELETE** /api/v1/invitations/{id} | Withdraw an invitation |



## invitationsAccept

> AcceptResponseBody invitationsAccept(acceptRequestBody)

Accept an invitation

Issues NO session. A new invitee chooses a password here and then signs in through /auth/login; an existing account gains the membership and signs in as it already could. Keeping session minting on the login path alone means lockout, MFA and the email-confirmation gate cannot be skipped by arriving through this door.

### Example

```ts
import {
  Configuration,
  InvitationsApi,
} from '@glueops/apiary-sdk';
import type { InvitationsAcceptRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const api = new InvitationsApi();

  const body = {
    // AcceptRequestBody
    acceptRequestBody: ...,
  } satisfies InvitationsAcceptRequest;

  try {
    const data = await api.invitationsAccept(body);
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
| **acceptRequestBody** | [AcceptRequestBody](AcceptRequestBody.md) |  | |

### Return type

[**AcceptResponseBody**](AcceptResponseBody.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## invitationsPreview

> PreviewResponseBody invitationsPreview(token)

Describe an invitation

Lets the accept page render before the invitee commits to anything: which organization, which address, and whether they need to choose a password or sign in. Holding the token is the authorization — it was mailed to the address it names.

### Example

```ts
import {
  Configuration,
  InvitationsApi,
} from '@glueops/apiary-sdk';
import type { InvitationsPreviewRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const api = new InvitationsApi();

  const body = {
    // string
    token: token_example,
  } satisfies InvitationsPreviewRequest;

  try {
    const data = await api.invitationsPreview(body);
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
| **token** | `string` |  | [Defaults to `undefined`] |

### Return type

[**PreviewResponseBody**](PreviewResponseBody.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## orgInvitationsCreate

> InvitationView orgInvitationsCreate(createRequestBody1)

Invite someone into this organization

Requires admin. Impersonated sessions are refused: an operator acting as a user must not be able to invite an account they control. The organization comes from the session, never from the request.

### Example

```ts
import {
  Configuration,
  InvitationsApi,
} from '@glueops/apiary-sdk';
import type { OrgInvitationsCreateRequest } from '@glueops/apiary-sdk';

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
  const api = new InvitationsApi(config);

  const body = {
    // CreateRequestBody1
    createRequestBody1: ...,
  } satisfies OrgInvitationsCreateRequest;

  try {
    const data = await api.orgInvitationsCreate(body);
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
| **createRequestBody1** | [CreateRequestBody1](CreateRequestBody1.md) |  | |

### Return type

[**InvitationView**](InvitationView.md)

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


## orgInvitationsList

> ListResponseBody1 orgInvitationsList()

List this organization\&#39;s invitations

### Example

```ts
import {
  Configuration,
  InvitationsApi,
} from '@glueops/apiary-sdk';
import type { OrgInvitationsListRequest } from '@glueops/apiary-sdk';

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
  const api = new InvitationsApi(config);

  try {
    const data = await api.orgInvitationsList();
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

[**ListResponseBody1**](ListResponseBody1.md)

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


## orgInvitationsRevoke

> RevokeResponseBody orgInvitationsRevoke(id)

Withdraw an invitation

The organization is taken from the session and used as a query predicate, so an id belonging to another tenant answers 404 rather than revoking anything.

### Example

```ts
import {
  Configuration,
  InvitationsApi,
} from '@glueops/apiary-sdk';
import type { OrgInvitationsRevokeRequest } from '@glueops/apiary-sdk';

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
  const api = new InvitationsApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
  } satisfies OrgInvitationsRevokeRequest;

  try {
    const data = await api.orgInvitationsRevoke(body);
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

[**RevokeResponseBody**](RevokeResponseBody.md)

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

