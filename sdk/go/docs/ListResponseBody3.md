# ListResponseBody3

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Keys** | [**[]KeyView**](KeyView.md) |  | 

## Methods

### NewListResponseBody3

`func NewListResponseBody3(keys []KeyView, ) *ListResponseBody3`

NewListResponseBody3 instantiates a new ListResponseBody3 object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListResponseBody3WithDefaults

`func NewListResponseBody3WithDefaults() *ListResponseBody3`

NewListResponseBody3WithDefaults instantiates a new ListResponseBody3 object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListResponseBody3) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListResponseBody3) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListResponseBody3) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListResponseBody3) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetKeys

`func (o *ListResponseBody3) GetKeys() []KeyView`

GetKeys returns the Keys field if non-nil, zero value otherwise.

### GetKeysOk

`func (o *ListResponseBody3) GetKeysOk() (*[]KeyView, bool)`

GetKeysOk returns a tuple with the Keys field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKeys

`func (o *ListResponseBody3) SetKeys(v []KeyView)`

SetKeys sets Keys field to given value.


### SetKeysNil

`func (o *ListResponseBody3) SetKeysNil(b bool)`

 SetKeysNil sets the value for Keys to be an explicit nil

### UnsetKeys
`func (o *ListResponseBody3) UnsetKeys()`

UnsetKeys ensures that no value is present for Keys, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


