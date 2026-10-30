# MaintenanceRunView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Error** | Pointer to **string** |  | [optional] 
**FinishedAt** | Pointer to **time.Time** |  | [optional] 
**HiveId** | Pointer to **string** |  | [optional] 
**Id** | **string** |  | 
**Nodes** | [**[]MaintenanceRunNodeView**](MaintenanceRunNodeView.md) |  | 
**Reboot** | **string** |  | 
**ScheduleId** | Pointer to **string** |  | [optional] 
**ScheduleName** | Pointer to **string** |  | [optional] 
**Standalone** | **bool** |  | 
**StartedAt** | **time.Time** |  | 
**Status** | **string** |  | 
**StopRequested** | **bool** |  | 
**WindowEnd** | **time.Time** |  | 
**WindowStart** | **time.Time** |  | 

## Methods

### NewMaintenanceRunView

`func NewMaintenanceRunView(id string, nodes []MaintenanceRunNodeView, reboot string, standalone bool, startedAt time.Time, status string, stopRequested bool, windowEnd time.Time, windowStart time.Time, ) *MaintenanceRunView`

NewMaintenanceRunView instantiates a new MaintenanceRunView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMaintenanceRunViewWithDefaults

`func NewMaintenanceRunViewWithDefaults() *MaintenanceRunView`

NewMaintenanceRunViewWithDefaults instantiates a new MaintenanceRunView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *MaintenanceRunView) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *MaintenanceRunView) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *MaintenanceRunView) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *MaintenanceRunView) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetError

`func (o *MaintenanceRunView) GetError() string`

GetError returns the Error field if non-nil, zero value otherwise.

### GetErrorOk

`func (o *MaintenanceRunView) GetErrorOk() (*string, bool)`

GetErrorOk returns a tuple with the Error field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetError

`func (o *MaintenanceRunView) SetError(v string)`

SetError sets Error field to given value.

### HasError

`func (o *MaintenanceRunView) HasError() bool`

HasError returns a boolean if a field has been set.

### GetFinishedAt

`func (o *MaintenanceRunView) GetFinishedAt() time.Time`

GetFinishedAt returns the FinishedAt field if non-nil, zero value otherwise.

### GetFinishedAtOk

`func (o *MaintenanceRunView) GetFinishedAtOk() (*time.Time, bool)`

GetFinishedAtOk returns a tuple with the FinishedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFinishedAt

`func (o *MaintenanceRunView) SetFinishedAt(v time.Time)`

SetFinishedAt sets FinishedAt field to given value.

### HasFinishedAt

`func (o *MaintenanceRunView) HasFinishedAt() bool`

HasFinishedAt returns a boolean if a field has been set.

### GetHiveId

`func (o *MaintenanceRunView) GetHiveId() string`

GetHiveId returns the HiveId field if non-nil, zero value otherwise.

### GetHiveIdOk

`func (o *MaintenanceRunView) GetHiveIdOk() (*string, bool)`

GetHiveIdOk returns a tuple with the HiveId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHiveId

`func (o *MaintenanceRunView) SetHiveId(v string)`

SetHiveId sets HiveId field to given value.

### HasHiveId

`func (o *MaintenanceRunView) HasHiveId() bool`

HasHiveId returns a boolean if a field has been set.

### GetId

`func (o *MaintenanceRunView) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *MaintenanceRunView) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *MaintenanceRunView) SetId(v string)`

SetId sets Id field to given value.


### GetNodes

`func (o *MaintenanceRunView) GetNodes() []MaintenanceRunNodeView`

GetNodes returns the Nodes field if non-nil, zero value otherwise.

### GetNodesOk

`func (o *MaintenanceRunView) GetNodesOk() (*[]MaintenanceRunNodeView, bool)`

GetNodesOk returns a tuple with the Nodes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNodes

`func (o *MaintenanceRunView) SetNodes(v []MaintenanceRunNodeView)`

SetNodes sets Nodes field to given value.


### SetNodesNil

`func (o *MaintenanceRunView) SetNodesNil(b bool)`

 SetNodesNil sets the value for Nodes to be an explicit nil

### UnsetNodes
`func (o *MaintenanceRunView) UnsetNodes()`

UnsetNodes ensures that no value is present for Nodes, not even an explicit nil
### GetReboot

`func (o *MaintenanceRunView) GetReboot() string`

GetReboot returns the Reboot field if non-nil, zero value otherwise.

### GetRebootOk

`func (o *MaintenanceRunView) GetRebootOk() (*string, bool)`

GetRebootOk returns a tuple with the Reboot field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReboot

`func (o *MaintenanceRunView) SetReboot(v string)`

SetReboot sets Reboot field to given value.


### GetScheduleId

`func (o *MaintenanceRunView) GetScheduleId() string`

GetScheduleId returns the ScheduleId field if non-nil, zero value otherwise.

### GetScheduleIdOk

`func (o *MaintenanceRunView) GetScheduleIdOk() (*string, bool)`

GetScheduleIdOk returns a tuple with the ScheduleId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetScheduleId

`func (o *MaintenanceRunView) SetScheduleId(v string)`

SetScheduleId sets ScheduleId field to given value.

### HasScheduleId

`func (o *MaintenanceRunView) HasScheduleId() bool`

HasScheduleId returns a boolean if a field has been set.

### GetScheduleName

`func (o *MaintenanceRunView) GetScheduleName() string`

GetScheduleName returns the ScheduleName field if non-nil, zero value otherwise.

### GetScheduleNameOk

`func (o *MaintenanceRunView) GetScheduleNameOk() (*string, bool)`

GetScheduleNameOk returns a tuple with the ScheduleName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetScheduleName

`func (o *MaintenanceRunView) SetScheduleName(v string)`

SetScheduleName sets ScheduleName field to given value.

### HasScheduleName

`func (o *MaintenanceRunView) HasScheduleName() bool`

HasScheduleName returns a boolean if a field has been set.

### GetStandalone

`func (o *MaintenanceRunView) GetStandalone() bool`

GetStandalone returns the Standalone field if non-nil, zero value otherwise.

### GetStandaloneOk

`func (o *MaintenanceRunView) GetStandaloneOk() (*bool, bool)`

GetStandaloneOk returns a tuple with the Standalone field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStandalone

`func (o *MaintenanceRunView) SetStandalone(v bool)`

SetStandalone sets Standalone field to given value.


### GetStartedAt

`func (o *MaintenanceRunView) GetStartedAt() time.Time`

GetStartedAt returns the StartedAt field if non-nil, zero value otherwise.

### GetStartedAtOk

`func (o *MaintenanceRunView) GetStartedAtOk() (*time.Time, bool)`

GetStartedAtOk returns a tuple with the StartedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStartedAt

`func (o *MaintenanceRunView) SetStartedAt(v time.Time)`

SetStartedAt sets StartedAt field to given value.


### GetStatus

`func (o *MaintenanceRunView) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *MaintenanceRunView) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *MaintenanceRunView) SetStatus(v string)`

SetStatus sets Status field to given value.


### GetStopRequested

`func (o *MaintenanceRunView) GetStopRequested() bool`

GetStopRequested returns the StopRequested field if non-nil, zero value otherwise.

### GetStopRequestedOk

`func (o *MaintenanceRunView) GetStopRequestedOk() (*bool, bool)`

GetStopRequestedOk returns a tuple with the StopRequested field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStopRequested

`func (o *MaintenanceRunView) SetStopRequested(v bool)`

SetStopRequested sets StopRequested field to given value.


### GetWindowEnd

`func (o *MaintenanceRunView) GetWindowEnd() time.Time`

GetWindowEnd returns the WindowEnd field if non-nil, zero value otherwise.

### GetWindowEndOk

`func (o *MaintenanceRunView) GetWindowEndOk() (*time.Time, bool)`

GetWindowEndOk returns a tuple with the WindowEnd field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWindowEnd

`func (o *MaintenanceRunView) SetWindowEnd(v time.Time)`

SetWindowEnd sets WindowEnd field to given value.


### GetWindowStart

`func (o *MaintenanceRunView) GetWindowStart() time.Time`

GetWindowStart returns the WindowStart field if non-nil, zero value otherwise.

### GetWindowStartOk

`func (o *MaintenanceRunView) GetWindowStartOk() (*time.Time, bool)`

GetWindowStartOk returns a tuple with the WindowStart field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWindowStart

`func (o *MaintenanceRunView) SetWindowStart(v time.Time)`

SetWindowStart sets WindowStart field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


