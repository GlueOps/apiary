# \AdminAPI

All URIs are relative to *http://127.0.0.1:8080*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AdminImpersonateEnd**](AdminAPI.md#AdminImpersonateEnd) | **Post** /api/v1/admin/impersonate/end | End the current impersonated session
[**AdminImpersonateStart**](AdminAPI.md#AdminImpersonateStart) | **Post** /api/v1/admin/impersonate | Begin impersonating a user



## AdminImpersonateEnd

> ImpersonateEndResponseBody AdminImpersonateEnd(ctx).ImpersonateEndRequestBody(impersonateEndRequestBody).XRealIp(xRealIp).UserAgent(userAgent).Execute()

End the current impersonated session



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
	impersonateEndRequestBody := *openapiclient.NewImpersonateEndRequestBody() // ImpersonateEndRequestBody | 
	xRealIp := "xRealIp_example" // string |  (optional)
	userAgent := "userAgent_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AdminAPI.AdminImpersonateEnd(context.Background()).ImpersonateEndRequestBody(impersonateEndRequestBody).XRealIp(xRealIp).UserAgent(userAgent).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AdminAPI.AdminImpersonateEnd``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AdminImpersonateEnd`: ImpersonateEndResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AdminAPI.AdminImpersonateEnd`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiAdminImpersonateEndRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **impersonateEndRequestBody** | [**ImpersonateEndRequestBody**](ImpersonateEndRequestBody.md) |  | 
 **xRealIp** | **string** |  | 
 **userAgent** | **string** |  | 

### Return type

[**ImpersonateEndResponseBody**](ImpersonateEndResponseBody.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AdminImpersonateStart

> ImpersonateStartResponseBody AdminImpersonateStart(ctx).ImpersonateStartRequestBody(impersonateStartRequestBody).XRealIp(xRealIp).UserAgent(userAgent).Execute()

Begin impersonating a user



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
	impersonateStartRequestBody := *openapiclient.NewImpersonateStartRequestBody("OrganizationId_example", "Reason_example", "UserId_example") // ImpersonateStartRequestBody | 
	xRealIp := "xRealIp_example" // string |  (optional)
	userAgent := "userAgent_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AdminAPI.AdminImpersonateStart(context.Background()).ImpersonateStartRequestBody(impersonateStartRequestBody).XRealIp(xRealIp).UserAgent(userAgent).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AdminAPI.AdminImpersonateStart``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AdminImpersonateStart`: ImpersonateStartResponseBody
	fmt.Fprintf(os.Stdout, "Response from `AdminAPI.AdminImpersonateStart`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiAdminImpersonateStartRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **impersonateStartRequestBody** | [**ImpersonateStartRequestBody**](ImpersonateStartRequestBody.md) |  | 
 **xRealIp** | **string** |  | 
 **userAgent** | **string** |  | 

### Return type

[**ImpersonateStartResponseBody**](ImpersonateStartResponseBody.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

