# PublishRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Reason** | Pointer to **string** | Why this policy is being published. Read later by whoever is deciding whether to back it out, which is the moment it matters most. | [optional] 

## Methods

### NewPublishRequestBody

`func NewPublishRequestBody() *PublishRequestBody`

NewPublishRequestBody instantiates a new PublishRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPublishRequestBodyWithDefaults

`func NewPublishRequestBodyWithDefaults() *PublishRequestBody`

NewPublishRequestBodyWithDefaults instantiates a new PublishRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *PublishRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *PublishRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *PublishRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *PublishRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetReason

`func (o *PublishRequestBody) GetReason() string`

GetReason returns the Reason field if non-nil, zero value otherwise.

### GetReasonOk

`func (o *PublishRequestBody) GetReasonOk() (*string, bool)`

GetReasonOk returns a tuple with the Reason field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReason

`func (o *PublishRequestBody) SetReason(v string)`

SetReason sets Reason field to given value.

### HasReason

`func (o *PublishRequestBody) HasReason() bool`

HasReason returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


