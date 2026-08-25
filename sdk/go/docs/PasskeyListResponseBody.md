# PasskeyListResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Passkeys** | [**[]PasskeyEntry**](PasskeyEntry.md) |  | 

## Methods

### NewPasskeyListResponseBody

`func NewPasskeyListResponseBody(passkeys []PasskeyEntry, ) *PasskeyListResponseBody`

NewPasskeyListResponseBody instantiates a new PasskeyListResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPasskeyListResponseBodyWithDefaults

`func NewPasskeyListResponseBodyWithDefaults() *PasskeyListResponseBody`

NewPasskeyListResponseBodyWithDefaults instantiates a new PasskeyListResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *PasskeyListResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *PasskeyListResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *PasskeyListResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *PasskeyListResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetPasskeys

`func (o *PasskeyListResponseBody) GetPasskeys() []PasskeyEntry`

GetPasskeys returns the Passkeys field if non-nil, zero value otherwise.

### GetPasskeysOk

`func (o *PasskeyListResponseBody) GetPasskeysOk() (*[]PasskeyEntry, bool)`

GetPasskeysOk returns a tuple with the Passkeys field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPasskeys

`func (o *PasskeyListResponseBody) SetPasskeys(v []PasskeyEntry)`

SetPasskeys sets Passkeys field to given value.


### SetPasskeysNil

`func (o *PasskeyListResponseBody) SetPasskeysNil(b bool)`

 SetPasskeysNil sets the value for Passkeys to be an explicit nil

### UnsetPasskeys
`func (o *PasskeyListResponseBody) UnsetPasskeys()`

UnsetPasskeys ensures that no value is present for Passkeys, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


