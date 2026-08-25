# PasskeyFinishLoginRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Challenge** | **string** |  | 
**Credential** | **interface{}** |  | 

## Methods

### NewPasskeyFinishLoginRequestBody

`func NewPasskeyFinishLoginRequestBody(challenge string, credential interface{}, ) *PasskeyFinishLoginRequestBody`

NewPasskeyFinishLoginRequestBody instantiates a new PasskeyFinishLoginRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPasskeyFinishLoginRequestBodyWithDefaults

`func NewPasskeyFinishLoginRequestBodyWithDefaults() *PasskeyFinishLoginRequestBody`

NewPasskeyFinishLoginRequestBodyWithDefaults instantiates a new PasskeyFinishLoginRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *PasskeyFinishLoginRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *PasskeyFinishLoginRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *PasskeyFinishLoginRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *PasskeyFinishLoginRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetChallenge

`func (o *PasskeyFinishLoginRequestBody) GetChallenge() string`

GetChallenge returns the Challenge field if non-nil, zero value otherwise.

### GetChallengeOk

`func (o *PasskeyFinishLoginRequestBody) GetChallengeOk() (*string, bool)`

GetChallengeOk returns a tuple with the Challenge field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetChallenge

`func (o *PasskeyFinishLoginRequestBody) SetChallenge(v string)`

SetChallenge sets Challenge field to given value.


### GetCredential

`func (o *PasskeyFinishLoginRequestBody) GetCredential() interface{}`

GetCredential returns the Credential field if non-nil, zero value otherwise.

### GetCredentialOk

`func (o *PasskeyFinishLoginRequestBody) GetCredentialOk() (*interface{}, bool)`

GetCredentialOk returns a tuple with the Credential field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCredential

`func (o *PasskeyFinishLoginRequestBody) SetCredential(v interface{})`

SetCredential sets Credential field to given value.


### SetCredentialNil

`func (o *PasskeyFinishLoginRequestBody) SetCredentialNil(b bool)`

 SetCredentialNil sets the value for Credential to be an explicit nil

### UnsetCredential
`func (o *PasskeyFinishLoginRequestBody) UnsetCredential()`

UnsetCredential ensures that no value is present for Credential, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


