# TOTPSetupResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Secret** | **string** |  | 
**Uri** | **string** |  | 

## Methods

### NewTOTPSetupResponseBody

`func NewTOTPSetupResponseBody(secret string, uri string, ) *TOTPSetupResponseBody`

NewTOTPSetupResponseBody instantiates a new TOTPSetupResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewTOTPSetupResponseBodyWithDefaults

`func NewTOTPSetupResponseBodyWithDefaults() *TOTPSetupResponseBody`

NewTOTPSetupResponseBodyWithDefaults instantiates a new TOTPSetupResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *TOTPSetupResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *TOTPSetupResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *TOTPSetupResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *TOTPSetupResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetSecret

`func (o *TOTPSetupResponseBody) GetSecret() string`

GetSecret returns the Secret field if non-nil, zero value otherwise.

### GetSecretOk

`func (o *TOTPSetupResponseBody) GetSecretOk() (*string, bool)`

GetSecretOk returns a tuple with the Secret field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecret

`func (o *TOTPSetupResponseBody) SetSecret(v string)`

SetSecret sets Secret field to given value.


### GetUri

`func (o *TOTPSetupResponseBody) GetUri() string`

GetUri returns the Uri field if non-nil, zero value otherwise.

### GetUriOk

`func (o *TOTPSetupResponseBody) GetUriOk() (*string, bool)`

GetUriOk returns a tuple with the Uri field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUri

`func (o *TOTPSetupResponseBody) SetUri(v string)`

SetUri sets Uri field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


