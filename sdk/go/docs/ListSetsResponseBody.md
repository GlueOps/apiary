# ListSetsResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Sets** | [**[]SetView**](SetView.md) |  | 

## Methods

### NewListSetsResponseBody

`func NewListSetsResponseBody(sets []SetView, ) *ListSetsResponseBody`

NewListSetsResponseBody instantiates a new ListSetsResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListSetsResponseBodyWithDefaults

`func NewListSetsResponseBodyWithDefaults() *ListSetsResponseBody`

NewListSetsResponseBodyWithDefaults instantiates a new ListSetsResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListSetsResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListSetsResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListSetsResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListSetsResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetSets

`func (o *ListSetsResponseBody) GetSets() []SetView`

GetSets returns the Sets field if non-nil, zero value otherwise.

### GetSetsOk

`func (o *ListSetsResponseBody) GetSetsOk() (*[]SetView, bool)`

GetSetsOk returns a tuple with the Sets field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSets

`func (o *ListSetsResponseBody) SetSets(v []SetView)`

SetSets sets Sets field to given value.


### SetSetsNil

`func (o *ListSetsResponseBody) SetSetsNil(b bool)`

 SetSetsNil sets the value for Sets to be an explicit nil

### UnsetSets
`func (o *ListSetsResponseBody) UnsetSets()`

UnsetSets ensures that no value is present for Sets, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


