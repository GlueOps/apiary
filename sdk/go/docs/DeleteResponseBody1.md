# DeleteResponseBody1

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**State** | **string** | Always &#39;inherit&#39; — deleting a policy is how a level withdraws its opinion. | 

## Methods

### NewDeleteResponseBody1

`func NewDeleteResponseBody1(state string, ) *DeleteResponseBody1`

NewDeleteResponseBody1 instantiates a new DeleteResponseBody1 object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDeleteResponseBody1WithDefaults

`func NewDeleteResponseBody1WithDefaults() *DeleteResponseBody1`

NewDeleteResponseBody1WithDefaults instantiates a new DeleteResponseBody1 object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *DeleteResponseBody1) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *DeleteResponseBody1) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *DeleteResponseBody1) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *DeleteResponseBody1) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetState

`func (o *DeleteResponseBody1) GetState() string`

GetState returns the State field if non-nil, zero value otherwise.

### GetStateOk

`func (o *DeleteResponseBody1) GetStateOk() (*string, bool)`

GetStateOk returns a tuple with the State field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetState

`func (o *DeleteResponseBody1) SetState(v string)`

SetState sets State field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


