# FleetApi

All URIs are relative to *http://127.0.0.1:8080*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**fleetBeeAgentBinary**](FleetApi.md#fleetbeeagentbinary) | **GET** /api/v1/agent/dist/{os}/{arch} | Download the agent binary for one platform |
| [**fleetBeeAgentDist**](FleetApi.md#fleetbeeagentdist) | **GET** /api/v1/agent/dist | What agent binaries this control plane can serve |
| [**fleetBeeAppendLogs**](FleetApi.md#fleetbeeappendlogs) | **POST** /api/v1/agent/tasks/{id}/logs | Stream task output back to the control plane |
| [**fleetBeeEnrol**](FleetApi.md#fleetbeeenrol) | **POST** /api/v1/agent/enrol | Enrol a fleet agent into an organization |
| [**fleetBeeFirewall**](FleetApi.md#fleetbeefirewall) | **GET** /api/v1/agent/firewall | Fetch the ruleset this host should enforce |
| [**fleetBeeHeartbeat**](FleetApi.md#fleetbeeheartbeat) | **POST** /api/v1/agent/heartbeat | Report liveness and collect work |
| [**fleetBeeReportTask**](FleetApi.md#fleetbeereporttask) | **POST** /api/v1/agent/tasks/{id}/status | Report progress or the outcome of a task |
| [**fleetBeesDelete**](FleetApi.md#fleetbeesdelete) | **DELETE** /api/v1/fleet/bees/{id} | Remove a fleet agent |
| [**fleetBeesList**](FleetApi.md#fleetbeeslist) | **GET** /api/v1/fleet/bees | List fleet agents |
| [**fleetEnrolmentTokensMint**](FleetApi.md#fleetenrolmenttokensmint) | **POST** /api/v1/fleet/enrolment-tokens | Mint an enrolment token |
| [**fleetHivesCreate**](FleetApi.md#fleethivescreate) | **POST** /api/v1/fleet/hives | Create a hive |
| [**fleetHivesDelete**](FleetApi.md#fleethivesdelete) | **DELETE** /api/v1/fleet/hives/{id} | Delete a hive |
| [**fleetHivesList**](FleetApi.md#fleethiveslist) | **GET** /api/v1/fleet/hives | List hives |
| [**fleetHivesSync**](FleetApi.md#fleethivessync) | **POST** /api/v1/fleet/hives/{id}/sync | Refresh a hive\&#39;s capacity observations now |
| [**fleetHivesUpdate**](FleetApi.md#fleethivesupdate) | **PATCH** /api/v1/fleet/hives/{id} | Update a hive\&#39;s configuration |
| [**fleetHostsArrive**](FleetApi.md#fleethostsarrive) | **POST** /api/v1/fleet/fleetless/{hypervisor_id}/arrive | Add a standalone machine to a hive |
| [**fleetHostsFleetless**](FleetApi.md#fleethostsfleetless) | **GET** /api/v1/fleet/fleetless | List machines belonging to no hive |
| [**fleetHypervisorsCordon**](FleetApi.md#fleethypervisorscordon) | **POST** /api/v1/fleet/hypervisors/{id}/cordon | Cordon or uncordon a hypervisor |
| [**fleetHypervisorsDepart**](FleetApi.md#fleethypervisorsdepart) | **POST** /api/v1/fleet/hypervisors/{id}/depart | Remove a hypervisor from its hive |
| [**fleetHypervisorsGet**](FleetApi.md#fleethypervisorsget) | **GET** /api/v1/fleet/hypervisors/{id} | Get one hypervisor |
| [**fleetHypervisorsList**](FleetApi.md#fleethypervisorslist) | **GET** /api/v1/fleet/hypervisors | List machines with their three status axes |
| [**fleetHypervisorsReprovision**](FleetApi.md#fleethypervisorsreprovision) | **POST** /api/v1/fleet/hypervisors/{id}/reprovision | Redrive a machine\&#39;s onboarding |
| [**fleetTasksCreate**](FleetApi.md#fleettaskscreate) | **POST** /api/v1/fleet/tasks | Queue work for a bee |
| [**fleetTasksLogs**](FleetApi.md#fleettaskslogs) | **GET** /api/v1/fleet/tasks/{id}/logs | Read a task\&#39;s streamed output |
| [**sshKeysList**](FleetApi.md#sshkeyslist) | **GET** /api/v1/ssh-keys | List fleet SSH keys |
| [**sshKeysMint**](FleetApi.md#sshkeysmint) | **POST** /api/v1/ssh-keys | Mint a fleet SSH key |
| [**sshKeysRevoke**](FleetApi.md#sshkeysrevoke) | **DELETE** /api/v1/ssh-keys/{id} | Revoke a fleet SSH key |



## fleetBeeAgentBinary

> string fleetBeeAgentBinary(os, arch)

Download the agent binary for one platform

Authenticated, even though the same bytes sit in the SPA\&#39;s embedded tree. Serving them as a static asset would hand the fleet\&#39;s agent binary to anyone who could reach the login page.  Verify the response against the sha256 from /agent/dist before running it.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetBeeAgentBinaryRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: AgentKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: AgentKeyAuth
    apiKey: "YOUR API KEY",
  });
  const api = new FleetApi(config);

  const body = {
    // string
    os: os_example,
    // string
    arch: arch_example,
  } satisfies FleetBeeAgentBinaryRequest;

  try {
    const data = await api.fleetBeeAgentBinary(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **os** | `string` |  | [Defaults to `undefined`] |
| **arch** | `string` |  | [Defaults to `undefined`] |

### Return type

**string**

### Authorization

[AgentKeySecret](../README.md#AgentKeySecret), [AgentKeyAuth](../README.md#AgentKeyAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  * Content-Disposition -  <br>  * Content-Type -  <br>  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetBeeAgentDist

> AgentDistResponseBody fleetBeeAgentDist()

What agent binaries this control plane can serve

A hypervisor in a managed estate has no route to the internet, so the control plane is the only host it can fetch a new agent from. Without this an upgrade means copying a binary to every machine by hand, which is how a fleet ends up running four different versions.  An EMPTY build list is not a failure. A server compiled without agent binaries is an ordinary development build, and an agent reads it as nothing to do.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetBeeAgentDistRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: AgentKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: AgentKeyAuth
    apiKey: "YOUR API KEY",
  });
  const api = new FleetApi(config);

  try {
    const data = await api.fleetBeeAgentDist();
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**AgentDistResponseBody**](AgentDistResponseBody.md)

### Authorization

[AgentKeySecret](../README.md#AgentKeySecret), [AgentKeyAuth](../README.md#AgentKeyAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetBeeAppendLogs

> LogsResponseBody fleetBeeAppendLogs(id, logsRequestBody)

Stream task output back to the control plane

So a fleet operation can be watched without SSH-ing to a hypervisor mid-rotation, which during an evacuation is often the one thing you must not do. Lines carry a per-task sequence assigned by the bee; resending a batch is a no-op.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetBeeAppendLogsRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: AgentKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: AgentKeyAuth
    apiKey: "YOUR API KEY",
  });
  const api = new FleetApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
    // LogsRequestBody
    logsRequestBody: ...,
  } satisfies FleetBeeAppendLogsRequest;

  try {
    const data = await api.fleetBeeAppendLogs(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **id** | `string` |  | [Defaults to `undefined`] |
| **logsRequestBody** | [LogsRequestBody](LogsRequestBody.md) |  | |

### Return type

[**LogsResponseBody**](LogsResponseBody.md)

### Authorization

[AgentKeySecret](../README.md#AgentKeySecret), [AgentKeyAuth](../README.md#AgentKeyAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetBeeEnrol

> EnrolResponseBody fleetBeeEnrol(enrolRequestBody)

Enrol a fleet agent into an organization

Redeems an enrolment token and returns this host\&#39;s agent-scoped key and secret. The host is admitted into an ORGANIZATION, not a fleet — joining one is a task, so a new host and a host migrating between fleets follow the same path. Idempotent on machine_id: a rebuilt hypervisor re-enrols onto the same MACHINE record with a fresh credential rather than leaving a permanently-offline ghost. A machine a capacity sync has already discovered in a cluster is adopted rather than duplicated, matched by the node name its hostname implies.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetBeeEnrolRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const api = new FleetApi();

  const body = {
    // EnrolRequestBody
    enrolRequestBody: ...,
  } satisfies FleetBeeEnrolRequest;

  try {
    const data = await api.fleetBeeEnrol(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **enrolRequestBody** | [EnrolRequestBody](EnrolRequestBody.md) |  | |

### Return type

[**EnrolResponseBody**](EnrolResponseBody.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **201** | Created |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetBeeFirewall

> AgentFirewallResponseBody fleetBeeFirewall()

Fetch the ruleset this host should enforce

The complete nft ruleset, not a patch — it begins with a flush, so a host that missed several revisions converges on this one rather than replaying a history it never saw.  An EMPTY response means hold what you have. It is returned for a machine in no fleet and for a fleet that has published nothing, and neither is an empty ruleset: flushing there would strip a host of its management access at the moment it is least reachable. Separate from the heartbeat because the body can be tens of kilobytes and would otherwise ride on every poll from every host to say \&#39;unchanged\&#39;.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetBeeFirewallRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: AgentKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: AgentKeyAuth
    apiKey: "YOUR API KEY",
  });
  const api = new FleetApi(config);

  try {
    const data = await api.fleetBeeFirewall();
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**AgentFirewallResponseBody**](AgentFirewallResponseBody.md)

### Authorization

[AgentKeySecret](../README.md#AgentKeySecret), [AgentKeyAuth](../README.md#AgentKeyAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetBeeHeartbeat

> HeartbeatResponseBody fleetBeeHeartbeat(heartbeatRequestBody)

Report liveness and collect work

One request carries both, deliberately: two endpoints would give a bee two ways to be \&#39;seen\&#39;, and a bee parked on a long poll would have to heartbeat separately to avoid looking dead. Declare unavailable_for_seconds BEFORE doing anything that interrupts the host — a reboot is indistinguishable from a failure unless the host says in advance that it is coming back.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetBeeHeartbeatRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: AgentKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: AgentKeyAuth
    apiKey: "YOUR API KEY",
  });
  const api = new FleetApi(config);

  const body = {
    // HeartbeatRequestBody
    heartbeatRequestBody: ...,
  } satisfies FleetBeeHeartbeatRequest;

  try {
    const data = await api.fleetBeeHeartbeat(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **heartbeatRequestBody** | [HeartbeatRequestBody](HeartbeatRequestBody.md) |  | |

### Return type

[**HeartbeatResponseBody**](HeartbeatResponseBody.md)

### Authorization

[AgentKeySecret](../README.md#AgentKeySecret), [AgentKeyAuth](../README.md#AgentKeyAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetBeeReportTask

> ReportResponseBody fleetBeeReportTask(id, reportRequestBody)

Report progress or the outcome of a task

Terminal reports are idempotent. A bee that finishes and dies before its report is acknowledged will report again on restart, and rejecting the duplicate would strand the task against a host that considers it done.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetBeeReportTaskRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: AgentKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: AgentKeyAuth
    apiKey: "YOUR API KEY",
  });
  const api = new FleetApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
    // ReportRequestBody
    reportRequestBody: ...,
  } satisfies FleetBeeReportTaskRequest;

  try {
    const data = await api.fleetBeeReportTask(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **id** | `string` |  | [Defaults to `undefined`] |
| **reportRequestBody** | [ReportRequestBody](ReportRequestBody.md) |  | |

### Return type

[**ReportResponseBody**](ReportResponseBody.md)

### Authorization

[AgentKeySecret](../README.md#AgentKeySecret), [AgentKeyAuth](../README.md#AgentKeyAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetBeesDelete

> DeleteBeeResponseBody fleetBeesDelete(id)

Remove a fleet agent

Revokes this agent\&#39;s credential and then deletes its record. The MACHINE survives as agentless — its cluster membership, address and onboarding history are on the hypervisor record, not on the agent — so this is \&#39;remove the agent\&#39;, never \&#39;forget the box\&#39;.  The credential is revoked FIRST and the revocation is verified before anything is deleted. It lives in the control database while the agent record lives in the tenant database, so no transaction covers both: deleting first and failing to revoke would leave a live agent key with nothing pointing at it, and nothing lists agent-scoped keys.  Refused with 409 while an onboarding worker holds a lease on this machine — it is installing and enrolling right now, and would either race the delete or resurrect the record behind it.  The agent process on the host is NOT stopped. It keeps running and starts failing authentication, which is the intended outcome: the credential is what this revokes.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetBeesDeleteRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new FleetApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
  } satisfies FleetBeesDeleteRequest;

  try {
    const data = await api.fleetBeesDelete(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **id** | `string` |  | [Defaults to `undefined`] |

### Return type

[**DeleteBeeResponseBody**](DeleteBeeResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetBeesList

> ListBeesResponseBody fleetBeesList(hiveId)

List fleet agents

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetBeesListRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new FleetApi(config);

  const body = {
    // string (optional)
    hiveId: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
  } satisfies FleetBeesListRequest;

  try {
    const data = await api.fleetBeesList(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **hiveId** | `string` |  | [Optional] [Defaults to `undefined`] |

### Return type

[**ListBeesResponseBody**](ListBeesResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetEnrolmentTokensMint

> MintTokenResponseBody fleetEnrolmentTokensMint(mintTokenRequestBody)

Mint an enrolment token

Admits a host into the ORGANIZATION, not into a cluster. Joining one is a task, so a brand-new host starts in the same fleetless state a migrating host passes through — which means an agent can be provisioned before any cluster exists.  Impersonated sessions are refused: an operator acting as somebody else must not be able to mint something that admits a machine to the fleet.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetEnrolmentTokensMintRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new FleetApi(config);

  const body = {
    // MintTokenRequestBody
    mintTokenRequestBody: ...,
  } satisfies FleetEnrolmentTokensMintRequest;

  try {
    const data = await api.fleetEnrolmentTokensMint(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **mintTokenRequestBody** | [MintTokenRequestBody](MintTokenRequestBody.md) |  | |

### Return type

[**MintTokenResponseBody**](MintTokenResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **201** | Created |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetHivesCreate

> HiveView fleetHivesCreate(createHiveRequestBody)

Create a hive

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetHivesCreateRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new FleetApi(config);

  const body = {
    // CreateHiveRequestBody
    createHiveRequestBody: ...,
  } satisfies FleetHivesCreateRequest;

  try {
    const data = await api.fleetHivesCreate(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **createHiveRequestBody** | [CreateHiveRequestBody](CreateHiveRequestBody.md) |  | |

### Return type

[**HiveView**](HiveView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **201** | Created |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetHivesDelete

> fleetHivesDelete(id)

Delete a hive

Refuses while the cluster still holds hypervisors. A hypervisor row is the durable record of a physical machine — its machine id, its onboarding history, the agent bound to it — so deleting the cluster out from under one would either destroy that or leave it pointing at a cluster that no longer exists.  Depart the nodes first; each one becomes fleetless and keeps its identity, and the cluster can then be removed.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetHivesDeleteRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new FleetApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
  } satisfies FleetHivesDeleteRequest;

  try {
    const data = await api.fleetHivesDelete(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **id** | `string` |  | [Defaults to `undefined`] |

### Return type

`void` (Empty response body)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **204** | No Content |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetHivesList

> ListHivesResponseBody fleetHivesList()

List hives

Every hive in the caller\&#39;s organization, with the node count and whether a capacity sync can run for it. A hive with no API URL or credential is never synced, so every node in it reads as never-observed and every departure refuses.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetHivesListRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new FleetApi(config);

  try {
    const data = await api.fleetHivesList();
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**ListHivesResponseBody**](ListHivesResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetHivesSync

> SyncHiveResponseBody fleetHivesSync(id)

Refresh a hive\&#39;s capacity observations now

The same job the schedule runs, enqueued immediately. Most often wanted after a departure refused with drain_unknown, which means the last observation is too stale to say whether a node is empty.  The job is unique per hive while queued or running, so asking twice gets the sync already on its way rather than a second one.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetHivesSyncRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new FleetApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
  } satisfies FleetHivesSyncRequest;

  try {
    const data = await api.fleetHivesSync(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **id** | `string` |  | [Defaults to `undefined`] |

### Return type

[**SyncHiveResponseBody**](SyncHiveResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetHivesUpdate

> HiveDetailView fleetHivesUpdate(id, updateHiveRequestBody)

Update a hive\&#39;s configuration

Only the fields present are changed. Setting an API URL and a credential is what makes a hive syncable.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetHivesUpdateRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new FleetApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
    // UpdateHiveRequestBody
    updateHiveRequestBody: ...,
  } satisfies FleetHivesUpdateRequest;

  try {
    const data = await api.fleetHivesUpdate(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **id** | `string` |  | [Defaults to `undefined`] |
| **updateHiveRequestBody** | [UpdateHiveRequestBody](UpdateHiveRequestBody.md) |  | |

### Return type

[**HiveDetailView**](HiveDetailView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetHostsArrive

> ArriveResponseBody fleetHostsArrive(hypervisorId, arriveRequestBody)

Add a standalone machine to a hive

Queues the arrival. Refused with 409 and reason codes when the host already belongs to a hive, still carries guests, or the target has no nodes to join to.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetHostsArriveRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new FleetApi(config);

  const body = {
    // string
    hypervisorId: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
    // ArriveRequestBody
    arriveRequestBody: ...,
  } satisfies FleetHostsArriveRequest;

  try {
    const data = await api.fleetHostsArrive(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **hypervisorId** | `string` |  | [Defaults to `undefined`] |
| **arriveRequestBody** | [ArriveRequestBody](ArriveRequestBody.md) |  | |

### Return type

[**ArriveResponseBody**](ArriveResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetHostsFleetless

> ListFleetlessResponseBody fleetHostsFleetless()

List machines belonging to no hive

Machines with no hive: onboarded but not yet joined to a cluster, or departed from one. Every migration passes through this state and a failed arrival leaves a machine sitting in it.  Each entry carries its onboarding state, because a machine that has not finished onboarding has no agent and an arrival pointed at it will fail at the SSH.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetHostsFleetlessRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new FleetApi(config);

  try {
    const data = await api.fleetHostsFleetless();
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**ListFleetlessResponseBody**](ListFleetlessResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetHypervisorsCordon

> HypervisorView fleetHypervisorsCordon(id, cordonRequestBody)

Cordon or uncordon a hypervisor

Cordoning withdraws the node from NEW placements. It does not evict anything: draining is a separate act, and conflating them would mean the only way to stop new work was to move existing work. A cordon is the first step of every departure.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetHypervisorsCordonRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new FleetApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
    // CordonRequestBody
    cordonRequestBody: ...,
  } satisfies FleetHypervisorsCordonRequest;

  try {
    const data = await api.fleetHypervisorsCordon(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **id** | `string` |  | [Defaults to `undefined`] |
| **cordonRequestBody** | [CordonRequestBody](CordonRequestBody.md) |  | |

### Return type

[**HypervisorView**](HypervisorView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetHypervisorsDepart

> DepartResponseBody fleetHypervisorsDepart(id)

Remove a hypervisor from its hive

Queues the departure. The host is left standalone and belonging to no fleet; it does NOT join anything. A migration is a departure and then an arrival, and the fleetless state between them is real rather than a window to hurry through.  Refused with 409 and a list of reason codes when the host is not drained, the hive is too small, or the capacity observation is too stale to decide. The same gate runs again inside the job, against state read in one transaction.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetHypervisorsDepartRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new FleetApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
  } satisfies FleetHypervisorsDepartRequest;

  try {
    const data = await api.fleetHypervisorsDepart(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **id** | `string` |  | [Defaults to `undefined`] |

### Return type

[**DepartResponseBody**](DepartResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetHypervisorsGet

> HypervisorView fleetHypervisorsGet(id)

Get one hypervisor

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetHypervisorsGetRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new FleetApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
  } satisfies FleetHypervisorsGetRequest;

  try {
    const data = await api.fleetHypervisorsGet(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **id** | `string` |  | [Defaults to `undefined`] |

### Return type

[**HypervisorView**](HypervisorView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetHypervisorsList

> ListHypervisorsResponseBody fleetHypervisorsList(hiveId)

List machines with their three status axes

Every MACHINE this organization owns, in a cluster or not. Each carries three independent statuses — provisioning_state (has the agent been installed), schedulable plus drain_state (may new work land here), and bee_status (is it answering right now) — because a machine can be provisioned AND cordoned AND offline at the same instant, and one badge cannot say that.  When a node cannot leave its hive, the reasons why are computed by the same gate the departure job runs, so a client never has to reimplement it. A machine in no hive reports not_a_member and appears in /fleet/fleetless as well.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetHypervisorsListRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new FleetApi(config);

  const body = {
    // string | Restrict to one hive. (optional)
    hiveId: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
  } satisfies FleetHypervisorsListRequest;

  try {
    const data = await api.fleetHypervisorsList(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **hiveId** | `string` | Restrict to one hive. | [Optional] [Defaults to `undefined`] |

### Return type

[**ListHypervisorsResponseBody**](ListHypervisorsResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetHypervisorsReprovision

> HypervisorView fleetHypervisorsReprovision(id)

Redrive a machine\&#39;s onboarding

Sets provisioning_state back to &#x60;pending&#x60; and clears the attempt count. This RECORDS INTENT; it does not queue a job. Onboarding is self-scheduling — a ticker claims pending rows under a lease — so there is nothing to enqueue and nothing that has to be running for the request to be meaningful.  Allowed from &#x60;failed&#x60; and from &#x60;provisioned&#x60; — the latter is \&#39;revive\&#39;, and it is safe because onboarding skips enrolment for a host that is already heartbeating, so a re-run does not invalidate a working agent\&#39;s credential.  Refused with 409 while a lease is live: redriving a row a worker holds puts two workers on one host, both installing keys and reloading sshd. An EXPIRED lease is redrivable — that is the recovery path for a worker killed mid-onboard.  provisioning_error is deliberately NOT cleared. An operator watching a machine retry wants to see what it is retrying past, and it is overwritten by the next real failure.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetHypervisorsReprovisionRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new FleetApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
  } satisfies FleetHypervisorsReprovisionRequest;

  try {
    const data = await api.fleetHypervisorsReprovision(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **id** | `string` |  | [Defaults to `undefined`] |

### Return type

[**HypervisorView**](HypervisorView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetTasksCreate

> CreateTaskResponseBody fleetTasksCreate(createTaskRequestBody)

Queue work for a bee

Refused with 409 while that bee already has an outstanding task. At most one at a time is what makes \&#39;what is this hypervisor doing\&#39; answerable from a single row.  fleet.datacenter.evict is issued to a node that will REMAIN, naming the one to remove — a node cannot remove itself from a cluster it is leaving.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetTasksCreateRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new FleetApi(config);

  const body = {
    // CreateTaskRequestBody
    createTaskRequestBody: ...,
  } satisfies FleetTasksCreateRequest;

  try {
    const data = await api.fleetTasksCreate(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **createTaskRequestBody** | [CreateTaskRequestBody](CreateTaskRequestBody.md) |  | |

### Return type

[**CreateTaskResponseBody**](CreateTaskResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **201** | Created |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## fleetTasksLogs

> TaskLogsResponseBody fleetTasksLogs(id, since)

Read a task\&#39;s streamed output

Pass &#x60;since&#x60; with the highest seq already held to tail cheaply.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { FleetTasksLogsRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new FleetApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
    // number | Return lines with seq greater than this. The UI polls with the highest seq it has, so a tail is one small query rather than a re-read. (optional)
    since: 789,
  } satisfies FleetTasksLogsRequest;

  try {
    const data = await api.fleetTasksLogs(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **id** | `string` |  | [Defaults to `undefined`] |
| **since** | `number` | Return lines with seq greater than this. The UI polls with the highest seq it has, so a tail is one small query rather than a re-read. | [Optional] [Defaults to `undefined`] |

### Return type

[**TaskLogsResponseBody**](TaskLogsResponseBody.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## sshKeysList

> ListResponseBody3 sshKeysList()

List fleet SSH keys

Public halves and fingerprints. The private half is never returned by any endpoint — it is unsealed only inside a River worker, so a fleet-wide credential never passes through a response, a log or an error tracker.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { SshKeysListRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new FleetApi(config);

  try {
    const data = await api.sshKeysList();
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**ListResponseBody3**](ListResponseBody3.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## sshKeysMint

> KeyView sshKeysMint(mintRequestBody)

Mint a fleet SSH key

Creates an Ed25519 keypair. The private half is sealed with this organization\&#39;s own key and never leaves the control plane; the public half is returned in authorized_keys form for whitelisting on hypervisors. Requires admin and refuses impersonated sessions: this key opens root on every host that trusts it, and an operator acting as a customer must not be able to mint one they could later use.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { SshKeysMintRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new FleetApi(config);

  const body = {
    // MintRequestBody
    mintRequestBody: ...,
  } satisfies SshKeysMintRequest;

  try {
    const data = await api.sshKeysMint(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **mintRequestBody** | [MintRequestBody](MintRequestBody.md) |  | |

### Return type

[**KeyView**](KeyView.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **201** | Created |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## sshKeysRevoke

> RevokeResponseBody1 sshKeysRevoke(id)

Revoke a fleet SSH key

Marks the key unusable. It does NOT remove the public half from any hypervisor — that happens when the agents next converge, so a revoked key stays whitelisted on a host that is offline. The row is kept rather than deleted, so a fingerprint found in an authorized_keys file later can still be attributed to something.

### Example

```ts
import {
  Configuration,
  FleetApi,
} from '@glueops/apiary-sdk';
import type { SshKeysRevokeRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new FleetApi(config);

  const body = {
    // string
    id: 38400000-8cf0-11bd-b23e-10b96e4ef00d,
  } satisfies SshKeysRevokeRequest;

  try {
    const data = await api.sshKeysRevoke(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **id** | `string` |  | [Defaults to `undefined`] |

### Return type

[**RevokeResponseBody1**](RevokeResponseBody1.md)

### Authorization

[OrgKeySecret](../README.md#OrgKeySecret), [OrgKeyAuth](../README.md#OrgKeyAuth), [ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`, `application/problem+json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK |  -  |
| **0** | Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

