# HypervisorView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Address** | Pointer to **string** |  | [optional] 
**Allocatable** | [**CapacityView**](CapacityView.md) | Ceiling after overcommit and reservation. | 
**BeeId** | Pointer to **string** |  | [optional] 
**BeeStatus** | Pointer to **string** | Reachability: derived from the agent&#39;s heartbeat, never stored. Absent when no agent is enrolled, which is not the same as offline. | [optional] 
**DepartureRefusals** | [**[]RefusalView**](RefusalView.md) |  | 
**DrainState** | **string** | Derived, never stored. &#39;unknown&#39; means the observation is too stale to say whether the node is empty — it is not a synonym for &#39;draining&#39;. | 
**HeldPlacements** | **int64** | Placements still holding capacity here, from this system&#39;s own ledger. Disagreement with observed_guests is meaningful in both directions. | 
**HiveId** | Pointer to **string** |  | [optional] 
**HiveName** | Pointer to **string** |  | [optional] 
**Id** | **string** |  | 
**LastSyncedAt** | Pointer to **time.Time** |  | [optional] 
**Name** | Pointer to **string** | The provider&#39;s node name, not a hostname we chose. Absent until this machine joins a cluster. | [optional] 
**ObservationStale** | **bool** |  | 
**Observed** | [**CapacityView**](CapacityView.md) | Allocated to guests, as the provider reports it. | 
**ObservedGuests** | **int64** | Guests the provider reports, STOPPED ONES INCLUDED. A stopped guest blocks a cluster join exactly as a running one does. | 
**ProvisioningAttempt** | Pointer to **int64** | How many onboarding runs this machine has cost. Capped; a machine at the cap needs an explicit redrive. | [optional] 
**ProvisioningError** | Pointer to **string** | The last onboarding failure, kept so &#39;failed&#39; is actionable without going to the logs. | [optional] 
**ProvisioningState** | **string** | Has the agent been installed? Stored, because it is a workflow rather than an observation. Independent of whether the machine is cordoned or reachable. | 
**Reserved** | [**CapacityView**](CapacityView.md) | Held back for the host itself, set by an operator. | 
**Schedulable** | **bool** | False means cordoned: the placement query excludes it, so no new guests land here. | 

## Methods

### NewHypervisorView

`func NewHypervisorView(allocatable CapacityView, departureRefusals []RefusalView, drainState string, heldPlacements int64, id string, observationStale bool, observed CapacityView, observedGuests int64, provisioningState string, reserved CapacityView, schedulable bool, ) *HypervisorView`

NewHypervisorView instantiates a new HypervisorView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewHypervisorViewWithDefaults

`func NewHypervisorViewWithDefaults() *HypervisorView`

NewHypervisorViewWithDefaults instantiates a new HypervisorView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *HypervisorView) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *HypervisorView) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *HypervisorView) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *HypervisorView) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetAddress

`func (o *HypervisorView) GetAddress() string`

GetAddress returns the Address field if non-nil, zero value otherwise.

### GetAddressOk

`func (o *HypervisorView) GetAddressOk() (*string, bool)`

GetAddressOk returns a tuple with the Address field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAddress

`func (o *HypervisorView) SetAddress(v string)`

SetAddress sets Address field to given value.

### HasAddress

`func (o *HypervisorView) HasAddress() bool`

HasAddress returns a boolean if a field has been set.

### GetAllocatable

`func (o *HypervisorView) GetAllocatable() CapacityView`

GetAllocatable returns the Allocatable field if non-nil, zero value otherwise.

### GetAllocatableOk

`func (o *HypervisorView) GetAllocatableOk() (*CapacityView, bool)`

GetAllocatableOk returns a tuple with the Allocatable field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAllocatable

`func (o *HypervisorView) SetAllocatable(v CapacityView)`

SetAllocatable sets Allocatable field to given value.


### GetBeeId

`func (o *HypervisorView) GetBeeId() string`

GetBeeId returns the BeeId field if non-nil, zero value otherwise.

### GetBeeIdOk

`func (o *HypervisorView) GetBeeIdOk() (*string, bool)`

GetBeeIdOk returns a tuple with the BeeId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBeeId

`func (o *HypervisorView) SetBeeId(v string)`

SetBeeId sets BeeId field to given value.

### HasBeeId

`func (o *HypervisorView) HasBeeId() bool`

HasBeeId returns a boolean if a field has been set.

### GetBeeStatus

`func (o *HypervisorView) GetBeeStatus() string`

GetBeeStatus returns the BeeStatus field if non-nil, zero value otherwise.

### GetBeeStatusOk

`func (o *HypervisorView) GetBeeStatusOk() (*string, bool)`

GetBeeStatusOk returns a tuple with the BeeStatus field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBeeStatus

`func (o *HypervisorView) SetBeeStatus(v string)`

SetBeeStatus sets BeeStatus field to given value.

### HasBeeStatus

`func (o *HypervisorView) HasBeeStatus() bool`

HasBeeStatus returns a boolean if a field has been set.

### GetDepartureRefusals

`func (o *HypervisorView) GetDepartureRefusals() []RefusalView`

GetDepartureRefusals returns the DepartureRefusals field if non-nil, zero value otherwise.

### GetDepartureRefusalsOk

`func (o *HypervisorView) GetDepartureRefusalsOk() (*[]RefusalView, bool)`

GetDepartureRefusalsOk returns a tuple with the DepartureRefusals field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDepartureRefusals

`func (o *HypervisorView) SetDepartureRefusals(v []RefusalView)`

SetDepartureRefusals sets DepartureRefusals field to given value.


### SetDepartureRefusalsNil

`func (o *HypervisorView) SetDepartureRefusalsNil(b bool)`

 SetDepartureRefusalsNil sets the value for DepartureRefusals to be an explicit nil

### UnsetDepartureRefusals
`func (o *HypervisorView) UnsetDepartureRefusals()`

UnsetDepartureRefusals ensures that no value is present for DepartureRefusals, not even an explicit nil
### GetDrainState

`func (o *HypervisorView) GetDrainState() string`

GetDrainState returns the DrainState field if non-nil, zero value otherwise.

### GetDrainStateOk

`func (o *HypervisorView) GetDrainStateOk() (*string, bool)`

GetDrainStateOk returns a tuple with the DrainState field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDrainState

`func (o *HypervisorView) SetDrainState(v string)`

SetDrainState sets DrainState field to given value.


### GetHeldPlacements

`func (o *HypervisorView) GetHeldPlacements() int64`

GetHeldPlacements returns the HeldPlacements field if non-nil, zero value otherwise.

### GetHeldPlacementsOk

`func (o *HypervisorView) GetHeldPlacementsOk() (*int64, bool)`

GetHeldPlacementsOk returns a tuple with the HeldPlacements field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHeldPlacements

`func (o *HypervisorView) SetHeldPlacements(v int64)`

SetHeldPlacements sets HeldPlacements field to given value.


### GetHiveId

`func (o *HypervisorView) GetHiveId() string`

GetHiveId returns the HiveId field if non-nil, zero value otherwise.

### GetHiveIdOk

`func (o *HypervisorView) GetHiveIdOk() (*string, bool)`

GetHiveIdOk returns a tuple with the HiveId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHiveId

`func (o *HypervisorView) SetHiveId(v string)`

SetHiveId sets HiveId field to given value.

### HasHiveId

`func (o *HypervisorView) HasHiveId() bool`

HasHiveId returns a boolean if a field has been set.

### GetHiveName

`func (o *HypervisorView) GetHiveName() string`

GetHiveName returns the HiveName field if non-nil, zero value otherwise.

### GetHiveNameOk

`func (o *HypervisorView) GetHiveNameOk() (*string, bool)`

GetHiveNameOk returns a tuple with the HiveName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHiveName

`func (o *HypervisorView) SetHiveName(v string)`

SetHiveName sets HiveName field to given value.

### HasHiveName

`func (o *HypervisorView) HasHiveName() bool`

HasHiveName returns a boolean if a field has been set.

### GetId

`func (o *HypervisorView) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *HypervisorView) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *HypervisorView) SetId(v string)`

SetId sets Id field to given value.


### GetLastSyncedAt

`func (o *HypervisorView) GetLastSyncedAt() time.Time`

GetLastSyncedAt returns the LastSyncedAt field if non-nil, zero value otherwise.

### GetLastSyncedAtOk

`func (o *HypervisorView) GetLastSyncedAtOk() (*time.Time, bool)`

GetLastSyncedAtOk returns a tuple with the LastSyncedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastSyncedAt

`func (o *HypervisorView) SetLastSyncedAt(v time.Time)`

SetLastSyncedAt sets LastSyncedAt field to given value.

### HasLastSyncedAt

`func (o *HypervisorView) HasLastSyncedAt() bool`

HasLastSyncedAt returns a boolean if a field has been set.

### GetName

`func (o *HypervisorView) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *HypervisorView) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *HypervisorView) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *HypervisorView) HasName() bool`

HasName returns a boolean if a field has been set.

### GetObservationStale

`func (o *HypervisorView) GetObservationStale() bool`

GetObservationStale returns the ObservationStale field if non-nil, zero value otherwise.

### GetObservationStaleOk

`func (o *HypervisorView) GetObservationStaleOk() (*bool, bool)`

GetObservationStaleOk returns a tuple with the ObservationStale field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetObservationStale

`func (o *HypervisorView) SetObservationStale(v bool)`

SetObservationStale sets ObservationStale field to given value.


### GetObserved

`func (o *HypervisorView) GetObserved() CapacityView`

GetObserved returns the Observed field if non-nil, zero value otherwise.

### GetObservedOk

`func (o *HypervisorView) GetObservedOk() (*CapacityView, bool)`

GetObservedOk returns a tuple with the Observed field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetObserved

`func (o *HypervisorView) SetObserved(v CapacityView)`

SetObserved sets Observed field to given value.


### GetObservedGuests

`func (o *HypervisorView) GetObservedGuests() int64`

GetObservedGuests returns the ObservedGuests field if non-nil, zero value otherwise.

### GetObservedGuestsOk

`func (o *HypervisorView) GetObservedGuestsOk() (*int64, bool)`

GetObservedGuestsOk returns a tuple with the ObservedGuests field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetObservedGuests

`func (o *HypervisorView) SetObservedGuests(v int64)`

SetObservedGuests sets ObservedGuests field to given value.


### GetProvisioningAttempt

`func (o *HypervisorView) GetProvisioningAttempt() int64`

GetProvisioningAttempt returns the ProvisioningAttempt field if non-nil, zero value otherwise.

### GetProvisioningAttemptOk

`func (o *HypervisorView) GetProvisioningAttemptOk() (*int64, bool)`

GetProvisioningAttemptOk returns a tuple with the ProvisioningAttempt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProvisioningAttempt

`func (o *HypervisorView) SetProvisioningAttempt(v int64)`

SetProvisioningAttempt sets ProvisioningAttempt field to given value.

### HasProvisioningAttempt

`func (o *HypervisorView) HasProvisioningAttempt() bool`

HasProvisioningAttempt returns a boolean if a field has been set.

### GetProvisioningError

`func (o *HypervisorView) GetProvisioningError() string`

GetProvisioningError returns the ProvisioningError field if non-nil, zero value otherwise.

### GetProvisioningErrorOk

`func (o *HypervisorView) GetProvisioningErrorOk() (*string, bool)`

GetProvisioningErrorOk returns a tuple with the ProvisioningError field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProvisioningError

`func (o *HypervisorView) SetProvisioningError(v string)`

SetProvisioningError sets ProvisioningError field to given value.

### HasProvisioningError

`func (o *HypervisorView) HasProvisioningError() bool`

HasProvisioningError returns a boolean if a field has been set.

### GetProvisioningState

`func (o *HypervisorView) GetProvisioningState() string`

GetProvisioningState returns the ProvisioningState field if non-nil, zero value otherwise.

### GetProvisioningStateOk

`func (o *HypervisorView) GetProvisioningStateOk() (*string, bool)`

GetProvisioningStateOk returns a tuple with the ProvisioningState field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProvisioningState

`func (o *HypervisorView) SetProvisioningState(v string)`

SetProvisioningState sets ProvisioningState field to given value.


### GetReserved

`func (o *HypervisorView) GetReserved() CapacityView`

GetReserved returns the Reserved field if non-nil, zero value otherwise.

### GetReservedOk

`func (o *HypervisorView) GetReservedOk() (*CapacityView, bool)`

GetReservedOk returns a tuple with the Reserved field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReserved

`func (o *HypervisorView) SetReserved(v CapacityView)`

SetReserved sets Reserved field to given value.


### GetSchedulable

`func (o *HypervisorView) GetSchedulable() bool`

GetSchedulable returns the Schedulable field if non-nil, zero value otherwise.

### GetSchedulableOk

`func (o *HypervisorView) GetSchedulableOk() (*bool, bool)`

GetSchedulableOk returns a tuple with the Schedulable field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchedulable

`func (o *HypervisorView) SetSchedulable(v bool)`

SetSchedulable sets Schedulable field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


