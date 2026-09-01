# OverridesResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Key** | **string** |  | 
**Overrides** | [**[]OverrideView**](OverrideView.md) |  | 

## Methods

### NewOverridesResponseBody

`func NewOverridesResponseBody(key string, overrides []OverrideView, ) *OverridesResponseBody`

NewOverridesResponseBody instantiates a new OverridesResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewOverridesResponseBodyWithDefaults

`func NewOverridesResponseBodyWithDefaults() *OverridesResponseBody`

NewOverridesResponseBodyWithDefaults instantiates a new OverridesResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *OverridesResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *OverridesResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *OverridesResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *OverridesResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetKey

`func (o *OverridesResponseBody) GetKey() string`

GetKey returns the Key field if non-nil, zero value otherwise.

### GetKeyOk

`func (o *OverridesResponseBody) GetKeyOk() (*string, bool)`

GetKeyOk returns a tuple with the Key field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKey

`func (o *OverridesResponseBody) SetKey(v string)`

SetKey sets Key field to given value.


### GetOverrides

`func (o *OverridesResponseBody) GetOverrides() []OverrideView`

GetOverrides returns the Overrides field if non-nil, zero value otherwise.

### GetOverridesOk

`func (o *OverridesResponseBody) GetOverridesOk() (*[]OverrideView, bool)`

GetOverridesOk returns a tuple with the Overrides field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOverrides

`func (o *OverridesResponseBody) SetOverrides(v []OverrideView)`

SetOverrides sets Overrides field to given value.


### SetOverridesNil

`func (o *OverridesResponseBody) SetOverridesNil(b bool)`

 SetOverridesNil sets the value for Overrides to be an explicit nil

### UnsetOverrides
`func (o *OverridesResponseBody) UnsetOverrides()`

UnsetOverrides ensures that no value is present for Overrides, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


