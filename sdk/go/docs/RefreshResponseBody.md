# RefreshResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**AccessToken** | **string** |  | 
**ExpiresIn** | **int64** |  | 
**Organization** | **string** |  | 
**OrganizationId** | **string** |  | 
**RefreshExpiresIn** | **int64** |  | 
**RefreshToken** | **string** |  | 
**TokenType** | **string** |  | 

## Methods

### NewRefreshResponseBody

`func NewRefreshResponseBody(accessToken string, expiresIn int64, organization string, organizationId string, refreshExpiresIn int64, refreshToken string, tokenType string, ) *RefreshResponseBody`

NewRefreshResponseBody instantiates a new RefreshResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRefreshResponseBodyWithDefaults

`func NewRefreshResponseBodyWithDefaults() *RefreshResponseBody`

NewRefreshResponseBodyWithDefaults instantiates a new RefreshResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *RefreshResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *RefreshResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *RefreshResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *RefreshResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetAccessToken

`func (o *RefreshResponseBody) GetAccessToken() string`

GetAccessToken returns the AccessToken field if non-nil, zero value otherwise.

### GetAccessTokenOk

`func (o *RefreshResponseBody) GetAccessTokenOk() (*string, bool)`

GetAccessTokenOk returns a tuple with the AccessToken field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAccessToken

`func (o *RefreshResponseBody) SetAccessToken(v string)`

SetAccessToken sets AccessToken field to given value.


### GetExpiresIn

`func (o *RefreshResponseBody) GetExpiresIn() int64`

GetExpiresIn returns the ExpiresIn field if non-nil, zero value otherwise.

### GetExpiresInOk

`func (o *RefreshResponseBody) GetExpiresInOk() (*int64, bool)`

GetExpiresInOk returns a tuple with the ExpiresIn field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExpiresIn

`func (o *RefreshResponseBody) SetExpiresIn(v int64)`

SetExpiresIn sets ExpiresIn field to given value.


### GetOrganization

`func (o *RefreshResponseBody) GetOrganization() string`

GetOrganization returns the Organization field if non-nil, zero value otherwise.

### GetOrganizationOk

`func (o *RefreshResponseBody) GetOrganizationOk() (*string, bool)`

GetOrganizationOk returns a tuple with the Organization field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrganization

`func (o *RefreshResponseBody) SetOrganization(v string)`

SetOrganization sets Organization field to given value.


### GetOrganizationId

`func (o *RefreshResponseBody) GetOrganizationId() string`

GetOrganizationId returns the OrganizationId field if non-nil, zero value otherwise.

### GetOrganizationIdOk

`func (o *RefreshResponseBody) GetOrganizationIdOk() (*string, bool)`

GetOrganizationIdOk returns a tuple with the OrganizationId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrganizationId

`func (o *RefreshResponseBody) SetOrganizationId(v string)`

SetOrganizationId sets OrganizationId field to given value.


### GetRefreshExpiresIn

`func (o *RefreshResponseBody) GetRefreshExpiresIn() int64`

GetRefreshExpiresIn returns the RefreshExpiresIn field if non-nil, zero value otherwise.

### GetRefreshExpiresInOk

`func (o *RefreshResponseBody) GetRefreshExpiresInOk() (*int64, bool)`

GetRefreshExpiresInOk returns a tuple with the RefreshExpiresIn field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRefreshExpiresIn

`func (o *RefreshResponseBody) SetRefreshExpiresIn(v int64)`

SetRefreshExpiresIn sets RefreshExpiresIn field to given value.


### GetRefreshToken

`func (o *RefreshResponseBody) GetRefreshToken() string`

GetRefreshToken returns the RefreshToken field if non-nil, zero value otherwise.

### GetRefreshTokenOk

`func (o *RefreshResponseBody) GetRefreshTokenOk() (*string, bool)`

GetRefreshTokenOk returns a tuple with the RefreshToken field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRefreshToken

`func (o *RefreshResponseBody) SetRefreshToken(v string)`

SetRefreshToken sets RefreshToken field to given value.


### GetTokenType

`func (o *RefreshResponseBody) GetTokenType() string`

GetTokenType returns the TokenType field if non-nil, zero value otherwise.

### GetTokenTypeOk

`func (o *RefreshResponseBody) GetTokenTypeOk() (*string, bool)`

GetTokenTypeOk returns a tuple with the TokenType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTokenType

`func (o *RefreshResponseBody) SetTokenType(v string)`

SetTokenType sets TokenType field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


