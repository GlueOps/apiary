# ChangePasswordRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**CurrentPassword** | **string** |  | 
**NewPassword** | **string** |  | 

## Methods

### NewChangePasswordRequestBody

`func NewChangePasswordRequestBody(currentPassword string, newPassword string, ) *ChangePasswordRequestBody`

NewChangePasswordRequestBody instantiates a new ChangePasswordRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewChangePasswordRequestBodyWithDefaults

`func NewChangePasswordRequestBodyWithDefaults() *ChangePasswordRequestBody`

NewChangePasswordRequestBodyWithDefaults instantiates a new ChangePasswordRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ChangePasswordRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ChangePasswordRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ChangePasswordRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ChangePasswordRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetCurrentPassword

`func (o *ChangePasswordRequestBody) GetCurrentPassword() string`

GetCurrentPassword returns the CurrentPassword field if non-nil, zero value otherwise.

### GetCurrentPasswordOk

`func (o *ChangePasswordRequestBody) GetCurrentPasswordOk() (*string, bool)`

GetCurrentPasswordOk returns a tuple with the CurrentPassword field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCurrentPassword

`func (o *ChangePasswordRequestBody) SetCurrentPassword(v string)`

SetCurrentPassword sets CurrentPassword field to given value.


### GetNewPassword

`func (o *ChangePasswordRequestBody) GetNewPassword() string`

GetNewPassword returns the NewPassword field if non-nil, zero value otherwise.

### GetNewPasswordOk

`func (o *ChangePasswordRequestBody) GetNewPasswordOk() (*string, bool)`

GetNewPasswordOk returns a tuple with the NewPassword field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNewPassword

`func (o *ChangePasswordRequestBody) SetNewPassword(v string)`

SetNewPassword sets NewPassword field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


