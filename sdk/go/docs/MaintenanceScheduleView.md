# MaintenanceScheduleView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**DurationMinutes** | **int64** |  | 
**Enabled** | **bool** |  | 
**HiveId** | Pointer to **string** |  | [optional] 
**HiveName** | Pointer to **string** |  | [optional] 
**Id** | **string** |  | 
**Kind** | **string** |  | 
**LastWindowAt** | Pointer to **time.Time** |  | [optional] 
**Name** | **string** |  | 
**NextOpenings** | **[]string** | The next three times the window opens, UTC. | 
**OnceAt** | Pointer to **time.Time** |  | [optional] 
**OpenNow** | **bool** | The window is open at this moment. | 
**Reboot** | **string** |  | 
**Standalone** | **bool** |  | 
**StartTime** | Pointer to **string** |  | [optional] 
**Timezone** | Pointer to **string** |  | [optional] 
**Weekdays** | **[]int64** |  | 

## Methods

### NewMaintenanceScheduleView

`func NewMaintenanceScheduleView(durationMinutes int64, enabled bool, id string, kind string, name string, nextOpenings []string, openNow bool, reboot string, standalone bool, weekdays []int64, ) *MaintenanceScheduleView`

NewMaintenanceScheduleView instantiates a new MaintenanceScheduleView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMaintenanceScheduleViewWithDefaults

`func NewMaintenanceScheduleViewWithDefaults() *MaintenanceScheduleView`

NewMaintenanceScheduleViewWithDefaults instantiates a new MaintenanceScheduleView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *MaintenanceScheduleView) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *MaintenanceScheduleView) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *MaintenanceScheduleView) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *MaintenanceScheduleView) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetDurationMinutes

`func (o *MaintenanceScheduleView) GetDurationMinutes() int64`

GetDurationMinutes returns the DurationMinutes field if non-nil, zero value otherwise.

### GetDurationMinutesOk

`func (o *MaintenanceScheduleView) GetDurationMinutesOk() (*int64, bool)`

GetDurationMinutesOk returns a tuple with the DurationMinutes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDurationMinutes

`func (o *MaintenanceScheduleView) SetDurationMinutes(v int64)`

SetDurationMinutes sets DurationMinutes field to given value.


### GetEnabled

`func (o *MaintenanceScheduleView) GetEnabled() bool`

GetEnabled returns the Enabled field if non-nil, zero value otherwise.

### GetEnabledOk

`func (o *MaintenanceScheduleView) GetEnabledOk() (*bool, bool)`

GetEnabledOk returns a tuple with the Enabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnabled

`func (o *MaintenanceScheduleView) SetEnabled(v bool)`

SetEnabled sets Enabled field to given value.


### GetHiveId

`func (o *MaintenanceScheduleView) GetHiveId() string`

GetHiveId returns the HiveId field if non-nil, zero value otherwise.

### GetHiveIdOk

`func (o *MaintenanceScheduleView) GetHiveIdOk() (*string, bool)`

GetHiveIdOk returns a tuple with the HiveId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHiveId

`func (o *MaintenanceScheduleView) SetHiveId(v string)`

SetHiveId sets HiveId field to given value.

### HasHiveId

`func (o *MaintenanceScheduleView) HasHiveId() bool`

HasHiveId returns a boolean if a field has been set.

### GetHiveName

`func (o *MaintenanceScheduleView) GetHiveName() string`

GetHiveName returns the HiveName field if non-nil, zero value otherwise.

### GetHiveNameOk

`func (o *MaintenanceScheduleView) GetHiveNameOk() (*string, bool)`

GetHiveNameOk returns a tuple with the HiveName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHiveName

`func (o *MaintenanceScheduleView) SetHiveName(v string)`

SetHiveName sets HiveName field to given value.

### HasHiveName

`func (o *MaintenanceScheduleView) HasHiveName() bool`

HasHiveName returns a boolean if a field has been set.

### GetId

`func (o *MaintenanceScheduleView) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *MaintenanceScheduleView) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *MaintenanceScheduleView) SetId(v string)`

SetId sets Id field to given value.


### GetKind

`func (o *MaintenanceScheduleView) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *MaintenanceScheduleView) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *MaintenanceScheduleView) SetKind(v string)`

SetKind sets Kind field to given value.


### GetLastWindowAt

`func (o *MaintenanceScheduleView) GetLastWindowAt() time.Time`

GetLastWindowAt returns the LastWindowAt field if non-nil, zero value otherwise.

### GetLastWindowAtOk

`func (o *MaintenanceScheduleView) GetLastWindowAtOk() (*time.Time, bool)`

GetLastWindowAtOk returns a tuple with the LastWindowAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastWindowAt

`func (o *MaintenanceScheduleView) SetLastWindowAt(v time.Time)`

SetLastWindowAt sets LastWindowAt field to given value.

### HasLastWindowAt

`func (o *MaintenanceScheduleView) HasLastWindowAt() bool`

HasLastWindowAt returns a boolean if a field has been set.

### GetName

`func (o *MaintenanceScheduleView) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *MaintenanceScheduleView) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *MaintenanceScheduleView) SetName(v string)`

SetName sets Name field to given value.


### GetNextOpenings

`func (o *MaintenanceScheduleView) GetNextOpenings() []string`

GetNextOpenings returns the NextOpenings field if non-nil, zero value otherwise.

### GetNextOpeningsOk

`func (o *MaintenanceScheduleView) GetNextOpeningsOk() (*[]string, bool)`

GetNextOpeningsOk returns a tuple with the NextOpenings field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNextOpenings

`func (o *MaintenanceScheduleView) SetNextOpenings(v []string)`

SetNextOpenings sets NextOpenings field to given value.


### SetNextOpeningsNil

`func (o *MaintenanceScheduleView) SetNextOpeningsNil(b bool)`

 SetNextOpeningsNil sets the value for NextOpenings to be an explicit nil

### UnsetNextOpenings
`func (o *MaintenanceScheduleView) UnsetNextOpenings()`

UnsetNextOpenings ensures that no value is present for NextOpenings, not even an explicit nil
### GetOnceAt

`func (o *MaintenanceScheduleView) GetOnceAt() time.Time`

GetOnceAt returns the OnceAt field if non-nil, zero value otherwise.

### GetOnceAtOk

`func (o *MaintenanceScheduleView) GetOnceAtOk() (*time.Time, bool)`

GetOnceAtOk returns a tuple with the OnceAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOnceAt

`func (o *MaintenanceScheduleView) SetOnceAt(v time.Time)`

SetOnceAt sets OnceAt field to given value.

### HasOnceAt

`func (o *MaintenanceScheduleView) HasOnceAt() bool`

HasOnceAt returns a boolean if a field has been set.

### GetOpenNow

`func (o *MaintenanceScheduleView) GetOpenNow() bool`

GetOpenNow returns the OpenNow field if non-nil, zero value otherwise.

### GetOpenNowOk

`func (o *MaintenanceScheduleView) GetOpenNowOk() (*bool, bool)`

GetOpenNowOk returns a tuple with the OpenNow field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOpenNow

`func (o *MaintenanceScheduleView) SetOpenNow(v bool)`

SetOpenNow sets OpenNow field to given value.


### GetReboot

`func (o *MaintenanceScheduleView) GetReboot() string`

GetReboot returns the Reboot field if non-nil, zero value otherwise.

### GetRebootOk

`func (o *MaintenanceScheduleView) GetRebootOk() (*string, bool)`

GetRebootOk returns a tuple with the Reboot field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReboot

`func (o *MaintenanceScheduleView) SetReboot(v string)`

SetReboot sets Reboot field to given value.


### GetStandalone

`func (o *MaintenanceScheduleView) GetStandalone() bool`

GetStandalone returns the Standalone field if non-nil, zero value otherwise.

### GetStandaloneOk

`func (o *MaintenanceScheduleView) GetStandaloneOk() (*bool, bool)`

GetStandaloneOk returns a tuple with the Standalone field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStandalone

`func (o *MaintenanceScheduleView) SetStandalone(v bool)`

SetStandalone sets Standalone field to given value.


### GetStartTime

`func (o *MaintenanceScheduleView) GetStartTime() string`

GetStartTime returns the StartTime field if non-nil, zero value otherwise.

### GetStartTimeOk

`func (o *MaintenanceScheduleView) GetStartTimeOk() (*string, bool)`

GetStartTimeOk returns a tuple with the StartTime field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStartTime

`func (o *MaintenanceScheduleView) SetStartTime(v string)`

SetStartTime sets StartTime field to given value.

### HasStartTime

`func (o *MaintenanceScheduleView) HasStartTime() bool`

HasStartTime returns a boolean if a field has been set.

### GetTimezone

`func (o *MaintenanceScheduleView) GetTimezone() string`

GetTimezone returns the Timezone field if non-nil, zero value otherwise.

### GetTimezoneOk

`func (o *MaintenanceScheduleView) GetTimezoneOk() (*string, bool)`

GetTimezoneOk returns a tuple with the Timezone field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimezone

`func (o *MaintenanceScheduleView) SetTimezone(v string)`

SetTimezone sets Timezone field to given value.

### HasTimezone

`func (o *MaintenanceScheduleView) HasTimezone() bool`

HasTimezone returns a boolean if a field has been set.

### GetWeekdays

`func (o *MaintenanceScheduleView) GetWeekdays() []int64`

GetWeekdays returns the Weekdays field if non-nil, zero value otherwise.

### GetWeekdaysOk

`func (o *MaintenanceScheduleView) GetWeekdaysOk() (*[]int64, bool)`

GetWeekdaysOk returns a tuple with the Weekdays field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWeekdays

`func (o *MaintenanceScheduleView) SetWeekdays(v []int64)`

SetWeekdays sets Weekdays field to given value.


### SetWeekdaysNil

`func (o *MaintenanceScheduleView) SetWeekdaysNil(b bool)`

 SetWeekdaysNil sets the value for Weekdays to be an explicit nil

### UnsetWeekdays
`func (o *MaintenanceScheduleView) UnsetWeekdays()`

UnsetWeekdays ensures that no value is present for Weekdays, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


