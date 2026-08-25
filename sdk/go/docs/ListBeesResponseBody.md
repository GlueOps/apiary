# ListBeesResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Bees** | [**[]BeeView**](BeeView.md) |  | 

## Methods

### NewListBeesResponseBody

`func NewListBeesResponseBody(bees []BeeView, ) *ListBeesResponseBody`

NewListBeesResponseBody instantiates a new ListBeesResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListBeesResponseBodyWithDefaults

`func NewListBeesResponseBodyWithDefaults() *ListBeesResponseBody`

NewListBeesResponseBodyWithDefaults instantiates a new ListBeesResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListBeesResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListBeesResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListBeesResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListBeesResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetBees

`func (o *ListBeesResponseBody) GetBees() []BeeView`

GetBees returns the Bees field if non-nil, zero value otherwise.

### GetBeesOk

`func (o *ListBeesResponseBody) GetBeesOk() (*[]BeeView, bool)`

GetBeesOk returns a tuple with the Bees field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBees

`func (o *ListBeesResponseBody) SetBees(v []BeeView)`

SetBees sets Bees field to given value.


### SetBeesNil

`func (o *ListBeesResponseBody) SetBeesNil(b bool)`

 SetBeesNil sets the value for Bees to be an explicit nil

### UnsetBees
`func (o *ListBeesResponseBody) UnsetBees()`

UnsetBees ensures that no value is present for Bees, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


