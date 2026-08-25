# CredentialCreation

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Mediation** | Pointer to **string** |  | [optional] 
**PublicKey** | [**PublicKeyCredentialCreationOptions**](PublicKeyCredentialCreationOptions.md) |  | 

## Methods

### NewCredentialCreation

`func NewCredentialCreation(publicKey PublicKeyCredentialCreationOptions, ) *CredentialCreation`

NewCredentialCreation instantiates a new CredentialCreation object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCredentialCreationWithDefaults

`func NewCredentialCreationWithDefaults() *CredentialCreation`

NewCredentialCreationWithDefaults instantiates a new CredentialCreation object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetMediation

`func (o *CredentialCreation) GetMediation() string`

GetMediation returns the Mediation field if non-nil, zero value otherwise.

### GetMediationOk

`func (o *CredentialCreation) GetMediationOk() (*string, bool)`

GetMediationOk returns a tuple with the Mediation field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMediation

`func (o *CredentialCreation) SetMediation(v string)`

SetMediation sets Mediation field to given value.

### HasMediation

`func (o *CredentialCreation) HasMediation() bool`

HasMediation returns a boolean if a field has been set.

### GetPublicKey

`func (o *CredentialCreation) GetPublicKey() PublicKeyCredentialCreationOptions`

GetPublicKey returns the PublicKey field if non-nil, zero value otherwise.

### GetPublicKeyOk

`func (o *CredentialCreation) GetPublicKeyOk() (*PublicKeyCredentialCreationOptions, bool)`

GetPublicKeyOk returns a tuple with the PublicKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPublicKey

`func (o *CredentialCreation) SetPublicKey(v PublicKeyCredentialCreationOptions)`

SetPublicKey sets PublicKey field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


