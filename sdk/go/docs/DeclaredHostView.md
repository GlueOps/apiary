# DeclaredHostView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Address** | **string** |  | 
**Created** | **bool** | False when this address was already known. Not a failure: re-pasting a list is ordinary, and the existing machine is left exactly as it was rather than pushed back to pending. | 
**Detail** | Pointer to **string** |  | [optional] 
**Id** | Pointer to **string** |  | [optional] 
**Queued** | **bool** | Whether this machine is now waiting for the onboarding ticker. | 

## Methods

### NewDeclaredHostView

`func NewDeclaredHostView(address string, created bool, queued bool, ) *DeclaredHostView`

NewDeclaredHostView instantiates a new DeclaredHostView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDeclaredHostViewWithDefaults

`func NewDeclaredHostViewWithDefaults() *DeclaredHostView`

NewDeclaredHostViewWithDefaults instantiates a new DeclaredHostView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAddress

`func (o *DeclaredHostView) GetAddress() string`

GetAddress returns the Address field if non-nil, zero value otherwise.

### GetAddressOk

`func (o *DeclaredHostView) GetAddressOk() (*string, bool)`

GetAddressOk returns a tuple with the Address field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAddress

`func (o *DeclaredHostView) SetAddress(v string)`

SetAddress sets Address field to given value.


### GetCreated

`func (o *DeclaredHostView) GetCreated() bool`

GetCreated returns the Created field if non-nil, zero value otherwise.

### GetCreatedOk

`func (o *DeclaredHostView) GetCreatedOk() (*bool, bool)`

GetCreatedOk returns a tuple with the Created field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreated

`func (o *DeclaredHostView) SetCreated(v bool)`

SetCreated sets Created field to given value.


### GetDetail

`func (o *DeclaredHostView) GetDetail() string`

GetDetail returns the Detail field if non-nil, zero value otherwise.

### GetDetailOk

`func (o *DeclaredHostView) GetDetailOk() (*string, bool)`

GetDetailOk returns a tuple with the Detail field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDetail

`func (o *DeclaredHostView) SetDetail(v string)`

SetDetail sets Detail field to given value.

### HasDetail

`func (o *DeclaredHostView) HasDetail() bool`

HasDetail returns a boolean if a field has been set.

### GetId

`func (o *DeclaredHostView) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *DeclaredHostView) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *DeclaredHostView) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *DeclaredHostView) HasId() bool`

HasId returns a boolean if a field has been set.

### GetQueued

`func (o *DeclaredHostView) GetQueued() bool`

GetQueued returns the Queued field if non-nil, zero value otherwise.

### GetQueuedOk

`func (o *DeclaredHostView) GetQueuedOk() (*bool, bool)`

GetQueuedOk returns a tuple with the Queued field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetQueued

`func (o *DeclaredHostView) SetQueued(v bool)`

SetQueued sets Queued field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


