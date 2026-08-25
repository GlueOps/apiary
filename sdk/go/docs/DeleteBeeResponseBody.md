# DeleteBeeResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Detail** | **string** |  | 
**HypervisorId** | Pointer to **string** |  | [optional] 

## Methods

### NewDeleteBeeResponseBody

`func NewDeleteBeeResponseBody(detail string, ) *DeleteBeeResponseBody`

NewDeleteBeeResponseBody instantiates a new DeleteBeeResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDeleteBeeResponseBodyWithDefaults

`func NewDeleteBeeResponseBodyWithDefaults() *DeleteBeeResponseBody`

NewDeleteBeeResponseBodyWithDefaults instantiates a new DeleteBeeResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *DeleteBeeResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *DeleteBeeResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *DeleteBeeResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *DeleteBeeResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetDetail

`func (o *DeleteBeeResponseBody) GetDetail() string`

GetDetail returns the Detail field if non-nil, zero value otherwise.

### GetDetailOk

`func (o *DeleteBeeResponseBody) GetDetailOk() (*string, bool)`

GetDetailOk returns a tuple with the Detail field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDetail

`func (o *DeleteBeeResponseBody) SetDetail(v string)`

SetDetail sets Detail field to given value.


### GetHypervisorId

`func (o *DeleteBeeResponseBody) GetHypervisorId() string`

GetHypervisorId returns the HypervisorId field if non-nil, zero value otherwise.

### GetHypervisorIdOk

`func (o *DeleteBeeResponseBody) GetHypervisorIdOk() (*string, bool)`

GetHypervisorIdOk returns a tuple with the HypervisorId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHypervisorId

`func (o *DeleteBeeResponseBody) SetHypervisorId(v string)`

SetHypervisorId sets HypervisorId field to given value.

### HasHypervisorId

`func (o *DeleteBeeResponseBody) HasHypervisorId() bool`

HasHypervisorId returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


