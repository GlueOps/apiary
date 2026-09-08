# PublicFlagsResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Flags** | [**[]PublicFlagView**](PublicFlagView.md) |  | 

## Methods

### NewPublicFlagsResponseBody

`func NewPublicFlagsResponseBody(flags []PublicFlagView, ) *PublicFlagsResponseBody`

NewPublicFlagsResponseBody instantiates a new PublicFlagsResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPublicFlagsResponseBodyWithDefaults

`func NewPublicFlagsResponseBodyWithDefaults() *PublicFlagsResponseBody`

NewPublicFlagsResponseBodyWithDefaults instantiates a new PublicFlagsResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *PublicFlagsResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *PublicFlagsResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *PublicFlagsResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *PublicFlagsResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetFlags

`func (o *PublicFlagsResponseBody) GetFlags() []PublicFlagView`

GetFlags returns the Flags field if non-nil, zero value otherwise.

### GetFlagsOk

`func (o *PublicFlagsResponseBody) GetFlagsOk() (*[]PublicFlagView, bool)`

GetFlagsOk returns a tuple with the Flags field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFlags

`func (o *PublicFlagsResponseBody) SetFlags(v []PublicFlagView)`

SetFlags sets Flags field to given value.


### SetFlagsNil

`func (o *PublicFlagsResponseBody) SetFlagsNil(b bool)`

 SetFlagsNil sets the value for Flags to be an explicit nil

### UnsetFlags
`func (o *PublicFlagsResponseBody) UnsetFlags()`

UnsetFlags ensures that no value is present for Flags, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


