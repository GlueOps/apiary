# SyncHiveResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Detail** | **string** |  | 
**Queued** | **bool** |  | 

## Methods

### NewSyncHiveResponseBody

`func NewSyncHiveResponseBody(detail string, queued bool, ) *SyncHiveResponseBody`

NewSyncHiveResponseBody instantiates a new SyncHiveResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSyncHiveResponseBodyWithDefaults

`func NewSyncHiveResponseBodyWithDefaults() *SyncHiveResponseBody`

NewSyncHiveResponseBodyWithDefaults instantiates a new SyncHiveResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *SyncHiveResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *SyncHiveResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *SyncHiveResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *SyncHiveResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetDetail

`func (o *SyncHiveResponseBody) GetDetail() string`

GetDetail returns the Detail field if non-nil, zero value otherwise.

### GetDetailOk

`func (o *SyncHiveResponseBody) GetDetailOk() (*string, bool)`

GetDetailOk returns a tuple with the Detail field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDetail

`func (o *SyncHiveResponseBody) SetDetail(v string)`

SetDetail sets Detail field to given value.


### GetQueued

`func (o *SyncHiveResponseBody) GetQueued() bool`

GetQueued returns the Queued field if non-nil, zero value otherwise.

### GetQueuedOk

`func (o *SyncHiveResponseBody) GetQueuedOk() (*bool, bool)`

GetQueuedOk returns a tuple with the Queued field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetQueued

`func (o *SyncHiveResponseBody) SetQueued(v bool)`

SetQueued sets Queued field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


