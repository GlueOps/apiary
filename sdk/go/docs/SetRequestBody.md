# SetRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Config** | Pointer to [**PolicyConfigView**](PolicyConfigView.md) |  | [optional] 
**State** | **string** | To return this level to &#39;inherit&#39;, DELETE the policy instead. | 

## Methods

### NewSetRequestBody

`func NewSetRequestBody(state string, ) *SetRequestBody`

NewSetRequestBody instantiates a new SetRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSetRequestBodyWithDefaults

`func NewSetRequestBodyWithDefaults() *SetRequestBody`

NewSetRequestBodyWithDefaults instantiates a new SetRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *SetRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *SetRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *SetRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *SetRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetConfig

`func (o *SetRequestBody) GetConfig() PolicyConfigView`

GetConfig returns the Config field if non-nil, zero value otherwise.

### GetConfigOk

`func (o *SetRequestBody) GetConfigOk() (*PolicyConfigView, bool)`

GetConfigOk returns a tuple with the Config field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConfig

`func (o *SetRequestBody) SetConfig(v PolicyConfigView)`

SetConfig sets Config field to given value.

### HasConfig

`func (o *SetRequestBody) HasConfig() bool`

HasConfig returns a boolean if a field has been set.

### GetState

`func (o *SetRequestBody) GetState() string`

GetState returns the State field if non-nil, zero value otherwise.

### GetStateOk

`func (o *SetRequestBody) GetStateOk() (*string, bool)`

GetStateOk returns a tuple with the State field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetState

`func (o *SetRequestBody) SetState(v string)`

SetState sets State field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


