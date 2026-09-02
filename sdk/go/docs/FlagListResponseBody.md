# FlagListResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Flags** | [**[]FlagView**](FlagView.md) |  | 

## Methods

### NewFlagListResponseBody

`func NewFlagListResponseBody(flags []FlagView, ) *FlagListResponseBody`

NewFlagListResponseBody instantiates a new FlagListResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewFlagListResponseBodyWithDefaults

`func NewFlagListResponseBodyWithDefaults() *FlagListResponseBody`

NewFlagListResponseBodyWithDefaults instantiates a new FlagListResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *FlagListResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *FlagListResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *FlagListResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *FlagListResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetFlags

`func (o *FlagListResponseBody) GetFlags() []FlagView`

GetFlags returns the Flags field if non-nil, zero value otherwise.

### GetFlagsOk

`func (o *FlagListResponseBody) GetFlagsOk() (*[]FlagView, bool)`

GetFlagsOk returns a tuple with the Flags field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFlags

`func (o *FlagListResponseBody) SetFlags(v []FlagView)`

SetFlags sets Flags field to given value.


### SetFlagsNil

`func (o *FlagListResponseBody) SetFlagsNil(b bool)`

 SetFlagsNil sets the value for Flags to be an explicit nil

### UnsetFlags
`func (o *FlagListResponseBody) UnsetFlags()`

UnsetFlags ensures that no value is present for Flags, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


