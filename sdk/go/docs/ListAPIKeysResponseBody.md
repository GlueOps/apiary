# ListAPIKeysResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Keys** | [**[]APIKeyView**](APIKeyView.md) |  | 

## Methods

### NewListAPIKeysResponseBody

`func NewListAPIKeysResponseBody(keys []APIKeyView, ) *ListAPIKeysResponseBody`

NewListAPIKeysResponseBody instantiates a new ListAPIKeysResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListAPIKeysResponseBodyWithDefaults

`func NewListAPIKeysResponseBodyWithDefaults() *ListAPIKeysResponseBody`

NewListAPIKeysResponseBodyWithDefaults instantiates a new ListAPIKeysResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListAPIKeysResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListAPIKeysResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListAPIKeysResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListAPIKeysResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetKeys

`func (o *ListAPIKeysResponseBody) GetKeys() []APIKeyView`

GetKeys returns the Keys field if non-nil, zero value otherwise.

### GetKeysOk

`func (o *ListAPIKeysResponseBody) GetKeysOk() (*[]APIKeyView, bool)`

GetKeysOk returns a tuple with the Keys field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKeys

`func (o *ListAPIKeysResponseBody) SetKeys(v []APIKeyView)`

SetKeys sets Keys field to given value.


### SetKeysNil

`func (o *ListAPIKeysResponseBody) SetKeysNil(b bool)`

 SetKeysNil sets the value for Keys to be an explicit nil

### UnsetKeys
`func (o *ListAPIKeysResponseBody) UnsetKeys()`

UnsetKeys ensures that no value is present for Keys, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


