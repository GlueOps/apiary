# UpdateBindingRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Enabled** | **bool** |  | 
**Priority** | Pointer to **int64** |  | [optional] 

## Methods

### NewUpdateBindingRequestBody

`func NewUpdateBindingRequestBody(enabled bool, ) *UpdateBindingRequestBody`

NewUpdateBindingRequestBody instantiates a new UpdateBindingRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewUpdateBindingRequestBodyWithDefaults

`func NewUpdateBindingRequestBodyWithDefaults() *UpdateBindingRequestBody`

NewUpdateBindingRequestBodyWithDefaults instantiates a new UpdateBindingRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *UpdateBindingRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *UpdateBindingRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *UpdateBindingRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *UpdateBindingRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetEnabled

`func (o *UpdateBindingRequestBody) GetEnabled() bool`

GetEnabled returns the Enabled field if non-nil, zero value otherwise.

### GetEnabledOk

`func (o *UpdateBindingRequestBody) GetEnabledOk() (*bool, bool)`

GetEnabledOk returns a tuple with the Enabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnabled

`func (o *UpdateBindingRequestBody) SetEnabled(v bool)`

SetEnabled sets Enabled field to given value.


### GetPriority

`func (o *UpdateBindingRequestBody) GetPriority() int64`

GetPriority returns the Priority field if non-nil, zero value otherwise.

### GetPriorityOk

`func (o *UpdateBindingRequestBody) GetPriorityOk() (*int64, bool)`

GetPriorityOk returns a tuple with the Priority field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPriority

`func (o *UpdateBindingRequestBody) SetPriority(v int64)`

SetPriority sets Priority field to given value.

### HasPriority

`func (o *UpdateBindingRequestBody) HasPriority() bool`

HasPriority returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


