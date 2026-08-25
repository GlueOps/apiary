# PublicKeyCredentialCreationOptions

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Attestation** | Pointer to **string** |  | [optional] 
**AttestationFormats** | Pointer to **[]string** |  | [optional] 
**AuthenticatorSelection** | Pointer to [**AuthenticatorSelection**](AuthenticatorSelection.md) |  | [optional] 
**Challenge** | **string** |  | 
**ExcludeCredentials** | Pointer to [**[]CredentialDescriptor**](CredentialDescriptor.md) |  | [optional] 
**Extensions** | Pointer to **map[string]interface{}** |  | [optional] 
**Hints** | Pointer to **[]string** |  | [optional] 
**PubKeyCredParams** | Pointer to [**[]CredentialParameter**](CredentialParameter.md) |  | [optional] 
**Rp** | [**RelyingPartyEntity**](RelyingPartyEntity.md) |  | 
**Timeout** | Pointer to **int64** |  | [optional] 
**User** | [**UserEntity**](UserEntity.md) |  | 

## Methods

### NewPublicKeyCredentialCreationOptions

`func NewPublicKeyCredentialCreationOptions(challenge string, rp RelyingPartyEntity, user UserEntity, ) *PublicKeyCredentialCreationOptions`

NewPublicKeyCredentialCreationOptions instantiates a new PublicKeyCredentialCreationOptions object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPublicKeyCredentialCreationOptionsWithDefaults

`func NewPublicKeyCredentialCreationOptionsWithDefaults() *PublicKeyCredentialCreationOptions`

NewPublicKeyCredentialCreationOptionsWithDefaults instantiates a new PublicKeyCredentialCreationOptions object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAttestation

`func (o *PublicKeyCredentialCreationOptions) GetAttestation() string`

GetAttestation returns the Attestation field if non-nil, zero value otherwise.

### GetAttestationOk

`func (o *PublicKeyCredentialCreationOptions) GetAttestationOk() (*string, bool)`

GetAttestationOk returns a tuple with the Attestation field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAttestation

`func (o *PublicKeyCredentialCreationOptions) SetAttestation(v string)`

SetAttestation sets Attestation field to given value.

### HasAttestation

`func (o *PublicKeyCredentialCreationOptions) HasAttestation() bool`

HasAttestation returns a boolean if a field has been set.

### GetAttestationFormats

`func (o *PublicKeyCredentialCreationOptions) GetAttestationFormats() []string`

GetAttestationFormats returns the AttestationFormats field if non-nil, zero value otherwise.

### GetAttestationFormatsOk

`func (o *PublicKeyCredentialCreationOptions) GetAttestationFormatsOk() (*[]string, bool)`

GetAttestationFormatsOk returns a tuple with the AttestationFormats field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAttestationFormats

`func (o *PublicKeyCredentialCreationOptions) SetAttestationFormats(v []string)`

SetAttestationFormats sets AttestationFormats field to given value.

### HasAttestationFormats

`func (o *PublicKeyCredentialCreationOptions) HasAttestationFormats() bool`

HasAttestationFormats returns a boolean if a field has been set.

### SetAttestationFormatsNil

`func (o *PublicKeyCredentialCreationOptions) SetAttestationFormatsNil(b bool)`

 SetAttestationFormatsNil sets the value for AttestationFormats to be an explicit nil

### UnsetAttestationFormats
`func (o *PublicKeyCredentialCreationOptions) UnsetAttestationFormats()`

UnsetAttestationFormats ensures that no value is present for AttestationFormats, not even an explicit nil
### GetAuthenticatorSelection

`func (o *PublicKeyCredentialCreationOptions) GetAuthenticatorSelection() AuthenticatorSelection`

GetAuthenticatorSelection returns the AuthenticatorSelection field if non-nil, zero value otherwise.

### GetAuthenticatorSelectionOk

`func (o *PublicKeyCredentialCreationOptions) GetAuthenticatorSelectionOk() (*AuthenticatorSelection, bool)`

GetAuthenticatorSelectionOk returns a tuple with the AuthenticatorSelection field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAuthenticatorSelection

`func (o *PublicKeyCredentialCreationOptions) SetAuthenticatorSelection(v AuthenticatorSelection)`

SetAuthenticatorSelection sets AuthenticatorSelection field to given value.

### HasAuthenticatorSelection

`func (o *PublicKeyCredentialCreationOptions) HasAuthenticatorSelection() bool`

HasAuthenticatorSelection returns a boolean if a field has been set.

### GetChallenge

`func (o *PublicKeyCredentialCreationOptions) GetChallenge() string`

GetChallenge returns the Challenge field if non-nil, zero value otherwise.

### GetChallengeOk

`func (o *PublicKeyCredentialCreationOptions) GetChallengeOk() (*string, bool)`

GetChallengeOk returns a tuple with the Challenge field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetChallenge

`func (o *PublicKeyCredentialCreationOptions) SetChallenge(v string)`

SetChallenge sets Challenge field to given value.


### GetExcludeCredentials

`func (o *PublicKeyCredentialCreationOptions) GetExcludeCredentials() []CredentialDescriptor`

GetExcludeCredentials returns the ExcludeCredentials field if non-nil, zero value otherwise.

### GetExcludeCredentialsOk

`func (o *PublicKeyCredentialCreationOptions) GetExcludeCredentialsOk() (*[]CredentialDescriptor, bool)`

GetExcludeCredentialsOk returns a tuple with the ExcludeCredentials field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExcludeCredentials

`func (o *PublicKeyCredentialCreationOptions) SetExcludeCredentials(v []CredentialDescriptor)`

SetExcludeCredentials sets ExcludeCredentials field to given value.

### HasExcludeCredentials

`func (o *PublicKeyCredentialCreationOptions) HasExcludeCredentials() bool`

HasExcludeCredentials returns a boolean if a field has been set.

### SetExcludeCredentialsNil

`func (o *PublicKeyCredentialCreationOptions) SetExcludeCredentialsNil(b bool)`

 SetExcludeCredentialsNil sets the value for ExcludeCredentials to be an explicit nil

### UnsetExcludeCredentials
`func (o *PublicKeyCredentialCreationOptions) UnsetExcludeCredentials()`

UnsetExcludeCredentials ensures that no value is present for ExcludeCredentials, not even an explicit nil
### GetExtensions

`func (o *PublicKeyCredentialCreationOptions) GetExtensions() map[string]*interface{}`

GetExtensions returns the Extensions field if non-nil, zero value otherwise.

### GetExtensionsOk

`func (o *PublicKeyCredentialCreationOptions) GetExtensionsOk() (*map[string]*interface{}, bool)`

GetExtensionsOk returns a tuple with the Extensions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExtensions

`func (o *PublicKeyCredentialCreationOptions) SetExtensions(v map[string]*interface{})`

SetExtensions sets Extensions field to given value.

### HasExtensions

`func (o *PublicKeyCredentialCreationOptions) HasExtensions() bool`

HasExtensions returns a boolean if a field has been set.

### GetHints

`func (o *PublicKeyCredentialCreationOptions) GetHints() []string`

GetHints returns the Hints field if non-nil, zero value otherwise.

### GetHintsOk

`func (o *PublicKeyCredentialCreationOptions) GetHintsOk() (*[]string, bool)`

GetHintsOk returns a tuple with the Hints field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHints

`func (o *PublicKeyCredentialCreationOptions) SetHints(v []string)`

SetHints sets Hints field to given value.

### HasHints

`func (o *PublicKeyCredentialCreationOptions) HasHints() bool`

HasHints returns a boolean if a field has been set.

### SetHintsNil

`func (o *PublicKeyCredentialCreationOptions) SetHintsNil(b bool)`

 SetHintsNil sets the value for Hints to be an explicit nil

### UnsetHints
`func (o *PublicKeyCredentialCreationOptions) UnsetHints()`

UnsetHints ensures that no value is present for Hints, not even an explicit nil
### GetPubKeyCredParams

`func (o *PublicKeyCredentialCreationOptions) GetPubKeyCredParams() []CredentialParameter`

GetPubKeyCredParams returns the PubKeyCredParams field if non-nil, zero value otherwise.

### GetPubKeyCredParamsOk

`func (o *PublicKeyCredentialCreationOptions) GetPubKeyCredParamsOk() (*[]CredentialParameter, bool)`

GetPubKeyCredParamsOk returns a tuple with the PubKeyCredParams field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPubKeyCredParams

`func (o *PublicKeyCredentialCreationOptions) SetPubKeyCredParams(v []CredentialParameter)`

SetPubKeyCredParams sets PubKeyCredParams field to given value.

### HasPubKeyCredParams

`func (o *PublicKeyCredentialCreationOptions) HasPubKeyCredParams() bool`

HasPubKeyCredParams returns a boolean if a field has been set.

### SetPubKeyCredParamsNil

`func (o *PublicKeyCredentialCreationOptions) SetPubKeyCredParamsNil(b bool)`

 SetPubKeyCredParamsNil sets the value for PubKeyCredParams to be an explicit nil

### UnsetPubKeyCredParams
`func (o *PublicKeyCredentialCreationOptions) UnsetPubKeyCredParams()`

UnsetPubKeyCredParams ensures that no value is present for PubKeyCredParams, not even an explicit nil
### GetRp

`func (o *PublicKeyCredentialCreationOptions) GetRp() RelyingPartyEntity`

GetRp returns the Rp field if non-nil, zero value otherwise.

### GetRpOk

`func (o *PublicKeyCredentialCreationOptions) GetRpOk() (*RelyingPartyEntity, bool)`

GetRpOk returns a tuple with the Rp field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRp

`func (o *PublicKeyCredentialCreationOptions) SetRp(v RelyingPartyEntity)`

SetRp sets Rp field to given value.


### GetTimeout

`func (o *PublicKeyCredentialCreationOptions) GetTimeout() int64`

GetTimeout returns the Timeout field if non-nil, zero value otherwise.

### GetTimeoutOk

`func (o *PublicKeyCredentialCreationOptions) GetTimeoutOk() (*int64, bool)`

GetTimeoutOk returns a tuple with the Timeout field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimeout

`func (o *PublicKeyCredentialCreationOptions) SetTimeout(v int64)`

SetTimeout sets Timeout field to given value.

### HasTimeout

`func (o *PublicKeyCredentialCreationOptions) HasTimeout() bool`

HasTimeout returns a boolean if a field has been set.

### GetUser

`func (o *PublicKeyCredentialCreationOptions) GetUser() UserEntity`

GetUser returns the User field if non-nil, zero value otherwise.

### GetUserOk

`func (o *PublicKeyCredentialCreationOptions) GetUserOk() (*UserEntity, bool)`

GetUserOk returns a tuple with the User field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUser

`func (o *PublicKeyCredentialCreationOptions) SetUser(v UserEntity)`

SetUser sets User field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


