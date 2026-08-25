# RevokeResponseBody1

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Note** | **string** |  | 
**Revoked** | **bool** |  | 

## Methods

### NewRevokeResponseBody1

`func NewRevokeResponseBody1(note string, revoked bool, ) *RevokeResponseBody1`

NewRevokeResponseBody1 instantiates a new RevokeResponseBody1 object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRevokeResponseBody1WithDefaults

`func NewRevokeResponseBody1WithDefaults() *RevokeResponseBody1`

NewRevokeResponseBody1WithDefaults instantiates a new RevokeResponseBody1 object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *RevokeResponseBody1) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *RevokeResponseBody1) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *RevokeResponseBody1) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *RevokeResponseBody1) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetNote

`func (o *RevokeResponseBody1) GetNote() string`

GetNote returns the Note field if non-nil, zero value otherwise.

### GetNoteOk

`func (o *RevokeResponseBody1) GetNoteOk() (*string, bool)`

GetNoteOk returns a tuple with the Note field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNote

`func (o *RevokeResponseBody1) SetNote(v string)`

SetNote sets Note field to given value.


### GetRevoked

`func (o *RevokeResponseBody1) GetRevoked() bool`

GetRevoked returns the Revoked field if non-nil, zero value otherwise.

### GetRevokedOk

`func (o *RevokeResponseBody1) GetRevokedOk() (*bool, bool)`

GetRevokedOk returns a tuple with the Revoked field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRevoked

`func (o *RevokeResponseBody1) SetRevoked(v bool)`

SetRevoked sets Revoked field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


