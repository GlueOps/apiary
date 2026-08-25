# BeeView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Address** | Pointer to **string** | How the control plane reaches the machine this agent runs on. | [optional] 
**AgentVersion** | Pointer to **string** |  | [optional] 
**HiveId** | Pointer to **string** |  | [optional] 
**HypervisorId** | Pointer to **string** |  | [optional] 
**Id** | **string** |  | 
**LastSeenAt** | Pointer to **time.Time** |  | [optional] 
**Status** | **string** | Derived, not stored. &#39;unavailable&#39; means the host DECLARED it would be away — a reboot in progress is not an outage. | 
**UnavailableReason** | Pointer to **string** |  | [optional] 
**UnavailableUntil** | Pointer to **time.Time** |  | [optional] 

## Methods

### NewBeeView

`func NewBeeView(id string, status string, ) *BeeView`

NewBeeView instantiates a new BeeView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewBeeViewWithDefaults

`func NewBeeViewWithDefaults() *BeeView`

NewBeeViewWithDefaults instantiates a new BeeView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAddress

`func (o *BeeView) GetAddress() string`

GetAddress returns the Address field if non-nil, zero value otherwise.

### GetAddressOk

`func (o *BeeView) GetAddressOk() (*string, bool)`

GetAddressOk returns a tuple with the Address field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAddress

`func (o *BeeView) SetAddress(v string)`

SetAddress sets Address field to given value.

### HasAddress

`func (o *BeeView) HasAddress() bool`

HasAddress returns a boolean if a field has been set.

### GetAgentVersion

`func (o *BeeView) GetAgentVersion() string`

GetAgentVersion returns the AgentVersion field if non-nil, zero value otherwise.

### GetAgentVersionOk

`func (o *BeeView) GetAgentVersionOk() (*string, bool)`

GetAgentVersionOk returns a tuple with the AgentVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgentVersion

`func (o *BeeView) SetAgentVersion(v string)`

SetAgentVersion sets AgentVersion field to given value.

### HasAgentVersion

`func (o *BeeView) HasAgentVersion() bool`

HasAgentVersion returns a boolean if a field has been set.

### GetHiveId

`func (o *BeeView) GetHiveId() string`

GetHiveId returns the HiveId field if non-nil, zero value otherwise.

### GetHiveIdOk

`func (o *BeeView) GetHiveIdOk() (*string, bool)`

GetHiveIdOk returns a tuple with the HiveId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHiveId

`func (o *BeeView) SetHiveId(v string)`

SetHiveId sets HiveId field to given value.

### HasHiveId

`func (o *BeeView) HasHiveId() bool`

HasHiveId returns a boolean if a field has been set.

### GetHypervisorId

`func (o *BeeView) GetHypervisorId() string`

GetHypervisorId returns the HypervisorId field if non-nil, zero value otherwise.

### GetHypervisorIdOk

`func (o *BeeView) GetHypervisorIdOk() (*string, bool)`

GetHypervisorIdOk returns a tuple with the HypervisorId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHypervisorId

`func (o *BeeView) SetHypervisorId(v string)`

SetHypervisorId sets HypervisorId field to given value.

### HasHypervisorId

`func (o *BeeView) HasHypervisorId() bool`

HasHypervisorId returns a boolean if a field has been set.

### GetId

`func (o *BeeView) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *BeeView) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *BeeView) SetId(v string)`

SetId sets Id field to given value.


### GetLastSeenAt

`func (o *BeeView) GetLastSeenAt() time.Time`

GetLastSeenAt returns the LastSeenAt field if non-nil, zero value otherwise.

### GetLastSeenAtOk

`func (o *BeeView) GetLastSeenAtOk() (*time.Time, bool)`

GetLastSeenAtOk returns a tuple with the LastSeenAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastSeenAt

`func (o *BeeView) SetLastSeenAt(v time.Time)`

SetLastSeenAt sets LastSeenAt field to given value.

### HasLastSeenAt

`func (o *BeeView) HasLastSeenAt() bool`

HasLastSeenAt returns a boolean if a field has been set.

### GetStatus

`func (o *BeeView) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *BeeView) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *BeeView) SetStatus(v string)`

SetStatus sets Status field to given value.


### GetUnavailableReason

`func (o *BeeView) GetUnavailableReason() string`

GetUnavailableReason returns the UnavailableReason field if non-nil, zero value otherwise.

### GetUnavailableReasonOk

`func (o *BeeView) GetUnavailableReasonOk() (*string, bool)`

GetUnavailableReasonOk returns a tuple with the UnavailableReason field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUnavailableReason

`func (o *BeeView) SetUnavailableReason(v string)`

SetUnavailableReason sets UnavailableReason field to given value.

### HasUnavailableReason

`func (o *BeeView) HasUnavailableReason() bool`

HasUnavailableReason returns a boolean if a field has been set.

### GetUnavailableUntil

`func (o *BeeView) GetUnavailableUntil() time.Time`

GetUnavailableUntil returns the UnavailableUntil field if non-nil, zero value otherwise.

### GetUnavailableUntilOk

`func (o *BeeView) GetUnavailableUntilOk() (*time.Time, bool)`

GetUnavailableUntilOk returns a tuple with the UnavailableUntil field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUnavailableUntil

`func (o *BeeView) SetUnavailableUntil(v time.Time)`

SetUnavailableUntil sets UnavailableUntil field to given value.

### HasUnavailableUntil

`func (o *BeeView) HasUnavailableUntil() bool`

HasUnavailableUntil returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


