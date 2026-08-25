# UpdateRoleResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Updated** | **bool** |  | 

## Methods

### NewUpdateRoleResponseBody

`func NewUpdateRoleResponseBody(updated bool, ) *UpdateRoleResponseBody`

NewUpdateRoleResponseBody instantiates a new UpdateRoleResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewUpdateRoleResponseBodyWithDefaults

`func NewUpdateRoleResponseBodyWithDefaults() *UpdateRoleResponseBody`

NewUpdateRoleResponseBodyWithDefaults instantiates a new UpdateRoleResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *UpdateRoleResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *UpdateRoleResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *UpdateRoleResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *UpdateRoleResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetUpdated

`func (o *UpdateRoleResponseBody) GetUpdated() bool`

GetUpdated returns the Updated field if non-nil, zero value otherwise.

### GetUpdatedOk

`func (o *UpdateRoleResponseBody) GetUpdatedOk() (*bool, bool)`

GetUpdatedOk returns a tuple with the Updated field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdated

`func (o *UpdateRoleResponseBody) SetUpdated(v bool)`

SetUpdated sets Updated field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


