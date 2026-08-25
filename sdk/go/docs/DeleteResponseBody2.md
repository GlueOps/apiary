# DeleteResponseBody2

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Deleted** | **bool** |  | 
**SeedRestoresOnRestart** | **bool** |  | 
**Warning** | Pointer to **string** |  | [optional] 

## Methods

### NewDeleteResponseBody2

`func NewDeleteResponseBody2(deleted bool, seedRestoresOnRestart bool, ) *DeleteResponseBody2`

NewDeleteResponseBody2 instantiates a new DeleteResponseBody2 object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDeleteResponseBody2WithDefaults

`func NewDeleteResponseBody2WithDefaults() *DeleteResponseBody2`

NewDeleteResponseBody2WithDefaults instantiates a new DeleteResponseBody2 object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *DeleteResponseBody2) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *DeleteResponseBody2) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *DeleteResponseBody2) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *DeleteResponseBody2) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetDeleted

`func (o *DeleteResponseBody2) GetDeleted() bool`

GetDeleted returns the Deleted field if non-nil, zero value otherwise.

### GetDeletedOk

`func (o *DeleteResponseBody2) GetDeletedOk() (*bool, bool)`

GetDeletedOk returns a tuple with the Deleted field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeleted

`func (o *DeleteResponseBody2) SetDeleted(v bool)`

SetDeleted sets Deleted field to given value.


### GetSeedRestoresOnRestart

`func (o *DeleteResponseBody2) GetSeedRestoresOnRestart() bool`

GetSeedRestoresOnRestart returns the SeedRestoresOnRestart field if non-nil, zero value otherwise.

### GetSeedRestoresOnRestartOk

`func (o *DeleteResponseBody2) GetSeedRestoresOnRestartOk() (*bool, bool)`

GetSeedRestoresOnRestartOk returns a tuple with the SeedRestoresOnRestart field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSeedRestoresOnRestart

`func (o *DeleteResponseBody2) SetSeedRestoresOnRestart(v bool)`

SetSeedRestoresOnRestart sets SeedRestoresOnRestart field to given value.


### GetWarning

`func (o *DeleteResponseBody2) GetWarning() string`

GetWarning returns the Warning field if non-nil, zero value otherwise.

### GetWarningOk

`func (o *DeleteResponseBody2) GetWarningOk() (*string, bool)`

GetWarningOk returns a tuple with the Warning field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWarning

`func (o *DeleteResponseBody2) SetWarning(v string)`

SetWarning sets Warning field to given value.

### HasWarning

`func (o *DeleteResponseBody2) HasWarning() bool`

HasWarning returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


