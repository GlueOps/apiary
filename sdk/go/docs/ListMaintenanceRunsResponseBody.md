# ListMaintenanceRunsResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Runs** | [**[]MaintenanceRunView**](MaintenanceRunView.md) |  | 

## Methods

### NewListMaintenanceRunsResponseBody

`func NewListMaintenanceRunsResponseBody(runs []MaintenanceRunView, ) *ListMaintenanceRunsResponseBody`

NewListMaintenanceRunsResponseBody instantiates a new ListMaintenanceRunsResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListMaintenanceRunsResponseBodyWithDefaults

`func NewListMaintenanceRunsResponseBodyWithDefaults() *ListMaintenanceRunsResponseBody`

NewListMaintenanceRunsResponseBodyWithDefaults instantiates a new ListMaintenanceRunsResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListMaintenanceRunsResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListMaintenanceRunsResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListMaintenanceRunsResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListMaintenanceRunsResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetRuns

`func (o *ListMaintenanceRunsResponseBody) GetRuns() []MaintenanceRunView`

GetRuns returns the Runs field if non-nil, zero value otherwise.

### GetRunsOk

`func (o *ListMaintenanceRunsResponseBody) GetRunsOk() (*[]MaintenanceRunView, bool)`

GetRunsOk returns a tuple with the Runs field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRuns

`func (o *ListMaintenanceRunsResponseBody) SetRuns(v []MaintenanceRunView)`

SetRuns sets Runs field to given value.


### SetRunsNil

`func (o *ListMaintenanceRunsResponseBody) SetRunsNil(b bool)`

 SetRunsNil sets the value for Runs to be an explicit nil

### UnsetRuns
`func (o *ListMaintenanceRunsResponseBody) UnsetRuns()`

UnsetRuns ensures that no value is present for Runs, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


