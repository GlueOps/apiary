# \PlatformAPI

All URIs are relative to *http://127.0.0.1:8080*

Method | HTTP request | Description
------------- | ------------- | -------------
[**PlatformAccessRulesCreate**](PlatformAPI.md#PlatformAccessRulesCreate) | **Post** /api/v1/platform/access-rules | Create an auth access rule
[**PlatformAccessRulesDelete**](PlatformAPI.md#PlatformAccessRulesDelete) | **Delete** /api/v1/platform/access-rules/{id} | Delete an auth access rule
[**PlatformAccessRulesList**](PlatformAPI.md#PlatformAccessRulesList) | **Get** /api/v1/platform/access-rules | List auth access rules
[**PlatformAccessRulesUpdate**](PlatformAPI.md#PlatformAccessRulesUpdate) | **Put** /api/v1/platform/access-rules/{id} | Change a rule&#39;s policy
[**PlatformPoliciesClear**](PlatformAPI.md#PlatformPoliciesClear) | **Delete** /api/v1/platform/policies/{kind} | Return a platform policy to inherit
[**PlatformPoliciesImpact**](PlatformAPI.md#PlatformPoliciesImpact) | **Get** /api/v1/platform/policies/{kind}/impact | How many users this policy would lock out
[**PlatformPoliciesList**](PlatformAPI.md#PlatformPoliciesList) | **Get** /api/v1/platform/policies | List platform policies
[**PlatformPoliciesSet**](PlatformAPI.md#PlatformPoliciesSet) | **Put** /api/v1/platform/policies/{kind} | Set a platform policy



## PlatformAccessRulesCreate

> AccessRuleView PlatformAccessRulesCreate(ctx).CreateRequestBody2(createRequestBody2).Execute()

Create an auth access rule



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
	createRequestBody2 := *openapiclient.NewCreateRequestBody2("Kind_example", "Policy_example", "Purpose_example", "Value_example") // CreateRequestBody2 | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.PlatformAPI.PlatformAccessRulesCreate(context.Background()).CreateRequestBody2(createRequestBody2).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `PlatformAPI.PlatformAccessRulesCreate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PlatformAccessRulesCreate`: AccessRuleView
	fmt.Fprintf(os.Stdout, "Response from `PlatformAPI.PlatformAccessRulesCreate`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiPlatformAccessRulesCreateRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **createRequestBody2** | [**CreateRequestBody2**](CreateRequestBody2.md) |  | 

### Return type

[**AccessRuleView**](AccessRuleView.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PlatformAccessRulesDelete

> DeleteResponseBody2 PlatformAccessRulesDelete(ctx, id).Execute()

Delete an auth access rule



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
	id := "38400000-8cf0-11bd-b23e-10b96e4ef00d" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.PlatformAPI.PlatformAccessRulesDelete(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `PlatformAPI.PlatformAccessRulesDelete``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PlatformAccessRulesDelete`: DeleteResponseBody2
	fmt.Fprintf(os.Stdout, "Response from `PlatformAPI.PlatformAccessRulesDelete`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiPlatformAccessRulesDeleteRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**DeleteResponseBody2**](DeleteResponseBody2.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PlatformAccessRulesList

> ListResponseBody4 PlatformAccessRulesList(ctx).Execute()

List auth access rules



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
	resp, r, err := apiClient.PlatformAPI.PlatformAccessRulesList(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `PlatformAPI.PlatformAccessRulesList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PlatformAccessRulesList`: ListResponseBody4
	fmt.Fprintf(os.Stdout, "Response from `PlatformAPI.PlatformAccessRulesList`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiPlatformAccessRulesListRequest struct via the builder pattern


### Return type

[**ListResponseBody4**](ListResponseBody4.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PlatformAccessRulesUpdate

> AccessRuleView PlatformAccessRulesUpdate(ctx, id).UpdateRequestBody(updateRequestBody).Execute()

Change a rule's policy



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
	id := "38400000-8cf0-11bd-b23e-10b96e4ef00d" // string | 
	updateRequestBody := *openapiclient.NewUpdateRequestBody("Policy_example") // UpdateRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.PlatformAPI.PlatformAccessRulesUpdate(context.Background(), id).UpdateRequestBody(updateRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `PlatformAPI.PlatformAccessRulesUpdate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PlatformAccessRulesUpdate`: AccessRuleView
	fmt.Fprintf(os.Stdout, "Response from `PlatformAPI.PlatformAccessRulesUpdate`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiPlatformAccessRulesUpdateRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **updateRequestBody** | [**UpdateRequestBody**](UpdateRequestBody.md) |  | 

### Return type

[**AccessRuleView**](AccessRuleView.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PlatformPoliciesClear

> DeleteResponseBody PlatformPoliciesClear(ctx, kind).Execute()

Return a platform policy to inherit



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
	kind := "kind_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.PlatformAPI.PlatformPoliciesClear(context.Background(), kind).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `PlatformAPI.PlatformPoliciesClear``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PlatformPoliciesClear`: DeleteResponseBody
	fmt.Fprintf(os.Stdout, "Response from `PlatformAPI.PlatformPoliciesClear`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**kind** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiPlatformPoliciesClearRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**DeleteResponseBody**](DeleteResponseBody.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PlatformPoliciesImpact

> ImpactResponseBody PlatformPoliciesImpact(ctx, kind).Execute()

How many users this policy would lock out



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
	kind := "kind_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.PlatformAPI.PlatformPoliciesImpact(context.Background(), kind).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `PlatformAPI.PlatformPoliciesImpact``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PlatformPoliciesImpact`: ImpactResponseBody
	fmt.Fprintf(os.Stdout, "Response from `PlatformAPI.PlatformPoliciesImpact`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**kind** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiPlatformPoliciesImpactRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**ImpactResponseBody**](ImpactResponseBody.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PlatformPoliciesList

> ListResponseBody PlatformPoliciesList(ctx).Execute()

List platform policies



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
	resp, r, err := apiClient.PlatformAPI.PlatformPoliciesList(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `PlatformAPI.PlatformPoliciesList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PlatformPoliciesList`: ListResponseBody
	fmt.Fprintf(os.Stdout, "Response from `PlatformAPI.PlatformPoliciesList`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiPlatformPoliciesListRequest struct via the builder pattern


### Return type

[**ListResponseBody**](ListResponseBody.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PlatformPoliciesSet

> PolicyView PlatformPoliciesSet(ctx, kind).SetRequestBody(setRequestBody).Execute()

Set a platform policy



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
	kind := "kind_example" // string | 
	setRequestBody := *openapiclient.NewSetRequestBody("State_example") // SetRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.PlatformAPI.PlatformPoliciesSet(context.Background(), kind).SetRequestBody(setRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `PlatformAPI.PlatformPoliciesSet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PlatformPoliciesSet`: PolicyView
	fmt.Fprintf(os.Stdout, "Response from `PlatformAPI.PlatformPoliciesSet`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**kind** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiPlatformPoliciesSetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **setRequestBody** | [**SetRequestBody**](SetRequestBody.md) |  | 

### Return type

[**PolicyView**](PolicyView.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

