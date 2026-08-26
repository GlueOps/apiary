# SnapshotsResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Snapshots** | [**[]SnapshotView**](SnapshotView.md) |  | 

## Methods

### NewSnapshotsResponseBody

`func NewSnapshotsResponseBody(snapshots []SnapshotView, ) *SnapshotsResponseBody`

NewSnapshotsResponseBody instantiates a new SnapshotsResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSnapshotsResponseBodyWithDefaults

`func NewSnapshotsResponseBodyWithDefaults() *SnapshotsResponseBody`

NewSnapshotsResponseBodyWithDefaults instantiates a new SnapshotsResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *SnapshotsResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *SnapshotsResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *SnapshotsResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *SnapshotsResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetSnapshots

`func (o *SnapshotsResponseBody) GetSnapshots() []SnapshotView`

GetSnapshots returns the Snapshots field if non-nil, zero value otherwise.

### GetSnapshotsOk

`func (o *SnapshotsResponseBody) GetSnapshotsOk() (*[]SnapshotView, bool)`

GetSnapshotsOk returns a tuple with the Snapshots field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSnapshots

`func (o *SnapshotsResponseBody) SetSnapshots(v []SnapshotView)`

SetSnapshots sets Snapshots field to given value.


### SetSnapshotsNil

`func (o *SnapshotsResponseBody) SetSnapshotsNil(b bool)`

 SetSnapshotsNil sets the value for Snapshots to be an explicit nil

### UnsetSnapshots
`func (o *SnapshotsResponseBody) UnsetSnapshots()`

UnsetSnapshots ensures that no value is present for Snapshots, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


