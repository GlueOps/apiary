# PasskeyBeginLoginResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Challenge** | **string** |  | 
**Options** | [**CredentialAssertion**](CredentialAssertion.md) |  | 

## Methods

### NewPasskeyBeginLoginResponseBody

`func NewPasskeyBeginLoginResponseBody(challenge string, options CredentialAssertion, ) *PasskeyBeginLoginResponseBody`

NewPasskeyBeginLoginResponseBody instantiates a new PasskeyBeginLoginResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPasskeyBeginLoginResponseBodyWithDefaults

`func NewPasskeyBeginLoginResponseBodyWithDefaults() *PasskeyBeginLoginResponseBody`

NewPasskeyBeginLoginResponseBodyWithDefaults instantiates a new PasskeyBeginLoginResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *PasskeyBeginLoginResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *PasskeyBeginLoginResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *PasskeyBeginLoginResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *PasskeyBeginLoginResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetChallenge

`func (o *PasskeyBeginLoginResponseBody) GetChallenge() string`

GetChallenge returns the Challenge field if non-nil, zero value otherwise.

### GetChallengeOk

`func (o *PasskeyBeginLoginResponseBody) GetChallengeOk() (*string, bool)`

GetChallengeOk returns a tuple with the Challenge field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetChallenge

`func (o *PasskeyBeginLoginResponseBody) SetChallenge(v string)`

SetChallenge sets Challenge field to given value.


### GetOptions

`func (o *PasskeyBeginLoginResponseBody) GetOptions() CredentialAssertion`

GetOptions returns the Options field if non-nil, zero value otherwise.

### GetOptionsOk

`func (o *PasskeyBeginLoginResponseBody) GetOptionsOk() (*CredentialAssertion, bool)`

GetOptionsOk returns a tuple with the Options field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOptions

`func (o *PasskeyBeginLoginResponseBody) SetOptions(v CredentialAssertion)`

SetOptions sets Options field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


