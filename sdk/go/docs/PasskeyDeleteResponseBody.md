# PasskeyDeleteResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Message** | **string** |  | 

## Methods

### NewPasskeyDeleteResponseBody

`func NewPasskeyDeleteResponseBody(message string, ) *PasskeyDeleteResponseBody`

NewPasskeyDeleteResponseBody instantiates a new PasskeyDeleteResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPasskeyDeleteResponseBodyWithDefaults

`func NewPasskeyDeleteResponseBodyWithDefaults() *PasskeyDeleteResponseBody`

NewPasskeyDeleteResponseBodyWithDefaults instantiates a new PasskeyDeleteResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *PasskeyDeleteResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *PasskeyDeleteResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *PasskeyDeleteResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *PasskeyDeleteResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetMessage

`func (o *PasskeyDeleteResponseBody) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *PasskeyDeleteResponseBody) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *PasskeyDeleteResponseBody) SetMessage(v string)`

SetMessage sets Message field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


