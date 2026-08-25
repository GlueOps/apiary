# ImpersonateStartResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**AccessToken** | **string** |  | 
**ExpiresIn** | **int64** |  | 
**OrganizationId** | **string** |  | 
**TargetEmail** | **string** |  | 
**TokenType** | **string** |  | 

## Methods

### NewImpersonateStartResponseBody

`func NewImpersonateStartResponseBody(accessToken string, expiresIn int64, organizationId string, targetEmail string, tokenType string, ) *ImpersonateStartResponseBody`

NewImpersonateStartResponseBody instantiates a new ImpersonateStartResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewImpersonateStartResponseBodyWithDefaults

`func NewImpersonateStartResponseBodyWithDefaults() *ImpersonateStartResponseBody`

NewImpersonateStartResponseBodyWithDefaults instantiates a new ImpersonateStartResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ImpersonateStartResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ImpersonateStartResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ImpersonateStartResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ImpersonateStartResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetAccessToken

`func (o *ImpersonateStartResponseBody) GetAccessToken() string`

GetAccessToken returns the AccessToken field if non-nil, zero value otherwise.

### GetAccessTokenOk

`func (o *ImpersonateStartResponseBody) GetAccessTokenOk() (*string, bool)`

GetAccessTokenOk returns a tuple with the AccessToken field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAccessToken

`func (o *ImpersonateStartResponseBody) SetAccessToken(v string)`

SetAccessToken sets AccessToken field to given value.


### GetExpiresIn

`func (o *ImpersonateStartResponseBody) GetExpiresIn() int64`

GetExpiresIn returns the ExpiresIn field if non-nil, zero value otherwise.

### GetExpiresInOk

`func (o *ImpersonateStartResponseBody) GetExpiresInOk() (*int64, bool)`

GetExpiresInOk returns a tuple with the ExpiresIn field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExpiresIn

`func (o *ImpersonateStartResponseBody) SetExpiresIn(v int64)`

SetExpiresIn sets ExpiresIn field to given value.


### GetOrganizationId

`func (o *ImpersonateStartResponseBody) GetOrganizationId() string`

GetOrganizationId returns the OrganizationId field if non-nil, zero value otherwise.

### GetOrganizationIdOk

`func (o *ImpersonateStartResponseBody) GetOrganizationIdOk() (*string, bool)`

GetOrganizationIdOk returns a tuple with the OrganizationId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrganizationId

`func (o *ImpersonateStartResponseBody) SetOrganizationId(v string)`

SetOrganizationId sets OrganizationId field to given value.


### GetTargetEmail

`func (o *ImpersonateStartResponseBody) GetTargetEmail() string`

GetTargetEmail returns the TargetEmail field if non-nil, zero value otherwise.

### GetTargetEmailOk

`func (o *ImpersonateStartResponseBody) GetTargetEmailOk() (*string, bool)`

GetTargetEmailOk returns a tuple with the TargetEmail field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTargetEmail

`func (o *ImpersonateStartResponseBody) SetTargetEmail(v string)`

SetTargetEmail sets TargetEmail field to given value.


### GetTokenType

`func (o *ImpersonateStartResponseBody) GetTokenType() string`

GetTokenType returns the TokenType field if non-nil, zero value otherwise.

### GetTokenTypeOk

`func (o *ImpersonateStartResponseBody) GetTokenTypeOk() (*string, bool)`

GetTokenTypeOk returns a tuple with the TokenType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTokenType

`func (o *ImpersonateStartResponseBody) SetTokenType(v string)`

SetTokenType sets TokenType field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


