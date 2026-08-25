# CreateBindingRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Enabled** | **bool** |  | [default to true]
**HiveId** | Pointer to **string** | Omit to bind ORG-WIDE: every host in the organization, whatever fleet it is in. That is what covers a host between fleets, which has no hive-scoped bindings at all. | [optional] 
**Priority** | Pointer to **int64** | Lower renders first, and outranks rule positions inside the ruleset. Ties break by scope, most specific first. | [optional] [default to 100]
**RulesetId** | **string** |  | 
**SwarmId** | Pointer to **string** | Narrow the binding to one group of hypervisors inside the fleet. Requires hive_id: swarm names are unique only within a fleet, so a swarm alone does not identify a scope. | [optional] 

## Methods

### NewCreateBindingRequestBody

`func NewCreateBindingRequestBody(enabled bool, rulesetId string, ) *CreateBindingRequestBody`

NewCreateBindingRequestBody instantiates a new CreateBindingRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCreateBindingRequestBodyWithDefaults

`func NewCreateBindingRequestBodyWithDefaults() *CreateBindingRequestBody`

NewCreateBindingRequestBodyWithDefaults instantiates a new CreateBindingRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *CreateBindingRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *CreateBindingRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *CreateBindingRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *CreateBindingRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetEnabled

`func (o *CreateBindingRequestBody) GetEnabled() bool`

GetEnabled returns the Enabled field if non-nil, zero value otherwise.

### GetEnabledOk

`func (o *CreateBindingRequestBody) GetEnabledOk() (*bool, bool)`

GetEnabledOk returns a tuple with the Enabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnabled

`func (o *CreateBindingRequestBody) SetEnabled(v bool)`

SetEnabled sets Enabled field to given value.


### GetHiveId

`func (o *CreateBindingRequestBody) GetHiveId() string`

GetHiveId returns the HiveId field if non-nil, zero value otherwise.

### GetHiveIdOk

`func (o *CreateBindingRequestBody) GetHiveIdOk() (*string, bool)`

GetHiveIdOk returns a tuple with the HiveId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHiveId

`func (o *CreateBindingRequestBody) SetHiveId(v string)`

SetHiveId sets HiveId field to given value.

### HasHiveId

`func (o *CreateBindingRequestBody) HasHiveId() bool`

HasHiveId returns a boolean if a field has been set.

### GetPriority

`func (o *CreateBindingRequestBody) GetPriority() int64`

GetPriority returns the Priority field if non-nil, zero value otherwise.

### GetPriorityOk

`func (o *CreateBindingRequestBody) GetPriorityOk() (*int64, bool)`

GetPriorityOk returns a tuple with the Priority field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPriority

`func (o *CreateBindingRequestBody) SetPriority(v int64)`

SetPriority sets Priority field to given value.

### HasPriority

`func (o *CreateBindingRequestBody) HasPriority() bool`

HasPriority returns a boolean if a field has been set.

### GetRulesetId

`func (o *CreateBindingRequestBody) GetRulesetId() string`

GetRulesetId returns the RulesetId field if non-nil, zero value otherwise.

### GetRulesetIdOk

`func (o *CreateBindingRequestBody) GetRulesetIdOk() (*string, bool)`

GetRulesetIdOk returns a tuple with the RulesetId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRulesetId

`func (o *CreateBindingRequestBody) SetRulesetId(v string)`

SetRulesetId sets RulesetId field to given value.


### GetSwarmId

`func (o *CreateBindingRequestBody) GetSwarmId() string`

GetSwarmId returns the SwarmId field if non-nil, zero value otherwise.

### GetSwarmIdOk

`func (o *CreateBindingRequestBody) GetSwarmIdOk() (*string, bool)`

GetSwarmIdOk returns a tuple with the SwarmId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSwarmId

`func (o *CreateBindingRequestBody) SetSwarmId(v string)`

SetSwarmId sets SwarmId field to given value.

### HasSwarmId

`func (o *CreateBindingRequestBody) HasSwarmId() bool`

HasSwarmId returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


