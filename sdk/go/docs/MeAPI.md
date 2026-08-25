# \MeAPI

All URIs are relative to *http://127.0.0.1:8080*

Method | HTTP request | Description
------------- | ------------- | -------------
[**MeDeleteAccount**](MeAPI.md#MeDeleteAccount) | **Delete** /api/v1/me | Delete your account
[**MeSoleOwnerships**](MeAPI.md#MeSoleOwnerships) | **Get** /api/v1/me/sole-ownerships | Organizations this account solely owns



## MeDeleteAccount

> DeleteAccountResponseBody MeDeleteAccount(ctx).DeleteAccountRequestBody(deleteAccountRequestBody).Execute()

Delete your account



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
	deleteAccountRequestBody := *openapiclient.NewDeleteAccountRequestBody("ConfirmEmail_example") // DeleteAccountRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.MeAPI.MeDeleteAccount(context.Background()).DeleteAccountRequestBody(deleteAccountRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `MeAPI.MeDeleteAccount``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `MeDeleteAccount`: DeleteAccountResponseBody
	fmt.Fprintf(os.Stdout, "Response from `MeAPI.MeDeleteAccount`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiMeDeleteAccountRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **deleteAccountRequestBody** | [**DeleteAccountRequestBody**](DeleteAccountRequestBody.md) |  | 

### Return type

[**DeleteAccountResponseBody**](DeleteAccountResponseBody.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## MeSoleOwnerships

> SoleOwnershipsResponseBody MeSoleOwnerships(ctx).Execute()

Organizations this account solely owns



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
	resp, r, err := apiClient.MeAPI.MeSoleOwnerships(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `MeAPI.MeSoleOwnerships``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `MeSoleOwnerships`: SoleOwnershipsResponseBody
	fmt.Fprintf(os.Stdout, "Response from `MeAPI.MeSoleOwnerships`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiMeSoleOwnershipsRequest struct via the builder pattern


### Return type

[**SoleOwnershipsResponseBody**](SoleOwnershipsResponseBody.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

