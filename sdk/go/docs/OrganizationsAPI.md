# \OrganizationsAPI

All URIs are relative to *http://127.0.0.1:8282*

Method | HTTP request | Description
------------- | ------------- | -------------
[**OrgDelete**](OrganizationsAPI.md#OrgDelete) | **Delete** /api/v1/orgs/current | Delete this organization



## OrgDelete

> DeleteOrgResponseBody OrgDelete(ctx).DeleteOrgRequestBody(deleteOrgRequestBody).Execute()

Delete this organization



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
	deleteOrgRequestBody := *openapiclient.NewDeleteOrgRequestBody("ConfirmName_example") // DeleteOrgRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.OrganizationsAPI.OrgDelete(context.Background()).DeleteOrgRequestBody(deleteOrgRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationsAPI.OrgDelete``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrgDelete`: DeleteOrgResponseBody
	fmt.Fprintf(os.Stdout, "Response from `OrganizationsAPI.OrgDelete`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiOrgDeleteRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **deleteOrgRequestBody** | [**DeleteOrgRequestBody**](DeleteOrgRequestBody.md) |  | 

### Return type

[**DeleteOrgResponseBody**](DeleteOrgResponseBody.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

