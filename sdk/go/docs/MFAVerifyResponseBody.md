# MFAVerifyResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**AccessToken** | **string** |  | 
**ExpiresIn** | **int64** |  | 
**OrganizationId** | **string** |  | 
**RefreshExpiresIn** | **int64** |  | 
**RefreshToken** | **string** |  | 
**TokenType** | **string** |  | 

## Methods

### NewMFAVerifyResponseBody

`func NewMFAVerifyResponseBody(accessToken string, expiresIn int64, organizationId string, refreshExpiresIn int64, refreshToken string, tokenType string, ) *MFAVerifyResponseBody`

NewMFAVerifyResponseBody instantiates a new MFAVerifyResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMFAVerifyResponseBodyWithDefaults

`func NewMFAVerifyResponseBodyWithDefaults() *MFAVerifyResponseBody`

NewMFAVerifyResponseBodyWithDefaults instantiates a new MFAVerifyResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *MFAVerifyResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *MFAVerifyResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *MFAVerifyResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *MFAVerifyResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetAccessToken

`func (o *MFAVerifyResponseBody) GetAccessToken() string`

GetAccessToken returns the AccessToken field if non-nil, zero value otherwise.

### GetAccessTokenOk

`func (o *MFAVerifyResponseBody) GetAccessTokenOk() (*string, bool)`

GetAccessTokenOk returns a tuple with the AccessToken field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAccessToken

`func (o *MFAVerifyResponseBody) SetAccessToken(v string)`

SetAccessToken sets AccessToken field to given value.


### GetExpiresIn

`func (o *MFAVerifyResponseBody) GetExpiresIn() int64`

GetExpiresIn returns the ExpiresIn field if non-nil, zero value otherwise.

### GetExpiresInOk

`func (o *MFAVerifyResponseBody) GetExpiresInOk() (*int64, bool)`

GetExpiresInOk returns a tuple with the ExpiresIn field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExpiresIn

`func (o *MFAVerifyResponseBody) SetExpiresIn(v int64)`

SetExpiresIn sets ExpiresIn field to given value.


### GetOrganizationId

`func (o *MFAVerifyResponseBody) GetOrganizationId() string`

GetOrganizationId returns the OrganizationId field if non-nil, zero value otherwise.

### GetOrganizationIdOk

`func (o *MFAVerifyResponseBody) GetOrganizationIdOk() (*string, bool)`

GetOrganizationIdOk returns a tuple with the OrganizationId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrganizationId

`func (o *MFAVerifyResponseBody) SetOrganizationId(v string)`

SetOrganizationId sets OrganizationId field to given value.


### GetRefreshExpiresIn

`func (o *MFAVerifyResponseBody) GetRefreshExpiresIn() int64`

GetRefreshExpiresIn returns the RefreshExpiresIn field if non-nil, zero value otherwise.

### GetRefreshExpiresInOk

`func (o *MFAVerifyResponseBody) GetRefreshExpiresInOk() (*int64, bool)`

GetRefreshExpiresInOk returns a tuple with the RefreshExpiresIn field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRefreshExpiresIn

`func (o *MFAVerifyResponseBody) SetRefreshExpiresIn(v int64)`

SetRefreshExpiresIn sets RefreshExpiresIn field to given value.


### GetRefreshToken

`func (o *MFAVerifyResponseBody) GetRefreshToken() string`

GetRefreshToken returns the RefreshToken field if non-nil, zero value otherwise.

### GetRefreshTokenOk

`func (o *MFAVerifyResponseBody) GetRefreshTokenOk() (*string, bool)`

GetRefreshTokenOk returns a tuple with the RefreshToken field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRefreshToken

`func (o *MFAVerifyResponseBody) SetRefreshToken(v string)`

SetRefreshToken sets RefreshToken field to given value.


### GetTokenType

`func (o *MFAVerifyResponseBody) GetTokenType() string`

GetTokenType returns the TokenType field if non-nil, zero value otherwise.

### GetTokenTypeOk

`func (o *MFAVerifyResponseBody) GetTokenTypeOk() (*string, bool)`

GetTokenTypeOk returns a tuple with the TokenType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTokenType

`func (o *MFAVerifyResponseBody) SetTokenType(v string)`

SetTokenType sets TokenType field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


