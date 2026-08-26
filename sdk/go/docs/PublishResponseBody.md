# PublishResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Bytes** | **int64** |  | 
**Digest** | **string** |  | 
**Revision** | **int64** |  | 
**Unchanged** | **bool** | True when the rendered policy matched the published one and no revision was created. Not a failure: publishing twice is the ordinary way to reach it. | 

## Methods

### NewPublishResponseBody

`func NewPublishResponseBody(bytes int64, digest string, revision int64, unchanged bool, ) *PublishResponseBody`

NewPublishResponseBody instantiates a new PublishResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPublishResponseBodyWithDefaults

`func NewPublishResponseBodyWithDefaults() *PublishResponseBody`

NewPublishResponseBodyWithDefaults instantiates a new PublishResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *PublishResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *PublishResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *PublishResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *PublishResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetBytes

`func (o *PublishResponseBody) GetBytes() int64`

GetBytes returns the Bytes field if non-nil, zero value otherwise.

### GetBytesOk

`func (o *PublishResponseBody) GetBytesOk() (*int64, bool)`

GetBytesOk returns a tuple with the Bytes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBytes

`func (o *PublishResponseBody) SetBytes(v int64)`

SetBytes sets Bytes field to given value.


### GetDigest

`func (o *PublishResponseBody) GetDigest() string`

GetDigest returns the Digest field if non-nil, zero value otherwise.

### GetDigestOk

`func (o *PublishResponseBody) GetDigestOk() (*string, bool)`

GetDigestOk returns a tuple with the Digest field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDigest

`func (o *PublishResponseBody) SetDigest(v string)`

SetDigest sets Digest field to given value.


### GetRevision

`func (o *PublishResponseBody) GetRevision() int64`

GetRevision returns the Revision field if non-nil, zero value otherwise.

### GetRevisionOk

`func (o *PublishResponseBody) GetRevisionOk() (*int64, bool)`

GetRevisionOk returns a tuple with the Revision field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRevision

`func (o *PublishResponseBody) SetRevision(v int64)`

SetRevision sets Revision field to given value.


### GetUnchanged

`func (o *PublishResponseBody) GetUnchanged() bool`

GetUnchanged returns the Unchanged field if non-nil, zero value otherwise.

### GetUnchangedOk

`func (o *PublishResponseBody) GetUnchangedOk() (*bool, bool)`

GetUnchangedOk returns a tuple with the Unchanged field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUnchanged

`func (o *PublishResponseBody) SetUnchanged(v bool)`

SetUnchanged sets Unchanged field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


