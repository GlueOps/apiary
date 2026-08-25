# StoredTaskLogLine

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**LoggedAt** | **time.Time** |  | 
**Message** | **string** |  | 
**Seq** | **int64** |  | 
**Stream** | **string** |  | 

## Methods

### NewStoredTaskLogLine

`func NewStoredTaskLogLine(loggedAt time.Time, message string, seq int64, stream string, ) *StoredTaskLogLine`

NewStoredTaskLogLine instantiates a new StoredTaskLogLine object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewStoredTaskLogLineWithDefaults

`func NewStoredTaskLogLineWithDefaults() *StoredTaskLogLine`

NewStoredTaskLogLineWithDefaults instantiates a new StoredTaskLogLine object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetLoggedAt

`func (o *StoredTaskLogLine) GetLoggedAt() time.Time`

GetLoggedAt returns the LoggedAt field if non-nil, zero value otherwise.

### GetLoggedAtOk

`func (o *StoredTaskLogLine) GetLoggedAtOk() (*time.Time, bool)`

GetLoggedAtOk returns a tuple with the LoggedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLoggedAt

`func (o *StoredTaskLogLine) SetLoggedAt(v time.Time)`

SetLoggedAt sets LoggedAt field to given value.


### GetMessage

`func (o *StoredTaskLogLine) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *StoredTaskLogLine) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *StoredTaskLogLine) SetMessage(v string)`

SetMessage sets Message field to given value.


### GetSeq

`func (o *StoredTaskLogLine) GetSeq() int64`

GetSeq returns the Seq field if non-nil, zero value otherwise.

### GetSeqOk

`func (o *StoredTaskLogLine) GetSeqOk() (*int64, bool)`

GetSeqOk returns a tuple with the Seq field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSeq

`func (o *StoredTaskLogLine) SetSeq(v int64)`

SetSeq sets Seq field to given value.


### GetStream

`func (o *StoredTaskLogLine) GetStream() string`

GetStream returns the Stream field if non-nil, zero value otherwise.

### GetStreamOk

`func (o *StoredTaskLogLine) GetStreamOk() (*string, bool)`

GetStreamOk returns a tuple with the Stream field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStream

`func (o *StoredTaskLogLine) SetStream(v string)`

SetStream sets Stream field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


