# UpdateRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Policy** | **string** |  | 
**Reason** | Pointer to **string** | Replaces the existing reason; send empty to clear it. | [optional] 

## Methods

### NewUpdateRequestBody

`func NewUpdateRequestBody(policy string, ) *UpdateRequestBody`

NewUpdateRequestBody instantiates a new UpdateRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewUpdateRequestBodyWithDefaults

`func NewUpdateRequestBodyWithDefaults() *UpdateRequestBody`

NewUpdateRequestBodyWithDefaults instantiates a new UpdateRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *UpdateRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *UpdateRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *UpdateRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *UpdateRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetPolicy

`func (o *UpdateRequestBody) GetPolicy() string`

GetPolicy returns the Policy field if non-nil, zero value otherwise.

### GetPolicyOk

`func (o *UpdateRequestBody) GetPolicyOk() (*string, bool)`

GetPolicyOk returns a tuple with the Policy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPolicy

`func (o *UpdateRequestBody) SetPolicy(v string)`

SetPolicy sets Policy field to given value.


### GetReason

`func (o *UpdateRequestBody) GetReason() string`

GetReason returns the Reason field if non-nil, zero value otherwise.

### GetReasonOk

`func (o *UpdateRequestBody) GetReasonOk() (*string, bool)`

GetReasonOk returns a tuple with the Reason field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReason

`func (o *UpdateRequestBody) SetReason(v string)`

SetReason sets Reason field to given value.

### HasReason

`func (o *UpdateRequestBody) HasReason() bool`

HasReason returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


