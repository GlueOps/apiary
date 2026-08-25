# BindingView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Enabled** | **bool** |  | 
**HiveId** | Pointer to **string** |  | [optional] 
**Id** | **string** |  | 
**Priority** | **int64** | Orders rulesets against each other. Lower renders first, and it outranks the positions inside them. | 
**RulesetId** | **string** |  | 
**RulesetName** | **string** |  | 
**SwarmId** | Pointer to **string** |  | [optional] 

## Methods

### NewBindingView

`func NewBindingView(enabled bool, id string, priority int64, rulesetId string, rulesetName string, ) *BindingView`

NewBindingView instantiates a new BindingView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewBindingViewWithDefaults

`func NewBindingViewWithDefaults() *BindingView`

NewBindingViewWithDefaults instantiates a new BindingView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *BindingView) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *BindingView) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *BindingView) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *BindingView) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetEnabled

`func (o *BindingView) GetEnabled() bool`

GetEnabled returns the Enabled field if non-nil, zero value otherwise.

### GetEnabledOk

`func (o *BindingView) GetEnabledOk() (*bool, bool)`

GetEnabledOk returns a tuple with the Enabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnabled

`func (o *BindingView) SetEnabled(v bool)`

SetEnabled sets Enabled field to given value.


### GetHiveId

`func (o *BindingView) GetHiveId() string`

GetHiveId returns the HiveId field if non-nil, zero value otherwise.

### GetHiveIdOk

`func (o *BindingView) GetHiveIdOk() (*string, bool)`

GetHiveIdOk returns a tuple with the HiveId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHiveId

`func (o *BindingView) SetHiveId(v string)`

SetHiveId sets HiveId field to given value.

### HasHiveId

`func (o *BindingView) HasHiveId() bool`

HasHiveId returns a boolean if a field has been set.

### GetId

`func (o *BindingView) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *BindingView) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *BindingView) SetId(v string)`

SetId sets Id field to given value.


### GetPriority

`func (o *BindingView) GetPriority() int64`

GetPriority returns the Priority field if non-nil, zero value otherwise.

### GetPriorityOk

`func (o *BindingView) GetPriorityOk() (*int64, bool)`

GetPriorityOk returns a tuple with the Priority field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPriority

`func (o *BindingView) SetPriority(v int64)`

SetPriority sets Priority field to given value.


### GetRulesetId

`func (o *BindingView) GetRulesetId() string`

GetRulesetId returns the RulesetId field if non-nil, zero value otherwise.

### GetRulesetIdOk

`func (o *BindingView) GetRulesetIdOk() (*string, bool)`

GetRulesetIdOk returns a tuple with the RulesetId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRulesetId

`func (o *BindingView) SetRulesetId(v string)`

SetRulesetId sets RulesetId field to given value.


### GetRulesetName

`func (o *BindingView) GetRulesetName() string`

GetRulesetName returns the RulesetName field if non-nil, zero value otherwise.

### GetRulesetNameOk

`func (o *BindingView) GetRulesetNameOk() (*string, bool)`

GetRulesetNameOk returns a tuple with the RulesetName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRulesetName

`func (o *BindingView) SetRulesetName(v string)`

SetRulesetName sets RulesetName field to given value.


### GetSwarmId

`func (o *BindingView) GetSwarmId() string`

GetSwarmId returns the SwarmId field if non-nil, zero value otherwise.

### GetSwarmIdOk

`func (o *BindingView) GetSwarmIdOk() (*string, bool)`

GetSwarmIdOk returns a tuple with the SwarmId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSwarmId

`func (o *BindingView) SetSwarmId(v string)`

SetSwarmId sets SwarmId field to given value.

### HasSwarmId

`func (o *BindingView) HasSwarmId() bool`

HasSwarmId returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


