# ImpactResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Message** | **string** |  | 
**NonCompliantUsers** | **int64** |  | 

## Methods

### NewImpactResponseBody

`func NewImpactResponseBody(message string, nonCompliantUsers int64, ) *ImpactResponseBody`

NewImpactResponseBody instantiates a new ImpactResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewImpactResponseBodyWithDefaults

`func NewImpactResponseBodyWithDefaults() *ImpactResponseBody`

NewImpactResponseBodyWithDefaults instantiates a new ImpactResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ImpactResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ImpactResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ImpactResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ImpactResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetMessage

`func (o *ImpactResponseBody) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *ImpactResponseBody) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *ImpactResponseBody) SetMessage(v string)`

SetMessage sets Message field to given value.


### GetNonCompliantUsers

`func (o *ImpactResponseBody) GetNonCompliantUsers() int64`

GetNonCompliantUsers returns the NonCompliantUsers field if non-nil, zero value otherwise.

### GetNonCompliantUsersOk

`func (o *ImpactResponseBody) GetNonCompliantUsersOk() (*int64, bool)`

GetNonCompliantUsersOk returns a tuple with the NonCompliantUsers field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNonCompliantUsers

`func (o *ImpactResponseBody) SetNonCompliantUsers(v int64)`

SetNonCompliantUsers sets NonCompliantUsers field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


