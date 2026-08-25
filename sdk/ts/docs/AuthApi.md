# AuthApi

All URIs are relative to *http://127.0.0.1:8282*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**authForgotPassword**](AuthApi.md#authforgotpassword) | **POST** /api/v1/auth/forgot-password | Request a password reset link |
| [**authLogin**](AuthApi.md#authlogin) | **POST** /api/v1/auth/login | Authenticate a user |
| [**authLogout**](AuthApi.md#authlogout) | **POST** /api/v1/auth/logout | Logout current session |
| [**authMfaVerify**](AuthApi.md#authmfaverify) | **POST** /api/v1/auth/mfa/verify | Complete MFA challenge |
| [**authPasskeyLoginBegin**](AuthApi.md#authpasskeyloginbegin) | **POST** /api/v1/auth/passkey/login/begin | Begin passkey login |
| [**authPasskeyLoginFinish**](AuthApi.md#authpasskeyloginfinish) | **POST** /api/v1/auth/passkey/login/finish | Complete passkey login |
| [**authPasskeyRegisterBegin**](AuthApi.md#authpasskeyregisterbegin) | **POST** /api/v1/auth/passkey/register/begin | Begin passkey registration |
| [**authPasskeyRegisterFinish**](AuthApi.md#authpasskeyregisterfinish) | **POST** /api/v1/auth/passkey/register/finish | Complete passkey registration |
| [**authPasskeysDelete**](AuthApi.md#authpasskeysdelete) | **DELETE** /api/v1/auth/passkeys/{id} | Remove a passkey |
| [**authPasskeysList**](AuthApi.md#authpasskeyslist) | **GET** /api/v1/auth/passkeys | List registered passkeys |
| [**authRefresh**](AuthApi.md#authrefresh) | **POST** /api/v1/auth/refresh | Refresh auth tokens |
| [**authRegister**](AuthApi.md#authregister) | **POST** /api/v1/auth/register | Create an account and its first organization |
| [**authResendVerification**](AuthApi.md#authresendverification) | **POST** /api/v1/auth/resend-verification | Send a new confirmation link |
| [**authResetPassword**](AuthApi.md#authresetpassword) | **POST** /api/v1/auth/reset-password | Set a new password using a reset token |
| [**authSocialCallback**](AuthApi.md#authsocialcallback) | **GET** /api/v1/auth/{provider}/callback | Handle social login callback |
| [**authSocialStart**](AuthApi.md#authsocialstart) | **POST** /api/v1/auth/{provider}/start | Begin social login |
| [**authTotpConfirm**](AuthApi.md#authtotpconfirm) | **POST** /api/v1/auth/totp/verify | Confirm TOTP setup |
| [**authTotpDisable**](AuthApi.md#authtotpdisable) | **DELETE** /api/v1/auth/totp | Disable TOTP |
| [**authTotpSetup**](AuthApi.md#authtotpsetup) | **POST** /api/v1/auth/totp/setup | Start TOTP setup |
| [**authVerifyEmail**](AuthApi.md#authverifyemail) | **POST** /api/v1/auth/verify-email | Confirm an email address |



## authForgotPassword

> ForgotPasswordResponseBody authForgotPassword(forgotPasswordRequestBody)

Request a password reset link

Sends a password reset link if the address belongs to an active account. The response is identical either way.

### Example

```ts
import {
  Configuration,
  AuthApi,
} from '@glueops/apiary-sdk';
import type { AuthForgotPasswordRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const api = new AuthApi();

  const body = {
    // ForgotPasswordRequestBody
    forgotPasswordRequestBody: ...,
  } satisfies AuthForgotPasswordRequest;

  try {
    const data = await api.authForgotPassword(body);
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
| **forgotPasswordRequestBody** | [ForgotPasswordRequestBody](ForgotPasswordRequestBody.md) |  | |

### Return type

[**ForgotPasswordResponseBody**](ForgotPasswordResponseBody.md)

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


## authLogin

> LoginResponseBody authLogin(loginRequestBody, userAgent)

Authenticate a user

Validates email and password, selects the user\&#39;s organization, and returns JWT access and refresh tokens.

### Example

```ts
import {
  Configuration,
  AuthApi,
} from '@glueops/apiary-sdk';
import type { AuthLoginRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const api = new AuthApi();

  const body = {
    // LoginRequestBody
    loginRequestBody: ...,
    // string (optional)
    userAgent: userAgent_example,
  } satisfies AuthLoginRequest;

  try {
    const data = await api.authLogin(body);
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
| **loginRequestBody** | [LoginRequestBody](LoginRequestBody.md) |  | |
| **userAgent** | `string` |  | [Optional] [Defaults to `undefined`] |

### Return type

[**LoginResponseBody**](LoginResponseBody.md)

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


## authLogout

> LogoutResponseBody authLogout(userAgent)

Logout current session

Revokes the current token session.

### Example

```ts
import {
  Configuration,
  AuthApi,
} from '@glueops/apiary-sdk';
import type { AuthLogoutRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new AuthApi(config);

  const body = {
    // string (optional)
    userAgent: userAgent_example,
  } satisfies AuthLogoutRequest;

  try {
    const data = await api.authLogout(body);
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
| **userAgent** | `string` |  | [Optional] [Defaults to `undefined`] |

### Return type

[**LogoutResponseBody**](LogoutResponseBody.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## authMfaVerify

> MFAVerifyResponseBody authMfaVerify(mFAVerifyRequestBody, userAgent)

Complete MFA challenge

Validates an MFA pending token and TOTP code, then returns full access/refresh tokens.

### Example

```ts
import {
  Configuration,
  AuthApi,
} from '@glueops/apiary-sdk';
import type { AuthMfaVerifyRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const api = new AuthApi();

  const body = {
    // MFAVerifyRequestBody
    mFAVerifyRequestBody: ...,
    // string (optional)
    userAgent: userAgent_example,
  } satisfies AuthMfaVerifyRequest;

  try {
    const data = await api.authMfaVerify(body);
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
| **mFAVerifyRequestBody** | [MFAVerifyRequestBody](MFAVerifyRequestBody.md) |  | |
| **userAgent** | `string` |  | [Optional] [Defaults to `undefined`] |

### Return type

[**MFAVerifyResponseBody**](MFAVerifyResponseBody.md)

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


## authPasskeyLoginBegin

> PasskeyBeginLoginResponseBody authPasskeyLoginBegin()

Begin passkey login

Starts the WebAuthn discoverable login flow. Returns options and a signed challenge.

### Example

```ts
import {
  Configuration,
  AuthApi,
} from '@glueops/apiary-sdk';
import type { AuthPasskeyLoginBeginRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const api = new AuthApi();

  try {
    const data = await api.authPasskeyLoginBegin();
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

[**PasskeyBeginLoginResponseBody**](PasskeyBeginLoginResponseBody.md)

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


## authPasskeyLoginFinish

> PasskeyFinishLoginResponseBody authPasskeyLoginFinish(passkeyFinishLoginRequestBody, userAgent)

Complete passkey login

Validates the WebAuthn assertion and returns access/refresh tokens.

### Example

```ts
import {
  Configuration,
  AuthApi,
} from '@glueops/apiary-sdk';
import type { AuthPasskeyLoginFinishRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const api = new AuthApi();

  const body = {
    // PasskeyFinishLoginRequestBody
    passkeyFinishLoginRequestBody: ...,
    // string (optional)
    userAgent: userAgent_example,
  } satisfies AuthPasskeyLoginFinishRequest;

  try {
    const data = await api.authPasskeyLoginFinish(body);
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
| **passkeyFinishLoginRequestBody** | [PasskeyFinishLoginRequestBody](PasskeyFinishLoginRequestBody.md) |  | |
| **userAgent** | `string` |  | [Optional] [Defaults to `undefined`] |

### Return type

[**PasskeyFinishLoginResponseBody**](PasskeyFinishLoginResponseBody.md)

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


## authPasskeyRegisterBegin

> PasskeyBeginRegisterResponseBody authPasskeyRegisterBegin()

Begin passkey registration

REACHABLE BEFORE POLICY IS SATISFIED. A session that has authenticated but not yet met a policy such as require_mfa can call this — that is the point, since the endpoint exists to let someone comply. Authentication still applies in full; only authorization is relaxed, and only for accounts holding NO verified factor. An account that already has one must present it first, or a stolen password could add its own authenticator. Starts the WebAuthn registration flow. Returns creation options and a signed challenge.

### Example

```ts
import {
  Configuration,
  AuthApi,
} from '@glueops/apiary-sdk';
import type { AuthPasskeyRegisterBeginRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new AuthApi(config);

  try {
    const data = await api.authPasskeyRegisterBegin();
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

[**PasskeyBeginRegisterResponseBody**](PasskeyBeginRegisterResponseBody.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## authPasskeyRegisterFinish

> PasskeyFinishRegisterResponseBody authPasskeyRegisterFinish(passkeyFinishRegisterRequestBody)

Complete passkey registration

REACHABLE BEFORE POLICY IS SATISFIED. A session that has authenticated but not yet met a policy such as require_mfa can call this — that is the point, since the endpoint exists to let someone comply. Authentication still applies in full; only authorization is relaxed, and only for accounts holding NO verified factor. An account that already has one must present it first, or a stolen password could add its own authenticator. Validates the WebAuthn attestation and stores the credential.

### Example

```ts
import {
  Configuration,
  AuthApi,
} from '@glueops/apiary-sdk';
import type { AuthPasskeyRegisterFinishRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new AuthApi(config);

  const body = {
    // PasskeyFinishRegisterRequestBody
    passkeyFinishRegisterRequestBody: ...,
  } satisfies AuthPasskeyRegisterFinishRequest;

  try {
    const data = await api.authPasskeyRegisterFinish(body);
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
| **passkeyFinishRegisterRequestBody** | [PasskeyFinishRegisterRequestBody](PasskeyFinishRegisterRequestBody.md) |  | |

### Return type

[**PasskeyFinishRegisterResponseBody**](PasskeyFinishRegisterResponseBody.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## authPasskeysDelete

> PasskeyDeleteResponseBody authPasskeysDelete(id)

Remove a passkey

Deletes a registered passkey. If no MFA methods remain, MFA is disabled.

### Example

```ts
import {
  Configuration,
  AuthApi,
} from '@glueops/apiary-sdk';
import type { AuthPasskeysDeleteRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new AuthApi(config);

  const body = {
    // string
    id: id_example,
  } satisfies AuthPasskeysDeleteRequest;

  try {
    const data = await api.authPasskeysDelete(body);
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

[**PasskeyDeleteResponseBody**](PasskeyDeleteResponseBody.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## authPasskeysList

> PasskeyListResponseBody authPasskeysList()

List registered passkeys

Returns all passkeys registered for the authenticated user.

### Example

```ts
import {
  Configuration,
  AuthApi,
} from '@glueops/apiary-sdk';
import type { AuthPasskeysListRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new AuthApi(config);

  try {
    const data = await api.authPasskeysList();
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

[**PasskeyListResponseBody**](PasskeyListResponseBody.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## authRefresh

> RefreshResponseBody authRefresh(refreshRequestBody, userAgent)

Refresh auth tokens

Validates a refresh token and returns a new access and refresh token pair.

### Example

```ts
import {
  Configuration,
  AuthApi,
} from '@glueops/apiary-sdk';
import type { AuthRefreshRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const api = new AuthApi();

  const body = {
    // RefreshRequestBody
    refreshRequestBody: ...,
    // string (optional)
    userAgent: userAgent_example,
  } satisfies AuthRefreshRequest;

  try {
    const data = await api.authRefresh(body);
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
| **refreshRequestBody** | [RefreshRequestBody](RefreshRequestBody.md) |  | |
| **userAgent** | `string` |  | [Optional] [Defaults to `undefined`] |

### Return type

[**RefreshResponseBody**](RefreshResponseBody.md)

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


## authRegister

> RegisterResponseBody authRegister(registerRequestBody, xRealIp, userAgent)

Create an account and its first organization

Creates the user, an organization, and the owner membership binding them — all in one transaction, because a user with no organization cannot sign in at all. Returns NO tokens: the address must be confirmed first, and confirming issues no session either — sign in through /auth/login afterwards. The organization\&#39;s tenant database is not provisioned until the address is confirmed, so an unconfirmed signup costs a row and nothing else.

### Example

```ts
import {
  Configuration,
  AuthApi,
} from '@glueops/apiary-sdk';
import type { AuthRegisterRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const api = new AuthApi();

  const body = {
    // RegisterRequestBody
    registerRequestBody: ...,
    // string (optional)
    xRealIp: xRealIp_example,
    // string (optional)
    userAgent: userAgent_example,
  } satisfies AuthRegisterRequest;

  try {
    const data = await api.authRegister(body);
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
| **registerRequestBody** | [RegisterRequestBody](RegisterRequestBody.md) |  | |
| **xRealIp** | `string` |  | [Optional] [Defaults to `undefined`] |
| **userAgent** | `string` |  | [Optional] [Defaults to `undefined`] |

### Return type

[**RegisterResponseBody**](RegisterResponseBody.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **201** | Created |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## authResendVerification

> ResendVerificationResponseBody authResendVerification(resendVerificationRequestBody)

Send a new confirmation link

Always answers 202, whatever the address. An unauthenticated endpoint that answered differently for a registered address than an unregistered one would be a membership oracle. Sending is throttled per account.

### Example

```ts
import {
  Configuration,
  AuthApi,
} from '@glueops/apiary-sdk';
import type { AuthResendVerificationRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const api = new AuthApi();

  const body = {
    // ResendVerificationRequestBody
    resendVerificationRequestBody: ...,
  } satisfies AuthResendVerificationRequest;

  try {
    const data = await api.authResendVerification(body);
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
| **resendVerificationRequestBody** | [ResendVerificationRequestBody](ResendVerificationRequestBody.md) |  | |

### Return type

[**ResendVerificationResponseBody**](ResendVerificationResponseBody.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **202** | Accepted |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## authResetPassword

> ResetPasswordResponseBody authResetPassword(resetPasswordRequestBody)

Set a new password using a reset token

Consumes a single-use reset token, sets the new password, and revokes every existing session for the account.

### Example

```ts
import {
  Configuration,
  AuthApi,
} from '@glueops/apiary-sdk';
import type { AuthResetPasswordRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const api = new AuthApi();

  const body = {
    // ResetPasswordRequestBody
    resetPasswordRequestBody: ...,
  } satisfies AuthResetPasswordRequest;

  try {
    const data = await api.authResetPassword(body);
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
| **resetPasswordRequestBody** | [ResetPasswordRequestBody](ResetPasswordRequestBody.md) |  | |

### Return type

[**ResetPasswordResponseBody**](ResetPasswordResponseBody.md)

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


## authSocialCallback

> string authSocialCallback(provider, code, state, userAgent, cookie)

Handle social login callback

Exchanges the provider code, links or creates the user account, auto-joins by email domain, and returns tokens.

### Example

```ts
import {
  Configuration,
  AuthApi,
} from '@glueops/apiary-sdk';
import type { AuthSocialCallbackRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const api = new AuthApi();

  const body = {
    // string
    provider: provider_example,
    // string
    code: code_example,
    // string (optional)
    state: state_example,
    // string (optional)
    userAgent: userAgent_example,
    // string (optional)
    cookie: cookie_example,
  } satisfies AuthSocialCallbackRequest;

  try {
    const data = await api.authSocialCallback(body);
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
| **provider** | `string` |  | [Defaults to `undefined`] |
| **code** | `string` |  | [Defaults to `undefined`] |
| **state** | `string` |  | [Optional] [Defaults to `undefined`] |
| **userAgent** | `string` |  | [Optional] [Defaults to `undefined`] |
| **cookie** | `string` |  | [Optional] [Defaults to `undefined`] |

### Return type

**string**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  * Content-Type -  <br>  * Set-Cookie -  <br>  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## authSocialStart

> StartAuthResponseBody authSocialStart(provider, mode, origin)

Begin social login

Returns the provider authorization URL for the frontend to redirect to.

### Example

```ts
import {
  Configuration,
  AuthApi,
} from '@glueops/apiary-sdk';
import type { AuthSocialStartRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const api = new AuthApi();

  const body = {
    // string
    provider: provider_example,
    // string (optional)
    mode: mode_example,
    // string (optional)
    origin: origin_example,
  } satisfies AuthSocialStartRequest;

  try {
    const data = await api.authSocialStart(body);
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
| **provider** | `string` |  | [Defaults to `undefined`] |
| **mode** | `string` |  | [Optional] [Defaults to `undefined`] |
| **origin** | `string` |  | [Optional] [Defaults to `undefined`] |

### Return type

[**StartAuthResponseBody**](StartAuthResponseBody.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  * Set-Cookie -  <br>  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## authTotpConfirm

> TOTPVerifyResponseBody authTotpConfirm(tOTPVerifyRequestBody)

Confirm TOTP setup

REACHABLE BEFORE POLICY IS SATISFIED. A session that has authenticated but not yet met a policy such as require_mfa can call this — that is the point, since the endpoint exists to let someone comply. Authentication still applies in full; only authorization is relaxed, and only for accounts holding NO verified factor. An account that already has one must present it first, or a stolen password could add its own authenticator. Confirms TOTP setup by validating a code. This activates MFA for the user.

### Example

```ts
import {
  Configuration,
  AuthApi,
} from '@glueops/apiary-sdk';
import type { AuthTotpConfirmRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new AuthApi(config);

  const body = {
    // TOTPVerifyRequestBody
    tOTPVerifyRequestBody: ...,
  } satisfies AuthTotpConfirmRequest;

  try {
    const data = await api.authTotpConfirm(body);
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
| **tOTPVerifyRequestBody** | [TOTPVerifyRequestBody](TOTPVerifyRequestBody.md) |  | |

### Return type

[**TOTPVerifyResponseBody**](TOTPVerifyResponseBody.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## authTotpDisable

> TOTPDisableResponseBody authTotpDisable(tOTPDisableRequestBody)

Disable TOTP

Disables TOTP MFA. Requires the user\&#39;s password or a valid TOTP code for verification.

### Example

```ts
import {
  Configuration,
  AuthApi,
} from '@glueops/apiary-sdk';
import type { AuthTotpDisableRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new AuthApi(config);

  const body = {
    // TOTPDisableRequestBody
    tOTPDisableRequestBody: ...,
  } satisfies AuthTotpDisableRequest;

  try {
    const data = await api.authTotpDisable(body);
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
| **tOTPDisableRequestBody** | [TOTPDisableRequestBody](TOTPDisableRequestBody.md) |  | |

### Return type

[**TOTPDisableResponseBody**](TOTPDisableResponseBody.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## authTotpSetup

> TOTPSetupResponseBody authTotpSetup()

Start TOTP setup

REACHABLE BEFORE POLICY IS SATISFIED. A session that has authenticated but not yet met a policy such as require_mfa can call this — that is the point, since the endpoint exists to let someone comply. Authentication still applies in full; only authorization is relaxed, and only for accounts holding NO verified factor. An account that already has one must present it first, or a stolen password could add its own authenticator. Generates a new TOTP secret and returns the otpauth URI for QR code rendering.

### Example

```ts
import {
  Configuration,
  AuthApi,
} from '@glueops/apiary-sdk';
import type { AuthTotpSetupRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new AuthApi(config);

  try {
    const data = await api.authTotpSetup();
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

[**TOTPSetupResponseBody**](TOTPSetupResponseBody.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## authVerifyEmail

> VerifyEmailResponseBody authVerifyEmail(verifyEmailRequestBody)

Confirm an email address

Redeems the token from a confirmation link. This is what releases provisioning of the organization\&#39;s tenant database — registration deliberately creates nothing but rows until an address is confirmed. No session is issued: sign in through /auth/login afterwards.

### Example

```ts
import {
  Configuration,
  AuthApi,
} from '@glueops/apiary-sdk';
import type { AuthVerifyEmailRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const api = new AuthApi();

  const body = {
    // VerifyEmailRequestBody
    verifyEmailRequestBody: ...,
  } satisfies AuthVerifyEmailRequest;

  try {
    const data = await api.authVerifyEmail(body);
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
| **verifyEmailRequestBody** | [VerifyEmailRequestBody](VerifyEmailRequestBody.md) |  | |

### Return type

[**VerifyEmailResponseBody**](VerifyEmailResponseBody.md)

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

