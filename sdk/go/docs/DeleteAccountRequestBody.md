# DeleteAccountRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**ConfirmEmail** | **string** | Your own email address, exactly. | 

## Methods

### NewDeleteAccountRequestBody

`func NewDeleteAccountRequestBody(confirmEmail string, ) *DeleteAccountRequestBody`

NewDeleteAccountRequestBody instantiates a new DeleteAccountRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDeleteAccountRequestBodyWithDefaults

`func NewDeleteAccountRequestBodyWithDefaults() *DeleteAccountRequestBody`

NewDeleteAccountRequestBodyWithDefaults instantiates a new DeleteAccountRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *DeleteAccountRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *DeleteAccountRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *DeleteAccountRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *DeleteAccountRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetConfirmEmail

`func (o *DeleteAccountRequestBody) GetConfirmEmail() string`

GetConfirmEmail returns the ConfirmEmail field if non-nil, zero value otherwise.

### GetConfirmEmailOk

`func (o *DeleteAccountRequestBody) GetConfirmEmailOk() (*string, bool)`

GetConfirmEmailOk returns a tuple with the ConfirmEmail field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConfirmEmail

`func (o *DeleteAccountRequestBody) SetConfirmEmail(v string)`

SetConfirmEmail sets ConfirmEmail field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


