# CredentialAssertion

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Mediation** | Pointer to **string** |  | [optional] 
**PublicKey** | [**PublicKeyCredentialRequestOptions**](PublicKeyCredentialRequestOptions.md) |  | 

## Methods

### NewCredentialAssertion

`func NewCredentialAssertion(publicKey PublicKeyCredentialRequestOptions, ) *CredentialAssertion`

NewCredentialAssertion instantiates a new CredentialAssertion object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCredentialAssertionWithDefaults

`func NewCredentialAssertionWithDefaults() *CredentialAssertion`

NewCredentialAssertionWithDefaults instantiates a new CredentialAssertion object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetMediation

`func (o *CredentialAssertion) GetMediation() string`

GetMediation returns the Mediation field if non-nil, zero value otherwise.

### GetMediationOk

`func (o *CredentialAssertion) GetMediationOk() (*string, bool)`

GetMediationOk returns a tuple with the Mediation field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMediation

`func (o *CredentialAssertion) SetMediation(v string)`

SetMediation sets Mediation field to given value.

### HasMediation

`func (o *CredentialAssertion) HasMediation() bool`

HasMediation returns a boolean if a field has been set.

### GetPublicKey

`func (o *CredentialAssertion) GetPublicKey() PublicKeyCredentialRequestOptions`

GetPublicKey returns the PublicKey field if non-nil, zero value otherwise.

### GetPublicKeyOk

`func (o *CredentialAssertion) GetPublicKeyOk() (*PublicKeyCredentialRequestOptions, bool)`

GetPublicKeyOk returns a tuple with the PublicKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPublicKey

`func (o *CredentialAssertion) SetPublicKey(v PublicKeyCredentialRequestOptions)`

SetPublicKey sets PublicKey field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


