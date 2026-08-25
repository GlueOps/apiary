# RevokeAPIKeyResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Revoked** | **bool** |  | 

## Methods

### NewRevokeAPIKeyResponseBody

`func NewRevokeAPIKeyResponseBody(revoked bool, ) *RevokeAPIKeyResponseBody`

NewRevokeAPIKeyResponseBody instantiates a new RevokeAPIKeyResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRevokeAPIKeyResponseBodyWithDefaults

`func NewRevokeAPIKeyResponseBodyWithDefaults() *RevokeAPIKeyResponseBody`

NewRevokeAPIKeyResponseBodyWithDefaults instantiates a new RevokeAPIKeyResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *RevokeAPIKeyResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *RevokeAPIKeyResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *RevokeAPIKeyResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *RevokeAPIKeyResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetRevoked

`func (o *RevokeAPIKeyResponseBody) GetRevoked() bool`

GetRevoked returns the Revoked field if non-nil, zero value otherwise.

### GetRevokedOk

`func (o *RevokeAPIKeyResponseBody) GetRevokedOk() (*bool, bool)`

GetRevokedOk returns a tuple with the Revoked field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRevoked

`func (o *RevokeAPIKeyResponseBody) SetRevoked(v bool)`

SetRevoked sets Revoked field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


