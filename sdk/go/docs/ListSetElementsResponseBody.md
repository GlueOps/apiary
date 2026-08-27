# ListSetElementsResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Elements** | [**[]SetElementView**](SetElementView.md) |  | 

## Methods

### NewListSetElementsResponseBody

`func NewListSetElementsResponseBody(elements []SetElementView, ) *ListSetElementsResponseBody`

NewListSetElementsResponseBody instantiates a new ListSetElementsResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListSetElementsResponseBodyWithDefaults

`func NewListSetElementsResponseBodyWithDefaults() *ListSetElementsResponseBody`

NewListSetElementsResponseBodyWithDefaults instantiates a new ListSetElementsResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListSetElementsResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListSetElementsResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListSetElementsResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListSetElementsResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetElements

`func (o *ListSetElementsResponseBody) GetElements() []SetElementView`

GetElements returns the Elements field if non-nil, zero value otherwise.

### GetElementsOk

`func (o *ListSetElementsResponseBody) GetElementsOk() (*[]SetElementView, bool)`

GetElementsOk returns a tuple with the Elements field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetElements

`func (o *ListSetElementsResponseBody) SetElements(v []SetElementView)`

SetElements sets Elements field to given value.


### SetElementsNil

`func (o *ListSetElementsResponseBody) SetElementsNil(b bool)`

 SetElementsNil sets the value for Elements to be an explicit nil

### UnsetElements
`func (o *ListSetElementsResponseBody) UnsetElements()`

UnsetElements ensures that no value is present for Elements, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


