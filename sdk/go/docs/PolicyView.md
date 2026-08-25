# PolicyView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Config** | [**PolicyConfigView**](PolicyConfigView.md) |  | 
**EffectiveConfig** | [**PolicyConfigView**](PolicyConfigView.md) |  | 
**EffectiveEnabled** | **bool** |  | 
**Kind** | **string** |  | 
**State** | **string** | inherit means this level expresses no opinion. For an ORGANIZATION that hands the decision to the platform. For the PLATFORM there is no level above, so it means the policy is simply not enforced — effective_enabled can never differ from the platform&#39;s own setting, because the effective view is computed with no organization. | 

## Methods

### NewPolicyView

`func NewPolicyView(config PolicyConfigView, effectiveConfig PolicyConfigView, effectiveEnabled bool, kind string, state string, ) *PolicyView`

NewPolicyView instantiates a new PolicyView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPolicyViewWithDefaults

`func NewPolicyViewWithDefaults() *PolicyView`

NewPolicyViewWithDefaults instantiates a new PolicyView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *PolicyView) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *PolicyView) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *PolicyView) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *PolicyView) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetConfig

`func (o *PolicyView) GetConfig() PolicyConfigView`

GetConfig returns the Config field if non-nil, zero value otherwise.

### GetConfigOk

`func (o *PolicyView) GetConfigOk() (*PolicyConfigView, bool)`

GetConfigOk returns a tuple with the Config field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConfig

`func (o *PolicyView) SetConfig(v PolicyConfigView)`

SetConfig sets Config field to given value.


### GetEffectiveConfig

`func (o *PolicyView) GetEffectiveConfig() PolicyConfigView`

GetEffectiveConfig returns the EffectiveConfig field if non-nil, zero value otherwise.

### GetEffectiveConfigOk

`func (o *PolicyView) GetEffectiveConfigOk() (*PolicyConfigView, bool)`

GetEffectiveConfigOk returns a tuple with the EffectiveConfig field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEffectiveConfig

`func (o *PolicyView) SetEffectiveConfig(v PolicyConfigView)`

SetEffectiveConfig sets EffectiveConfig field to given value.


### GetEffectiveEnabled

`func (o *PolicyView) GetEffectiveEnabled() bool`

GetEffectiveEnabled returns the EffectiveEnabled field if non-nil, zero value otherwise.

### GetEffectiveEnabledOk

`func (o *PolicyView) GetEffectiveEnabledOk() (*bool, bool)`

GetEffectiveEnabledOk returns a tuple with the EffectiveEnabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEffectiveEnabled

`func (o *PolicyView) SetEffectiveEnabled(v bool)`

SetEffectiveEnabled sets EffectiveEnabled field to given value.


### GetKind

`func (o *PolicyView) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *PolicyView) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *PolicyView) SetKind(v string)`

SetKind sets Kind field to given value.


### GetState

`func (o *PolicyView) GetState() string`

GetState returns the State field if non-nil, zero value otherwise.

### GetStateOk

`func (o *PolicyView) GetStateOk() (*string, bool)`

GetStateOk returns a tuple with the State field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetState

`func (o *PolicyView) SetState(v string)`

SetState sets State field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


