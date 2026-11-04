# CreateMaintenanceScheduleRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**DurationMinutes** | **int64** | How long the window stays open. A node already rebooting when it closes is allowed to finish; no new node starts. | 
**Enabled** | **bool** |  | 
**HiveId** | Pointer to **string** | The cluster this window maintains. | [optional] 
**Kind** | **string** |  | 
**Name** | **string** |  | 
**OnceAt** | Pointer to **time.Time** | When a one-off window opens. A time in the past still opens the window now, provided the window has not already closed. | [optional] 
**Reboot** | **string** | Reboot policy each node&#39;s dist-upgrade runs with. | 
**Standalone** | Pointer to **bool** | True maintains the standalone pool instead: every hypervisor in no cluster, resolved when each run starts. | [optional] 
**StartTime** | Pointer to **string** | Local time the window opens, HH:MM 24-hour, in timezone. | [optional] 
**Timezone** | Pointer to **string** | IANA zone name. Required for a recurring window: 02:00 means 02:00 local in summer and in winter. | [optional] 
**Weekdays** | Pointer to **[]int64** | Days the window opens on, 0 &#x3D; Sunday … 6 &#x3D; Saturday, in timezone. | [optional] 

## Methods

### NewCreateMaintenanceScheduleRequestBody

`func NewCreateMaintenanceScheduleRequestBody(durationMinutes int64, enabled bool, kind string, name string, reboot string, ) *CreateMaintenanceScheduleRequestBody`

NewCreateMaintenanceScheduleRequestBody instantiates a new CreateMaintenanceScheduleRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCreateMaintenanceScheduleRequestBodyWithDefaults

`func NewCreateMaintenanceScheduleRequestBodyWithDefaults() *CreateMaintenanceScheduleRequestBody`

NewCreateMaintenanceScheduleRequestBodyWithDefaults instantiates a new CreateMaintenanceScheduleRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *CreateMaintenanceScheduleRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *CreateMaintenanceScheduleRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *CreateMaintenanceScheduleRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *CreateMaintenanceScheduleRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetDurationMinutes

`func (o *CreateMaintenanceScheduleRequestBody) GetDurationMinutes() int64`

GetDurationMinutes returns the DurationMinutes field if non-nil, zero value otherwise.

### GetDurationMinutesOk

`func (o *CreateMaintenanceScheduleRequestBody) GetDurationMinutesOk() (*int64, bool)`

GetDurationMinutesOk returns a tuple with the DurationMinutes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDurationMinutes

`func (o *CreateMaintenanceScheduleRequestBody) SetDurationMinutes(v int64)`

SetDurationMinutes sets DurationMinutes field to given value.


### GetEnabled

`func (o *CreateMaintenanceScheduleRequestBody) GetEnabled() bool`

GetEnabled returns the Enabled field if non-nil, zero value otherwise.

### GetEnabledOk

`func (o *CreateMaintenanceScheduleRequestBody) GetEnabledOk() (*bool, bool)`

GetEnabledOk returns a tuple with the Enabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnabled

`func (o *CreateMaintenanceScheduleRequestBody) SetEnabled(v bool)`

SetEnabled sets Enabled field to given value.


### GetHiveId

`func (o *CreateMaintenanceScheduleRequestBody) GetHiveId() string`

GetHiveId returns the HiveId field if non-nil, zero value otherwise.

### GetHiveIdOk

`func (o *CreateMaintenanceScheduleRequestBody) GetHiveIdOk() (*string, bool)`

GetHiveIdOk returns a tuple with the HiveId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHiveId

`func (o *CreateMaintenanceScheduleRequestBody) SetHiveId(v string)`

SetHiveId sets HiveId field to given value.

### HasHiveId

`func (o *CreateMaintenanceScheduleRequestBody) HasHiveId() bool`

HasHiveId returns a boolean if a field has been set.

### GetKind

`func (o *CreateMaintenanceScheduleRequestBody) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *CreateMaintenanceScheduleRequestBody) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *CreateMaintenanceScheduleRequestBody) SetKind(v string)`

SetKind sets Kind field to given value.


### GetName

`func (o *CreateMaintenanceScheduleRequestBody) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *CreateMaintenanceScheduleRequestBody) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *CreateMaintenanceScheduleRequestBody) SetName(v string)`

SetName sets Name field to given value.


### GetOnceAt

`func (o *CreateMaintenanceScheduleRequestBody) GetOnceAt() time.Time`

GetOnceAt returns the OnceAt field if non-nil, zero value otherwise.

### GetOnceAtOk

`func (o *CreateMaintenanceScheduleRequestBody) GetOnceAtOk() (*time.Time, bool)`

GetOnceAtOk returns a tuple with the OnceAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOnceAt

`func (o *CreateMaintenanceScheduleRequestBody) SetOnceAt(v time.Time)`

SetOnceAt sets OnceAt field to given value.

### HasOnceAt

`func (o *CreateMaintenanceScheduleRequestBody) HasOnceAt() bool`

HasOnceAt returns a boolean if a field has been set.

### GetReboot

`func (o *CreateMaintenanceScheduleRequestBody) GetReboot() string`

GetReboot returns the Reboot field if non-nil, zero value otherwise.

### GetRebootOk

`func (o *CreateMaintenanceScheduleRequestBody) GetRebootOk() (*string, bool)`

GetRebootOk returns a tuple with the Reboot field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReboot

`func (o *CreateMaintenanceScheduleRequestBody) SetReboot(v string)`

SetReboot sets Reboot field to given value.


### GetStandalone

`func (o *CreateMaintenanceScheduleRequestBody) GetStandalone() bool`

GetStandalone returns the Standalone field if non-nil, zero value otherwise.

### GetStandaloneOk

`func (o *CreateMaintenanceScheduleRequestBody) GetStandaloneOk() (*bool, bool)`

GetStandaloneOk returns a tuple with the Standalone field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStandalone

`func (o *CreateMaintenanceScheduleRequestBody) SetStandalone(v bool)`

SetStandalone sets Standalone field to given value.

### HasStandalone

`func (o *CreateMaintenanceScheduleRequestBody) HasStandalone() bool`

HasStandalone returns a boolean if a field has been set.

### GetStartTime

`func (o *CreateMaintenanceScheduleRequestBody) GetStartTime() string`

GetStartTime returns the StartTime field if non-nil, zero value otherwise.

### GetStartTimeOk

`func (o *CreateMaintenanceScheduleRequestBody) GetStartTimeOk() (*string, bool)`

GetStartTimeOk returns a tuple with the StartTime field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStartTime

`func (o *CreateMaintenanceScheduleRequestBody) SetStartTime(v string)`

SetStartTime sets StartTime field to given value.

### HasStartTime

`func (o *CreateMaintenanceScheduleRequestBody) HasStartTime() bool`

HasStartTime returns a boolean if a field has been set.

### GetTimezone

`func (o *CreateMaintenanceScheduleRequestBody) GetTimezone() string`

GetTimezone returns the Timezone field if non-nil, zero value otherwise.

### GetTimezoneOk

`func (o *CreateMaintenanceScheduleRequestBody) GetTimezoneOk() (*string, bool)`

GetTimezoneOk returns a tuple with the Timezone field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimezone

`func (o *CreateMaintenanceScheduleRequestBody) SetTimezone(v string)`

SetTimezone sets Timezone field to given value.

### HasTimezone

`func (o *CreateMaintenanceScheduleRequestBody) HasTimezone() bool`

HasTimezone returns a boolean if a field has been set.

### GetWeekdays

`func (o *CreateMaintenanceScheduleRequestBody) GetWeekdays() []int64`

GetWeekdays returns the Weekdays field if non-nil, zero value otherwise.

### GetWeekdaysOk

`func (o *CreateMaintenanceScheduleRequestBody) GetWeekdaysOk() (*[]int64, bool)`

GetWeekdaysOk returns a tuple with the Weekdays field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWeekdays

`func (o *CreateMaintenanceScheduleRequestBody) SetWeekdays(v []int64)`

SetWeekdays sets Weekdays field to given value.

### HasWeekdays

`func (o *CreateMaintenanceScheduleRequestBody) HasWeekdays() bool`

HasWeekdays returns a boolean if a field has been set.

### SetWeekdaysNil

`func (o *CreateMaintenanceScheduleRequestBody) SetWeekdaysNil(b bool)`

 SetWeekdaysNil sets the value for Weekdays to be an explicit nil

### UnsetWeekdays
`func (o *CreateMaintenanceScheduleRequestBody) UnsetWeekdays()`

UnsetWeekdays ensures that no value is present for Weekdays, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


