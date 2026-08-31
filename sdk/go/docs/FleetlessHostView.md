# FleetlessHostView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Address** | **string** |  | 
**ArrivalRefusals** | [**[]RefusalView**](RefusalView.md) |  | 
**BeeId** | Pointer to **string** |  | [optional] 
**Disarm** | Pointer to [**DisarmView**](DisarmView.md) |  | [optional] 
**HypervisorId** | **string** |  | 
**LastSeenAt** | Pointer to **time.Time** |  | [optional] 
**ProvisioningError** | Pointer to **string** |  | [optional] 
**ProvisioningState** | **string** |  | 
**Status** | Pointer to **string** |  | [optional] 

## Methods

### NewFleetlessHostView

`func NewFleetlessHostView(address string, arrivalRefusals []RefusalView, hypervisorId string, provisioningState string, ) *FleetlessHostView`

NewFleetlessHostView instantiates a new FleetlessHostView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewFleetlessHostViewWithDefaults

`func NewFleetlessHostViewWithDefaults() *FleetlessHostView`

NewFleetlessHostViewWithDefaults instantiates a new FleetlessHostView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAddress

`func (o *FleetlessHostView) GetAddress() string`

GetAddress returns the Address field if non-nil, zero value otherwise.

### GetAddressOk

`func (o *FleetlessHostView) GetAddressOk() (*string, bool)`

GetAddressOk returns a tuple with the Address field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAddress

`func (o *FleetlessHostView) SetAddress(v string)`

SetAddress sets Address field to given value.


### GetArrivalRefusals

`func (o *FleetlessHostView) GetArrivalRefusals() []RefusalView`

GetArrivalRefusals returns the ArrivalRefusals field if non-nil, zero value otherwise.

### GetArrivalRefusalsOk

`func (o *FleetlessHostView) GetArrivalRefusalsOk() (*[]RefusalView, bool)`

GetArrivalRefusalsOk returns a tuple with the ArrivalRefusals field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArrivalRefusals

`func (o *FleetlessHostView) SetArrivalRefusals(v []RefusalView)`

SetArrivalRefusals sets ArrivalRefusals field to given value.


### SetArrivalRefusalsNil

`func (o *FleetlessHostView) SetArrivalRefusalsNil(b bool)`

 SetArrivalRefusalsNil sets the value for ArrivalRefusals to be an explicit nil

### UnsetArrivalRefusals
`func (o *FleetlessHostView) UnsetArrivalRefusals()`

UnsetArrivalRefusals ensures that no value is present for ArrivalRefusals, not even an explicit nil
### GetBeeId

`func (o *FleetlessHostView) GetBeeId() string`

GetBeeId returns the BeeId field if non-nil, zero value otherwise.

### GetBeeIdOk

`func (o *FleetlessHostView) GetBeeIdOk() (*string, bool)`

GetBeeIdOk returns a tuple with the BeeId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBeeId

`func (o *FleetlessHostView) SetBeeId(v string)`

SetBeeId sets BeeId field to given value.

### HasBeeId

`func (o *FleetlessHostView) HasBeeId() bool`

HasBeeId returns a boolean if a field has been set.

### GetDisarm

`func (o *FleetlessHostView) GetDisarm() DisarmView`

GetDisarm returns the Disarm field if non-nil, zero value otherwise.

### GetDisarmOk

`func (o *FleetlessHostView) GetDisarmOk() (*DisarmView, bool)`

GetDisarmOk returns a tuple with the Disarm field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDisarm

`func (o *FleetlessHostView) SetDisarm(v DisarmView)`

SetDisarm sets Disarm field to given value.

### HasDisarm

`func (o *FleetlessHostView) HasDisarm() bool`

HasDisarm returns a boolean if a field has been set.

### GetHypervisorId

`func (o *FleetlessHostView) GetHypervisorId() string`

GetHypervisorId returns the HypervisorId field if non-nil, zero value otherwise.

### GetHypervisorIdOk

`func (o *FleetlessHostView) GetHypervisorIdOk() (*string, bool)`

GetHypervisorIdOk returns a tuple with the HypervisorId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHypervisorId

`func (o *FleetlessHostView) SetHypervisorId(v string)`

SetHypervisorId sets HypervisorId field to given value.


### GetLastSeenAt

`func (o *FleetlessHostView) GetLastSeenAt() time.Time`

GetLastSeenAt returns the LastSeenAt field if non-nil, zero value otherwise.

### GetLastSeenAtOk

`func (o *FleetlessHostView) GetLastSeenAtOk() (*time.Time, bool)`

GetLastSeenAtOk returns a tuple with the LastSeenAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastSeenAt

`func (o *FleetlessHostView) SetLastSeenAt(v time.Time)`

SetLastSeenAt sets LastSeenAt field to given value.

### HasLastSeenAt

`func (o *FleetlessHostView) HasLastSeenAt() bool`

HasLastSeenAt returns a boolean if a field has been set.

### GetProvisioningError

`func (o *FleetlessHostView) GetProvisioningError() string`

GetProvisioningError returns the ProvisioningError field if non-nil, zero value otherwise.

### GetProvisioningErrorOk

`func (o *FleetlessHostView) GetProvisioningErrorOk() (*string, bool)`

GetProvisioningErrorOk returns a tuple with the ProvisioningError field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProvisioningError

`func (o *FleetlessHostView) SetProvisioningError(v string)`

SetProvisioningError sets ProvisioningError field to given value.

### HasProvisioningError

`func (o *FleetlessHostView) HasProvisioningError() bool`

HasProvisioningError returns a boolean if a field has been set.

### GetProvisioningState

`func (o *FleetlessHostView) GetProvisioningState() string`

GetProvisioningState returns the ProvisioningState field if non-nil, zero value otherwise.

### GetProvisioningStateOk

`func (o *FleetlessHostView) GetProvisioningStateOk() (*string, bool)`

GetProvisioningStateOk returns a tuple with the ProvisioningState field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProvisioningState

`func (o *FleetlessHostView) SetProvisioningState(v string)`

SetProvisioningState sets ProvisioningState field to given value.


### GetStatus

`func (o *FleetlessHostView) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *FleetlessHostView) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *FleetlessHostView) SetStatus(v string)`

SetStatus sets Status field to given value.

### HasStatus

`func (o *FleetlessHostView) HasStatus() bool`

HasStatus returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


