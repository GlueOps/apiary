# TOTPDisableRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Code** | Pointer to **string** |  | [optional] 
**Password** | Pointer to **string** |  | [optional] 

## Methods

### NewTOTPDisableRequestBody

`func NewTOTPDisableRequestBody() *TOTPDisableRequestBody`

NewTOTPDisableRequestBody instantiates a new TOTPDisableRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewTOTPDisableRequestBodyWithDefaults

`func NewTOTPDisableRequestBodyWithDefaults() *TOTPDisableRequestBody`

NewTOTPDisableRequestBodyWithDefaults instantiates a new TOTPDisableRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *TOTPDisableRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *TOTPDisableRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *TOTPDisableRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *TOTPDisableRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetCode

`func (o *TOTPDisableRequestBody) GetCode() string`

GetCode returns the Code field if non-nil, zero value otherwise.

### GetCodeOk

`func (o *TOTPDisableRequestBody) GetCodeOk() (*string, bool)`

GetCodeOk returns a tuple with the Code field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCode

`func (o *TOTPDisableRequestBody) SetCode(v string)`

SetCode sets Code field to given value.

### HasCode

`func (o *TOTPDisableRequestBody) HasCode() bool`

HasCode returns a boolean if a field has been set.

### GetPassword

`func (o *TOTPDisableRequestBody) GetPassword() string`

GetPassword returns the Password field if non-nil, zero value otherwise.

### GetPasswordOk

`func (o *TOTPDisableRequestBody) GetPasswordOk() (*string, bool)`

GetPasswordOk returns a tuple with the Password field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPassword

`func (o *TOTPDisableRequestBody) SetPassword(v string)`

SetPassword sets Password field to given value.

### HasPassword

`func (o *TOTPDisableRequestBody) HasPassword() bool`

HasPassword returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


