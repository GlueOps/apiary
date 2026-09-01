# ListResponseBody4

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Keys** | [**[]KeyView**](KeyView.md) |  | 

## Methods

### NewListResponseBody4

`func NewListResponseBody4(keys []KeyView, ) *ListResponseBody4`

NewListResponseBody4 instantiates a new ListResponseBody4 object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListResponseBody4WithDefaults

`func NewListResponseBody4WithDefaults() *ListResponseBody4`

NewListResponseBody4WithDefaults instantiates a new ListResponseBody4 object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListResponseBody4) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListResponseBody4) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListResponseBody4) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListResponseBody4) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetKeys

`func (o *ListResponseBody4) GetKeys() []KeyView`

GetKeys returns the Keys field if non-nil, zero value otherwise.

### GetKeysOk

`func (o *ListResponseBody4) GetKeysOk() (*[]KeyView, bool)`

GetKeysOk returns a tuple with the Keys field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKeys

`func (o *ListResponseBody4) SetKeys(v []KeyView)`

SetKeys sets Keys field to given value.


### SetKeysNil

`func (o *ListResponseBody4) SetKeysNil(b bool)`

 SetKeysNil sets the value for Keys to be an explicit nil

### UnsetKeys
`func (o *ListResponseBody4) UnsetKeys()`

UnsetKeys ensures that no value is present for Keys, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


