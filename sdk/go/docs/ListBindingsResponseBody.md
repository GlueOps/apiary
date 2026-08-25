# ListBindingsResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Bindings** | [**[]BindingView**](BindingView.md) |  | 

## Methods

### NewListBindingsResponseBody

`func NewListBindingsResponseBody(bindings []BindingView, ) *ListBindingsResponseBody`

NewListBindingsResponseBody instantiates a new ListBindingsResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListBindingsResponseBodyWithDefaults

`func NewListBindingsResponseBodyWithDefaults() *ListBindingsResponseBody`

NewListBindingsResponseBodyWithDefaults instantiates a new ListBindingsResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListBindingsResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListBindingsResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListBindingsResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListBindingsResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetBindings

`func (o *ListBindingsResponseBody) GetBindings() []BindingView`

GetBindings returns the Bindings field if non-nil, zero value otherwise.

### GetBindingsOk

`func (o *ListBindingsResponseBody) GetBindingsOk() (*[]BindingView, bool)`

GetBindingsOk returns a tuple with the Bindings field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBindings

`func (o *ListBindingsResponseBody) SetBindings(v []BindingView)`

SetBindings sets Bindings field to given value.


### SetBindingsNil

`func (o *ListBindingsResponseBody) SetBindingsNil(b bool)`

 SetBindingsNil sets the value for Bindings to be an explicit nil

### UnsetBindings
`func (o *ListBindingsResponseBody) UnsetBindings()`

UnsetBindings ensures that no value is present for Bindings, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


