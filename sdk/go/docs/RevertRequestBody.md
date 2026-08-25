# RevertRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Reason** | Pointer to **string** |  | [optional] 
**ToRevision** | **int64** | The revision whose BODY to republish. The new revision number is higher than the current one — a backout never moves the number down. | 

## Methods

### NewRevertRequestBody

`func NewRevertRequestBody(toRevision int64, ) *RevertRequestBody`

NewRevertRequestBody instantiates a new RevertRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRevertRequestBodyWithDefaults

`func NewRevertRequestBodyWithDefaults() *RevertRequestBody`

NewRevertRequestBodyWithDefaults instantiates a new RevertRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *RevertRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *RevertRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *RevertRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *RevertRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetReason

`func (o *RevertRequestBody) GetReason() string`

GetReason returns the Reason field if non-nil, zero value otherwise.

### GetReasonOk

`func (o *RevertRequestBody) GetReasonOk() (*string, bool)`

GetReasonOk returns a tuple with the Reason field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReason

`func (o *RevertRequestBody) SetReason(v string)`

SetReason sets Reason field to given value.

### HasReason

`func (o *RevertRequestBody) HasReason() bool`

HasReason returns a boolean if a field has been set.

### GetToRevision

`func (o *RevertRequestBody) GetToRevision() int64`

GetToRevision returns the ToRevision field if non-nil, zero value otherwise.

### GetToRevisionOk

`func (o *RevertRequestBody) GetToRevisionOk() (*int64, bool)`

GetToRevisionOk returns a tuple with the ToRevision field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetToRevision

`func (o *RevertRequestBody) SetToRevision(v int64)`

SetToRevision sets ToRevision field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


