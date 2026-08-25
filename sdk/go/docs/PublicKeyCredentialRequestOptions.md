# PublicKeyCredentialRequestOptions

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AllowCredentials** | Pointer to [**[]CredentialDescriptor**](CredentialDescriptor.md) |  | [optional] 
**Challenge** | **string** |  | 
**Extensions** | Pointer to **map[string]interface{}** |  | [optional] 
**Hints** | Pointer to **[]string** |  | [optional] 
**RpId** | Pointer to **string** |  | [optional] 
**Timeout** | Pointer to **int64** |  | [optional] 
**UserVerification** | Pointer to **string** |  | [optional] 

## Methods

### NewPublicKeyCredentialRequestOptions

`func NewPublicKeyCredentialRequestOptions(challenge string, ) *PublicKeyCredentialRequestOptions`

NewPublicKeyCredentialRequestOptions instantiates a new PublicKeyCredentialRequestOptions object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPublicKeyCredentialRequestOptionsWithDefaults

`func NewPublicKeyCredentialRequestOptionsWithDefaults() *PublicKeyCredentialRequestOptions`

NewPublicKeyCredentialRequestOptionsWithDefaults instantiates a new PublicKeyCredentialRequestOptions object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAllowCredentials

`func (o *PublicKeyCredentialRequestOptions) GetAllowCredentials() []CredentialDescriptor`

GetAllowCredentials returns the AllowCredentials field if non-nil, zero value otherwise.

### GetAllowCredentialsOk

`func (o *PublicKeyCredentialRequestOptions) GetAllowCredentialsOk() (*[]CredentialDescriptor, bool)`

GetAllowCredentialsOk returns a tuple with the AllowCredentials field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAllowCredentials

`func (o *PublicKeyCredentialRequestOptions) SetAllowCredentials(v []CredentialDescriptor)`

SetAllowCredentials sets AllowCredentials field to given value.

### HasAllowCredentials

`func (o *PublicKeyCredentialRequestOptions) HasAllowCredentials() bool`

HasAllowCredentials returns a boolean if a field has been set.

### SetAllowCredentialsNil

`func (o *PublicKeyCredentialRequestOptions) SetAllowCredentialsNil(b bool)`

 SetAllowCredentialsNil sets the value for AllowCredentials to be an explicit nil

### UnsetAllowCredentials
`func (o *PublicKeyCredentialRequestOptions) UnsetAllowCredentials()`

UnsetAllowCredentials ensures that no value is present for AllowCredentials, not even an explicit nil
### GetChallenge

`func (o *PublicKeyCredentialRequestOptions) GetChallenge() string`

GetChallenge returns the Challenge field if non-nil, zero value otherwise.

### GetChallengeOk

`func (o *PublicKeyCredentialRequestOptions) GetChallengeOk() (*string, bool)`

GetChallengeOk returns a tuple with the Challenge field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetChallenge

`func (o *PublicKeyCredentialRequestOptions) SetChallenge(v string)`

SetChallenge sets Challenge field to given value.


### GetExtensions

`func (o *PublicKeyCredentialRequestOptions) GetExtensions() map[string]*interface{}`

GetExtensions returns the Extensions field if non-nil, zero value otherwise.

### GetExtensionsOk

`func (o *PublicKeyCredentialRequestOptions) GetExtensionsOk() (*map[string]*interface{}, bool)`

GetExtensionsOk returns a tuple with the Extensions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExtensions

`func (o *PublicKeyCredentialRequestOptions) SetExtensions(v map[string]*interface{})`

SetExtensions sets Extensions field to given value.

### HasExtensions

`func (o *PublicKeyCredentialRequestOptions) HasExtensions() bool`

HasExtensions returns a boolean if a field has been set.

### GetHints

`func (o *PublicKeyCredentialRequestOptions) GetHints() []string`

GetHints returns the Hints field if non-nil, zero value otherwise.

### GetHintsOk

`func (o *PublicKeyCredentialRequestOptions) GetHintsOk() (*[]string, bool)`

GetHintsOk returns a tuple with the Hints field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHints

`func (o *PublicKeyCredentialRequestOptions) SetHints(v []string)`

SetHints sets Hints field to given value.

### HasHints

`func (o *PublicKeyCredentialRequestOptions) HasHints() bool`

HasHints returns a boolean if a field has been set.

### SetHintsNil

`func (o *PublicKeyCredentialRequestOptions) SetHintsNil(b bool)`

 SetHintsNil sets the value for Hints to be an explicit nil

### UnsetHints
`func (o *PublicKeyCredentialRequestOptions) UnsetHints()`

UnsetHints ensures that no value is present for Hints, not even an explicit nil
### GetRpId

`func (o *PublicKeyCredentialRequestOptions) GetRpId() string`

GetRpId returns the RpId field if non-nil, zero value otherwise.

### GetRpIdOk

`func (o *PublicKeyCredentialRequestOptions) GetRpIdOk() (*string, bool)`

GetRpIdOk returns a tuple with the RpId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRpId

`func (o *PublicKeyCredentialRequestOptions) SetRpId(v string)`

SetRpId sets RpId field to given value.

### HasRpId

`func (o *PublicKeyCredentialRequestOptions) HasRpId() bool`

HasRpId returns a boolean if a field has been set.

### GetTimeout

`func (o *PublicKeyCredentialRequestOptions) GetTimeout() int64`

GetTimeout returns the Timeout field if non-nil, zero value otherwise.

### GetTimeoutOk

`func (o *PublicKeyCredentialRequestOptions) GetTimeoutOk() (*int64, bool)`

GetTimeoutOk returns a tuple with the Timeout field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimeout

`func (o *PublicKeyCredentialRequestOptions) SetTimeout(v int64)`

SetTimeout sets Timeout field to given value.

### HasTimeout

`func (o *PublicKeyCredentialRequestOptions) HasTimeout() bool`

HasTimeout returns a boolean if a field has been set.

### GetUserVerification

`func (o *PublicKeyCredentialRequestOptions) GetUserVerification() string`

GetUserVerification returns the UserVerification field if non-nil, zero value otherwise.

### GetUserVerificationOk

`func (o *PublicKeyCredentialRequestOptions) GetUserVerificationOk() (*string, bool)`

GetUserVerificationOk returns a tuple with the UserVerification field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUserVerification

`func (o *PublicKeyCredentialRequestOptions) SetUserVerification(v string)`

SetUserVerification sets UserVerification field to given value.

### HasUserVerification

`func (o *PublicKeyCredentialRequestOptions) HasUserVerification() bool`

HasUserVerification returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


