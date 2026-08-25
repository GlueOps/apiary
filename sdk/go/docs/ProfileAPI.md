# \ProfileAPI

All URIs are relative to *http://127.0.0.1:8282*

Method | HTTP request | Description
------------- | ------------- | -------------
[**MeChangePassword**](ProfileAPI.md#MeChangePassword) | **Put** /api/v1/me/password | Change password
[**MeGet**](ProfileAPI.md#MeGet) | **Get** /api/v1/me | Get current user profile
[**MeOrgsList**](ProfileAPI.md#MeOrgsList) | **Get** /api/v1/me/orgs | List my organizations
[**MeUpdate**](ProfileAPI.md#MeUpdate) | **Patch** /api/v1/me | Update current user profile



## MeChangePassword

> ChangePasswordResponseBody MeChangePassword(ctx).ChangePasswordRequestBody(changePasswordRequestBody).Execute()

Change password



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
	changePasswordRequestBody := *openapiclient.NewChangePasswordRequestBody("CurrentPassword_example", "NewPassword_example") // ChangePasswordRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.ProfileAPI.MeChangePassword(context.Background()).ChangePasswordRequestBody(changePasswordRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `ProfileAPI.MeChangePassword``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `MeChangePassword`: ChangePasswordResponseBody
	fmt.Fprintf(os.Stdout, "Response from `ProfileAPI.MeChangePassword`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiMeChangePasswordRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **changePasswordRequestBody** | [**ChangePasswordRequestBody**](ChangePasswordRequestBody.md) |  | 

### Return type

[**ChangePasswordResponseBody**](ChangePasswordResponseBody.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## MeGet

> GetMeResponseBody MeGet(ctx).Execute()

Get current user profile



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
	resp, r, err := apiClient.ProfileAPI.MeGet(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `ProfileAPI.MeGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `MeGet`: GetMeResponseBody
	fmt.Fprintf(os.Stdout, "Response from `ProfileAPI.MeGet`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiMeGetRequest struct via the builder pattern


### Return type

[**GetMeResponseBody**](GetMeResponseBody.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## MeOrgsList

> GetMyOrganizationsResponseBody MeOrgsList(ctx).Execute()

List my organizations



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
	resp, r, err := apiClient.ProfileAPI.MeOrgsList(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `ProfileAPI.MeOrgsList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `MeOrgsList`: GetMyOrganizationsResponseBody
	fmt.Fprintf(os.Stdout, "Response from `ProfileAPI.MeOrgsList`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiMeOrgsListRequest struct via the builder pattern


### Return type

[**GetMyOrganizationsResponseBody**](GetMyOrganizationsResponseBody.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## MeUpdate

> UpdateMeResponseBody MeUpdate(ctx).UpdateMeRequestBody(updateMeRequestBody).Execute()

Update current user profile



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
	updateMeRequestBody := *openapiclient.NewUpdateMeRequestBody() // UpdateMeRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.ProfileAPI.MeUpdate(context.Background()).UpdateMeRequestBody(updateMeRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `ProfileAPI.MeUpdate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `MeUpdate`: UpdateMeResponseBody
	fmt.Fprintf(os.Stdout, "Response from `ProfileAPI.MeUpdate`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiMeUpdateRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **updateMeRequestBody** | [**UpdateMeRequestBody**](UpdateMeRequestBody.md) |  | 

### Return type

[**UpdateMeResponseBody**](UpdateMeResponseBody.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

