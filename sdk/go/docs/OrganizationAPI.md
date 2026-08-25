# \OrganizationAPI

All URIs are relative to *http://127.0.0.1:8282*

Method | HTTP request | Description
------------- | ------------- | -------------
[**OrgMembersList**](OrganizationAPI.md#OrgMembersList) | **Get** /api/v1/organization/members | List the members of this organization
[**OrgMembersRemove**](OrganizationAPI.md#OrgMembersRemove) | **Delete** /api/v1/organization/members/{id} | Remove a member from this organization
[**OrgMembersUpdateRole**](OrganizationAPI.md#OrgMembersUpdateRole) | **Patch** /api/v1/organization/members/{id} | Change a member&#39;s role
[**OrgPoliciesClear**](OrganizationAPI.md#OrgPoliciesClear) | **Delete** /api/v1/organization/policies/{kind} | Return a policy to inherit
[**OrgPoliciesImpact**](OrganizationAPI.md#OrgPoliciesImpact) | **Get** /api/v1/organization/policies/{kind}/impact | How many members this policy would lock out
[**OrgPoliciesList**](OrganizationAPI.md#OrgPoliciesList) | **Get** /api/v1/organization/policies | List this organization&#39;s policies
[**OrgPoliciesSet**](OrganizationAPI.md#OrgPoliciesSet) | **Put** /api/v1/organization/policies/{kind} | Set a policy for this organization



## OrgMembersList

> ListMembersResponseBody OrgMembersList(ctx).Execute()

List the members of this organization



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
	resp, r, err := apiClient.OrganizationAPI.OrgMembersList(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrgMembersList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrgMembersList`: ListMembersResponseBody
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrgMembersList`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiOrgMembersListRequest struct via the builder pattern


### Return type

[**ListMembersResponseBody**](ListMembersResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## OrgMembersRemove

> RemoveMemberResponseBody OrgMembersRemove(ctx, id).Execute()

Remove a member from this organization



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
	resp, r, err := apiClient.OrganizationAPI.OrgMembersRemove(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrgMembersRemove``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrgMembersRemove`: RemoveMemberResponseBody
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrgMembersRemove`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrgMembersRemoveRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**RemoveMemberResponseBody**](RemoveMemberResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## OrgMembersUpdateRole

> UpdateRoleResponseBody OrgMembersUpdateRole(ctx, id).UpdateRoleRequestBody(updateRoleRequestBody).Execute()

Change a member's role



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
	id := "38400000-8cf0-11bd-b23e-10b96e4ef00d" // string | The membership id, not the user id.
	updateRoleRequestBody := *openapiclient.NewUpdateRoleRequestBody("Role_example") // UpdateRoleRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.OrganizationAPI.OrgMembersUpdateRole(context.Background(), id).UpdateRoleRequestBody(updateRoleRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrgMembersUpdateRole``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrgMembersUpdateRole`: UpdateRoleResponseBody
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrgMembersUpdateRole`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** | The membership id, not the user id. | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrgMembersUpdateRoleRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **updateRoleRequestBody** | [**UpdateRoleRequestBody**](UpdateRoleRequestBody.md) |  | 

### Return type

[**UpdateRoleResponseBody**](UpdateRoleResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## OrgPoliciesClear

> DeleteResponseBody1 OrgPoliciesClear(ctx, kind).Execute()

Return a policy to inherit

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
	resp, r, err := apiClient.OrganizationAPI.OrgPoliciesClear(context.Background(), kind).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrgPoliciesClear``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrgPoliciesClear`: DeleteResponseBody1
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrgPoliciesClear`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**kind** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrgPoliciesClearRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**DeleteResponseBody1**](DeleteResponseBody1.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## OrgPoliciesImpact

> ImpactResponseBody OrgPoliciesImpact(ctx, kind).Execute()

How many members this policy would lock out



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
	resp, r, err := apiClient.OrganizationAPI.OrgPoliciesImpact(context.Background(), kind).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrgPoliciesImpact``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrgPoliciesImpact`: ImpactResponseBody
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrgPoliciesImpact`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**kind** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrgPoliciesImpactRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**ImpactResponseBody**](ImpactResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## OrgPoliciesList

> ListResponseBody2 OrgPoliciesList(ctx).Execute()

List this organization's policies



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
	resp, r, err := apiClient.OrganizationAPI.OrgPoliciesList(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrgPoliciesList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrgPoliciesList`: ListResponseBody2
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrgPoliciesList`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiOrgPoliciesListRequest struct via the builder pattern


### Return type

[**ListResponseBody2**](ListResponseBody2.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## OrgPoliciesSet

> PolicyView OrgPoliciesSet(ctx, kind).SetRequestBody(setRequestBody).Execute()

Set a policy for this organization



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
	resp, r, err := apiClient.OrganizationAPI.OrgPoliciesSet(context.Background(), kind).SetRequestBody(setRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrgPoliciesSet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrgPoliciesSet`: PolicyView
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrgPoliciesSet`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**kind** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrgPoliciesSetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **setRequestBody** | [**SetRequestBody**](SetRequestBody.md) |  | 

### Return type

[**PolicyView**](PolicyView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

