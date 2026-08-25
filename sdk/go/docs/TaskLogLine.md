# TaskLogLine

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**LoggedAt** | Pointer to **time.Time** |  | [optional] 
**Message** | **string** |  | 
**Seq** | **int64** | Monotonic per task, assigned by the bee. Resending a batch is a no-op, which is what makes the stream correct when a bee cannot confirm a write. | 
**Stream** | Pointer to **string** |  | [optional] 

## Methods

### NewTaskLogLine

`func NewTaskLogLine(message string, seq int64, ) *TaskLogLine`

NewTaskLogLine instantiates a new TaskLogLine object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewTaskLogLineWithDefaults

`func NewTaskLogLineWithDefaults() *TaskLogLine`

NewTaskLogLineWithDefaults instantiates a new TaskLogLine object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetLoggedAt

`func (o *TaskLogLine) GetLoggedAt() time.Time`

GetLoggedAt returns the LoggedAt field if non-nil, zero value otherwise.

### GetLoggedAtOk

`func (o *TaskLogLine) GetLoggedAtOk() (*time.Time, bool)`

GetLoggedAtOk returns a tuple with the LoggedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLoggedAt

`func (o *TaskLogLine) SetLoggedAt(v time.Time)`

SetLoggedAt sets LoggedAt field to given value.

### HasLoggedAt

`func (o *TaskLogLine) HasLoggedAt() bool`

HasLoggedAt returns a boolean if a field has been set.

### GetMessage

`func (o *TaskLogLine) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *TaskLogLine) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *TaskLogLine) SetMessage(v string)`

SetMessage sets Message field to given value.


### GetSeq

`func (o *TaskLogLine) GetSeq() int64`

GetSeq returns the Seq field if non-nil, zero value otherwise.

### GetSeqOk

`func (o *TaskLogLine) GetSeqOk() (*int64, bool)`

GetSeqOk returns a tuple with the Seq field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSeq

`func (o *TaskLogLine) SetSeq(v int64)`

SetSeq sets Seq field to given value.


### GetStream

`func (o *TaskLogLine) GetStream() string`

GetStream returns the Stream field if non-nil, zero value otherwise.

### GetStreamOk

`func (o *TaskLogLine) GetStreamOk() (*string, bool)`

GetStreamOk returns a tuple with the Stream field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStream

`func (o *TaskLogLine) SetStream(v string)`

SetStream sets Stream field to given value.

### HasStream

`func (o *TaskLogLine) HasStream() bool`

HasStream returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


