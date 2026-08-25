# ForgotPasswordRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Email** | **string** |  | 

## Methods

### NewForgotPasswordRequestBody

`func NewForgotPasswordRequestBody(email string, ) *ForgotPasswordRequestBody`

NewForgotPasswordRequestBody instantiates a new ForgotPasswordRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewForgotPasswordRequestBodyWithDefaults

`func NewForgotPasswordRequestBodyWithDefaults() *ForgotPasswordRequestBody`

NewForgotPasswordRequestBodyWithDefaults instantiates a new ForgotPasswordRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ForgotPasswordRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ForgotPasswordRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ForgotPasswordRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ForgotPasswordRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetEmail

`func (o *ForgotPasswordRequestBody) GetEmail() string`

GetEmail returns the Email field if non-nil, zero value otherwise.

### GetEmailOk

`func (o *ForgotPasswordRequestBody) GetEmailOk() (*string, bool)`

GetEmailOk returns a tuple with the Email field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEmail

`func (o *ForgotPasswordRequestBody) SetEmail(v string)`

SetEmail sets Email field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


