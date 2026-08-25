# MintAPIKeyResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Credential** | [**APIKeyView**](APIKeyView.md) |  | 
**Key** | **string** | The identifying half. For a user-scoped key this IS the whole credential; for an org-scoped key it is the HTTP Basic username. Returned once. | 
**Secret** | Pointer to **string** | The secret half of an org-scoped key, presented as the HTTP Basic password. Returned ONCE and stored only as a hash — no endpoint reissues it. Absent for user-scoped keys, which are a single opaque value. | [optional] 

## Methods

### NewMintAPIKeyResponseBody

`func NewMintAPIKeyResponseBody(credential APIKeyView, key string, ) *MintAPIKeyResponseBody`

NewMintAPIKeyResponseBody instantiates a new MintAPIKeyResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMintAPIKeyResponseBodyWithDefaults

`func NewMintAPIKeyResponseBodyWithDefaults() *MintAPIKeyResponseBody`

NewMintAPIKeyResponseBodyWithDefaults instantiates a new MintAPIKeyResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *MintAPIKeyResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *MintAPIKeyResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *MintAPIKeyResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *MintAPIKeyResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetCredential

`func (o *MintAPIKeyResponseBody) GetCredential() APIKeyView`

GetCredential returns the Credential field if non-nil, zero value otherwise.

### GetCredentialOk

`func (o *MintAPIKeyResponseBody) GetCredentialOk() (*APIKeyView, bool)`

GetCredentialOk returns a tuple with the Credential field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCredential

`func (o *MintAPIKeyResponseBody) SetCredential(v APIKeyView)`

SetCredential sets Credential field to given value.


### GetKey

`func (o *MintAPIKeyResponseBody) GetKey() string`

GetKey returns the Key field if non-nil, zero value otherwise.

### GetKeyOk

`func (o *MintAPIKeyResponseBody) GetKeyOk() (*string, bool)`

GetKeyOk returns a tuple with the Key field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKey

`func (o *MintAPIKeyResponseBody) SetKey(v string)`

SetKey sets Key field to given value.


### GetSecret

`func (o *MintAPIKeyResponseBody) GetSecret() string`

GetSecret returns the Secret field if non-nil, zero value otherwise.

### GetSecretOk

`func (o *MintAPIKeyResponseBody) GetSecretOk() (*string, bool)`

GetSecretOk returns a tuple with the Secret field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecret

`func (o *MintAPIKeyResponseBody) SetSecret(v string)`

SetSecret sets Secret field to given value.

### HasSecret

`func (o *MintAPIKeyResponseBody) HasSecret() bool`

HasSecret returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


