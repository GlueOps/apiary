# DeleteOrgRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**ConfirmName** | **string** | The organization&#39;s name, exactly. This call takes an entire tenant out of service; typing the name is the difference between meaning it and clicking the wrong row. | 

## Methods

### NewDeleteOrgRequestBody

`func NewDeleteOrgRequestBody(confirmName string, ) *DeleteOrgRequestBody`

NewDeleteOrgRequestBody instantiates a new DeleteOrgRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDeleteOrgRequestBodyWithDefaults

`func NewDeleteOrgRequestBodyWithDefaults() *DeleteOrgRequestBody`

NewDeleteOrgRequestBodyWithDefaults instantiates a new DeleteOrgRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *DeleteOrgRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *DeleteOrgRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *DeleteOrgRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *DeleteOrgRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetConfirmName

`func (o *DeleteOrgRequestBody) GetConfirmName() string`

GetConfirmName returns the ConfirmName field if non-nil, zero value otherwise.

### GetConfirmNameOk

`func (o *DeleteOrgRequestBody) GetConfirmNameOk() (*string, bool)`

GetConfirmNameOk returns a tuple with the ConfirmName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConfirmName

`func (o *DeleteOrgRequestBody) SetConfirmName(v string)`

SetConfirmName sets ConfirmName field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


