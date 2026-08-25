# MFAVerifyRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Code** | **string** |  | 
**MfaToken** | **string** |  | 

## Methods

### NewMFAVerifyRequestBody

`func NewMFAVerifyRequestBody(code string, mfaToken string, ) *MFAVerifyRequestBody`

NewMFAVerifyRequestBody instantiates a new MFAVerifyRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMFAVerifyRequestBodyWithDefaults

`func NewMFAVerifyRequestBodyWithDefaults() *MFAVerifyRequestBody`

NewMFAVerifyRequestBodyWithDefaults instantiates a new MFAVerifyRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *MFAVerifyRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *MFAVerifyRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *MFAVerifyRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *MFAVerifyRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetCode

`func (o *MFAVerifyRequestBody) GetCode() string`

GetCode returns the Code field if non-nil, zero value otherwise.

### GetCodeOk

`func (o *MFAVerifyRequestBody) GetCodeOk() (*string, bool)`

GetCodeOk returns a tuple with the Code field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCode

`func (o *MFAVerifyRequestBody) SetCode(v string)`

SetCode sets Code field to given value.


### GetMfaToken

`func (o *MFAVerifyRequestBody) GetMfaToken() string`

GetMfaToken returns the MfaToken field if non-nil, zero value otherwise.

### GetMfaTokenOk

`func (o *MFAVerifyRequestBody) GetMfaTokenOk() (*string, bool)`

GetMfaTokenOk returns a tuple with the MfaToken field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMfaToken

`func (o *MFAVerifyRequestBody) SetMfaToken(v string)`

SetMfaToken sets MfaToken field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


