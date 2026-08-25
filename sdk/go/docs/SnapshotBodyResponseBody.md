# SnapshotBodyResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Content** | **string** |  | 
**Digest** | **string** |  | 
**Revision** | **int64** |  | 

## Methods

### NewSnapshotBodyResponseBody

`func NewSnapshotBodyResponseBody(content string, digest string, revision int64, ) *SnapshotBodyResponseBody`

NewSnapshotBodyResponseBody instantiates a new SnapshotBodyResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSnapshotBodyResponseBodyWithDefaults

`func NewSnapshotBodyResponseBodyWithDefaults() *SnapshotBodyResponseBody`

NewSnapshotBodyResponseBodyWithDefaults instantiates a new SnapshotBodyResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *SnapshotBodyResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *SnapshotBodyResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *SnapshotBodyResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *SnapshotBodyResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetContent

`func (o *SnapshotBodyResponseBody) GetContent() string`

GetContent returns the Content field if non-nil, zero value otherwise.

### GetContentOk

`func (o *SnapshotBodyResponseBody) GetContentOk() (*string, bool)`

GetContentOk returns a tuple with the Content field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetContent

`func (o *SnapshotBodyResponseBody) SetContent(v string)`

SetContent sets Content field to given value.


### GetDigest

`func (o *SnapshotBodyResponseBody) GetDigest() string`

GetDigest returns the Digest field if non-nil, zero value otherwise.

### GetDigestOk

`func (o *SnapshotBodyResponseBody) GetDigestOk() (*string, bool)`

GetDigestOk returns a tuple with the Digest field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDigest

`func (o *SnapshotBodyResponseBody) SetDigest(v string)`

SetDigest sets Digest field to given value.


### GetRevision

`func (o *SnapshotBodyResponseBody) GetRevision() int64`

GetRevision returns the Revision field if non-nil, zero value otherwise.

### GetRevisionOk

`func (o *SnapshotBodyResponseBody) GetRevisionOk() (*int64, bool)`

GetRevisionOk returns a tuple with the Revision field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRevision

`func (o *SnapshotBodyResponseBody) SetRevision(v int64)`

SetRevision sets Revision field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


