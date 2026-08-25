# CordonRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Schedulable** | **bool** | False cordons the node: it stops being elected for new placements. Existing guests are NOT evicted — draining is a separate act. | 

## Methods

### NewCordonRequestBody

`func NewCordonRequestBody(schedulable bool, ) *CordonRequestBody`

NewCordonRequestBody instantiates a new CordonRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCordonRequestBodyWithDefaults

`func NewCordonRequestBodyWithDefaults() *CordonRequestBody`

NewCordonRequestBodyWithDefaults instantiates a new CordonRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *CordonRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *CordonRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *CordonRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *CordonRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetSchedulable

`func (o *CordonRequestBody) GetSchedulable() bool`

GetSchedulable returns the Schedulable field if non-nil, zero value otherwise.

### GetSchedulableOk

`func (o *CordonRequestBody) GetSchedulableOk() (*bool, bool)`

GetSchedulableOk returns a tuple with the Schedulable field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchedulable

`func (o *CordonRequestBody) SetSchedulable(v bool)`

SetSchedulable sets Schedulable field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


