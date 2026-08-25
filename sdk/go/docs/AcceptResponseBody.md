# AcceptResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Accepted** | **bool** |  | 
**AccountCreated** | **bool** |  | 
**Email** | **string** |  | 
**Message** | **string** |  | 
**Organization** | **string** |  | 

## Methods

### NewAcceptResponseBody

`func NewAcceptResponseBody(accepted bool, accountCreated bool, email string, message string, organization string, ) *AcceptResponseBody`

NewAcceptResponseBody instantiates a new AcceptResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAcceptResponseBodyWithDefaults

`func NewAcceptResponseBodyWithDefaults() *AcceptResponseBody`

NewAcceptResponseBodyWithDefaults instantiates a new AcceptResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *AcceptResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *AcceptResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *AcceptResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *AcceptResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetAccepted

`func (o *AcceptResponseBody) GetAccepted() bool`

GetAccepted returns the Accepted field if non-nil, zero value otherwise.

### GetAcceptedOk

`func (o *AcceptResponseBody) GetAcceptedOk() (*bool, bool)`

GetAcceptedOk returns a tuple with the Accepted field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAccepted

`func (o *AcceptResponseBody) SetAccepted(v bool)`

SetAccepted sets Accepted field to given value.


### GetAccountCreated

`func (o *AcceptResponseBody) GetAccountCreated() bool`

GetAccountCreated returns the AccountCreated field if non-nil, zero value otherwise.

### GetAccountCreatedOk

`func (o *AcceptResponseBody) GetAccountCreatedOk() (*bool, bool)`

GetAccountCreatedOk returns a tuple with the AccountCreated field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAccountCreated

`func (o *AcceptResponseBody) SetAccountCreated(v bool)`

SetAccountCreated sets AccountCreated field to given value.


### GetEmail

`func (o *AcceptResponseBody) GetEmail() string`

GetEmail returns the Email field if non-nil, zero value otherwise.

### GetEmailOk

`func (o *AcceptResponseBody) GetEmailOk() (*string, bool)`

GetEmailOk returns a tuple with the Email field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEmail

`func (o *AcceptResponseBody) SetEmail(v string)`

SetEmail sets Email field to given value.


### GetMessage

`func (o *AcceptResponseBody) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *AcceptResponseBody) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *AcceptResponseBody) SetMessage(v string)`

SetMessage sets Message field to given value.


### GetOrganization

`func (o *AcceptResponseBody) GetOrganization() string`

GetOrganization returns the Organization field if non-nil, zero value otherwise.

### GetOrganizationOk

`func (o *AcceptResponseBody) GetOrganizationOk() (*string, bool)`

GetOrganizationOk returns a tuple with the Organization field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrganization

`func (o *AcceptResponseBody) SetOrganization(v string)`

SetOrganization sets Organization field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


