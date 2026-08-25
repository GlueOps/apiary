# DeleteOrgResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Deleted** | **bool** |  | 
**Note** | **string** |  | 

## Methods

### NewDeleteOrgResponseBody

`func NewDeleteOrgResponseBody(deleted bool, note string, ) *DeleteOrgResponseBody`

NewDeleteOrgResponseBody instantiates a new DeleteOrgResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDeleteOrgResponseBodyWithDefaults

`func NewDeleteOrgResponseBodyWithDefaults() *DeleteOrgResponseBody`

NewDeleteOrgResponseBodyWithDefaults instantiates a new DeleteOrgResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *DeleteOrgResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *DeleteOrgResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *DeleteOrgResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *DeleteOrgResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetDeleted

`func (o *DeleteOrgResponseBody) GetDeleted() bool`

GetDeleted returns the Deleted field if non-nil, zero value otherwise.

### GetDeletedOk

`func (o *DeleteOrgResponseBody) GetDeletedOk() (*bool, bool)`

GetDeletedOk returns a tuple with the Deleted field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeleted

`func (o *DeleteOrgResponseBody) SetDeleted(v bool)`

SetDeleted sets Deleted field to given value.


### GetNote

`func (o *DeleteOrgResponseBody) GetNote() string`

GetNote returns the Note field if non-nil, zero value otherwise.

### GetNoteOk

`func (o *DeleteOrgResponseBody) GetNoteOk() (*string, bool)`

GetNoteOk returns a tuple with the Note field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNote

`func (o *DeleteOrgResponseBody) SetNote(v string)`

SetNote sets Note field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


