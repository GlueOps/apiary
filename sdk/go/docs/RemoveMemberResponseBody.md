# RemoveMemberResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Removed** | **bool** |  | 

## Methods

### NewRemoveMemberResponseBody

`func NewRemoveMemberResponseBody(removed bool, ) *RemoveMemberResponseBody`

NewRemoveMemberResponseBody instantiates a new RemoveMemberResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRemoveMemberResponseBodyWithDefaults

`func NewRemoveMemberResponseBodyWithDefaults() *RemoveMemberResponseBody`

NewRemoveMemberResponseBodyWithDefaults instantiates a new RemoveMemberResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *RemoveMemberResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *RemoveMemberResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *RemoveMemberResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *RemoveMemberResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetRemoved

`func (o *RemoveMemberResponseBody) GetRemoved() bool`

GetRemoved returns the Removed field if non-nil, zero value otherwise.

### GetRemovedOk

`func (o *RemoveMemberResponseBody) GetRemovedOk() (*bool, bool)`

GetRemovedOk returns a tuple with the Removed field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRemoved

`func (o *RemoveMemberResponseBody) SetRemoved(v bool)`

SetRemoved sets Removed field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


