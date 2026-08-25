# \FleetAPI

All URIs are relative to *http://127.0.0.1:8080*

Method | HTTP request | Description
------------- | ------------- | -------------
[**FleetBeeAppendLogs**](FleetAPI.md#FleetBeeAppendLogs) | **Post** /api/v1/agent/tasks/{id}/logs | Stream task output back to the control plane
[**FleetBeeEnrol**](FleetAPI.md#FleetBeeEnrol) | **Post** /api/v1/agent/enrol | Enrol a fleet agent into an organization
[**FleetBeeFirewall**](FleetAPI.md#FleetBeeFirewall) | **Get** /api/v1/agent/firewall | Fetch the ruleset this host should enforce
[**FleetBeeHeartbeat**](FleetAPI.md#FleetBeeHeartbeat) | **Post** /api/v1/agent/heartbeat | Report liveness and collect work
[**FleetBeeReportTask**](FleetAPI.md#FleetBeeReportTask) | **Post** /api/v1/agent/tasks/{id}/status | Report progress or the outcome of a task
[**FleetBeesDelete**](FleetAPI.md#FleetBeesDelete) | **Delete** /api/v1/fleet/bees/{id} | Remove a fleet agent
[**FleetBeesList**](FleetAPI.md#FleetBeesList) | **Get** /api/v1/fleet/bees | List fleet agents
[**FleetEnrolmentTokensMint**](FleetAPI.md#FleetEnrolmentTokensMint) | **Post** /api/v1/fleet/enrolment-tokens | Mint an enrolment token
[**FleetHivesCreate**](FleetAPI.md#FleetHivesCreate) | **Post** /api/v1/fleet/hives | Create a hive
[**FleetHivesDelete**](FleetAPI.md#FleetHivesDelete) | **Delete** /api/v1/fleet/hives/{id} | Delete a hive
[**FleetHivesList**](FleetAPI.md#FleetHivesList) | **Get** /api/v1/fleet/hives | List hives
[**FleetHivesSync**](FleetAPI.md#FleetHivesSync) | **Post** /api/v1/fleet/hives/{id}/sync | Refresh a hive&#39;s capacity observations now
[**FleetHivesUpdate**](FleetAPI.md#FleetHivesUpdate) | **Patch** /api/v1/fleet/hives/{id} | Update a hive&#39;s configuration
[**FleetHostsArrive**](FleetAPI.md#FleetHostsArrive) | **Post** /api/v1/fleet/fleetless/{hypervisor_id}/arrive | Add a standalone machine to a hive
[**FleetHostsFleetless**](FleetAPI.md#FleetHostsFleetless) | **Get** /api/v1/fleet/fleetless | List machines belonging to no hive
[**FleetHypervisorsCordon**](FleetAPI.md#FleetHypervisorsCordon) | **Post** /api/v1/fleet/hypervisors/{id}/cordon | Cordon or uncordon a hypervisor
[**FleetHypervisorsDepart**](FleetAPI.md#FleetHypervisorsDepart) | **Post** /api/v1/fleet/hypervisors/{id}/depart | Remove a hypervisor from its hive
[**FleetHypervisorsGet**](FleetAPI.md#FleetHypervisorsGet) | **Get** /api/v1/fleet/hypervisors/{id} | Get one hypervisor
[**FleetHypervisorsList**](FleetAPI.md#FleetHypervisorsList) | **Get** /api/v1/fleet/hypervisors | List machines with their three status axes
[**FleetHypervisorsReprovision**](FleetAPI.md#FleetHypervisorsReprovision) | **Post** /api/v1/fleet/hypervisors/{id}/reprovision | Redrive a machine&#39;s onboarding
[**FleetTasksCreate**](FleetAPI.md#FleetTasksCreate) | **Post** /api/v1/fleet/tasks | Queue work for a bee
[**FleetTasksLogs**](FleetAPI.md#FleetTasksLogs) | **Get** /api/v1/fleet/tasks/{id}/logs | Read a task&#39;s streamed output
[**SshKeysList**](FleetAPI.md#SshKeysList) | **Get** /api/v1/ssh-keys | List fleet SSH keys
[**SshKeysMint**](FleetAPI.md#SshKeysMint) | **Post** /api/v1/ssh-keys | Mint a fleet SSH key
[**SshKeysRevoke**](FleetAPI.md#SshKeysRevoke) | **Delete** /api/v1/ssh-keys/{id} | Revoke a fleet SSH key



## FleetBeeAppendLogs

> LogsResponseBody FleetBeeAppendLogs(ctx, id).LogsRequestBody(logsRequestBody).Execute()

Stream task output back to the control plane



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
	logsRequestBody := *openapiclient.NewLogsRequestBody([]openapiclient.TaskLogLine{*openapiclient.NewTaskLogLine("Message_example", int64(123))}) // LogsRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetBeeAppendLogs(context.Background(), id).LogsRequestBody(logsRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetBeeAppendLogs``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetBeeAppendLogs`: LogsResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetBeeAppendLogs`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFleetBeeAppendLogsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **logsRequestBody** | [**LogsRequestBody**](LogsRequestBody.md) |  | 

### Return type

[**LogsResponseBody**](LogsResponseBody.md)

### Authorization

[AgentKeySecret](../README.md#AgentKeySecret), [AgentKeyAuth](../README.md#AgentKeyAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetBeeEnrol

> EnrolResponseBody FleetBeeEnrol(ctx).EnrolRequestBody(enrolRequestBody).Execute()

Enrol a fleet agent into an organization



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
	enrolRequestBody := *openapiclient.NewEnrolRequestBody("EnrolmentToken_example", "Hostname_example", "MachineId_example") // EnrolRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetBeeEnrol(context.Background()).EnrolRequestBody(enrolRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetBeeEnrol``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetBeeEnrol`: EnrolResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetBeeEnrol`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiFleetBeeEnrolRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **enrolRequestBody** | [**EnrolRequestBody**](EnrolRequestBody.md) |  | 

### Return type

[**EnrolResponseBody**](EnrolResponseBody.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetBeeFirewall

> AgentFirewallResponseBody FleetBeeFirewall(ctx).Execute()

Fetch the ruleset this host should enforce



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
	resp, r, err := apiClient.FleetAPI.FleetBeeFirewall(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetBeeFirewall``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetBeeFirewall`: AgentFirewallResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetBeeFirewall`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiFleetBeeFirewallRequest struct via the builder pattern


### Return type

[**AgentFirewallResponseBody**](AgentFirewallResponseBody.md)

### Authorization

[AgentKeySecret](../README.md#AgentKeySecret), [AgentKeyAuth](../README.md#AgentKeyAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetBeeHeartbeat

> HeartbeatResponseBody FleetBeeHeartbeat(ctx).HeartbeatRequestBody(heartbeatRequestBody).Execute()

Report liveness and collect work



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
	heartbeatRequestBody := *openapiclient.NewHeartbeatRequestBody() // HeartbeatRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetBeeHeartbeat(context.Background()).HeartbeatRequestBody(heartbeatRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetBeeHeartbeat``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetBeeHeartbeat`: HeartbeatResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetBeeHeartbeat`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiFleetBeeHeartbeatRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **heartbeatRequestBody** | [**HeartbeatRequestBody**](HeartbeatRequestBody.md) |  | 

### Return type

[**HeartbeatResponseBody**](HeartbeatResponseBody.md)

### Authorization

[AgentKeySecret](../README.md#AgentKeySecret), [AgentKeyAuth](../README.md#AgentKeyAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetBeeReportTask

> ReportResponseBody FleetBeeReportTask(ctx, id).ReportRequestBody(reportRequestBody).Execute()

Report progress or the outcome of a task



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
	reportRequestBody := *openapiclient.NewReportRequestBody("Status_example") // ReportRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetBeeReportTask(context.Background(), id).ReportRequestBody(reportRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetBeeReportTask``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetBeeReportTask`: ReportResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetBeeReportTask`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFleetBeeReportTaskRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **reportRequestBody** | [**ReportRequestBody**](ReportRequestBody.md) |  | 

### Return type

[**ReportResponseBody**](ReportResponseBody.md)

### Authorization

[AgentKeySecret](../README.md#AgentKeySecret), [AgentKeyAuth](../README.md#AgentKeyAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetBeesDelete

> DeleteBeeResponseBody FleetBeesDelete(ctx, id).Execute()

Remove a fleet agent



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
	resp, r, err := apiClient.FleetAPI.FleetBeesDelete(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetBeesDelete``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetBeesDelete`: DeleteBeeResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetBeesDelete`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFleetBeesDeleteRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**DeleteBeeResponseBody**](DeleteBeeResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetBeesList

> ListBeesResponseBody FleetBeesList(ctx).HiveId(hiveId).Execute()

List fleet agents

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
	hiveId := "38400000-8cf0-11bd-b23e-10b96e4ef00d" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetBeesList(context.Background()).HiveId(hiveId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetBeesList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetBeesList`: ListBeesResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetBeesList`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiFleetBeesListRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **hiveId** | **string** |  | 

### Return type

[**ListBeesResponseBody**](ListBeesResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetEnrolmentTokensMint

> MintTokenResponseBody FleetEnrolmentTokensMint(ctx).MintTokenRequestBody(mintTokenRequestBody).Execute()

Mint an enrolment token



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
	mintTokenRequestBody := *openapiclient.NewMintTokenRequestBody() // MintTokenRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetEnrolmentTokensMint(context.Background()).MintTokenRequestBody(mintTokenRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetEnrolmentTokensMint``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetEnrolmentTokensMint`: MintTokenResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetEnrolmentTokensMint`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiFleetEnrolmentTokensMintRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **mintTokenRequestBody** | [**MintTokenRequestBody**](MintTokenRequestBody.md) |  | 

### Return type

[**MintTokenResponseBody**](MintTokenResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetHivesCreate

> HiveView FleetHivesCreate(ctx).CreateHiveRequestBody(createHiveRequestBody).Execute()

Create a hive

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
	createHiveRequestBody := *openapiclient.NewCreateHiveRequestBody("Name_example") // CreateHiveRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetHivesCreate(context.Background()).CreateHiveRequestBody(createHiveRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetHivesCreate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetHivesCreate`: HiveView
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetHivesCreate`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiFleetHivesCreateRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **createHiveRequestBody** | [**CreateHiveRequestBody**](CreateHiveRequestBody.md) |  | 

### Return type

[**HiveView**](HiveView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetHivesDelete

> FleetHivesDelete(ctx, id).Execute()

Delete a hive



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
	r, err := apiClient.FleetAPI.FleetHivesDelete(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetHivesDelete``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFleetHivesDeleteRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

 (empty response body)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetHivesList

> ListHivesResponseBody FleetHivesList(ctx).Execute()

List hives



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
	resp, r, err := apiClient.FleetAPI.FleetHivesList(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetHivesList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetHivesList`: ListHivesResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetHivesList`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiFleetHivesListRequest struct via the builder pattern


### Return type

[**ListHivesResponseBody**](ListHivesResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetHivesSync

> SyncHiveResponseBody FleetHivesSync(ctx, id).Execute()

Refresh a hive's capacity observations now



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
	resp, r, err := apiClient.FleetAPI.FleetHivesSync(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetHivesSync``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetHivesSync`: SyncHiveResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetHivesSync`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFleetHivesSyncRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**SyncHiveResponseBody**](SyncHiveResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetHivesUpdate

> HiveDetailView FleetHivesUpdate(ctx, id).UpdateHiveRequestBody(updateHiveRequestBody).Execute()

Update a hive's configuration



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
	updateHiveRequestBody := *openapiclient.NewUpdateHiveRequestBody() // UpdateHiveRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetHivesUpdate(context.Background(), id).UpdateHiveRequestBody(updateHiveRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetHivesUpdate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetHivesUpdate`: HiveDetailView
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetHivesUpdate`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFleetHivesUpdateRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **updateHiveRequestBody** | [**UpdateHiveRequestBody**](UpdateHiveRequestBody.md) |  | 

### Return type

[**HiveDetailView**](HiveDetailView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetHostsArrive

> ArriveResponseBody FleetHostsArrive(ctx, hypervisorId).ArriveRequestBody(arriveRequestBody).Execute()

Add a standalone machine to a hive



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
	hypervisorId := "38400000-8cf0-11bd-b23e-10b96e4ef00d" // string | 
	arriveRequestBody := *openapiclient.NewArriveRequestBody([]openapiclient.CorosyncLinkInput{*openapiclient.NewCorosyncLinkInput("Address_example", int64(123))}, "TargetHiveId_example") // ArriveRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetHostsArrive(context.Background(), hypervisorId).ArriveRequestBody(arriveRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetHostsArrive``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetHostsArrive`: ArriveResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetHostsArrive`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**hypervisorId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFleetHostsArriveRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **arriveRequestBody** | [**ArriveRequestBody**](ArriveRequestBody.md) |  | 

### Return type

[**ArriveResponseBody**](ArriveResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetHostsFleetless

> ListFleetlessResponseBody FleetHostsFleetless(ctx).Execute()

List machines belonging to no hive



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
	resp, r, err := apiClient.FleetAPI.FleetHostsFleetless(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetHostsFleetless``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetHostsFleetless`: ListFleetlessResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetHostsFleetless`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiFleetHostsFleetlessRequest struct via the builder pattern


### Return type

[**ListFleetlessResponseBody**](ListFleetlessResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetHypervisorsCordon

> HypervisorView FleetHypervisorsCordon(ctx, id).CordonRequestBody(cordonRequestBody).Execute()

Cordon or uncordon a hypervisor



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
	cordonRequestBody := *openapiclient.NewCordonRequestBody(false) // CordonRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetHypervisorsCordon(context.Background(), id).CordonRequestBody(cordonRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetHypervisorsCordon``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetHypervisorsCordon`: HypervisorView
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetHypervisorsCordon`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFleetHypervisorsCordonRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **cordonRequestBody** | [**CordonRequestBody**](CordonRequestBody.md) |  | 

### Return type

[**HypervisorView**](HypervisorView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetHypervisorsDepart

> DepartResponseBody FleetHypervisorsDepart(ctx, id).Execute()

Remove a hypervisor from its hive



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
	resp, r, err := apiClient.FleetAPI.FleetHypervisorsDepart(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetHypervisorsDepart``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetHypervisorsDepart`: DepartResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetHypervisorsDepart`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFleetHypervisorsDepartRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**DepartResponseBody**](DepartResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetHypervisorsGet

> HypervisorView FleetHypervisorsGet(ctx, id).Execute()

Get one hypervisor

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
	resp, r, err := apiClient.FleetAPI.FleetHypervisorsGet(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetHypervisorsGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetHypervisorsGet`: HypervisorView
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetHypervisorsGet`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFleetHypervisorsGetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**HypervisorView**](HypervisorView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetHypervisorsList

> ListHypervisorsResponseBody FleetHypervisorsList(ctx).HiveId(hiveId).Execute()

List machines with their three status axes



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
	hiveId := "38400000-8cf0-11bd-b23e-10b96e4ef00d" // string | Restrict to one hive. (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetHypervisorsList(context.Background()).HiveId(hiveId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetHypervisorsList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetHypervisorsList`: ListHypervisorsResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetHypervisorsList`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiFleetHypervisorsListRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **hiveId** | **string** | Restrict to one hive. | 

### Return type

[**ListHypervisorsResponseBody**](ListHypervisorsResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetHypervisorsReprovision

> HypervisorView FleetHypervisorsReprovision(ctx, id).Execute()

Redrive a machine's onboarding



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
	resp, r, err := apiClient.FleetAPI.FleetHypervisorsReprovision(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetHypervisorsReprovision``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetHypervisorsReprovision`: HypervisorView
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetHypervisorsReprovision`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFleetHypervisorsReprovisionRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**HypervisorView**](HypervisorView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetTasksCreate

> CreateTaskResponseBody FleetTasksCreate(ctx).CreateTaskRequestBody(createTaskRequestBody).Execute()

Queue work for a bee



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
	createTaskRequestBody := *openapiclient.NewCreateTaskRequestBody("BeeId_example", "Kind_example") // CreateTaskRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetTasksCreate(context.Background()).CreateTaskRequestBody(createTaskRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetTasksCreate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetTasksCreate`: CreateTaskResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetTasksCreate`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiFleetTasksCreateRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **createTaskRequestBody** | [**CreateTaskRequestBody**](CreateTaskRequestBody.md) |  | 

### Return type

[**CreateTaskResponseBody**](CreateTaskResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetTasksLogs

> TaskLogsResponseBody FleetTasksLogs(ctx, id).Since(since).Execute()

Read a task's streamed output



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
	since := int64(789) // int64 | Return lines with seq greater than this. The UI polls with the highest seq it has, so a tail is one small query rather than a re-read. (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetTasksLogs(context.Background(), id).Since(since).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetTasksLogs``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetTasksLogs`: TaskLogsResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetTasksLogs`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFleetTasksLogsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **since** | **int64** | Return lines with seq greater than this. The UI polls with the highest seq it has, so a tail is one small query rather than a re-read. | 

### Return type

[**TaskLogsResponseBody**](TaskLogsResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SshKeysList

> ListResponseBody3 SshKeysList(ctx).Execute()

List fleet SSH keys



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
	resp, r, err := apiClient.FleetAPI.SshKeysList(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.SshKeysList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `SshKeysList`: ListResponseBody3
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.SshKeysList`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiSshKeysListRequest struct via the builder pattern


### Return type

[**ListResponseBody3**](ListResponseBody3.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SshKeysMint

> KeyView SshKeysMint(ctx).MintRequestBody(mintRequestBody).Execute()

Mint a fleet SSH key



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
	mintRequestBody := *openapiclient.NewMintRequestBody("Name_example") // MintRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.SshKeysMint(context.Background()).MintRequestBody(mintRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.SshKeysMint``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `SshKeysMint`: KeyView
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.SshKeysMint`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiSshKeysMintRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **mintRequestBody** | [**MintRequestBody**](MintRequestBody.md) |  | 

### Return type

[**KeyView**](KeyView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SshKeysRevoke

> RevokeResponseBody1 SshKeysRevoke(ctx, id).Execute()

Revoke a fleet SSH key



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
	resp, r, err := apiClient.FleetAPI.SshKeysRevoke(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.SshKeysRevoke``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `SshKeysRevoke`: RevokeResponseBody1
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.SshKeysRevoke`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiSshKeysRevokeRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**RevokeResponseBody1**](RevokeResponseBody1.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

