# AuthenticationExtensions

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Appid** | Pointer to **string** |  | [optional] 
**AppidExclude** | Pointer to **string** |  | [optional] 
**CredBlob** | Pointer to **string** |  | [optional] 
**CredProps** | Pointer to **bool** |  | [optional] 
**CredentialProtectionPolicy** | Pointer to **string** |  | [optional] 
**EnforceCredentialProtectionPolicy** | Pointer to **bool** |  | [optional] 
**RequestCredBlob** | Pointer to **bool** |  | [optional] 
**HmacCreateSecret** | Pointer to **bool** |  | [optional] 
**HmacGetSecret** | Pointer to [**HMACGetSecretInputs**](HMACGetSecretInputs.md) |  | [optional] 
**LargeBlob** | Pointer to [**LargeBlobInputs**](LargeBlobInputs.md) |  | [optional] 
**MinPinLength** | Pointer to **bool** |  | [optional] 
**Prf** | Pointer to [**PRFInputs**](PRFInputs.md) |  | [optional] 
**RemoteClientDataJSON** | Pointer to **string** |  | [optional] 
**Uvm** | Pointer to **bool** |  | [optional] 

## Methods

### NewAuthenticationExtensions

`func NewAuthenticationExtensions() *AuthenticationExtensions`

NewAuthenticationExtensions instantiates a new AuthenticationExtensions object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAuthenticationExtensionsWithDefaults

`func NewAuthenticationExtensionsWithDefaults() *AuthenticationExtensions`

NewAuthenticationExtensionsWithDefaults instantiates a new AuthenticationExtensions object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAppid

`func (o *AuthenticationExtensions) GetAppid() string`

GetAppid returns the Appid field if non-nil, zero value otherwise.

### GetAppidOk

`func (o *AuthenticationExtensions) GetAppidOk() (*string, bool)`

GetAppidOk returns a tuple with the Appid field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAppid

`func (o *AuthenticationExtensions) SetAppid(v string)`

SetAppid sets Appid field to given value.

### HasAppid

`func (o *AuthenticationExtensions) HasAppid() bool`

HasAppid returns a boolean if a field has been set.

### GetAppidExclude

`func (o *AuthenticationExtensions) GetAppidExclude() string`

GetAppidExclude returns the AppidExclude field if non-nil, zero value otherwise.

### GetAppidExcludeOk

`func (o *AuthenticationExtensions) GetAppidExcludeOk() (*string, bool)`

GetAppidExcludeOk returns a tuple with the AppidExclude field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAppidExclude

`func (o *AuthenticationExtensions) SetAppidExclude(v string)`

SetAppidExclude sets AppidExclude field to given value.

### HasAppidExclude

`func (o *AuthenticationExtensions) HasAppidExclude() bool`

HasAppidExclude returns a boolean if a field has been set.

### GetCredBlob

`func (o *AuthenticationExtensions) GetCredBlob() string`

GetCredBlob returns the CredBlob field if non-nil, zero value otherwise.

### GetCredBlobOk

`func (o *AuthenticationExtensions) GetCredBlobOk() (*string, bool)`

GetCredBlobOk returns a tuple with the CredBlob field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCredBlob

`func (o *AuthenticationExtensions) SetCredBlob(v string)`

SetCredBlob sets CredBlob field to given value.

### HasCredBlob

`func (o *AuthenticationExtensions) HasCredBlob() bool`

HasCredBlob returns a boolean if a field has been set.

### GetCredProps

`func (o *AuthenticationExtensions) GetCredProps() bool`

GetCredProps returns the CredProps field if non-nil, zero value otherwise.

### GetCredPropsOk

`func (o *AuthenticationExtensions) GetCredPropsOk() (*bool, bool)`

GetCredPropsOk returns a tuple with the CredProps field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCredProps

`func (o *AuthenticationExtensions) SetCredProps(v bool)`

SetCredProps sets CredProps field to given value.

### HasCredProps

`func (o *AuthenticationExtensions) HasCredProps() bool`

HasCredProps returns a boolean if a field has been set.

### GetCredentialProtectionPolicy

`func (o *AuthenticationExtensions) GetCredentialProtectionPolicy() string`

GetCredentialProtectionPolicy returns the CredentialProtectionPolicy field if non-nil, zero value otherwise.

### GetCredentialProtectionPolicyOk

`func (o *AuthenticationExtensions) GetCredentialProtectionPolicyOk() (*string, bool)`

GetCredentialProtectionPolicyOk returns a tuple with the CredentialProtectionPolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCredentialProtectionPolicy

`func (o *AuthenticationExtensions) SetCredentialProtectionPolicy(v string)`

SetCredentialProtectionPolicy sets CredentialProtectionPolicy field to given value.

### HasCredentialProtectionPolicy

`func (o *AuthenticationExtensions) HasCredentialProtectionPolicy() bool`

HasCredentialProtectionPolicy returns a boolean if a field has been set.

### GetEnforceCredentialProtectionPolicy

`func (o *AuthenticationExtensions) GetEnforceCredentialProtectionPolicy() bool`

GetEnforceCredentialProtectionPolicy returns the EnforceCredentialProtectionPolicy field if non-nil, zero value otherwise.

### GetEnforceCredentialProtectionPolicyOk

`func (o *AuthenticationExtensions) GetEnforceCredentialProtectionPolicyOk() (*bool, bool)`

GetEnforceCredentialProtectionPolicyOk returns a tuple with the EnforceCredentialProtectionPolicy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnforceCredentialProtectionPolicy

`func (o *AuthenticationExtensions) SetEnforceCredentialProtectionPolicy(v bool)`

SetEnforceCredentialProtectionPolicy sets EnforceCredentialProtectionPolicy field to given value.

### HasEnforceCredentialProtectionPolicy

`func (o *AuthenticationExtensions) HasEnforceCredentialProtectionPolicy() bool`

HasEnforceCredentialProtectionPolicy returns a boolean if a field has been set.

### GetRequestCredBlob

`func (o *AuthenticationExtensions) GetRequestCredBlob() bool`

GetRequestCredBlob returns the RequestCredBlob field if non-nil, zero value otherwise.

### GetRequestCredBlobOk

`func (o *AuthenticationExtensions) GetRequestCredBlobOk() (*bool, bool)`

GetRequestCredBlobOk returns a tuple with the RequestCredBlob field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRequestCredBlob

`func (o *AuthenticationExtensions) SetRequestCredBlob(v bool)`

SetRequestCredBlob sets RequestCredBlob field to given value.

### HasRequestCredBlob

`func (o *AuthenticationExtensions) HasRequestCredBlob() bool`

HasRequestCredBlob returns a boolean if a field has been set.

### GetHmacCreateSecret

`func (o *AuthenticationExtensions) GetHmacCreateSecret() bool`

GetHmacCreateSecret returns the HmacCreateSecret field if non-nil, zero value otherwise.

### GetHmacCreateSecretOk

`func (o *AuthenticationExtensions) GetHmacCreateSecretOk() (*bool, bool)`

GetHmacCreateSecretOk returns a tuple with the HmacCreateSecret field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHmacCreateSecret

`func (o *AuthenticationExtensions) SetHmacCreateSecret(v bool)`

SetHmacCreateSecret sets HmacCreateSecret field to given value.

### HasHmacCreateSecret

`func (o *AuthenticationExtensions) HasHmacCreateSecret() bool`

HasHmacCreateSecret returns a boolean if a field has been set.

### GetHmacGetSecret

`func (o *AuthenticationExtensions) GetHmacGetSecret() HMACGetSecretInputs`

GetHmacGetSecret returns the HmacGetSecret field if non-nil, zero value otherwise.

### GetHmacGetSecretOk

`func (o *AuthenticationExtensions) GetHmacGetSecretOk() (*HMACGetSecretInputs, bool)`

GetHmacGetSecretOk returns a tuple with the HmacGetSecret field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHmacGetSecret

`func (o *AuthenticationExtensions) SetHmacGetSecret(v HMACGetSecretInputs)`

SetHmacGetSecret sets HmacGetSecret field to given value.

### HasHmacGetSecret

`func (o *AuthenticationExtensions) HasHmacGetSecret() bool`

HasHmacGetSecret returns a boolean if a field has been set.

### GetLargeBlob

`func (o *AuthenticationExtensions) GetLargeBlob() LargeBlobInputs`

GetLargeBlob returns the LargeBlob field if non-nil, zero value otherwise.

### GetLargeBlobOk

`func (o *AuthenticationExtensions) GetLargeBlobOk() (*LargeBlobInputs, bool)`

GetLargeBlobOk returns a tuple with the LargeBlob field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLargeBlob

`func (o *AuthenticationExtensions) SetLargeBlob(v LargeBlobInputs)`

SetLargeBlob sets LargeBlob field to given value.

### HasLargeBlob

`func (o *AuthenticationExtensions) HasLargeBlob() bool`

HasLargeBlob returns a boolean if a field has been set.

### GetMinPinLength

`func (o *AuthenticationExtensions) GetMinPinLength() bool`

GetMinPinLength returns the MinPinLength field if non-nil, zero value otherwise.

### GetMinPinLengthOk

`func (o *AuthenticationExtensions) GetMinPinLengthOk() (*bool, bool)`

GetMinPinLengthOk returns a tuple with the MinPinLength field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMinPinLength

`func (o *AuthenticationExtensions) SetMinPinLength(v bool)`

SetMinPinLength sets MinPinLength field to given value.

### HasMinPinLength

`func (o *AuthenticationExtensions) HasMinPinLength() bool`

HasMinPinLength returns a boolean if a field has been set.

### GetPrf

`func (o *AuthenticationExtensions) GetPrf() PRFInputs`

GetPrf returns the Prf field if non-nil, zero value otherwise.

### GetPrfOk

`func (o *AuthenticationExtensions) GetPrfOk() (*PRFInputs, bool)`

GetPrfOk returns a tuple with the Prf field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPrf

`func (o *AuthenticationExtensions) SetPrf(v PRFInputs)`

SetPrf sets Prf field to given value.

### HasPrf

`func (o *AuthenticationExtensions) HasPrf() bool`

HasPrf returns a boolean if a field has been set.

### GetRemoteClientDataJSON

`func (o *AuthenticationExtensions) GetRemoteClientDataJSON() string`

GetRemoteClientDataJSON returns the RemoteClientDataJSON field if non-nil, zero value otherwise.

### GetRemoteClientDataJSONOk

`func (o *AuthenticationExtensions) GetRemoteClientDataJSONOk() (*string, bool)`

GetRemoteClientDataJSONOk returns a tuple with the RemoteClientDataJSON field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRemoteClientDataJSON

`func (o *AuthenticationExtensions) SetRemoteClientDataJSON(v string)`

SetRemoteClientDataJSON sets RemoteClientDataJSON field to given value.

### HasRemoteClientDataJSON

`func (o *AuthenticationExtensions) HasRemoteClientDataJSON() bool`

HasRemoteClientDataJSON returns a boolean if a field has been set.

### GetUvm

`func (o *AuthenticationExtensions) GetUvm() bool`

GetUvm returns the Uvm field if non-nil, zero value otherwise.

### GetUvmOk

`func (o *AuthenticationExtensions) GetUvmOk() (*bool, bool)`

GetUvmOk returns a tuple with the Uvm field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUvm

`func (o *AuthenticationExtensions) SetUvm(v bool)`

SetUvm sets Uvm field to given value.

### HasUvm

`func (o *AuthenticationExtensions) HasUvm() bool`

HasUvm returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


