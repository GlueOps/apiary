# AddSetElementRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Comment** | Pointer to **string** |  | [optional] 
**Label** | Pointer to **string** |  | [optional] 
**Value** | **string** | An address or a prefix. A bare 192.168.10.60 is accepted and treated as /32 — an operator writing a management address should not have to spell that out. | 

## Methods

### NewAddSetElementRequestBody

`func NewAddSetElementRequestBody(value string, ) *AddSetElementRequestBody`

NewAddSetElementRequestBody instantiates a new AddSetElementRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAddSetElementRequestBodyWithDefaults

`func NewAddSetElementRequestBodyWithDefaults() *AddSetElementRequestBody`

NewAddSetElementRequestBodyWithDefaults instantiates a new AddSetElementRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *AddSetElementRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *AddSetElementRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *AddSetElementRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *AddSetElementRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetComment

`func (o *AddSetElementRequestBody) GetComment() string`

GetComment returns the Comment field if non-nil, zero value otherwise.

### GetCommentOk

`func (o *AddSetElementRequestBody) GetCommentOk() (*string, bool)`

GetCommentOk returns a tuple with the Comment field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetComment

`func (o *AddSetElementRequestBody) SetComment(v string)`

SetComment sets Comment field to given value.

### HasComment

`func (o *AddSetElementRequestBody) HasComment() bool`

HasComment returns a boolean if a field has been set.

### GetLabel

`func (o *AddSetElementRequestBody) GetLabel() string`

GetLabel returns the Label field if non-nil, zero value otherwise.

### GetLabelOk

`func (o *AddSetElementRequestBody) GetLabelOk() (*string, bool)`

GetLabelOk returns a tuple with the Label field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabel

`func (o *AddSetElementRequestBody) SetLabel(v string)`

SetLabel sets Label field to given value.

### HasLabel

`func (o *AddSetElementRequestBody) HasLabel() bool`

HasLabel returns a boolean if a field has been set.

### GetValue

`func (o *AddSetElementRequestBody) GetValue() string`

GetValue returns the Value field if non-nil, zero value otherwise.

### GetValueOk

`func (o *AddSetElementRequestBody) GetValueOk() (*string, bool)`

GetValueOk returns a tuple with the Value field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetValue

`func (o *AddSetElementRequestBody) SetValue(v string)`

SetValue sets Value field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


