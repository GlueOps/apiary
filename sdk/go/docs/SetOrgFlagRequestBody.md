# SetOrgFlagRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Value** | **bool** |  | 

## Methods

### NewSetOrgFlagRequestBody

`func NewSetOrgFlagRequestBody(value bool, ) *SetOrgFlagRequestBody`

NewSetOrgFlagRequestBody instantiates a new SetOrgFlagRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSetOrgFlagRequestBodyWithDefaults

`func NewSetOrgFlagRequestBodyWithDefaults() *SetOrgFlagRequestBody`

NewSetOrgFlagRequestBodyWithDefaults instantiates a new SetOrgFlagRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *SetOrgFlagRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *SetOrgFlagRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *SetOrgFlagRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *SetOrgFlagRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetValue

`func (o *SetOrgFlagRequestBody) GetValue() bool`

GetValue returns the Value field if non-nil, zero value otherwise.

### GetValueOk

`func (o *SetOrgFlagRequestBody) GetValueOk() (*bool, bool)`

GetValueOk returns a tuple with the Value field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetValue

`func (o *SetOrgFlagRequestBody) SetValue(v bool)`

SetValue sets Value field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


