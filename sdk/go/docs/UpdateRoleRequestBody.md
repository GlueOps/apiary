# UpdateRoleRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Role** | **string** |  | 

## Methods

### NewUpdateRoleRequestBody

`func NewUpdateRoleRequestBody(role string, ) *UpdateRoleRequestBody`

NewUpdateRoleRequestBody instantiates a new UpdateRoleRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewUpdateRoleRequestBodyWithDefaults

`func NewUpdateRoleRequestBodyWithDefaults() *UpdateRoleRequestBody`

NewUpdateRoleRequestBodyWithDefaults instantiates a new UpdateRoleRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *UpdateRoleRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *UpdateRoleRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *UpdateRoleRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *UpdateRoleRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetRole

`func (o *UpdateRoleRequestBody) GetRole() string`

GetRole returns the Role field if non-nil, zero value otherwise.

### GetRoleOk

`func (o *UpdateRoleRequestBody) GetRoleOk() (*string, bool)`

GetRoleOk returns a tuple with the Role field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRole

`func (o *UpdateRoleRequestBody) SetRole(v string)`

SetRole sets Role field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


