# RegisterRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Email** | **string** |  | 
**FirstName** | **string** |  | 
**LastName** | Pointer to **string** |  | [optional] 
**OrganizationName** | Pointer to **string** | Blank derives a workspace name from the email. | [optional] 
**Password** | **string** |  | 

## Methods

### NewRegisterRequestBody

`func NewRegisterRequestBody(email string, firstName string, password string, ) *RegisterRequestBody`

NewRegisterRequestBody instantiates a new RegisterRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRegisterRequestBodyWithDefaults

`func NewRegisterRequestBodyWithDefaults() *RegisterRequestBody`

NewRegisterRequestBodyWithDefaults instantiates a new RegisterRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *RegisterRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *RegisterRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *RegisterRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *RegisterRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetEmail

`func (o *RegisterRequestBody) GetEmail() string`

GetEmail returns the Email field if non-nil, zero value otherwise.

### GetEmailOk

`func (o *RegisterRequestBody) GetEmailOk() (*string, bool)`

GetEmailOk returns a tuple with the Email field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEmail

`func (o *RegisterRequestBody) SetEmail(v string)`

SetEmail sets Email field to given value.


### GetFirstName

`func (o *RegisterRequestBody) GetFirstName() string`

GetFirstName returns the FirstName field if non-nil, zero value otherwise.

### GetFirstNameOk

`func (o *RegisterRequestBody) GetFirstNameOk() (*string, bool)`

GetFirstNameOk returns a tuple with the FirstName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFirstName

`func (o *RegisterRequestBody) SetFirstName(v string)`

SetFirstName sets FirstName field to given value.


### GetLastName

`func (o *RegisterRequestBody) GetLastName() string`

GetLastName returns the LastName field if non-nil, zero value otherwise.

### GetLastNameOk

`func (o *RegisterRequestBody) GetLastNameOk() (*string, bool)`

GetLastNameOk returns a tuple with the LastName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastName

`func (o *RegisterRequestBody) SetLastName(v string)`

SetLastName sets LastName field to given value.

### HasLastName

`func (o *RegisterRequestBody) HasLastName() bool`

HasLastName returns a boolean if a field has been set.

### GetOrganizationName

`func (o *RegisterRequestBody) GetOrganizationName() string`

GetOrganizationName returns the OrganizationName field if non-nil, zero value otherwise.

### GetOrganizationNameOk

`func (o *RegisterRequestBody) GetOrganizationNameOk() (*string, bool)`

GetOrganizationNameOk returns a tuple with the OrganizationName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrganizationName

`func (o *RegisterRequestBody) SetOrganizationName(v string)`

SetOrganizationName sets OrganizationName field to given value.

### HasOrganizationName

`func (o *RegisterRequestBody) HasOrganizationName() bool`

HasOrganizationName returns a boolean if a field has been set.

### GetPassword

`func (o *RegisterRequestBody) GetPassword() string`

GetPassword returns the Password field if non-nil, zero value otherwise.

### GetPasswordOk

`func (o *RegisterRequestBody) GetPasswordOk() (*string, bool)`

GetPasswordOk returns a tuple with the Password field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPassword

`func (o *RegisterRequestBody) SetPassword(v string)`

SetPassword sets Password field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


