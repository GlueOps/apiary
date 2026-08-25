# ListResponseBody2

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Policies** | [**[]PolicyView**](PolicyView.md) |  | 

## Methods

### NewListResponseBody2

`func NewListResponseBody2(policies []PolicyView, ) *ListResponseBody2`

NewListResponseBody2 instantiates a new ListResponseBody2 object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListResponseBody2WithDefaults

`func NewListResponseBody2WithDefaults() *ListResponseBody2`

NewListResponseBody2WithDefaults instantiates a new ListResponseBody2 object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListResponseBody2) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListResponseBody2) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListResponseBody2) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListResponseBody2) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetPolicies

`func (o *ListResponseBody2) GetPolicies() []PolicyView`

GetPolicies returns the Policies field if non-nil, zero value otherwise.

### GetPoliciesOk

`func (o *ListResponseBody2) GetPoliciesOk() (*[]PolicyView, bool)`

GetPoliciesOk returns a tuple with the Policies field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPolicies

`func (o *ListResponseBody2) SetPolicies(v []PolicyView)`

SetPolicies sets Policies field to given value.


### SetPoliciesNil

`func (o *ListResponseBody2) SetPoliciesNil(b bool)`

 SetPoliciesNil sets the value for Policies to be an explicit nil

### UnsetPolicies
`func (o *ListResponseBody2) UnsetPolicies()`

UnsetPolicies ensures that no value is present for Policies, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


