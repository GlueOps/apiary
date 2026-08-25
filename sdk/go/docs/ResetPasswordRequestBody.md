# ResetPasswordRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**NewPassword** | **string** |  | 
**Token** | **string** |  | 

## Methods

### NewResetPasswordRequestBody

`func NewResetPasswordRequestBody(newPassword string, token string, ) *ResetPasswordRequestBody`

NewResetPasswordRequestBody instantiates a new ResetPasswordRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewResetPasswordRequestBodyWithDefaults

`func NewResetPasswordRequestBodyWithDefaults() *ResetPasswordRequestBody`

NewResetPasswordRequestBodyWithDefaults instantiates a new ResetPasswordRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ResetPasswordRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ResetPasswordRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ResetPasswordRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ResetPasswordRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetNewPassword

`func (o *ResetPasswordRequestBody) GetNewPassword() string`

GetNewPassword returns the NewPassword field if non-nil, zero value otherwise.

### GetNewPasswordOk

`func (o *ResetPasswordRequestBody) GetNewPasswordOk() (*string, bool)`

GetNewPasswordOk returns a tuple with the NewPassword field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNewPassword

`func (o *ResetPasswordRequestBody) SetNewPassword(v string)`

SetNewPassword sets NewPassword field to given value.


### GetToken

`func (o *ResetPasswordRequestBody) GetToken() string`

GetToken returns the Token field if non-nil, zero value otherwise.

### GetTokenOk

`func (o *ResetPasswordRequestBody) GetTokenOk() (*string, bool)`

GetTokenOk returns a tuple with the Token field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetToken

`func (o *ResetPasswordRequestBody) SetToken(v string)`

SetToken sets Token field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


