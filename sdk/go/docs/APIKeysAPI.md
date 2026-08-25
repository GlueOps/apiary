# \APIKeysAPI

All URIs are relative to *http://127.0.0.1:8282*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ApiKeysCreate**](APIKeysAPI.md#ApiKeysCreate) | **Post** /api/v1/api-keys | Mint an API key
[**ApiKeysList**](APIKeysAPI.md#ApiKeysList) | **Get** /api/v1/api-keys | List API keys
[**ApiKeysRevoke**](APIKeysAPI.md#ApiKeysRevoke) | **Delete** /api/v1/api-keys/{id} | Revoke an API key



## ApiKeysCreate

> MintAPIKeyResponseBody ApiKeysCreate(ctx).MintAPIKeyRequestBody(mintAPIKeyRequestBody).Execute()

Mint an API key



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
	mintAPIKeyRequestBody := *openapiclient.NewMintAPIKeyRequestBody("Name_example", "Scope_example") // MintAPIKeyRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.APIKeysAPI.ApiKeysCreate(context.Background()).MintAPIKeyRequestBody(mintAPIKeyRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `APIKeysAPI.ApiKeysCreate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiKeysCreate`: MintAPIKeyResponseBody
	fmt.Fprintf(os.Stdout, "Response from `APIKeysAPI.ApiKeysCreate`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiApiKeysCreateRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **mintAPIKeyRequestBody** | [**MintAPIKeyRequestBody**](MintAPIKeyRequestBody.md) |  | 

### Return type

[**MintAPIKeyResponseBody**](MintAPIKeyResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiKeysList

> ListAPIKeysResponseBody ApiKeysList(ctx).Execute()

List API keys



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
	resp, r, err := apiClient.APIKeysAPI.ApiKeysList(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `APIKeysAPI.ApiKeysList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiKeysList`: ListAPIKeysResponseBody
	fmt.Fprintf(os.Stdout, "Response from `APIKeysAPI.ApiKeysList`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiApiKeysListRequest struct via the builder pattern


### Return type

[**ListAPIKeysResponseBody**](ListAPIKeysResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ApiKeysRevoke

> RevokeAPIKeyResponseBody ApiKeysRevoke(ctx, id).Execute()

Revoke an API key



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
	resp, r, err := apiClient.APIKeysAPI.ApiKeysRevoke(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `APIKeysAPI.ApiKeysRevoke``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApiKeysRevoke`: RevokeAPIKeyResponseBody
	fmt.Fprintf(os.Stdout, "Response from `APIKeysAPI.ApiKeysRevoke`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiApiKeysRevokeRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**RevokeAPIKeyResponseBody**](RevokeAPIKeyResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

