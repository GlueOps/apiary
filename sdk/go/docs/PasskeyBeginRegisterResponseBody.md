# PasskeyBeginRegisterResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Challenge** | **string** |  | 
**Options** | [**CredentialCreation**](CredentialCreation.md) |  | 

## Methods

### NewPasskeyBeginRegisterResponseBody

`func NewPasskeyBeginRegisterResponseBody(challenge string, options CredentialCreation, ) *PasskeyBeginRegisterResponseBody`

NewPasskeyBeginRegisterResponseBody instantiates a new PasskeyBeginRegisterResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPasskeyBeginRegisterResponseBodyWithDefaults

`func NewPasskeyBeginRegisterResponseBodyWithDefaults() *PasskeyBeginRegisterResponseBody`

NewPasskeyBeginRegisterResponseBodyWithDefaults instantiates a new PasskeyBeginRegisterResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *PasskeyBeginRegisterResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *PasskeyBeginRegisterResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *PasskeyBeginRegisterResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *PasskeyBeginRegisterResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetChallenge

`func (o *PasskeyBeginRegisterResponseBody) GetChallenge() string`

GetChallenge returns the Challenge field if non-nil, zero value otherwise.

### GetChallengeOk

`func (o *PasskeyBeginRegisterResponseBody) GetChallengeOk() (*string, bool)`

GetChallengeOk returns a tuple with the Challenge field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetChallenge

`func (o *PasskeyBeginRegisterResponseBody) SetChallenge(v string)`

SetChallenge sets Challenge field to given value.


### GetOptions

`func (o *PasskeyBeginRegisterResponseBody) GetOptions() CredentialCreation`

GetOptions returns the Options field if non-nil, zero value otherwise.

### GetOptionsOk

`func (o *PasskeyBeginRegisterResponseBody) GetOptionsOk() (*CredentialCreation, bool)`

GetOptionsOk returns a tuple with the Options field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOptions

`func (o *PasskeyBeginRegisterResponseBody) SetOptions(v CredentialCreation)`

SetOptions sets Options field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


