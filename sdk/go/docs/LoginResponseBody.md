# LoginResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**AccessToken** | Pointer to **string** |  | [optional] 
**ExpiresIn** | Pointer to **int64** |  | [optional] 
**MfaRequired** | Pointer to **bool** |  | [optional] 
**MfaToken** | Pointer to **string** |  | [optional] 
**Organization** | **string** |  | 
**OrganizationId** | **string** |  | 
**RefreshExpiresIn** | Pointer to **int64** |  | [optional] 
**RefreshToken** | Pointer to **string** |  | [optional] 
**TokenType** | Pointer to **string** |  | [optional] 

## Methods

### NewLoginResponseBody

`func NewLoginResponseBody(organization string, organizationId string, ) *LoginResponseBody`

NewLoginResponseBody instantiates a new LoginResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewLoginResponseBodyWithDefaults

`func NewLoginResponseBodyWithDefaults() *LoginResponseBody`

NewLoginResponseBodyWithDefaults instantiates a new LoginResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *LoginResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *LoginResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *LoginResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *LoginResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetAccessToken

`func (o *LoginResponseBody) GetAccessToken() string`

GetAccessToken returns the AccessToken field if non-nil, zero value otherwise.

### GetAccessTokenOk

`func (o *LoginResponseBody) GetAccessTokenOk() (*string, bool)`

GetAccessTokenOk returns a tuple with the AccessToken field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAccessToken

`func (o *LoginResponseBody) SetAccessToken(v string)`

SetAccessToken sets AccessToken field to given value.

### HasAccessToken

`func (o *LoginResponseBody) HasAccessToken() bool`

HasAccessToken returns a boolean if a field has been set.

### GetExpiresIn

`func (o *LoginResponseBody) GetExpiresIn() int64`

GetExpiresIn returns the ExpiresIn field if non-nil, zero value otherwise.

### GetExpiresInOk

`func (o *LoginResponseBody) GetExpiresInOk() (*int64, bool)`

GetExpiresInOk returns a tuple with the ExpiresIn field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExpiresIn

`func (o *LoginResponseBody) SetExpiresIn(v int64)`

SetExpiresIn sets ExpiresIn field to given value.

### HasExpiresIn

`func (o *LoginResponseBody) HasExpiresIn() bool`

HasExpiresIn returns a boolean if a field has been set.

### GetMfaRequired

`func (o *LoginResponseBody) GetMfaRequired() bool`

GetMfaRequired returns the MfaRequired field if non-nil, zero value otherwise.

### GetMfaRequiredOk

`func (o *LoginResponseBody) GetMfaRequiredOk() (*bool, bool)`

GetMfaRequiredOk returns a tuple with the MfaRequired field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMfaRequired

`func (o *LoginResponseBody) SetMfaRequired(v bool)`

SetMfaRequired sets MfaRequired field to given value.

### HasMfaRequired

`func (o *LoginResponseBody) HasMfaRequired() bool`

HasMfaRequired returns a boolean if a field has been set.

### GetMfaToken

`func (o *LoginResponseBody) GetMfaToken() string`

GetMfaToken returns the MfaToken field if non-nil, zero value otherwise.

### GetMfaTokenOk

`func (o *LoginResponseBody) GetMfaTokenOk() (*string, bool)`

GetMfaTokenOk returns a tuple with the MfaToken field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMfaToken

`func (o *LoginResponseBody) SetMfaToken(v string)`

SetMfaToken sets MfaToken field to given value.

### HasMfaToken

`func (o *LoginResponseBody) HasMfaToken() bool`

HasMfaToken returns a boolean if a field has been set.

### GetOrganization

`func (o *LoginResponseBody) GetOrganization() string`

GetOrganization returns the Organization field if non-nil, zero value otherwise.

### GetOrganizationOk

`func (o *LoginResponseBody) GetOrganizationOk() (*string, bool)`

GetOrganizationOk returns a tuple with the Organization field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrganization

`func (o *LoginResponseBody) SetOrganization(v string)`

SetOrganization sets Organization field to given value.


### GetOrganizationId

`func (o *LoginResponseBody) GetOrganizationId() string`

GetOrganizationId returns the OrganizationId field if non-nil, zero value otherwise.

### GetOrganizationIdOk

`func (o *LoginResponseBody) GetOrganizationIdOk() (*string, bool)`

GetOrganizationIdOk returns a tuple with the OrganizationId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrganizationId

`func (o *LoginResponseBody) SetOrganizationId(v string)`

SetOrganizationId sets OrganizationId field to given value.


### GetRefreshExpiresIn

`func (o *LoginResponseBody) GetRefreshExpiresIn() int64`

GetRefreshExpiresIn returns the RefreshExpiresIn field if non-nil, zero value otherwise.

### GetRefreshExpiresInOk

`func (o *LoginResponseBody) GetRefreshExpiresInOk() (*int64, bool)`

GetRefreshExpiresInOk returns a tuple with the RefreshExpiresIn field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRefreshExpiresIn

`func (o *LoginResponseBody) SetRefreshExpiresIn(v int64)`

SetRefreshExpiresIn sets RefreshExpiresIn field to given value.

### HasRefreshExpiresIn

`func (o *LoginResponseBody) HasRefreshExpiresIn() bool`

HasRefreshExpiresIn returns a boolean if a field has been set.

### GetRefreshToken

`func (o *LoginResponseBody) GetRefreshToken() string`

GetRefreshToken returns the RefreshToken field if non-nil, zero value otherwise.

### GetRefreshTokenOk

`func (o *LoginResponseBody) GetRefreshTokenOk() (*string, bool)`

GetRefreshTokenOk returns a tuple with the RefreshToken field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRefreshToken

`func (o *LoginResponseBody) SetRefreshToken(v string)`

SetRefreshToken sets RefreshToken field to given value.

### HasRefreshToken

`func (o *LoginResponseBody) HasRefreshToken() bool`

HasRefreshToken returns a boolean if a field has been set.

### GetTokenType

`func (o *LoginResponseBody) GetTokenType() string`

GetTokenType returns the TokenType field if non-nil, zero value otherwise.

### GetTokenTypeOk

`func (o *LoginResponseBody) GetTokenTypeOk() (*string, bool)`

GetTokenTypeOk returns a tuple with the TokenType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTokenType

`func (o *LoginResponseBody) SetTokenType(v string)`

SetTokenType sets TokenType field to given value.

### HasTokenType

`func (o *LoginResponseBody) HasTokenType() bool`

HasTokenType returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


