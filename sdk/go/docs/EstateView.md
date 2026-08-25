# EstateView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**CredentialId** | **string** |  | 
**CredentialKind** | **string** |  | 
**CredentialName** | **string** |  | 
**Fingerprint** | **string** | Complete SHA-256 of the credential&#39;s identifying component, in ssh encoding. Compare it against your provider console to confirm which credential is installed. | 
**Id** | **string** |  | 
**Name** | **string** |  | 
**Provider** | **string** |  | 
**Scope** | **map[string]string** | Provider-specific target, e.g. {\&quot;account\&quot;:\&quot;111\&quot;,\&quot;region\&quot;:\&quot;eu-west-1\&quot;} | 
**ScopeVersion** | **int64** |  | 

## Methods

### NewEstateView

`func NewEstateView(credentialId string, credentialKind string, credentialName string, fingerprint string, id string, name string, provider string, scope map[string]string, scopeVersion int64, ) *EstateView`

NewEstateView instantiates a new EstateView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewEstateViewWithDefaults

`func NewEstateViewWithDefaults() *EstateView`

NewEstateViewWithDefaults instantiates a new EstateView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *EstateView) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *EstateView) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *EstateView) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *EstateView) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetCredentialId

`func (o *EstateView) GetCredentialId() string`

GetCredentialId returns the CredentialId field if non-nil, zero value otherwise.

### GetCredentialIdOk

`func (o *EstateView) GetCredentialIdOk() (*string, bool)`

GetCredentialIdOk returns a tuple with the CredentialId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCredentialId

`func (o *EstateView) SetCredentialId(v string)`

SetCredentialId sets CredentialId field to given value.


### GetCredentialKind

`func (o *EstateView) GetCredentialKind() string`

GetCredentialKind returns the CredentialKind field if non-nil, zero value otherwise.

### GetCredentialKindOk

`func (o *EstateView) GetCredentialKindOk() (*string, bool)`

GetCredentialKindOk returns a tuple with the CredentialKind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCredentialKind

`func (o *EstateView) SetCredentialKind(v string)`

SetCredentialKind sets CredentialKind field to given value.


### GetCredentialName

`func (o *EstateView) GetCredentialName() string`

GetCredentialName returns the CredentialName field if non-nil, zero value otherwise.

### GetCredentialNameOk

`func (o *EstateView) GetCredentialNameOk() (*string, bool)`

GetCredentialNameOk returns a tuple with the CredentialName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCredentialName

`func (o *EstateView) SetCredentialName(v string)`

SetCredentialName sets CredentialName field to given value.


### GetFingerprint

`func (o *EstateView) GetFingerprint() string`

GetFingerprint returns the Fingerprint field if non-nil, zero value otherwise.

### GetFingerprintOk

`func (o *EstateView) GetFingerprintOk() (*string, bool)`

GetFingerprintOk returns a tuple with the Fingerprint field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFingerprint

`func (o *EstateView) SetFingerprint(v string)`

SetFingerprint sets Fingerprint field to given value.


### GetId

`func (o *EstateView) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *EstateView) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *EstateView) SetId(v string)`

SetId sets Id field to given value.


### GetName

`func (o *EstateView) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *EstateView) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *EstateView) SetName(v string)`

SetName sets Name field to given value.


### GetProvider

`func (o *EstateView) GetProvider() string`

GetProvider returns the Provider field if non-nil, zero value otherwise.

### GetProviderOk

`func (o *EstateView) GetProviderOk() (*string, bool)`

GetProviderOk returns a tuple with the Provider field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProvider

`func (o *EstateView) SetProvider(v string)`

SetProvider sets Provider field to given value.


### GetScope

`func (o *EstateView) GetScope() map[string]string`

GetScope returns the Scope field if non-nil, zero value otherwise.

### GetScopeOk

`func (o *EstateView) GetScopeOk() (*map[string]string, bool)`

GetScopeOk returns a tuple with the Scope field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetScope

`func (o *EstateView) SetScope(v map[string]string)`

SetScope sets Scope field to given value.


### GetScopeVersion

`func (o *EstateView) GetScopeVersion() int64`

GetScopeVersion returns the ScopeVersion field if non-nil, zero value otherwise.

### GetScopeVersionOk

`func (o *EstateView) GetScopeVersionOk() (*int64, bool)`

GetScopeVersionOk returns a tuple with the ScopeVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetScopeVersion

`func (o *EstateView) SetScopeVersion(v int64)`

SetScopeVersion sets ScopeVersion field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


