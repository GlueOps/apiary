# AcceptRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**FirstName** | Pointer to **string** |  | [optional] 
**LastName** | Pointer to **string** |  | [optional] 
**Password** | Pointer to **string** | Only for an address that has no account yet. Supplying one for an existing account is refused, not ignored. | [optional] 
**Token** | **string** |  | 

## Methods

### NewAcceptRequestBody

`func NewAcceptRequestBody(token string, ) *AcceptRequestBody`

NewAcceptRequestBody instantiates a new AcceptRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAcceptRequestBodyWithDefaults

`func NewAcceptRequestBodyWithDefaults() *AcceptRequestBody`

NewAcceptRequestBodyWithDefaults instantiates a new AcceptRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *AcceptRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *AcceptRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *AcceptRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *AcceptRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetFirstName

`func (o *AcceptRequestBody) GetFirstName() string`

GetFirstName returns the FirstName field if non-nil, zero value otherwise.

### GetFirstNameOk

`func (o *AcceptRequestBody) GetFirstNameOk() (*string, bool)`

GetFirstNameOk returns a tuple with the FirstName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFirstName

`func (o *AcceptRequestBody) SetFirstName(v string)`

SetFirstName sets FirstName field to given value.

### HasFirstName

`func (o *AcceptRequestBody) HasFirstName() bool`

HasFirstName returns a boolean if a field has been set.

### GetLastName

`func (o *AcceptRequestBody) GetLastName() string`

GetLastName returns the LastName field if non-nil, zero value otherwise.

### GetLastNameOk

`func (o *AcceptRequestBody) GetLastNameOk() (*string, bool)`

GetLastNameOk returns a tuple with the LastName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastName

`func (o *AcceptRequestBody) SetLastName(v string)`

SetLastName sets LastName field to given value.

### HasLastName

`func (o *AcceptRequestBody) HasLastName() bool`

HasLastName returns a boolean if a field has been set.

### GetPassword

`func (o *AcceptRequestBody) GetPassword() string`

GetPassword returns the Password field if non-nil, zero value otherwise.

### GetPasswordOk

`func (o *AcceptRequestBody) GetPasswordOk() (*string, bool)`

GetPasswordOk returns a tuple with the Password field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPassword

`func (o *AcceptRequestBody) SetPassword(v string)`

SetPassword sets Password field to given value.

### HasPassword

`func (o *AcceptRequestBody) HasPassword() bool`

HasPassword returns a boolean if a field has been set.

### GetToken

`func (o *AcceptRequestBody) GetToken() string`

GetToken returns the Token field if non-nil, zero value otherwise.

### GetTokenOk

`func (o *AcceptRequestBody) GetTokenOk() (*string, bool)`

GetTokenOk returns a tuple with the Token field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetToken

`func (o *AcceptRequestBody) SetToken(v string)`

SetToken sets Token field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


