# VerifyEmailRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Token** | **string** | The token from the confirmation link. | 

## Methods

### NewVerifyEmailRequestBody

`func NewVerifyEmailRequestBody(token string, ) *VerifyEmailRequestBody`

NewVerifyEmailRequestBody instantiates a new VerifyEmailRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewVerifyEmailRequestBodyWithDefaults

`func NewVerifyEmailRequestBodyWithDefaults() *VerifyEmailRequestBody`

NewVerifyEmailRequestBodyWithDefaults instantiates a new VerifyEmailRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *VerifyEmailRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *VerifyEmailRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *VerifyEmailRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *VerifyEmailRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetToken

`func (o *VerifyEmailRequestBody) GetToken() string`

GetToken returns the Token field if non-nil, zero value otherwise.

### GetTokenOk

`func (o *VerifyEmailRequestBody) GetTokenOk() (*string, bool)`

GetTokenOk returns a tuple with the Token field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetToken

`func (o *VerifyEmailRequestBody) SetToken(v string)`

SetToken sets Token field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


