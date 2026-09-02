# \PlatformAPI

All URIs are relative to *http://127.0.0.1:8080*

Method | HTTP request | Description
------------- | ------------- | -------------
[**PlatformAccessRulesCreate**](PlatformAPI.md#PlatformAccessRulesCreate) | **Post** /api/v1/platform/access-rules | Create an auth access rule
[**PlatformAccessRulesDelete**](PlatformAPI.md#PlatformAccessRulesDelete) | **Delete** /api/v1/platform/access-rules/{id} | Delete an auth access rule
[**PlatformAccessRulesList**](PlatformAPI.md#PlatformAccessRulesList) | **Get** /api/v1/platform/access-rules | List auth access rules
[**PlatformAccessRulesUpdate**](PlatformAPI.md#PlatformAccessRulesUpdate) | **Put** /api/v1/platform/access-rules/{id} | Change a rule&#39;s policy
[**PlatformFlagClear**](PlatformAPI.md#PlatformFlagClear) | **Delete** /api/v1/platform/flags/{key} | Drop the installation-wide value, restoring the built-in default
[**PlatformFlagOverrides**](PlatformAPI.md#PlatformFlagOverrides) | **Get** /api/v1/platform/flags/{key}/overrides | Which organizations have set this flag themselves
[**PlatformFlagSet**](PlatformAPI.md#PlatformFlagSet) | **Put** /api/v1/platform/flags/{key} | Set a flag for the whole installation
[**PlatformFlagsList**](PlatformAPI.md#PlatformFlagsList) | **Get** /api/v1/platform/flags | List every declared feature flag
[**PlatformOrgFlagClear**](PlatformAPI.md#PlatformOrgFlagClear) | **Delete** /api/v1/platform/organizations/{org_id}/flags/{key} | Drop one organization&#39;s override
[**PlatformOrgFlagSet**](PlatformAPI.md#PlatformOrgFlagSet) | **Put** /api/v1/platform/organizations/{org_id}/flags/{key} | Set a flag for one organization
[**PlatformOrganizationsList**](PlatformAPI.md#PlatformOrganizationsList) | **Get** /api/v1/platform/organizations | The organizations on this installation
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


## PlatformFlagClear

> FlagView PlatformFlagClear(ctx, key).Execute()

Drop the installation-wide value, restoring the built-in default



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
	key := "key_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.PlatformAPI.PlatformFlagClear(context.Background(), key).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `PlatformAPI.PlatformFlagClear``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PlatformFlagClear`: FlagView
	fmt.Fprintf(os.Stdout, "Response from `PlatformAPI.PlatformFlagClear`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**key** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiPlatformFlagClearRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**FlagView**](FlagView.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PlatformFlagOverrides

> FlagOverridesResponseBody PlatformFlagOverrides(ctx, key).Execute()

Which organizations have set this flag themselves



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
	key := "key_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.PlatformAPI.PlatformFlagOverrides(context.Background(), key).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `PlatformAPI.PlatformFlagOverrides``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PlatformFlagOverrides`: FlagOverridesResponseBody
	fmt.Fprintf(os.Stdout, "Response from `PlatformAPI.PlatformFlagOverrides`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**key** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiPlatformFlagOverridesRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**FlagOverridesResponseBody**](FlagOverridesResponseBody.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PlatformFlagSet

> FlagView PlatformFlagSet(ctx, key).SetFlagRequestBody(setFlagRequestBody).Execute()

Set a flag for the whole installation



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
	key := "key_example" // string | 
	setFlagRequestBody := *openapiclient.NewSetFlagRequestBody(false) // SetFlagRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.PlatformAPI.PlatformFlagSet(context.Background(), key).SetFlagRequestBody(setFlagRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `PlatformAPI.PlatformFlagSet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PlatformFlagSet`: FlagView
	fmt.Fprintf(os.Stdout, "Response from `PlatformAPI.PlatformFlagSet`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**key** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiPlatformFlagSetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **setFlagRequestBody** | [**SetFlagRequestBody**](SetFlagRequestBody.md) |  | 

### Return type

[**FlagView**](FlagView.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PlatformFlagsList

> FlagListResponseBody PlatformFlagsList(ctx).Execute()

List every declared feature flag



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
	resp, r, err := apiClient.PlatformAPI.PlatformFlagsList(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `PlatformAPI.PlatformFlagsList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PlatformFlagsList`: FlagListResponseBody
	fmt.Fprintf(os.Stdout, "Response from `PlatformAPI.PlatformFlagsList`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiPlatformFlagsListRequest struct via the builder pattern


### Return type

[**FlagListResponseBody**](FlagListResponseBody.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PlatformOrgFlagClear

> FlagView PlatformOrgFlagClear(ctx, orgId, key).Execute()

Drop one organization's override



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
	orgId := "38400000-8cf0-11bd-b23e-10b96e4ef00d" // string | 
	key := "key_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.PlatformAPI.PlatformOrgFlagClear(context.Background(), orgId, key).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `PlatformAPI.PlatformOrgFlagClear``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PlatformOrgFlagClear`: FlagView
	fmt.Fprintf(os.Stdout, "Response from `PlatformAPI.PlatformOrgFlagClear`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **string** |  | 
**key** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiPlatformOrgFlagClearRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**FlagView**](FlagView.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PlatformOrgFlagSet

> FlagView PlatformOrgFlagSet(ctx, orgId, key).SetOrgFlagRequestBody(setOrgFlagRequestBody).Execute()

Set a flag for one organization



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
	orgId := "38400000-8cf0-11bd-b23e-10b96e4ef00d" // string | 
	key := "key_example" // string | 
	setOrgFlagRequestBody := *openapiclient.NewSetOrgFlagRequestBody(false) // SetOrgFlagRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.PlatformAPI.PlatformOrgFlagSet(context.Background(), orgId, key).SetOrgFlagRequestBody(setOrgFlagRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `PlatformAPI.PlatformOrgFlagSet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PlatformOrgFlagSet`: FlagView
	fmt.Fprintf(os.Stdout, "Response from `PlatformAPI.PlatformOrgFlagSet`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **string** |  | 
**key** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiPlatformOrgFlagSetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **setOrgFlagRequestBody** | [**SetOrgFlagRequestBody**](SetOrgFlagRequestBody.md) |  | 

### Return type

[**FlagView**](FlagView.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PlatformOrganizationsList

> ListOrganizationsResponseBody PlatformOrganizationsList(ctx).Q(q).IncludeDeleted(includeDeleted).Limit(limit).Offset(offset).Execute()

The organizations on this installation



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
	q := "q_example" // string | Match the name or domain, case-insensitively. (optional)
	includeDeleted := true // bool | Include soft-deleted organizations. Off by default: a deleted tenant is not something to target. (optional)
	limit := int64(789) // int64 | Default 50. (optional)
	offset := int64(789) // int64 |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.PlatformAPI.PlatformOrganizationsList(context.Background()).Q(q).IncludeDeleted(includeDeleted).Limit(limit).Offset(offset).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `PlatformAPI.PlatformOrganizationsList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PlatformOrganizationsList`: ListOrganizationsResponseBody
	fmt.Fprintf(os.Stdout, "Response from `PlatformAPI.PlatformOrganizationsList`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiPlatformOrganizationsListRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **q** | **string** | Match the name or domain, case-insensitively. | 
 **includeDeleted** | **bool** | Include soft-deleted organizations. Off by default: a deleted tenant is not something to target. | 
 **limit** | **int64** | Default 50. | 
 **offset** | **int64** |  | 

### Return type

[**ListOrganizationsResponseBody**](ListOrganizationsResponseBody.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
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

