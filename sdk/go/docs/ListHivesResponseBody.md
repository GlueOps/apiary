# ListHivesResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Hives** | [**[]HiveDetailView**](HiveDetailView.md) |  | 

## Methods

### NewListHivesResponseBody

`func NewListHivesResponseBody(hives []HiveDetailView, ) *ListHivesResponseBody`

NewListHivesResponseBody instantiates a new ListHivesResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListHivesResponseBodyWithDefaults

`func NewListHivesResponseBodyWithDefaults() *ListHivesResponseBody`

NewListHivesResponseBodyWithDefaults instantiates a new ListHivesResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListHivesResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListHivesResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListHivesResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListHivesResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetHives

`func (o *ListHivesResponseBody) GetHives() []HiveDetailView`

GetHives returns the Hives field if non-nil, zero value otherwise.

### GetHivesOk

`func (o *ListHivesResponseBody) GetHivesOk() (*[]HiveDetailView, bool)`

GetHivesOk returns a tuple with the Hives field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHives

`func (o *ListHivesResponseBody) SetHives(v []HiveDetailView)`

SetHives sets Hives field to given value.


### SetHivesNil

`func (o *ListHivesResponseBody) SetHivesNil(b bool)`

 SetHivesNil sets the value for Hives to be an explicit nil

### UnsetHives
`func (o *ListHivesResponseBody) UnsetHives()`

UnsetHives ensures that no value is present for Hives, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


