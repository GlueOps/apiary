# DisarmView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Detail** | Pointer to **string** |  | [optional] 
**Disarmed** | **bool** |  | 
**Disposition** | Pointer to **string** |  | [optional] 
**HeldSecrets** | **int64** |  | 
**Rotated** | Pointer to **bool** |  | [optional] 

## Methods

### NewDisarmView

`func NewDisarmView(disarmed bool, heldSecrets int64, ) *DisarmView`

NewDisarmView instantiates a new DisarmView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDisarmViewWithDefaults

`func NewDisarmViewWithDefaults() *DisarmView`

NewDisarmViewWithDefaults instantiates a new DisarmView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDetail

`func (o *DisarmView) GetDetail() string`

GetDetail returns the Detail field if non-nil, zero value otherwise.

### GetDetailOk

`func (o *DisarmView) GetDetailOk() (*string, bool)`

GetDetailOk returns a tuple with the Detail field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDetail

`func (o *DisarmView) SetDetail(v string)`

SetDetail sets Detail field to given value.

### HasDetail

`func (o *DisarmView) HasDetail() bool`

HasDetail returns a boolean if a field has been set.

### GetDisarmed

`func (o *DisarmView) GetDisarmed() bool`

GetDisarmed returns the Disarmed field if non-nil, zero value otherwise.

### GetDisarmedOk

`func (o *DisarmView) GetDisarmedOk() (*bool, bool)`

GetDisarmedOk returns a tuple with the Disarmed field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDisarmed

`func (o *DisarmView) SetDisarmed(v bool)`

SetDisarmed sets Disarmed field to given value.


### GetDisposition

`func (o *DisarmView) GetDisposition() string`

GetDisposition returns the Disposition field if non-nil, zero value otherwise.

### GetDispositionOk

`func (o *DisarmView) GetDispositionOk() (*string, bool)`

GetDispositionOk returns a tuple with the Disposition field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDisposition

`func (o *DisarmView) SetDisposition(v string)`

SetDisposition sets Disposition field to given value.

### HasDisposition

`func (o *DisarmView) HasDisposition() bool`

HasDisposition returns a boolean if a field has been set.

### GetHeldSecrets

`func (o *DisarmView) GetHeldSecrets() int64`

GetHeldSecrets returns the HeldSecrets field if non-nil, zero value otherwise.

### GetHeldSecretsOk

`func (o *DisarmView) GetHeldSecretsOk() (*int64, bool)`

GetHeldSecretsOk returns a tuple with the HeldSecrets field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHeldSecrets

`func (o *DisarmView) SetHeldSecrets(v int64)`

SetHeldSecrets sets HeldSecrets field to given value.


### GetRotated

`func (o *DisarmView) GetRotated() bool`

GetRotated returns the Rotated field if non-nil, zero value otherwise.

### GetRotatedOk

`func (o *DisarmView) GetRotatedOk() (*bool, bool)`

GetRotatedOk returns a tuple with the Rotated field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRotated

`func (o *DisarmView) SetRotated(v bool)`

SetRotated sets Rotated field to given value.

### HasRotated

`func (o *DisarmView) HasRotated() bool`

HasRotated returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


