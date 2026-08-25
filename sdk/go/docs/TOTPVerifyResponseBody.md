# TOTPVerifyResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Message** | **string** |  | 

## Methods

### NewTOTPVerifyResponseBody

`func NewTOTPVerifyResponseBody(message string, ) *TOTPVerifyResponseBody`

NewTOTPVerifyResponseBody instantiates a new TOTPVerifyResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewTOTPVerifyResponseBodyWithDefaults

`func NewTOTPVerifyResponseBodyWithDefaults() *TOTPVerifyResponseBody`

NewTOTPVerifyResponseBodyWithDefaults instantiates a new TOTPVerifyResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *TOTPVerifyResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *TOTPVerifyResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *TOTPVerifyResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *TOTPVerifyResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetMessage

`func (o *TOTPVerifyResponseBody) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *TOTPVerifyResponseBody) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *TOTPVerifyResponseBody) SetMessage(v string)`

SetMessage sets Message field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


