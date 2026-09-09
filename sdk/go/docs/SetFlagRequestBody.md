# SetFlagRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**AcknowledgeLockout** | Pointer to **bool** | Proceed even though disabling this would leave no way to sign in. Only consulted for a flag that gates sign-in. | [optional] 
**Value** | **bool** |  | 

## Methods

### NewSetFlagRequestBody

`func NewSetFlagRequestBody(value bool, ) *SetFlagRequestBody`

NewSetFlagRequestBody instantiates a new SetFlagRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSetFlagRequestBodyWithDefaults

`func NewSetFlagRequestBodyWithDefaults() *SetFlagRequestBody`

NewSetFlagRequestBodyWithDefaults instantiates a new SetFlagRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *SetFlagRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *SetFlagRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *SetFlagRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *SetFlagRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetAcknowledgeLockout

`func (o *SetFlagRequestBody) GetAcknowledgeLockout() bool`

GetAcknowledgeLockout returns the AcknowledgeLockout field if non-nil, zero value otherwise.

### GetAcknowledgeLockoutOk

`func (o *SetFlagRequestBody) GetAcknowledgeLockoutOk() (*bool, bool)`

GetAcknowledgeLockoutOk returns a tuple with the AcknowledgeLockout field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAcknowledgeLockout

`func (o *SetFlagRequestBody) SetAcknowledgeLockout(v bool)`

SetAcknowledgeLockout sets AcknowledgeLockout field to given value.

### HasAcknowledgeLockout

`func (o *SetFlagRequestBody) HasAcknowledgeLockout() bool`

HasAcknowledgeLockout returns a boolean if a field has been set.

### GetValue

`func (o *SetFlagRequestBody) GetValue() bool`

GetValue returns the Value field if non-nil, zero value otherwise.

### GetValueOk

`func (o *SetFlagRequestBody) GetValueOk() (*bool, bool)`

GetValueOk returns a tuple with the Value field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetValue

`func (o *SetFlagRequestBody) SetValue(v bool)`

SetValue sets Value field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


