# RegisterResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Message** | **string** |  | 
**Organization** | **string** |  | 
**OrganizationId** | **string** |  | 
**VerificationSent** | **bool** | False means the confirmation email could not be handed to the mailer; the account exists and /auth/resend-verification is the recovery path. | 

## Methods

### NewRegisterResponseBody

`func NewRegisterResponseBody(message string, organization string, organizationId string, verificationSent bool, ) *RegisterResponseBody`

NewRegisterResponseBody instantiates a new RegisterResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRegisterResponseBodyWithDefaults

`func NewRegisterResponseBodyWithDefaults() *RegisterResponseBody`

NewRegisterResponseBodyWithDefaults instantiates a new RegisterResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *RegisterResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *RegisterResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *RegisterResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *RegisterResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetMessage

`func (o *RegisterResponseBody) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *RegisterResponseBody) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *RegisterResponseBody) SetMessage(v string)`

SetMessage sets Message field to given value.


### GetOrganization

`func (o *RegisterResponseBody) GetOrganization() string`

GetOrganization returns the Organization field if non-nil, zero value otherwise.

### GetOrganizationOk

`func (o *RegisterResponseBody) GetOrganizationOk() (*string, bool)`

GetOrganizationOk returns a tuple with the Organization field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrganization

`func (o *RegisterResponseBody) SetOrganization(v string)`

SetOrganization sets Organization field to given value.


### GetOrganizationId

`func (o *RegisterResponseBody) GetOrganizationId() string`

GetOrganizationId returns the OrganizationId field if non-nil, zero value otherwise.

### GetOrganizationIdOk

`func (o *RegisterResponseBody) GetOrganizationIdOk() (*string, bool)`

GetOrganizationIdOk returns a tuple with the OrganizationId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrganizationId

`func (o *RegisterResponseBody) SetOrganizationId(v string)`

SetOrganizationId sets OrganizationId field to given value.


### GetVerificationSent

`func (o *RegisterResponseBody) GetVerificationSent() bool`

GetVerificationSent returns the VerificationSent field if non-nil, zero value otherwise.

### GetVerificationSentOk

`func (o *RegisterResponseBody) GetVerificationSentOk() (*bool, bool)`

GetVerificationSentOk returns a tuple with the VerificationSent field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVerificationSent

`func (o *RegisterResponseBody) SetVerificationSent(v bool)`

SetVerificationSent sets VerificationSent field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


