# \AuthAPI

All URIs are relative to *http://127.0.0.1:8080*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AuthForgotPassword**](AuthAPI.md#AuthForgotPassword) | **Post** /api/v1/auth/forgot-password | Request a password reset link
[**AuthLogin**](AuthAPI.md#AuthLogin) | **Post** /api/v1/auth/login | Authenticate a user
[**AuthLogout**](AuthAPI.md#AuthLogout) | **Post** /api/v1/auth/logout | Logout current session
[**AuthMfaVerify**](AuthAPI.md#AuthMfaVerify) | **Post** /api/v1/auth/mfa/verify | Complete MFA challenge
[**AuthPasskeyLoginBegin**](AuthAPI.md#AuthPasskeyLoginBegin) | **Post** /api/v1/auth/passkey/login/begin | Begin passkey login
[**AuthPasskeyLoginFinish**](AuthAPI.md#AuthPasskeyLoginFinish) | **Post** /api/v1/auth/passkey/login/finish | Complete passkey login
[**AuthPasskeyRegisterBegin**](AuthAPI.md#AuthPasskeyRegisterBegin) | **Post** /api/v1/auth/passkey/register/begin | Begin passkey registration
[**AuthPasskeyRegisterFinish**](AuthAPI.md#AuthPasskeyRegisterFinish) | **Post** /api/v1/auth/passkey/register/finish | Complete passkey registration
[**AuthPasskeysDelete**](AuthAPI.md#AuthPasskeysDelete) | **Delete** /api/v1/auth/passkeys/{id} | Remove a passkey
[**AuthPasskeysList**](AuthAPI.md#AuthPasskeysList) | **Get** /api/v1/auth/passkeys | List registered passkeys
[**AuthRefresh**](AuthAPI.md#AuthRefresh) | **Post** /api/v1/auth/refresh | Refresh auth tokens
[**AuthRegister**](AuthAPI.md#AuthRegister) | **Post** /api/v1/auth/register | Create an account and its first organization
[**AuthResendVerification**](AuthAPI.md#AuthResendVerification) | **Post** /api/v1/auth/resend-verification | Send a new confirmation link
[**AuthResetPassword**](AuthAPI.md#AuthResetPassword) | **Post** /api/v1/auth/reset-password | Set a new password using a reset token
[**AuthSocialCallback**](AuthAPI.md#AuthSocialCallback) | **Get** /api/v1/auth/{provider}/callback | Handle social login callback
[**AuthSocialStart**](AuthAPI.md#AuthSocialStart) | **Post** /api/v1/auth/{provider}/start | Begin social login
[**AuthTotpConfirm**](AuthAPI.md#AuthTotpConfirm) | **Post** /api/v1/auth/totp/verify | Confirm TOTP setup
[**AuthTotpDisable**](AuthAPI.md#AuthTotpDisable) | **Delete** /api/v1/auth/totp | Disable TOTP
[**AuthTotpSetup**](AuthAPI.md#AuthTotpSetup) | **Post** /api/v1/auth/totp/setup | Start TOTP setup
[**AuthVerifyEmail**](AuthAPI.md#AuthVerifyEmail) | **Post** /api/v1/auth/verify-email | Confirm an email address



## AuthForgotPassword

> ForgotPasswordResponseBody AuthForgotPassword(ctx).ForgotPasswordRequestBody(forgotPasswordRequestBody).Execute()

Request a password reset link



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/glueops/apiary/sdk/go/apiary"
)

func main() {
	forgotPasswordRequestBody := *openapiclient.NewForgotPasswordRequestBody("Email_example") // ForgotPasswordRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.AuthForgotPassword(context.Background()).ForgotPasswordRequestBody(forgotPasswordRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.AuthForgotPassword``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthForgotPassword`: ForgotPasswordResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.AuthForgotPassword`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiAuthForgotPasswordRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **forgotPasswordRequestBody** | [**ForgotPasswordRequestBody**](ForgotPasswordRequestBody.md) |  | 

### Return type

[**ForgotPasswordResponseBody**](ForgotPasswordResponseBody.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthLogin

> LoginResponseBody AuthLogin(ctx).LoginRequestBody(loginRequestBody).UserAgent(userAgent).Execute()

Authenticate a user



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/glueops/apiary/sdk/go/apiary"
)

func main() {
	loginRequestBody := *openapiclient.NewLoginRequestBody("Email_example", "Password_example") // LoginRequestBody | 
	userAgent := "userAgent_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.AuthLogin(context.Background()).LoginRequestBody(loginRequestBody).UserAgent(userAgent).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.AuthLogin``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthLogin`: LoginResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.AuthLogin`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiAuthLoginRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **loginRequestBody** | [**LoginRequestBody**](LoginRequestBody.md) |  | 
 **userAgent** | **string** |  | 

### Return type

[**LoginResponseBody**](LoginResponseBody.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthLogout

> LogoutResponseBody AuthLogout(ctx).UserAgent(userAgent).Execute()

Logout current session



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/glueops/apiary/sdk/go/apiary"
)

func main() {
	userAgent := "userAgent_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.AuthLogout(context.Background()).UserAgent(userAgent).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.AuthLogout``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthLogout`: LogoutResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.AuthLogout`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiAuthLogoutRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **userAgent** | **string** |  | 

### Return type

[**LogoutResponseBody**](LogoutResponseBody.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthMfaVerify

> MFAVerifyResponseBody AuthMfaVerify(ctx).MFAVerifyRequestBody(mFAVerifyRequestBody).UserAgent(userAgent).Execute()

Complete MFA challenge



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/glueops/apiary/sdk/go/apiary"
)

func main() {
	mFAVerifyRequestBody := *openapiclient.NewMFAVerifyRequestBody("Code_example", "MfaToken_example") // MFAVerifyRequestBody | 
	userAgent := "userAgent_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.AuthMfaVerify(context.Background()).MFAVerifyRequestBody(mFAVerifyRequestBody).UserAgent(userAgent).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.AuthMfaVerify``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthMfaVerify`: MFAVerifyResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.AuthMfaVerify`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiAuthMfaVerifyRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **mFAVerifyRequestBody** | [**MFAVerifyRequestBody**](MFAVerifyRequestBody.md) |  | 
 **userAgent** | **string** |  | 

### Return type

[**MFAVerifyResponseBody**](MFAVerifyResponseBody.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthPasskeyLoginBegin

> PasskeyBeginLoginResponseBody AuthPasskeyLoginBegin(ctx).Execute()

Begin passkey login



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/glueops/apiary/sdk/go/apiary"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.AuthPasskeyLoginBegin(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.AuthPasskeyLoginBegin``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthPasskeyLoginBegin`: PasskeyBeginLoginResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.AuthPasskeyLoginBegin`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiAuthPasskeyLoginBeginRequest struct via the builder pattern


### Return type

[**PasskeyBeginLoginResponseBody**](PasskeyBeginLoginResponseBody.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthPasskeyLoginFinish

> PasskeyFinishLoginResponseBody AuthPasskeyLoginFinish(ctx).PasskeyFinishLoginRequestBody(passkeyFinishLoginRequestBody).UserAgent(userAgent).Execute()

Complete passkey login



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/glueops/apiary/sdk/go/apiary"
)

func main() {
	passkeyFinishLoginRequestBody := *openapiclient.NewPasskeyFinishLoginRequestBody("Challenge_example", interface{}(123)) // PasskeyFinishLoginRequestBody | 
	userAgent := "userAgent_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.AuthPasskeyLoginFinish(context.Background()).PasskeyFinishLoginRequestBody(passkeyFinishLoginRequestBody).UserAgent(userAgent).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.AuthPasskeyLoginFinish``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthPasskeyLoginFinish`: PasskeyFinishLoginResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.AuthPasskeyLoginFinish`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiAuthPasskeyLoginFinishRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **passkeyFinishLoginRequestBody** | [**PasskeyFinishLoginRequestBody**](PasskeyFinishLoginRequestBody.md) |  | 
 **userAgent** | **string** |  | 

### Return type

[**PasskeyFinishLoginResponseBody**](PasskeyFinishLoginResponseBody.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthPasskeyRegisterBegin

> PasskeyBeginRegisterResponseBody AuthPasskeyRegisterBegin(ctx).Execute()

Begin passkey registration



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/glueops/apiary/sdk/go/apiary"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.AuthPasskeyRegisterBegin(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.AuthPasskeyRegisterBegin``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthPasskeyRegisterBegin`: PasskeyBeginRegisterResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.AuthPasskeyRegisterBegin`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiAuthPasskeyRegisterBeginRequest struct via the builder pattern


### Return type

[**PasskeyBeginRegisterResponseBody**](PasskeyBeginRegisterResponseBody.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthPasskeyRegisterFinish

> PasskeyFinishRegisterResponseBody AuthPasskeyRegisterFinish(ctx).PasskeyFinishRegisterRequestBody(passkeyFinishRegisterRequestBody).Execute()

Complete passkey registration



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/glueops/apiary/sdk/go/apiary"
)

func main() {
	passkeyFinishRegisterRequestBody := *openapiclient.NewPasskeyFinishRegisterRequestBody("Challenge_example", interface{}(123), "Name_example") // PasskeyFinishRegisterRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.AuthPasskeyRegisterFinish(context.Background()).PasskeyFinishRegisterRequestBody(passkeyFinishRegisterRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.AuthPasskeyRegisterFinish``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthPasskeyRegisterFinish`: PasskeyFinishRegisterResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.AuthPasskeyRegisterFinish`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiAuthPasskeyRegisterFinishRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **passkeyFinishRegisterRequestBody** | [**PasskeyFinishRegisterRequestBody**](PasskeyFinishRegisterRequestBody.md) |  | 

### Return type

[**PasskeyFinishRegisterResponseBody**](PasskeyFinishRegisterResponseBody.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthPasskeysDelete

> PasskeyDeleteResponseBody AuthPasskeysDelete(ctx, id).Execute()

Remove a passkey



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/glueops/apiary/sdk/go/apiary"
)

func main() {
	id := "id_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.AuthPasskeysDelete(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.AuthPasskeysDelete``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthPasskeysDelete`: PasskeyDeleteResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.AuthPasskeysDelete`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiAuthPasskeysDeleteRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**PasskeyDeleteResponseBody**](PasskeyDeleteResponseBody.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthPasskeysList

> PasskeyListResponseBody AuthPasskeysList(ctx).Execute()

List registered passkeys



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/glueops/apiary/sdk/go/apiary"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.AuthPasskeysList(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.AuthPasskeysList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthPasskeysList`: PasskeyListResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.AuthPasskeysList`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiAuthPasskeysListRequest struct via the builder pattern


### Return type

[**PasskeyListResponseBody**](PasskeyListResponseBody.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthRefresh

> RefreshResponseBody AuthRefresh(ctx).RefreshRequestBody(refreshRequestBody).UserAgent(userAgent).Execute()

Refresh auth tokens



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/glueops/apiary/sdk/go/apiary"
)

func main() {
	refreshRequestBody := *openapiclient.NewRefreshRequestBody("RefreshToken_example") // RefreshRequestBody | 
	userAgent := "userAgent_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.AuthRefresh(context.Background()).RefreshRequestBody(refreshRequestBody).UserAgent(userAgent).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.AuthRefresh``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthRefresh`: RefreshResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.AuthRefresh`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiAuthRefreshRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **refreshRequestBody** | [**RefreshRequestBody**](RefreshRequestBody.md) |  | 
 **userAgent** | **string** |  | 

### Return type

[**RefreshResponseBody**](RefreshResponseBody.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthRegister

> RegisterResponseBody AuthRegister(ctx).RegisterRequestBody(registerRequestBody).XRealIp(xRealIp).UserAgent(userAgent).Execute()

Create an account and its first organization



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/glueops/apiary/sdk/go/apiary"
)

func main() {
	registerRequestBody := *openapiclient.NewRegisterRequestBody("Email_example", "FirstName_example", "Password_example") // RegisterRequestBody | 
	xRealIp := "xRealIp_example" // string |  (optional)
	userAgent := "userAgent_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.AuthRegister(context.Background()).RegisterRequestBody(registerRequestBody).XRealIp(xRealIp).UserAgent(userAgent).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.AuthRegister``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthRegister`: RegisterResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.AuthRegister`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiAuthRegisterRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **registerRequestBody** | [**RegisterRequestBody**](RegisterRequestBody.md) |  | 
 **xRealIp** | **string** |  | 
 **userAgent** | **string** |  | 

### Return type

[**RegisterResponseBody**](RegisterResponseBody.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthResendVerification

> ResendVerificationResponseBody AuthResendVerification(ctx).ResendVerificationRequestBody(resendVerificationRequestBody).Execute()

Send a new confirmation link



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/glueops/apiary/sdk/go/apiary"
)

func main() {
	resendVerificationRequestBody := *openapiclient.NewResendVerificationRequestBody("Email_example") // ResendVerificationRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.AuthResendVerification(context.Background()).ResendVerificationRequestBody(resendVerificationRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.AuthResendVerification``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthResendVerification`: ResendVerificationResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.AuthResendVerification`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiAuthResendVerificationRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **resendVerificationRequestBody** | [**ResendVerificationRequestBody**](ResendVerificationRequestBody.md) |  | 

### Return type

[**ResendVerificationResponseBody**](ResendVerificationResponseBody.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthResetPassword

> ResetPasswordResponseBody AuthResetPassword(ctx).ResetPasswordRequestBody(resetPasswordRequestBody).Execute()

Set a new password using a reset token



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/glueops/apiary/sdk/go/apiary"
)

func main() {
	resetPasswordRequestBody := *openapiclient.NewResetPasswordRequestBody("NewPassword_example", "Token_example") // ResetPasswordRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.AuthResetPassword(context.Background()).ResetPasswordRequestBody(resetPasswordRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.AuthResetPassword``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthResetPassword`: ResetPasswordResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.AuthResetPassword`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiAuthResetPasswordRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **resetPasswordRequestBody** | [**ResetPasswordRequestBody**](ResetPasswordRequestBody.md) |  | 

### Return type

[**ResetPasswordResponseBody**](ResetPasswordResponseBody.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthSocialCallback

> string AuthSocialCallback(ctx, provider).Code(code).State(state).UserAgent(userAgent).Cookie(cookie).Execute()

Handle social login callback



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/glueops/apiary/sdk/go/apiary"
)

func main() {
	provider := "provider_example" // string | 
	code := "code_example" // string | 
	state := "state_example" // string |  (optional)
	userAgent := "userAgent_example" // string |  (optional)
	cookie := "cookie_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.AuthSocialCallback(context.Background(), provider).Code(code).State(state).UserAgent(userAgent).Cookie(cookie).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.AuthSocialCallback``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthSocialCallback`: string
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.AuthSocialCallback`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**provider** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiAuthSocialCallbackRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **code** | **string** |  | 
 **state** | **string** |  | 
 **userAgent** | **string** |  | 
 **cookie** | **string** |  | 

### Return type

**string**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthSocialStart

> StartAuthResponseBody AuthSocialStart(ctx, provider).Mode(mode).Origin(origin).Execute()

Begin social login



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/glueops/apiary/sdk/go/apiary"
)

func main() {
	provider := "provider_example" // string | 
	mode := "mode_example" // string |  (optional)
	origin := "origin_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.AuthSocialStart(context.Background(), provider).Mode(mode).Origin(origin).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.AuthSocialStart``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthSocialStart`: StartAuthResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.AuthSocialStart`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**provider** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiAuthSocialStartRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **mode** | **string** |  | 
 **origin** | **string** |  | 

### Return type

[**StartAuthResponseBody**](StartAuthResponseBody.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthTotpConfirm

> TOTPVerifyResponseBody AuthTotpConfirm(ctx).TOTPVerifyRequestBody(tOTPVerifyRequestBody).Execute()

Confirm TOTP setup



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/glueops/apiary/sdk/go/apiary"
)

func main() {
	tOTPVerifyRequestBody := *openapiclient.NewTOTPVerifyRequestBody("Code_example") // TOTPVerifyRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.AuthTotpConfirm(context.Background()).TOTPVerifyRequestBody(tOTPVerifyRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.AuthTotpConfirm``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthTotpConfirm`: TOTPVerifyResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.AuthTotpConfirm`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiAuthTotpConfirmRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **tOTPVerifyRequestBody** | [**TOTPVerifyRequestBody**](TOTPVerifyRequestBody.md) |  | 

### Return type

[**TOTPVerifyResponseBody**](TOTPVerifyResponseBody.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthTotpDisable

> TOTPDisableResponseBody AuthTotpDisable(ctx).TOTPDisableRequestBody(tOTPDisableRequestBody).Execute()

Disable TOTP



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/glueops/apiary/sdk/go/apiary"
)

func main() {
	tOTPDisableRequestBody := *openapiclient.NewTOTPDisableRequestBody() // TOTPDisableRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.AuthTotpDisable(context.Background()).TOTPDisableRequestBody(tOTPDisableRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.AuthTotpDisable``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthTotpDisable`: TOTPDisableResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.AuthTotpDisable`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiAuthTotpDisableRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **tOTPDisableRequestBody** | [**TOTPDisableRequestBody**](TOTPDisableRequestBody.md) |  | 

### Return type

[**TOTPDisableResponseBody**](TOTPDisableResponseBody.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthTotpSetup

> TOTPSetupResponseBody AuthTotpSetup(ctx).Execute()

Start TOTP setup



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/glueops/apiary/sdk/go/apiary"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.AuthTotpSetup(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.AuthTotpSetup``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthTotpSetup`: TOTPSetupResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.AuthTotpSetup`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiAuthTotpSetupRequest struct via the builder pattern


### Return type

[**TOTPSetupResponseBody**](TOTPSetupResponseBody.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthVerifyEmail

> VerifyEmailResponseBody AuthVerifyEmail(ctx).VerifyEmailRequestBody(verifyEmailRequestBody).Execute()

Confirm an email address



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/glueops/apiary/sdk/go/apiary"
)

func main() {
	verifyEmailRequestBody := *openapiclient.NewVerifyEmailRequestBody("Token_example") // VerifyEmailRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.AuthVerifyEmail(context.Background()).VerifyEmailRequestBody(verifyEmailRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.AuthVerifyEmail``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthVerifyEmail`: VerifyEmailResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.AuthVerifyEmail`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiAuthVerifyEmailRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **verifyEmailRequestBody** | [**VerifyEmailRequestBody**](VerifyEmailRequestBody.md) |  | 

### Return type

[**VerifyEmailResponseBody**](VerifyEmailResponseBody.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

