# \FirewallAPI

All URIs are relative to *http://127.0.0.1:8282*

Method | HTTP request | Description
------------- | ------------- | -------------
[**FirewallBindingsCreate**](FirewallAPI.md#FirewallBindingsCreate) | **Post** /api/v1/firewall/bindings | Bind a ruleset to a fleet, or to the whole organization
[**FirewallBindingsDelete**](FirewallAPI.md#FirewallBindingsDelete) | **Delete** /api/v1/firewall/bindings/{id} | Remove a binding
[**FirewallBindingsList**](FirewallAPI.md#FirewallBindingsList) | **Get** /api/v1/firewall/bindings | List ruleset bindings
[**FirewallBindingsUpdate**](FirewallAPI.md#FirewallBindingsUpdate) | **Put** /api/v1/firewall/bindings/{id} | Reprioritise or disable a binding
[**FirewallPreview**](FirewallAPI.md#FirewallPreview) | **Get** /api/v1/firewall/preview | Preview the rules a fleet would receive
[**FirewallRulesCreate**](FirewallAPI.md#FirewallRulesCreate) | **Post** /api/v1/firewall/rulesets/{id}/rules | Add a rule to a ruleset
[**FirewallRulesDelete**](FirewallAPI.md#FirewallRulesDelete) | **Delete** /api/v1/firewall/rules/{id} | Delete a rule
[**FirewallRulesList**](FirewallAPI.md#FirewallRulesList) | **Get** /api/v1/firewall/rulesets/{id}/rules | List a ruleset&#39;s rules
[**FirewallRulesUpdate**](FirewallAPI.md#FirewallRulesUpdate) | **Put** /api/v1/firewall/rules/{id} | Replace a rule
[**FirewallRulesetsCreate**](FirewallAPI.md#FirewallRulesetsCreate) | **Post** /api/v1/firewall/rulesets | Create a firewall ruleset
[**FirewallRulesetsDelete**](FirewallAPI.md#FirewallRulesetsDelete) | **Delete** /api/v1/firewall/rulesets/{id} | Delete a firewall ruleset
[**FirewallRulesetsList**](FirewallAPI.md#FirewallRulesetsList) | **Get** /api/v1/firewall/rulesets | List firewall rulesets
[**FirewallRulesetsUpdate**](FirewallAPI.md#FirewallRulesetsUpdate) | **Put** /api/v1/firewall/rulesets/{id} | Rename a firewall ruleset



## FirewallBindingsCreate

> BindingView FirewallBindingsCreate(ctx).CreateBindingRequestBody(createBindingRequestBody).Execute()

Bind a ruleset to a fleet, or to the whole organization

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
	createBindingRequestBody := *openapiclient.NewCreateBindingRequestBody(false, "RulesetId_example") // CreateBindingRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FirewallAPI.FirewallBindingsCreate(context.Background()).CreateBindingRequestBody(createBindingRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FirewallAPI.FirewallBindingsCreate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FirewallBindingsCreate`: BindingView
	fmt.Fprintf(os.Stdout, "Response from `FirewallAPI.FirewallBindingsCreate`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiFirewallBindingsCreateRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **createBindingRequestBody** | [**CreateBindingRequestBody**](CreateBindingRequestBody.md) |  | 

### Return type

[**BindingView**](BindingView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FirewallBindingsDelete

> DeletedResponseBody FirewallBindingsDelete(ctx, id).Execute()

Remove a binding

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
	resp, r, err := apiClient.FirewallAPI.FirewallBindingsDelete(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FirewallAPI.FirewallBindingsDelete``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FirewallBindingsDelete`: DeletedResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FirewallAPI.FirewallBindingsDelete`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFirewallBindingsDeleteRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**DeletedResponseBody**](DeletedResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FirewallBindingsList

> ListBindingsResponseBody FirewallBindingsList(ctx).Execute()

List ruleset bindings



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
	resp, r, err := apiClient.FirewallAPI.FirewallBindingsList(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FirewallAPI.FirewallBindingsList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FirewallBindingsList`: ListBindingsResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FirewallAPI.FirewallBindingsList`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiFirewallBindingsListRequest struct via the builder pattern


### Return type

[**ListBindingsResponseBody**](ListBindingsResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FirewallBindingsUpdate

> DeletedResponseBody FirewallBindingsUpdate(ctx, id).UpdateBindingRequestBody(updateBindingRequestBody).Execute()

Reprioritise or disable a binding

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
	updateBindingRequestBody := *openapiclient.NewUpdateBindingRequestBody(false) // UpdateBindingRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FirewallAPI.FirewallBindingsUpdate(context.Background(), id).UpdateBindingRequestBody(updateBindingRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FirewallAPI.FirewallBindingsUpdate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FirewallBindingsUpdate`: DeletedResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FirewallAPI.FirewallBindingsUpdate`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFirewallBindingsUpdateRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **updateBindingRequestBody** | [**UpdateBindingRequestBody**](UpdateBindingRequestBody.md) |  | 

### Return type

[**DeletedResponseBody**](DeletedResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FirewallPreview

> PreviewResponseBody1 FirewallPreview(ctx).HiveId(hiveId).Execute()

Preview the rules a fleet would receive



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
	hiveId := "38400000-8cf0-11bd-b23e-10b96e4ef00d" // string | Omit to preview a host that belongs to NO fleet — the state every migration passes through. (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FirewallAPI.FirewallPreview(context.Background()).HiveId(hiveId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FirewallAPI.FirewallPreview``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FirewallPreview`: PreviewResponseBody1
	fmt.Fprintf(os.Stdout, "Response from `FirewallAPI.FirewallPreview`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiFirewallPreviewRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **hiveId** | **string** | Omit to preview a host that belongs to NO fleet — the state every migration passes through. | 

### Return type

[**PreviewResponseBody1**](PreviewResponseBody1.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FirewallRulesCreate

> RuleView FirewallRulesCreate(ctx, id).RuleBody(ruleBody).Execute()

Add a rule to a ruleset



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
	ruleBody := *openapiclient.NewRuleBody(false, "Name_example") // RuleBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FirewallAPI.FirewallRulesCreate(context.Background(), id).RuleBody(ruleBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FirewallAPI.FirewallRulesCreate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FirewallRulesCreate`: RuleView
	fmt.Fprintf(os.Stdout, "Response from `FirewallAPI.FirewallRulesCreate`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFirewallRulesCreateRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ruleBody** | [**RuleBody**](RuleBody.md) |  | 

### Return type

[**RuleView**](RuleView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FirewallRulesDelete

> DeletedResponseBody FirewallRulesDelete(ctx, id).Execute()

Delete a rule

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
	resp, r, err := apiClient.FirewallAPI.FirewallRulesDelete(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FirewallAPI.FirewallRulesDelete``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FirewallRulesDelete`: DeletedResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FirewallAPI.FirewallRulesDelete`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFirewallRulesDeleteRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**DeletedResponseBody**](DeletedResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FirewallRulesList

> ListRulesResponseBody FirewallRulesList(ctx, id).Execute()

List a ruleset's rules



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
	resp, r, err := apiClient.FirewallAPI.FirewallRulesList(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FirewallAPI.FirewallRulesList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FirewallRulesList`: ListRulesResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FirewallAPI.FirewallRulesList`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFirewallRulesListRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**ListRulesResponseBody**](ListRulesResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FirewallRulesUpdate

> RuleView FirewallRulesUpdate(ctx, id).RuleBody(ruleBody).Execute()

Replace a rule



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
	ruleBody := *openapiclient.NewRuleBody(false, "Name_example") // RuleBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FirewallAPI.FirewallRulesUpdate(context.Background(), id).RuleBody(ruleBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FirewallAPI.FirewallRulesUpdate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FirewallRulesUpdate`: RuleView
	fmt.Fprintf(os.Stdout, "Response from `FirewallAPI.FirewallRulesUpdate`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFirewallRulesUpdateRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **ruleBody** | [**RuleBody**](RuleBody.md) |  | 

### Return type

[**RuleView**](RuleView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FirewallRulesetsCreate

> RulesetView FirewallRulesetsCreate(ctx).CreateRulesetRequestBody(createRulesetRequestBody).Execute()

Create a firewall ruleset

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
	createRulesetRequestBody := *openapiclient.NewCreateRulesetRequestBody("Name_example") // CreateRulesetRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FirewallAPI.FirewallRulesetsCreate(context.Background()).CreateRulesetRequestBody(createRulesetRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FirewallAPI.FirewallRulesetsCreate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FirewallRulesetsCreate`: RulesetView
	fmt.Fprintf(os.Stdout, "Response from `FirewallAPI.FirewallRulesetsCreate`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiFirewallRulesetsCreateRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **createRulesetRequestBody** | [**CreateRulesetRequestBody**](CreateRulesetRequestBody.md) |  | 

### Return type

[**RulesetView**](RulesetView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FirewallRulesetsDelete

> DeletedResponseBody FirewallRulesetsDelete(ctx, id).Execute()

Delete a firewall ruleset



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
	resp, r, err := apiClient.FirewallAPI.FirewallRulesetsDelete(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FirewallAPI.FirewallRulesetsDelete``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FirewallRulesetsDelete`: DeletedResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FirewallAPI.FirewallRulesetsDelete`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFirewallRulesetsDeleteRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**DeletedResponseBody**](DeletedResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FirewallRulesetsList

> ListRulesetsResponseBody FirewallRulesetsList(ctx).Execute()

List firewall rulesets



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
	resp, r, err := apiClient.FirewallAPI.FirewallRulesetsList(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FirewallAPI.FirewallRulesetsList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FirewallRulesetsList`: ListRulesetsResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FirewallAPI.FirewallRulesetsList`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiFirewallRulesetsListRequest struct via the builder pattern


### Return type

[**ListRulesetsResponseBody**](ListRulesetsResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FirewallRulesetsUpdate

> RulesetView FirewallRulesetsUpdate(ctx, id).UpdateRulesetRequestBody(updateRulesetRequestBody).Execute()

Rename a firewall ruleset

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
	updateRulesetRequestBody := *openapiclient.NewUpdateRulesetRequestBody("Name_example") // UpdateRulesetRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FirewallAPI.FirewallRulesetsUpdate(context.Background(), id).UpdateRulesetRequestBody(updateRulesetRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FirewallAPI.FirewallRulesetsUpdate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FirewallRulesetsUpdate`: RulesetView
	fmt.Fprintf(os.Stdout, "Response from `FirewallAPI.FirewallRulesetsUpdate`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFirewallRulesetsUpdateRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **updateRulesetRequestBody** | [**UpdateRulesetRequestBody**](UpdateRulesetRequestBody.md) |  | 

### Return type

[**RulesetView**](RulesetView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

