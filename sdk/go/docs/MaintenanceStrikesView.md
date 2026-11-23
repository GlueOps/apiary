# MaintenanceStrikesView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Flagged** | **bool** | Maintenance skips it until an admin acknowledges it. | 
**HiveId** | Pointer to **string** |  | [optional] 
**HypervisorId** | **string** |  | 
**LatestAt** | **time.Time** |  | 
**LatestReason** | **string** |  | 
**LatestRunId** | **string** |  | 
**Name** | **string** |  | 
**Schedulable** | **bool** | False while it is still cordoned. Acknowledging does not uncordon it. | 
**Strikes** | **int64** | Maintenance runs in a row in which its upgrade started and then failed. | 

## Methods

### NewMaintenanceStrikesView

`func NewMaintenanceStrikesView(flagged bool, hypervisorId string, latestAt time.Time, latestReason string, latestRunId string, name string, schedulable bool, strikes int64, ) *MaintenanceStrikesView`

NewMaintenanceStrikesView instantiates a new MaintenanceStrikesView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMaintenanceStrikesViewWithDefaults

`func NewMaintenanceStrikesViewWithDefaults() *MaintenanceStrikesView`

NewMaintenanceStrikesViewWithDefaults instantiates a new MaintenanceStrikesView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetFlagged

`func (o *MaintenanceStrikesView) GetFlagged() bool`

GetFlagged returns the Flagged field if non-nil, zero value otherwise.

### GetFlaggedOk

`func (o *MaintenanceStrikesView) GetFlaggedOk() (*bool, bool)`

GetFlaggedOk returns a tuple with the Flagged field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFlagged

`func (o *MaintenanceStrikesView) SetFlagged(v bool)`

SetFlagged sets Flagged field to given value.


### GetHiveId

`func (o *MaintenanceStrikesView) GetHiveId() string`

GetHiveId returns the HiveId field if non-nil, zero value otherwise.

### GetHiveIdOk

`func (o *MaintenanceStrikesView) GetHiveIdOk() (*string, bool)`

GetHiveIdOk returns a tuple with the HiveId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHiveId

`func (o *MaintenanceStrikesView) SetHiveId(v string)`

SetHiveId sets HiveId field to given value.

### HasHiveId

`func (o *MaintenanceStrikesView) HasHiveId() bool`

HasHiveId returns a boolean if a field has been set.

### GetHypervisorId

`func (o *MaintenanceStrikesView) GetHypervisorId() string`

GetHypervisorId returns the HypervisorId field if non-nil, zero value otherwise.

### GetHypervisorIdOk

`func (o *MaintenanceStrikesView) GetHypervisorIdOk() (*string, bool)`

GetHypervisorIdOk returns a tuple with the HypervisorId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHypervisorId

`func (o *MaintenanceStrikesView) SetHypervisorId(v string)`

SetHypervisorId sets HypervisorId field to given value.


### GetLatestAt

`func (o *MaintenanceStrikesView) GetLatestAt() time.Time`

GetLatestAt returns the LatestAt field if non-nil, zero value otherwise.

### GetLatestAtOk

`func (o *MaintenanceStrikesView) GetLatestAtOk() (*time.Time, bool)`

GetLatestAtOk returns a tuple with the LatestAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLatestAt

`func (o *MaintenanceStrikesView) SetLatestAt(v time.Time)`

SetLatestAt sets LatestAt field to given value.


### GetLatestReason

`func (o *MaintenanceStrikesView) GetLatestReason() string`

GetLatestReason returns the LatestReason field if non-nil, zero value otherwise.

### GetLatestReasonOk

`func (o *MaintenanceStrikesView) GetLatestReasonOk() (*string, bool)`

GetLatestReasonOk returns a tuple with the LatestReason field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLatestReason

`func (o *MaintenanceStrikesView) SetLatestReason(v string)`

SetLatestReason sets LatestReason field to given value.


### GetLatestRunId

`func (o *MaintenanceStrikesView) GetLatestRunId() string`

GetLatestRunId returns the LatestRunId field if non-nil, zero value otherwise.

### GetLatestRunIdOk

`func (o *MaintenanceStrikesView) GetLatestRunIdOk() (*string, bool)`

GetLatestRunIdOk returns a tuple with the LatestRunId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLatestRunId

`func (o *MaintenanceStrikesView) SetLatestRunId(v string)`

SetLatestRunId sets LatestRunId field to given value.


### GetName

`func (o *MaintenanceStrikesView) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *MaintenanceStrikesView) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *MaintenanceStrikesView) SetName(v string)`

SetName sets Name field to given value.


### GetSchedulable

`func (o *MaintenanceStrikesView) GetSchedulable() bool`

GetSchedulable returns the Schedulable field if non-nil, zero value otherwise.

### GetSchedulableOk

`func (o *MaintenanceStrikesView) GetSchedulableOk() (*bool, bool)`

GetSchedulableOk returns a tuple with the Schedulable field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchedulable

`func (o *MaintenanceStrikesView) SetSchedulable(v bool)`

SetSchedulable sets Schedulable field to given value.


### GetStrikes

`func (o *MaintenanceStrikesView) GetStrikes() int64`

GetStrikes returns the Strikes field if non-nil, zero value otherwise.

### GetStrikesOk

`func (o *MaintenanceStrikesView) GetStrikesOk() (*int64, bool)`

GetStrikesOk returns a tuple with the Strikes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStrikes

`func (o *MaintenanceStrikesView) SetStrikes(v int64)`

SetStrikes sets Strikes field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


