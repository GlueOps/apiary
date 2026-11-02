# ListMaintenanceSchedulesResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Schedules** | [**[]MaintenanceScheduleView**](MaintenanceScheduleView.md) |  | 

## Methods

### NewListMaintenanceSchedulesResponseBody

`func NewListMaintenanceSchedulesResponseBody(schedules []MaintenanceScheduleView, ) *ListMaintenanceSchedulesResponseBody`

NewListMaintenanceSchedulesResponseBody instantiates a new ListMaintenanceSchedulesResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListMaintenanceSchedulesResponseBodyWithDefaults

`func NewListMaintenanceSchedulesResponseBodyWithDefaults() *ListMaintenanceSchedulesResponseBody`

NewListMaintenanceSchedulesResponseBodyWithDefaults instantiates a new ListMaintenanceSchedulesResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListMaintenanceSchedulesResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListMaintenanceSchedulesResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListMaintenanceSchedulesResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListMaintenanceSchedulesResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetSchedules

`func (o *ListMaintenanceSchedulesResponseBody) GetSchedules() []MaintenanceScheduleView`

GetSchedules returns the Schedules field if non-nil, zero value otherwise.

### GetSchedulesOk

`func (o *ListMaintenanceSchedulesResponseBody) GetSchedulesOk() (*[]MaintenanceScheduleView, bool)`

GetSchedulesOk returns a tuple with the Schedules field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchedules

`func (o *ListMaintenanceSchedulesResponseBody) SetSchedules(v []MaintenanceScheduleView)`

SetSchedules sets Schedules field to given value.


### SetSchedulesNil

`func (o *ListMaintenanceSchedulesResponseBody) SetSchedulesNil(b bool)`

 SetSchedulesNil sets the value for Schedules to be an explicit nil

### UnsetSchedules
`func (o *ListMaintenanceSchedulesResponseBody) UnsetSchedules()`

UnsetSchedules ensures that no value is present for Schedules, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


