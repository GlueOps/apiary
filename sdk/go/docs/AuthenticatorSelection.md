# AuthenticatorSelection

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AuthenticatorAttachment** | Pointer to **string** |  | [optional] 
**RequireResidentKey** | Pointer to **bool** |  | [optional] 
**ResidentKey** | Pointer to **string** |  | [optional] 
**UserVerification** | Pointer to **string** |  | [optional] 

## Methods

### NewAuthenticatorSelection

`func NewAuthenticatorSelection() *AuthenticatorSelection`

NewAuthenticatorSelection instantiates a new AuthenticatorSelection object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAuthenticatorSelectionWithDefaults

`func NewAuthenticatorSelectionWithDefaults() *AuthenticatorSelection`

NewAuthenticatorSelectionWithDefaults instantiates a new AuthenticatorSelection object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAuthenticatorAttachment

`func (o *AuthenticatorSelection) GetAuthenticatorAttachment() string`

GetAuthenticatorAttachment returns the AuthenticatorAttachment field if non-nil, zero value otherwise.

### GetAuthenticatorAttachmentOk

`func (o *AuthenticatorSelection) GetAuthenticatorAttachmentOk() (*string, bool)`

GetAuthenticatorAttachmentOk returns a tuple with the AuthenticatorAttachment field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAuthenticatorAttachment

`func (o *AuthenticatorSelection) SetAuthenticatorAttachment(v string)`

SetAuthenticatorAttachment sets AuthenticatorAttachment field to given value.

### HasAuthenticatorAttachment

`func (o *AuthenticatorSelection) HasAuthenticatorAttachment() bool`

HasAuthenticatorAttachment returns a boolean if a field has been set.

### GetRequireResidentKey

`func (o *AuthenticatorSelection) GetRequireResidentKey() bool`

GetRequireResidentKey returns the RequireResidentKey field if non-nil, zero value otherwise.

### GetRequireResidentKeyOk

`func (o *AuthenticatorSelection) GetRequireResidentKeyOk() (*bool, bool)`

GetRequireResidentKeyOk returns a tuple with the RequireResidentKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRequireResidentKey

`func (o *AuthenticatorSelection) SetRequireResidentKey(v bool)`

SetRequireResidentKey sets RequireResidentKey field to given value.

### HasRequireResidentKey

`func (o *AuthenticatorSelection) HasRequireResidentKey() bool`

HasRequireResidentKey returns a boolean if a field has been set.

### GetResidentKey

`func (o *AuthenticatorSelection) GetResidentKey() string`

GetResidentKey returns the ResidentKey field if non-nil, zero value otherwise.

### GetResidentKeyOk

`func (o *AuthenticatorSelection) GetResidentKeyOk() (*string, bool)`

GetResidentKeyOk returns a tuple with the ResidentKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResidentKey

`func (o *AuthenticatorSelection) SetResidentKey(v string)`

SetResidentKey sets ResidentKey field to given value.

### HasResidentKey

`func (o *AuthenticatorSelection) HasResidentKey() bool`

HasResidentKey returns a boolean if a field has been set.

### GetUserVerification

`func (o *AuthenticatorSelection) GetUserVerification() string`

GetUserVerification returns the UserVerification field if non-nil, zero value otherwise.

### GetUserVerificationOk

`func (o *AuthenticatorSelection) GetUserVerificationOk() (*string, bool)`

GetUserVerificationOk returns a tuple with the UserVerification field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUserVerification

`func (o *AuthenticatorSelection) SetUserVerification(v string)`

SetUserVerification sets UserVerification field to given value.

### HasUserVerification

`func (o *AuthenticatorSelection) HasUserVerification() bool`

HasUserVerification returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


