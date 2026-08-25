# KeyView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**CreatedAt** | **time.Time** |  | 
**ExpiresAt** | Pointer to **time.Time** |  | [optional] 
**Fingerprint** | **string** |  | 
**Id** | **string** |  | 
**Name** | **string** |  | 
**PublicKey** | **string** |  | 
**Purpose** | **string** |  | 
**Revoked** | **bool** |  | 
**Usable** | **bool** | Whether this key may still be used: not revoked, not expired, and correctly scoped if it is a join key. | 

## Methods

### NewKeyView

`func NewKeyView(createdAt time.Time, fingerprint string, id string, name string, publicKey string, purpose string, revoked bool, usable bool, ) *KeyView`

NewKeyView instantiates a new KeyView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewKeyViewWithDefaults

`func NewKeyViewWithDefaults() *KeyView`

NewKeyViewWithDefaults instantiates a new KeyView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *KeyView) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *KeyView) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *KeyView) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *KeyView) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetCreatedAt

`func (o *KeyView) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *KeyView) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *KeyView) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.


### GetExpiresAt

`func (o *KeyView) GetExpiresAt() time.Time`

GetExpiresAt returns the ExpiresAt field if non-nil, zero value otherwise.

### GetExpiresAtOk

`func (o *KeyView) GetExpiresAtOk() (*time.Time, bool)`

GetExpiresAtOk returns a tuple with the ExpiresAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExpiresAt

`func (o *KeyView) SetExpiresAt(v time.Time)`

SetExpiresAt sets ExpiresAt field to given value.

### HasExpiresAt

`func (o *KeyView) HasExpiresAt() bool`

HasExpiresAt returns a boolean if a field has been set.

### GetFingerprint

`func (o *KeyView) GetFingerprint() string`

GetFingerprint returns the Fingerprint field if non-nil, zero value otherwise.

### GetFingerprintOk

`func (o *KeyView) GetFingerprintOk() (*string, bool)`

GetFingerprintOk returns a tuple with the Fingerprint field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFingerprint

`func (o *KeyView) SetFingerprint(v string)`

SetFingerprint sets Fingerprint field to given value.


### GetId

`func (o *KeyView) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *KeyView) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *KeyView) SetId(v string)`

SetId sets Id field to given value.


### GetName

`func (o *KeyView) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *KeyView) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *KeyView) SetName(v string)`

SetName sets Name field to given value.


### GetPublicKey

`func (o *KeyView) GetPublicKey() string`

GetPublicKey returns the PublicKey field if non-nil, zero value otherwise.

### GetPublicKeyOk

`func (o *KeyView) GetPublicKeyOk() (*string, bool)`

GetPublicKeyOk returns a tuple with the PublicKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPublicKey

`func (o *KeyView) SetPublicKey(v string)`

SetPublicKey sets PublicKey field to given value.


### GetPurpose

`func (o *KeyView) GetPurpose() string`

GetPurpose returns the Purpose field if non-nil, zero value otherwise.

### GetPurposeOk

`func (o *KeyView) GetPurposeOk() (*string, bool)`

GetPurposeOk returns a tuple with the Purpose field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPurpose

`func (o *KeyView) SetPurpose(v string)`

SetPurpose sets Purpose field to given value.


### GetRevoked

`func (o *KeyView) GetRevoked() bool`

GetRevoked returns the Revoked field if non-nil, zero value otherwise.

### GetRevokedOk

`func (o *KeyView) GetRevokedOk() (*bool, bool)`

GetRevokedOk returns a tuple with the Revoked field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRevoked

`func (o *KeyView) SetRevoked(v bool)`

SetRevoked sets Revoked field to given value.


### GetUsable

`func (o *KeyView) GetUsable() bool`

GetUsable returns the Usable field if non-nil, zero value otherwise.

### GetUsableOk

`func (o *KeyView) GetUsableOk() (*bool, bool)`

GetUsableOk returns a tuple with the Usable field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUsable

`func (o *KeyView) SetUsable(v bool)`

SetUsable sets Usable field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


