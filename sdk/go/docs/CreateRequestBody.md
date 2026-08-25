# CreateRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**CredentialKind** | Pointer to **string** |  | [optional] [default to "static_secret"]
**CredentialName** | **string** |  | 
**Identity** | **string** | The NON-SECRET identifying component — AWS access key id, Proxmox token id, or role ARN. This is what gets fingerprinted. | 
**Name** | **string** |  | 
**Provider** | **string** |  | 
**Scope** | **map[string]string** | Provider-specific target. Keys are validated against the provider; unknown keys are refused. | 
**Secret** | **string** | Credential material. Sealed at rest and never returned by any endpoint. | 

## Methods

### NewCreateRequestBody

`func NewCreateRequestBody(credentialName string, identity string, name string, provider string, scope map[string]string, secret string, ) *CreateRequestBody`

NewCreateRequestBody instantiates a new CreateRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCreateRequestBodyWithDefaults

`func NewCreateRequestBodyWithDefaults() *CreateRequestBody`

NewCreateRequestBodyWithDefaults instantiates a new CreateRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *CreateRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *CreateRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *CreateRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *CreateRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetCredentialKind

`func (o *CreateRequestBody) GetCredentialKind() string`

GetCredentialKind returns the CredentialKind field if non-nil, zero value otherwise.

### GetCredentialKindOk

`func (o *CreateRequestBody) GetCredentialKindOk() (*string, bool)`

GetCredentialKindOk returns a tuple with the CredentialKind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCredentialKind

`func (o *CreateRequestBody) SetCredentialKind(v string)`

SetCredentialKind sets CredentialKind field to given value.

### HasCredentialKind

`func (o *CreateRequestBody) HasCredentialKind() bool`

HasCredentialKind returns a boolean if a field has been set.

### GetCredentialName

`func (o *CreateRequestBody) GetCredentialName() string`

GetCredentialName returns the CredentialName field if non-nil, zero value otherwise.

### GetCredentialNameOk

`func (o *CreateRequestBody) GetCredentialNameOk() (*string, bool)`

GetCredentialNameOk returns a tuple with the CredentialName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCredentialName

`func (o *CreateRequestBody) SetCredentialName(v string)`

SetCredentialName sets CredentialName field to given value.


### GetIdentity

`func (o *CreateRequestBody) GetIdentity() string`

GetIdentity returns the Identity field if non-nil, zero value otherwise.

### GetIdentityOk

`func (o *CreateRequestBody) GetIdentityOk() (*string, bool)`

GetIdentityOk returns a tuple with the Identity field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIdentity

`func (o *CreateRequestBody) SetIdentity(v string)`

SetIdentity sets Identity field to given value.


### GetName

`func (o *CreateRequestBody) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *CreateRequestBody) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *CreateRequestBody) SetName(v string)`

SetName sets Name field to given value.


### GetProvider

`func (o *CreateRequestBody) GetProvider() string`

GetProvider returns the Provider field if non-nil, zero value otherwise.

### GetProviderOk

`func (o *CreateRequestBody) GetProviderOk() (*string, bool)`

GetProviderOk returns a tuple with the Provider field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProvider

`func (o *CreateRequestBody) SetProvider(v string)`

SetProvider sets Provider field to given value.


### GetScope

`func (o *CreateRequestBody) GetScope() map[string]string`

GetScope returns the Scope field if non-nil, zero value otherwise.

### GetScopeOk

`func (o *CreateRequestBody) GetScopeOk() (*map[string]string, bool)`

GetScopeOk returns a tuple with the Scope field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetScope

`func (o *CreateRequestBody) SetScope(v map[string]string)`

SetScope sets Scope field to given value.


### GetSecret

`func (o *CreateRequestBody) GetSecret() string`

GetSecret returns the Secret field if non-nil, zero value otherwise.

### GetSecretOk

`func (o *CreateRequestBody) GetSecretOk() (*string, bool)`

GetSecretOk returns a tuple with the Secret field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecret

`func (o *CreateRequestBody) SetSecret(v string)`

SetSecret sets Secret field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


