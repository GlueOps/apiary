# GetMeResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**CreatedAt** | **time.Time** |  | 
**Email** | **string** |  | 
**FirstName** | **string** |  | 
**Id** | **string** |  | 
**IsActive** | **bool** |  | 
**LastLoginAt** | Pointer to **time.Time** |  | [optional] 
**LastName** | **string** |  | 
**Mfa** | [**MFAStruct**](MFAStruct.md) |  | 
**Organization** | [**OrganizationStruct**](OrganizationStruct.md) |  | 
**Role** | **string** |  | 

## Methods

### NewGetMeResponseBody

`func NewGetMeResponseBody(createdAt time.Time, email string, firstName string, id string, isActive bool, lastName string, mfa MFAStruct, organization OrganizationStruct, role string, ) *GetMeResponseBody`

NewGetMeResponseBody instantiates a new GetMeResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGetMeResponseBodyWithDefaults

`func NewGetMeResponseBodyWithDefaults() *GetMeResponseBody`

NewGetMeResponseBodyWithDefaults instantiates a new GetMeResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *GetMeResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *GetMeResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *GetMeResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *GetMeResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetCreatedAt

`func (o *GetMeResponseBody) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *GetMeResponseBody) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *GetMeResponseBody) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.


### GetEmail

`func (o *GetMeResponseBody) GetEmail() string`

GetEmail returns the Email field if non-nil, zero value otherwise.

### GetEmailOk

`func (o *GetMeResponseBody) GetEmailOk() (*string, bool)`

GetEmailOk returns a tuple with the Email field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEmail

`func (o *GetMeResponseBody) SetEmail(v string)`

SetEmail sets Email field to given value.


### GetFirstName

`func (o *GetMeResponseBody) GetFirstName() string`

GetFirstName returns the FirstName field if non-nil, zero value otherwise.

### GetFirstNameOk

`func (o *GetMeResponseBody) GetFirstNameOk() (*string, bool)`

GetFirstNameOk returns a tuple with the FirstName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFirstName

`func (o *GetMeResponseBody) SetFirstName(v string)`

SetFirstName sets FirstName field to given value.


### GetId

`func (o *GetMeResponseBody) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *GetMeResponseBody) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *GetMeResponseBody) SetId(v string)`

SetId sets Id field to given value.


### GetIsActive

`func (o *GetMeResponseBody) GetIsActive() bool`

GetIsActive returns the IsActive field if non-nil, zero value otherwise.

### GetIsActiveOk

`func (o *GetMeResponseBody) GetIsActiveOk() (*bool, bool)`

GetIsActiveOk returns a tuple with the IsActive field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIsActive

`func (o *GetMeResponseBody) SetIsActive(v bool)`

SetIsActive sets IsActive field to given value.


### GetLastLoginAt

`func (o *GetMeResponseBody) GetLastLoginAt() time.Time`

GetLastLoginAt returns the LastLoginAt field if non-nil, zero value otherwise.

### GetLastLoginAtOk

`func (o *GetMeResponseBody) GetLastLoginAtOk() (*time.Time, bool)`

GetLastLoginAtOk returns a tuple with the LastLoginAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastLoginAt

`func (o *GetMeResponseBody) SetLastLoginAt(v time.Time)`

SetLastLoginAt sets LastLoginAt field to given value.

### HasLastLoginAt

`func (o *GetMeResponseBody) HasLastLoginAt() bool`

HasLastLoginAt returns a boolean if a field has been set.

### GetLastName

`func (o *GetMeResponseBody) GetLastName() string`

GetLastName returns the LastName field if non-nil, zero value otherwise.

### GetLastNameOk

`func (o *GetMeResponseBody) GetLastNameOk() (*string, bool)`

GetLastNameOk returns a tuple with the LastName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastName

`func (o *GetMeResponseBody) SetLastName(v string)`

SetLastName sets LastName field to given value.


### GetMfa

`func (o *GetMeResponseBody) GetMfa() MFAStruct`

GetMfa returns the Mfa field if non-nil, zero value otherwise.

### GetMfaOk

`func (o *GetMeResponseBody) GetMfaOk() (*MFAStruct, bool)`

GetMfaOk returns a tuple with the Mfa field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMfa

`func (o *GetMeResponseBody) SetMfa(v MFAStruct)`

SetMfa sets Mfa field to given value.


### GetOrganization

`func (o *GetMeResponseBody) GetOrganization() OrganizationStruct`

GetOrganization returns the Organization field if non-nil, zero value otherwise.

### GetOrganizationOk

`func (o *GetMeResponseBody) GetOrganizationOk() (*OrganizationStruct, bool)`

GetOrganizationOk returns a tuple with the Organization field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrganization

`func (o *GetMeResponseBody) SetOrganization(v OrganizationStruct)`

SetOrganization sets Organization field to given value.


### GetRole

`func (o *GetMeResponseBody) GetRole() string`

GetRole returns the Role field if non-nil, zero value otherwise.

### GetRoleOk

`func (o *GetMeResponseBody) GetRoleOk() (*string, bool)`

GetRoleOk returns a tuple with the Role field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRole

`func (o *GetMeResponseBody) SetRole(v string)`

SetRole sets Role field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


