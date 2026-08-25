# TaskLogsResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Lines** | [**[]StoredTaskLogLine**](StoredTaskLogLine.md) |  | 

## Methods

### NewTaskLogsResponseBody

`func NewTaskLogsResponseBody(lines []StoredTaskLogLine, ) *TaskLogsResponseBody`

NewTaskLogsResponseBody instantiates a new TaskLogsResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewTaskLogsResponseBodyWithDefaults

`func NewTaskLogsResponseBodyWithDefaults() *TaskLogsResponseBody`

NewTaskLogsResponseBodyWithDefaults instantiates a new TaskLogsResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *TaskLogsResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *TaskLogsResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *TaskLogsResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *TaskLogsResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetLines

`func (o *TaskLogsResponseBody) GetLines() []StoredTaskLogLine`

GetLines returns the Lines field if non-nil, zero value otherwise.

### GetLinesOk

`func (o *TaskLogsResponseBody) GetLinesOk() (*[]StoredTaskLogLine, bool)`

GetLinesOk returns a tuple with the Lines field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLines

`func (o *TaskLogsResponseBody) SetLines(v []StoredTaskLogLine)`

SetLines sets Lines field to given value.


### SetLinesNil

`func (o *TaskLogsResponseBody) SetLinesNil(b bool)`

 SetLinesNil sets the value for Lines to be an explicit nil

### UnsetLines
`func (o *TaskLogsResponseBody) UnsetLines()`

UnsetLines ensures that no value is present for Lines, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


