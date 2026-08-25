# ListResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Estates** | [**[]EstateView**](EstateView.md) |  | 

## Methods

### NewListResponseBody

`func NewListResponseBody(estates []EstateView, ) *ListResponseBody`

NewListResponseBody instantiates a new ListResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListResponseBodyWithDefaults

`func NewListResponseBodyWithDefaults() *ListResponseBody`

NewListResponseBodyWithDefaults instantiates a new ListResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetEstates

`func (o *ListResponseBody) GetEstates() []EstateView`

GetEstates returns the Estates field if non-nil, zero value otherwise.

### GetEstatesOk

`func (o *ListResponseBody) GetEstatesOk() (*[]EstateView, bool)`

GetEstatesOk returns a tuple with the Estates field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEstates

`func (o *ListResponseBody) SetEstates(v []EstateView)`

SetEstates sets Estates field to given value.


### SetEstatesNil

`func (o *ListResponseBody) SetEstatesNil(b bool)`

 SetEstatesNil sets the value for Estates to be an explicit nil

### UnsetEstates
`func (o *ListResponseBody) UnsetEstates()`

UnsetEstates ensures that no value is present for Estates, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


