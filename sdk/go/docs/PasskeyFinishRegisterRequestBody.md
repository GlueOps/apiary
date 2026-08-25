# PasskeyFinishRegisterRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Challenge** | **string** |  | 
**Credential** | **interface{}** |  | 
**Name** | **string** |  | 

## Methods

### NewPasskeyFinishRegisterRequestBody

`func NewPasskeyFinishRegisterRequestBody(challenge string, credential interface{}, name string, ) *PasskeyFinishRegisterRequestBody`

NewPasskeyFinishRegisterRequestBody instantiates a new PasskeyFinishRegisterRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPasskeyFinishRegisterRequestBodyWithDefaults

`func NewPasskeyFinishRegisterRequestBodyWithDefaults() *PasskeyFinishRegisterRequestBody`

NewPasskeyFinishRegisterRequestBodyWithDefaults instantiates a new PasskeyFinishRegisterRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *PasskeyFinishRegisterRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *PasskeyFinishRegisterRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *PasskeyFinishRegisterRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *PasskeyFinishRegisterRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetChallenge

`func (o *PasskeyFinishRegisterRequestBody) GetChallenge() string`

GetChallenge returns the Challenge field if non-nil, zero value otherwise.

### GetChallengeOk

`func (o *PasskeyFinishRegisterRequestBody) GetChallengeOk() (*string, bool)`

GetChallengeOk returns a tuple with the Challenge field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetChallenge

`func (o *PasskeyFinishRegisterRequestBody) SetChallenge(v string)`

SetChallenge sets Challenge field to given value.


### GetCredential

`func (o *PasskeyFinishRegisterRequestBody) GetCredential() interface{}`

GetCredential returns the Credential field if non-nil, zero value otherwise.

### GetCredentialOk

`func (o *PasskeyFinishRegisterRequestBody) GetCredentialOk() (*interface{}, bool)`

GetCredentialOk returns a tuple with the Credential field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCredential

`func (o *PasskeyFinishRegisterRequestBody) SetCredential(v interface{})`

SetCredential sets Credential field to given value.


### SetCredentialNil

`func (o *PasskeyFinishRegisterRequestBody) SetCredentialNil(b bool)`

 SetCredentialNil sets the value for Credential to be an explicit nil

### UnsetCredential
`func (o *PasskeyFinishRegisterRequestBody) UnsetCredential()`

UnsetCredential ensures that no value is present for Credential, not even an explicit nil
### GetName

`func (o *PasskeyFinishRegisterRequestBody) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *PasskeyFinishRegisterRequestBody) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *PasskeyFinishRegisterRequestBody) SetName(v string)`

SetName sets Name field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


