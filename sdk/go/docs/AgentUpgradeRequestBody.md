# AgentUpgradeRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Force** | Pointer to **bool** | Queue agents that already report the target version. For repairing a damaged binary, not for routine use: it restarts every agent in scope. | [optional] 
**HiveId** | Pointer to **string** | Limit to one fleet. Omit to cover every agent in the organization, including hosts in no fleet — which are the ones most likely to be forgotten. | [optional] 

## Methods

### NewAgentUpgradeRequestBody

`func NewAgentUpgradeRequestBody() *AgentUpgradeRequestBody`

NewAgentUpgradeRequestBody instantiates a new AgentUpgradeRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAgentUpgradeRequestBodyWithDefaults

`func NewAgentUpgradeRequestBodyWithDefaults() *AgentUpgradeRequestBody`

NewAgentUpgradeRequestBodyWithDefaults instantiates a new AgentUpgradeRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *AgentUpgradeRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *AgentUpgradeRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *AgentUpgradeRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *AgentUpgradeRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetForce

`func (o *AgentUpgradeRequestBody) GetForce() bool`

GetForce returns the Force field if non-nil, zero value otherwise.

### GetForceOk

`func (o *AgentUpgradeRequestBody) GetForceOk() (*bool, bool)`

GetForceOk returns a tuple with the Force field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetForce

`func (o *AgentUpgradeRequestBody) SetForce(v bool)`

SetForce sets Force field to given value.

### HasForce

`func (o *AgentUpgradeRequestBody) HasForce() bool`

HasForce returns a boolean if a field has been set.

### GetHiveId

`func (o *AgentUpgradeRequestBody) GetHiveId() string`

GetHiveId returns the HiveId field if non-nil, zero value otherwise.

### GetHiveIdOk

`func (o *AgentUpgradeRequestBody) GetHiveIdOk() (*string, bool)`

GetHiveIdOk returns a tuple with the HiveId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHiveId

`func (o *AgentUpgradeRequestBody) SetHiveId(v string)`

SetHiveId sets HiveId field to given value.

### HasHiveId

`func (o *AgentUpgradeRequestBody) HasHiveId() bool`

HasHiveId returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


