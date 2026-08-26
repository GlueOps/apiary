# SnapshotView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Bytes** | **int64** |  | 
**CreatedAt** | **time.Time** |  | 
**Current** | **bool** | Whether this is the revision the fleet is being told to converge on. | 
**Digest** | **string** |  | 
**Reason** | Pointer to **string** |  | [optional] 
**RevertOf** | Pointer to **int64** |  | [optional] 
**Revision** | **int64** |  | 

## Methods

### NewSnapshotView

`func NewSnapshotView(bytes int64, createdAt time.Time, current bool, digest string, revision int64, ) *SnapshotView`

NewSnapshotView instantiates a new SnapshotView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSnapshotViewWithDefaults

`func NewSnapshotViewWithDefaults() *SnapshotView`

NewSnapshotViewWithDefaults instantiates a new SnapshotView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetBytes

`func (o *SnapshotView) GetBytes() int64`

GetBytes returns the Bytes field if non-nil, zero value otherwise.

### GetBytesOk

`func (o *SnapshotView) GetBytesOk() (*int64, bool)`

GetBytesOk returns a tuple with the Bytes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBytes

`func (o *SnapshotView) SetBytes(v int64)`

SetBytes sets Bytes field to given value.


### GetCreatedAt

`func (o *SnapshotView) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *SnapshotView) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *SnapshotView) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.


### GetCurrent

`func (o *SnapshotView) GetCurrent() bool`

GetCurrent returns the Current field if non-nil, zero value otherwise.

### GetCurrentOk

`func (o *SnapshotView) GetCurrentOk() (*bool, bool)`

GetCurrentOk returns a tuple with the Current field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCurrent

`func (o *SnapshotView) SetCurrent(v bool)`

SetCurrent sets Current field to given value.


### GetDigest

`func (o *SnapshotView) GetDigest() string`

GetDigest returns the Digest field if non-nil, zero value otherwise.

### GetDigestOk

`func (o *SnapshotView) GetDigestOk() (*string, bool)`

GetDigestOk returns a tuple with the Digest field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDigest

`func (o *SnapshotView) SetDigest(v string)`

SetDigest sets Digest field to given value.


### GetReason

`func (o *SnapshotView) GetReason() string`

GetReason returns the Reason field if non-nil, zero value otherwise.

### GetReasonOk

`func (o *SnapshotView) GetReasonOk() (*string, bool)`

GetReasonOk returns a tuple with the Reason field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReason

`func (o *SnapshotView) SetReason(v string)`

SetReason sets Reason field to given value.

### HasReason

`func (o *SnapshotView) HasReason() bool`

HasReason returns a boolean if a field has been set.

### GetRevertOf

`func (o *SnapshotView) GetRevertOf() int64`

GetRevertOf returns the RevertOf field if non-nil, zero value otherwise.

### GetRevertOfOk

`func (o *SnapshotView) GetRevertOfOk() (*int64, bool)`

GetRevertOfOk returns a tuple with the RevertOf field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRevertOf

`func (o *SnapshotView) SetRevertOf(v int64)`

SetRevertOf sets RevertOf field to given value.

### HasRevertOf

`func (o *SnapshotView) HasRevertOf() bool`

HasRevertOf returns a boolean if a field has been set.

### GetRevision

`func (o *SnapshotView) GetRevision() int64`

GetRevision returns the Revision field if non-nil, zero value otherwise.

### GetRevisionOk

`func (o *SnapshotView) GetRevisionOk() (*int64, bool)`

GetRevisionOk returns a tuple with the Revision field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRevision

`func (o *SnapshotView) SetRevision(v int64)`

SetRevision sets Revision field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


