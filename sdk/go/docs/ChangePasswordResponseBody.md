# ChangePasswordResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Message** | **string** |  | 

## Methods

### NewChangePasswordResponseBody

`func NewChangePasswordResponseBody(message string, ) *ChangePasswordResponseBody`

NewChangePasswordResponseBody instantiates a new ChangePasswordResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewChangePasswordResponseBodyWithDefaults

`func NewChangePasswordResponseBodyWithDefaults() *ChangePasswordResponseBody`

NewChangePasswordResponseBodyWithDefaults instantiates a new ChangePasswordResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ChangePasswordResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ChangePasswordResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ChangePasswordResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ChangePasswordResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetMessage

`func (o *ChangePasswordResponseBody) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *ChangePasswordResponseBody) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *ChangePasswordResponseBody) SetMessage(v string)`

SetMessage sets Message field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


