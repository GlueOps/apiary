# CorosyncLinkInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Address** | **string** | This node&#39;s address on that ring. | 
**Number** | **int64** | Ring number. Proxmox supports link0-link7. | 
**Priority** | Pointer to **int64** | knet link preference; higher wins. Absent leaves corosync&#39;s default, which is not the same as zero. | [optional] 

## Methods

### NewCorosyncLinkInput

`func NewCorosyncLinkInput(address string, number int64, ) *CorosyncLinkInput`

NewCorosyncLinkInput instantiates a new CorosyncLinkInput object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCorosyncLinkInputWithDefaults

`func NewCorosyncLinkInputWithDefaults() *CorosyncLinkInput`

NewCorosyncLinkInputWithDefaults instantiates a new CorosyncLinkInput object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAddress

`func (o *CorosyncLinkInput) GetAddress() string`

GetAddress returns the Address field if non-nil, zero value otherwise.

### GetAddressOk

`func (o *CorosyncLinkInput) GetAddressOk() (*string, bool)`

GetAddressOk returns a tuple with the Address field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAddress

`func (o *CorosyncLinkInput) SetAddress(v string)`

SetAddress sets Address field to given value.


### GetNumber

`func (o *CorosyncLinkInput) GetNumber() int64`

GetNumber returns the Number field if non-nil, zero value otherwise.

### GetNumberOk

`func (o *CorosyncLinkInput) GetNumberOk() (*int64, bool)`

GetNumberOk returns a tuple with the Number field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNumber

`func (o *CorosyncLinkInput) SetNumber(v int64)`

SetNumber sets Number field to given value.


### GetPriority

`func (o *CorosyncLinkInput) GetPriority() int64`

GetPriority returns the Priority field if non-nil, zero value otherwise.

### GetPriorityOk

`func (o *CorosyncLinkInput) GetPriorityOk() (*int64, bool)`

GetPriorityOk returns a tuple with the Priority field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPriority

`func (o *CorosyncLinkInput) SetPriority(v int64)`

SetPriority sets Priority field to given value.

### HasPriority

`func (o *CorosyncLinkInput) HasPriority() bool`

HasPriority returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


