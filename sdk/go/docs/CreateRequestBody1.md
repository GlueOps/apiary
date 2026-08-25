# CreateRequestBody1

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Email** | **string** |  | 
**Role** | Pointer to **string** | Owner is deliberately not invitable; ownership is transferred by an existing owner. | [optional] [default to "member"]

## Methods

### NewCreateRequestBody1

`func NewCreateRequestBody1(email string, ) *CreateRequestBody1`

NewCreateRequestBody1 instantiates a new CreateRequestBody1 object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCreateRequestBody1WithDefaults

`func NewCreateRequestBody1WithDefaults() *CreateRequestBody1`

NewCreateRequestBody1WithDefaults instantiates a new CreateRequestBody1 object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *CreateRequestBody1) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *CreateRequestBody1) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *CreateRequestBody1) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *CreateRequestBody1) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetEmail

`func (o *CreateRequestBody1) GetEmail() string`

GetEmail returns the Email field if non-nil, zero value otherwise.

### GetEmailOk

`func (o *CreateRequestBody1) GetEmailOk() (*string, bool)`

GetEmailOk returns a tuple with the Email field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEmail

`func (o *CreateRequestBody1) SetEmail(v string)`

SetEmail sets Email field to given value.


### GetRole

`func (o *CreateRequestBody1) GetRole() string`

GetRole returns the Role field if non-nil, zero value otherwise.

### GetRoleOk

`func (o *CreateRequestBody1) GetRoleOk() (*string, bool)`

GetRoleOk returns a tuple with the Role field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRole

`func (o *CreateRequestBody1) SetRole(v string)`

SetRole sets Role field to given value.

### HasRole

`func (o *CreateRequestBody1) HasRole() bool`

HasRole returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


