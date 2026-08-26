# AgentUpgradeResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Agents** | [**[]UpgradeCandidateView**](UpgradeCandidateView.md) |  | 
**Queued** | **int64** |  | 
**Version** | **string** | The version agents are being moved to, which is this control plane&#39;s own. | 

## Methods

### NewAgentUpgradeResponseBody

`func NewAgentUpgradeResponseBody(agents []UpgradeCandidateView, queued int64, version string, ) *AgentUpgradeResponseBody`

NewAgentUpgradeResponseBody instantiates a new AgentUpgradeResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAgentUpgradeResponseBodyWithDefaults

`func NewAgentUpgradeResponseBodyWithDefaults() *AgentUpgradeResponseBody`

NewAgentUpgradeResponseBodyWithDefaults instantiates a new AgentUpgradeResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *AgentUpgradeResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *AgentUpgradeResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *AgentUpgradeResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *AgentUpgradeResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetAgents

`func (o *AgentUpgradeResponseBody) GetAgents() []UpgradeCandidateView`

GetAgents returns the Agents field if non-nil, zero value otherwise.

### GetAgentsOk

`func (o *AgentUpgradeResponseBody) GetAgentsOk() (*[]UpgradeCandidateView, bool)`

GetAgentsOk returns a tuple with the Agents field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgents

`func (o *AgentUpgradeResponseBody) SetAgents(v []UpgradeCandidateView)`

SetAgents sets Agents field to given value.


### SetAgentsNil

`func (o *AgentUpgradeResponseBody) SetAgentsNil(b bool)`

 SetAgentsNil sets the value for Agents to be an explicit nil

### UnsetAgents
`func (o *AgentUpgradeResponseBody) UnsetAgents()`

UnsetAgents ensures that no value is present for Agents, not even an explicit nil
### GetQueued

`func (o *AgentUpgradeResponseBody) GetQueued() int64`

GetQueued returns the Queued field if non-nil, zero value otherwise.

### GetQueuedOk

`func (o *AgentUpgradeResponseBody) GetQueuedOk() (*int64, bool)`

GetQueuedOk returns a tuple with the Queued field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetQueued

`func (o *AgentUpgradeResponseBody) SetQueued(v int64)`

SetQueued sets Queued field to given value.


### GetVersion

`func (o *AgentUpgradeResponseBody) GetVersion() string`

GetVersion returns the Version field if non-nil, zero value otherwise.

### GetVersionOk

`func (o *AgentUpgradeResponseBody) GetVersionOk() (*string, bool)`

GetVersionOk returns a tuple with the Version field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVersion

`func (o *AgentUpgradeResponseBody) SetVersion(v string)`

SetVersion sets Version field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


