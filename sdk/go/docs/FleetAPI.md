# \FleetAPI

All URIs are relative to *http://127.0.0.1:8080*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AuditEventsList**](FleetAPI.md#AuditEventsList) | **Get** /api/v1/audit/events | What has been done in this organization, and by whom
[**FleetActivityList**](FleetAPI.md#FleetActivityList) | **Get** /api/v1/fleet/activity | What the fleet&#39;s agents have been asked to do, and what came of it
[**FleetAgentUpgrade**](FleetAPI.md#FleetAgentUpgrade) | **Post** /api/v1/fleet/agent-upgrades | Upgrade every agent in a fleet
[**FleetBeeAgentBinary**](FleetAPI.md#FleetBeeAgentBinary) | **Get** /api/v1/agent/dist/{os}/{arch} | Download the agent binary for one platform
[**FleetBeeAgentDist**](FleetAPI.md#FleetBeeAgentDist) | **Get** /api/v1/agent/dist | What agent binaries this control plane can serve
[**FleetBeeAppendLogs**](FleetAPI.md#FleetBeeAppendLogs) | **Post** /api/v1/agent/tasks/{id}/logs | Stream task output back to the control plane
[**FleetBeeEnrol**](FleetAPI.md#FleetBeeEnrol) | **Post** /api/v1/agent/enrol | Enrol a fleet agent into an organization
[**FleetBeeFirewall**](FleetAPI.md#FleetBeeFirewall) | **Get** /api/v1/agent/firewall | Fetch the ruleset this host should enforce
[**FleetBeeHeartbeat**](FleetAPI.md#FleetBeeHeartbeat) | **Post** /api/v1/agent/heartbeat | Report liveness and collect work
[**FleetBeeMembership**](FleetAPI.md#FleetBeeMembership) | **Get** /api/v1/agent/membership | Fetch the cluster composition this host&#39;s hive should have
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
[**FleetHostsBootstrap**](FleetAPI.md#FleetHostsBootstrap) | **Post** /api/v1/fleet/fleetless/{hypervisor_id}/bootstrap | Create a cluster on a standalone machine
[**FleetHostsFleetless**](FleetAPI.md#FleetHostsFleetless) | **Get** /api/v1/fleet/fleetless | List machines belonging to no hive
[**FleetHypervisorsCordon**](FleetAPI.md#FleetHypervisorsCordon) | **Post** /api/v1/fleet/hypervisors/{id}/cordon | Cordon or uncordon a hypervisor
[**FleetHypervisorsDeclare**](FleetAPI.md#FleetHypervisorsDeclare) | **Post** /api/v1/fleet/hypervisors | Bring hosts into the fleet by address
[**FleetHypervisorsDepart**](FleetAPI.md#FleetHypervisorsDepart) | **Post** /api/v1/fleet/hypervisors/{id}/depart | Remove a hypervisor from its hive
[**FleetHypervisorsDisarm**](FleetAPI.md#FleetHypervisorsDisarm) | **Post** /api/v1/fleet/hypervisors/{id}/disarm | Destroy the cluster secrets a departed machine still holds
[**FleetHypervisorsGet**](FleetAPI.md#FleetHypervisorsGet) | **Get** /api/v1/fleet/hypervisors/{id} | Get one hypervisor
[**FleetHypervisorsList**](FleetAPI.md#FleetHypervisorsList) | **Get** /api/v1/fleet/hypervisors | List machines with their three status axes
[**FleetHypervisorsReprovision**](FleetAPI.md#FleetHypervisorsReprovision) | **Post** /api/v1/fleet/hypervisors/{id}/reprovision | Redrive a machine&#39;s onboarding
[**FleetMaintenanceRunsList**](FleetAPI.md#FleetMaintenanceRunsList) | **Get** /api/v1/fleet/maintenance/runs | List recent maintenance runs
[**FleetMaintenanceRunsStop**](FleetAPI.md#FleetMaintenanceRunsStop) | **Post** /api/v1/fleet/maintenance/runs/{id}/stop | Stop a maintenance run
[**FleetMaintenanceSchedulesCreate**](FleetAPI.md#FleetMaintenanceSchedulesCreate) | **Post** /api/v1/fleet/maintenance/schedules | Schedule a maintenance window
[**FleetMaintenanceSchedulesDelete**](FleetAPI.md#FleetMaintenanceSchedulesDelete) | **Delete** /api/v1/fleet/maintenance/schedules/{id} | Delete a maintenance window
[**FleetMaintenanceSchedulesList**](FleetAPI.md#FleetMaintenanceSchedulesList) | **Get** /api/v1/fleet/maintenance/schedules | List maintenance windows
[**FleetMaintenanceSchedulesUpdate**](FleetAPI.md#FleetMaintenanceSchedulesUpdate) | **Put** /api/v1/fleet/maintenance/schedules/{id} | Change a maintenance window
[**FleetTasksCreate**](FleetAPI.md#FleetTasksCreate) | **Post** /api/v1/fleet/tasks | Queue work for a bee
[**FleetTasksLogs**](FleetAPI.md#FleetTasksLogs) | **Get** /api/v1/fleet/tasks/{id}/logs | Read a task&#39;s streamed output
[**SshKeysList**](FleetAPI.md#SshKeysList) | **Get** /api/v1/ssh-keys | List fleet SSH keys
[**SshKeysMint**](FleetAPI.md#SshKeysMint) | **Post** /api/v1/ssh-keys | Mint a fleet SSH key
[**SshKeysRevoke**](FleetAPI.md#SshKeysRevoke) | **Delete** /api/v1/ssh-keys/{id} | Revoke a fleet SSH key



## AuditEventsList

> ListAuditResponseBody AuditEventsList(ctx).Action(action).TargetKind(targetKind).TargetId(targetId).Limit(limit).Offset(offset).Execute()

What has been done in this organization, and by whom



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
	action := "action_example" // string | Exact action, e.g. hypervisor.cordon. (optional)
	targetKind := "targetKind_example" // string | hypervisor, hive, agent, firewall, estate or fleet. (optional)
	targetId := "targetId_example" // string | Narrow to one resource. (optional)
	limit := int64(789) // int64 | Default 50. (optional)
	offset := int64(789) // int64 |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.AuditEventsList(context.Background()).Action(action).TargetKind(targetKind).TargetId(targetId).Limit(limit).Offset(offset).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.AuditEventsList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuditEventsList`: ListAuditResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.AuditEventsList`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiAuditEventsListRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **action** | **string** | Exact action, e.g. hypervisor.cordon. | 
 **targetKind** | **string** | hypervisor, hive, agent, firewall, estate or fleet. | 
 **targetId** | **string** | Narrow to one resource. | 
 **limit** | **int64** | Default 50. | 
 **offset** | **int64** |  | 

### Return type

[**ListAuditResponseBody**](ListAuditResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetActivityList

> FleetActivityResponseBody FleetActivityList(ctx).Kind(kind).Status(status).HypervisorId(hypervisorId).Limit(limit).Offset(offset).Execute()

What the fleet's agents have been asked to do, and what came of it



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
	kind := "kind_example" // string | Exact task kind, e.g. fleet.datacenter.disarm. (optional)
	status := "status_example" // string | pending, running, succeeded, failed or cancelled. (optional)
	hypervisorId := "hypervisorId_example" // string | Narrow to one machine. (optional)
	limit := int64(789) // int64 | Default 50. (optional)
	offset := int64(789) // int64 |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetActivityList(context.Background()).Kind(kind).Status(status).HypervisorId(hypervisorId).Limit(limit).Offset(offset).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetActivityList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetActivityList`: FleetActivityResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetActivityList`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiFleetActivityListRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **kind** | **string** | Exact task kind, e.g. fleet.datacenter.disarm. | 
 **status** | **string** | pending, running, succeeded, failed or cancelled. | 
 **hypervisorId** | **string** | Narrow to one machine. | 
 **limit** | **int64** | Default 50. | 
 **offset** | **int64** |  | 

### Return type

[**FleetActivityResponseBody**](FleetActivityResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetAgentUpgrade

> AgentUpgradeResponseBody FleetAgentUpgrade(ctx).AgentUpgradeRequestBody(agentUpgradeRequestBody).Execute()

Upgrade every agent in a fleet



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
	agentUpgradeRequestBody := *openapiclient.NewAgentUpgradeRequestBody() // AgentUpgradeRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetAgentUpgrade(context.Background()).AgentUpgradeRequestBody(agentUpgradeRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetAgentUpgrade``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetAgentUpgrade`: AgentUpgradeResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetAgentUpgrade`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiFleetAgentUpgradeRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **agentUpgradeRequestBody** | [**AgentUpgradeRequestBody**](AgentUpgradeRequestBody.md) |  | 

### Return type

[**AgentUpgradeResponseBody**](AgentUpgradeResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetBeeAgentBinary

> string FleetBeeAgentBinary(ctx, os, arch).Execute()

Download the agent binary for one platform



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
	os := "os_example" // string | 
	arch := "arch_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetBeeAgentBinary(context.Background(), os, arch).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetBeeAgentBinary``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetBeeAgentBinary`: string
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetBeeAgentBinary`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**os** | **string** |  | 
**arch** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFleetBeeAgentBinaryRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

**string**

### Authorization

[AgentKeySecret](../README.md#AgentKeySecret), [AgentKeyAuth](../README.md#AgentKeyAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetBeeAgentDist

> AgentDistResponseBody FleetBeeAgentDist(ctx).Execute()

What agent binaries this control plane can serve



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
	resp, r, err := apiClient.FleetAPI.FleetBeeAgentDist(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetBeeAgentDist``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetBeeAgentDist`: AgentDistResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetBeeAgentDist`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiFleetBeeAgentDistRequest struct via the builder pattern


### Return type

[**AgentDistResponseBody**](AgentDistResponseBody.md)

### Authorization

[AgentKeySecret](../README.md#AgentKeySecret), [AgentKeyAuth](../README.md#AgentKeyAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


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


## FleetBeeMembership

> AgentMembershipResponseBody FleetBeeMembership(ctx).Execute()

Fetch the cluster composition this host's hive should have



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
	resp, r, err := apiClient.FleetAPI.FleetBeeMembership(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetBeeMembership``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetBeeMembership`: AgentMembershipResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetBeeMembership`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiFleetBeeMembershipRequest struct via the builder pattern


### Return type

[**AgentMembershipResponseBody**](AgentMembershipResponseBody.md)

### Authorization

[AgentKeySecret](../README.md#AgentKeySecret), [AgentKeyAuth](../README.md#AgentKeyAuth)

### HTTP request headers

- **Content-Type**: Not defined
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


## FleetHostsBootstrap

> BootstrapResponseBody FleetHostsBootstrap(ctx, hypervisorId).BootstrapRequestBody(bootstrapRequestBody).Execute()

Create a cluster on a standalone machine



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
	bootstrapRequestBody := *openapiclient.NewBootstrapRequestBody([]openapiclient.CorosyncLinkInput{*openapiclient.NewCorosyncLinkInput("Address_example", int64(123))}, "TargetHiveId_example") // BootstrapRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetHostsBootstrap(context.Background(), hypervisorId).BootstrapRequestBody(bootstrapRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetHostsBootstrap``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetHostsBootstrap`: BootstrapResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetHostsBootstrap`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**hypervisorId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFleetHostsBootstrapRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **bootstrapRequestBody** | [**BootstrapRequestBody**](BootstrapRequestBody.md) |  | 

### Return type

[**BootstrapResponseBody**](BootstrapResponseBody.md)

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


## FleetHypervisorsDeclare

> DeclareHostsResponseBody FleetHypervisorsDeclare(ctx).DeclareHostsRequestBody(declareHostsRequestBody).Execute()

Bring hosts into the fleet by address



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
	declareHostsRequestBody := *openapiclient.NewDeclareHostsRequestBody() // DeclareHostsRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetHypervisorsDeclare(context.Background()).DeclareHostsRequestBody(declareHostsRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetHypervisorsDeclare``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetHypervisorsDeclare`: DeclareHostsResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetHypervisorsDeclare`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiFleetHypervisorsDeclareRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **declareHostsRequestBody** | [**DeclareHostsRequestBody**](DeclareHostsRequestBody.md) |  | 

### Return type

[**DeclareHostsResponseBody**](DeclareHostsResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetHypervisorsDepart

> DepartResponseBody FleetHypervisorsDepart(ctx, id).DepartRequestBody(departRequestBody).Execute()

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
	departRequestBody := *openapiclient.NewDepartRequestBody("CredentialDisposition_example") // DepartRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetHypervisorsDepart(context.Background(), id).DepartRequestBody(departRequestBody).Execute()
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

 **departRequestBody** | [**DepartRequestBody**](DepartRequestBody.md) |  | 

### Return type

[**DepartResponseBody**](DepartResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetHypervisorsDisarm

> DisarmResponseBody FleetHypervisorsDisarm(ctx, id).DisarmRequestBody(disarmRequestBody).Execute()

Destroy the cluster secrets a departed machine still holds



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
	disarmRequestBody := *openapiclient.NewDisarmRequestBody() // DisarmRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetHypervisorsDisarm(context.Background(), id).DisarmRequestBody(disarmRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetHypervisorsDisarm``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetHypervisorsDisarm`: DisarmResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetHypervisorsDisarm`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFleetHypervisorsDisarmRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **disarmRequestBody** | [**DisarmRequestBody**](DisarmRequestBody.md) |  | 

### Return type

[**DisarmResponseBody**](DisarmResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
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


## FleetMaintenanceRunsList

> ListMaintenanceRunsResponseBody FleetMaintenanceRunsList(ctx).Limit(limit).Execute()

List recent maintenance runs



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
	limit := int64(789) // int64 |  (optional) (default to 20)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetMaintenanceRunsList(context.Background()).Limit(limit).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetMaintenanceRunsList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetMaintenanceRunsList`: ListMaintenanceRunsResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetMaintenanceRunsList`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiFleetMaintenanceRunsListRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **limit** | **int64** |  | [default to 20]

### Return type

[**ListMaintenanceRunsResponseBody**](ListMaintenanceRunsResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetMaintenanceRunsStop

> MaintenanceRunView FleetMaintenanceRunsStop(ctx, id).Execute()

Stop a maintenance run



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
	resp, r, err := apiClient.FleetAPI.FleetMaintenanceRunsStop(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetMaintenanceRunsStop``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetMaintenanceRunsStop`: MaintenanceRunView
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetMaintenanceRunsStop`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFleetMaintenanceRunsStopRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**MaintenanceRunView**](MaintenanceRunView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetMaintenanceSchedulesCreate

> MaintenanceScheduleView FleetMaintenanceSchedulesCreate(ctx).CreateMaintenanceScheduleRequestBody(createMaintenanceScheduleRequestBody).Execute()

Schedule a maintenance window



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
	createMaintenanceScheduleRequestBody := *openapiclient.NewCreateMaintenanceScheduleRequestBody(int64(123), false, "Kind_example", "Name_example", "Reboot_example") // CreateMaintenanceScheduleRequestBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetMaintenanceSchedulesCreate(context.Background()).CreateMaintenanceScheduleRequestBody(createMaintenanceScheduleRequestBody).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetMaintenanceSchedulesCreate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetMaintenanceSchedulesCreate`: MaintenanceScheduleView
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetMaintenanceSchedulesCreate`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiFleetMaintenanceSchedulesCreateRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **createMaintenanceScheduleRequestBody** | [**CreateMaintenanceScheduleRequestBody**](CreateMaintenanceScheduleRequestBody.md) |  | 

### Return type

[**MaintenanceScheduleView**](MaintenanceScheduleView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetMaintenanceSchedulesDelete

> FleetMaintenanceSchedulesDelete(ctx, id).Execute()

Delete a maintenance window



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
	r, err := apiClient.FleetAPI.FleetMaintenanceSchedulesDelete(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetMaintenanceSchedulesDelete``: %v\n", err)
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

Other parameters are passed through a pointer to a apiFleetMaintenanceSchedulesDeleteRequest struct via the builder pattern


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


## FleetMaintenanceSchedulesList

> ListMaintenanceSchedulesResponseBody FleetMaintenanceSchedulesList(ctx).Execute()

List maintenance windows

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
	resp, r, err := apiClient.FleetAPI.FleetMaintenanceSchedulesList(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetMaintenanceSchedulesList``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetMaintenanceSchedulesList`: ListMaintenanceSchedulesResponseBody
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetMaintenanceSchedulesList`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiFleetMaintenanceSchedulesListRequest struct via the builder pattern


### Return type

[**ListMaintenanceSchedulesResponseBody**](ListMaintenanceSchedulesResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## FleetMaintenanceSchedulesUpdate

> MaintenanceScheduleView FleetMaintenanceSchedulesUpdate(ctx, id).MaintenanceScheduleInput(maintenanceScheduleInput).Execute()

Change a maintenance window



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
	maintenanceScheduleInput := *openapiclient.NewMaintenanceScheduleInput(int64(123), false, "Kind_example", "Name_example", "Reboot_example") // MaintenanceScheduleInput | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.FleetAPI.FleetMaintenanceSchedulesUpdate(context.Background(), id).MaintenanceScheduleInput(maintenanceScheduleInput).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `FleetAPI.FleetMaintenanceSchedulesUpdate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `FleetMaintenanceSchedulesUpdate`: MaintenanceScheduleView
	fmt.Fprintf(os.Stdout, "Response from `FleetAPI.FleetMaintenanceSchedulesUpdate`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiFleetMaintenanceSchedulesUpdateRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **maintenanceScheduleInput** | [**MaintenanceScheduleInput**](MaintenanceScheduleInput.md) |  | 

### Return type

[**MaintenanceScheduleView**](MaintenanceScheduleView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
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

