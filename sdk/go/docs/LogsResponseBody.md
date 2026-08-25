# LogsResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Accepted** | **int64** | How many lines were NEW. A resent batch reports zero, which is how a bee can tell progress from retry. | 

## Methods

### NewLogsResponseBody

`func NewLogsResponseBody(accepted int64, ) *LogsResponseBody`

NewLogsResponseBody instantiates a new LogsResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewLogsResponseBodyWithDefaults

`func NewLogsResponseBodyWithDefaults() *LogsResponseBody`

NewLogsResponseBodyWithDefaults instantiates a new LogsResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *LogsResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *LogsResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *LogsResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *LogsResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetAccepted

`func (o *LogsResponseBody) GetAccepted() int64`

GetAccepted returns the Accepted field if non-nil, zero value otherwise.

### GetAcceptedOk

`func (o *LogsResponseBody) GetAcceptedOk() (*int64, bool)`

GetAcceptedOk returns a tuple with the Accepted field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAccepted

`func (o *LogsResponseBody) SetAccepted(v int64)`

SetAccepted sets Accepted field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


