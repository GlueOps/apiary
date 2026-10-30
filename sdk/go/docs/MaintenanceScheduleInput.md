# MaintenanceScheduleInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**DurationMinutes** | **int64** | How long the window stays open. A node already rebooting when it closes is allowed to finish; no new node starts. | 
**Enabled** | **bool** |  | 
**Kind** | **string** |  | 
**Name** | **string** |  | 
**OnceAt** | Pointer to **time.Time** | When a one-off window opens. A time in the past still opens the window now, provided the window has not already closed. | [optional] 
**Reboot** | **string** | Reboot policy each node&#39;s dist-upgrade runs with. | 
**StartTime** | Pointer to **string** | Local time the window opens, HH:MM 24-hour, in timezone. | [optional] 
**Timezone** | Pointer to **string** | IANA zone name. Required for a recurring window: 02:00 means 02:00 local in summer and in winter. | [optional] 
**Weekdays** | Pointer to **[]int64** | Days the window opens on, 0 &#x3D; Sunday … 6 &#x3D; Saturday, in timezone. | [optional] 

## Methods

### NewMaintenanceScheduleInput

`func NewMaintenanceScheduleInput(durationMinutes int64, enabled bool, kind string, name string, reboot string, ) *MaintenanceScheduleInput`

NewMaintenanceScheduleInput instantiates a new MaintenanceScheduleInput object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMaintenanceScheduleInputWithDefaults

`func NewMaintenanceScheduleInputWithDefaults() *MaintenanceScheduleInput`

NewMaintenanceScheduleInputWithDefaults instantiates a new MaintenanceScheduleInput object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *MaintenanceScheduleInput) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *MaintenanceScheduleInput) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *MaintenanceScheduleInput) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *MaintenanceScheduleInput) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetDurationMinutes

`func (o *MaintenanceScheduleInput) GetDurationMinutes() int64`

GetDurationMinutes returns the DurationMinutes field if non-nil, zero value otherwise.

### GetDurationMinutesOk

`func (o *MaintenanceScheduleInput) GetDurationMinutesOk() (*int64, bool)`

GetDurationMinutesOk returns a tuple with the DurationMinutes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDurationMinutes

`func (o *MaintenanceScheduleInput) SetDurationMinutes(v int64)`

SetDurationMinutes sets DurationMinutes field to given value.


### GetEnabled

`func (o *MaintenanceScheduleInput) GetEnabled() bool`

GetEnabled returns the Enabled field if non-nil, zero value otherwise.

### GetEnabledOk

`func (o *MaintenanceScheduleInput) GetEnabledOk() (*bool, bool)`

GetEnabledOk returns a tuple with the Enabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnabled

`func (o *MaintenanceScheduleInput) SetEnabled(v bool)`

SetEnabled sets Enabled field to given value.


### GetKind

`func (o *MaintenanceScheduleInput) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *MaintenanceScheduleInput) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *MaintenanceScheduleInput) SetKind(v string)`

SetKind sets Kind field to given value.


### GetName

`func (o *MaintenanceScheduleInput) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *MaintenanceScheduleInput) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *MaintenanceScheduleInput) SetName(v string)`

SetName sets Name field to given value.


### GetOnceAt

`func (o *MaintenanceScheduleInput) GetOnceAt() time.Time`

GetOnceAt returns the OnceAt field if non-nil, zero value otherwise.

### GetOnceAtOk

`func (o *MaintenanceScheduleInput) GetOnceAtOk() (*time.Time, bool)`

GetOnceAtOk returns a tuple with the OnceAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOnceAt

`func (o *MaintenanceScheduleInput) SetOnceAt(v time.Time)`

SetOnceAt sets OnceAt field to given value.

### HasOnceAt

`func (o *MaintenanceScheduleInput) HasOnceAt() bool`

HasOnceAt returns a boolean if a field has been set.

### GetReboot

`func (o *MaintenanceScheduleInput) GetReboot() string`

GetReboot returns the Reboot field if non-nil, zero value otherwise.

### GetRebootOk

`func (o *MaintenanceScheduleInput) GetRebootOk() (*string, bool)`

GetRebootOk returns a tuple with the Reboot field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReboot

`func (o *MaintenanceScheduleInput) SetReboot(v string)`

SetReboot sets Reboot field to given value.


### GetStartTime

`func (o *MaintenanceScheduleInput) GetStartTime() string`

GetStartTime returns the StartTime field if non-nil, zero value otherwise.

### GetStartTimeOk

`func (o *MaintenanceScheduleInput) GetStartTimeOk() (*string, bool)`

GetStartTimeOk returns a tuple with the StartTime field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStartTime

`func (o *MaintenanceScheduleInput) SetStartTime(v string)`

SetStartTime sets StartTime field to given value.

### HasStartTime

`func (o *MaintenanceScheduleInput) HasStartTime() bool`

HasStartTime returns a boolean if a field has been set.

### GetTimezone

`func (o *MaintenanceScheduleInput) GetTimezone() string`

GetTimezone returns the Timezone field if non-nil, zero value otherwise.

### GetTimezoneOk

`func (o *MaintenanceScheduleInput) GetTimezoneOk() (*string, bool)`

GetTimezoneOk returns a tuple with the Timezone field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimezone

`func (o *MaintenanceScheduleInput) SetTimezone(v string)`

SetTimezone sets Timezone field to given value.

### HasTimezone

`func (o *MaintenanceScheduleInput) HasTimezone() bool`

HasTimezone returns a boolean if a field has been set.

### GetWeekdays

`func (o *MaintenanceScheduleInput) GetWeekdays() []int64`

GetWeekdays returns the Weekdays field if non-nil, zero value otherwise.

### GetWeekdaysOk

`func (o *MaintenanceScheduleInput) GetWeekdaysOk() (*[]int64, bool)`

GetWeekdaysOk returns a tuple with the Weekdays field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWeekdays

`func (o *MaintenanceScheduleInput) SetWeekdays(v []int64)`

SetWeekdays sets Weekdays field to given value.

### HasWeekdays

`func (o *MaintenanceScheduleInput) HasWeekdays() bool`

HasWeekdays returns a boolean if a field has been set.

### SetWeekdaysNil

`func (o *MaintenanceScheduleInput) SetWeekdaysNil(b bool)`

 SetWeekdaysNil sets the value for Weekdays to be an explicit nil

### UnsetWeekdays
`func (o *MaintenanceScheduleInput) UnsetWeekdays()`

UnsetWeekdays ensures that no value is present for Weekdays, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


