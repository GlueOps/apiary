# MintTokenResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**BaseUrl** | **string** | The address the agent should enrol against. Empty if the control plane has no absolute BASE_URL configured, in which case --base-url must be supplied by hand. | 
**ExpiresAt** | **time.Time** |  | 
**Token** | **string** | Returned once. Only its hash is stored — a leak of the table must not yield a token that can enrol a machine into the fleet. | 

## Methods

### NewMintTokenResponseBody

`func NewMintTokenResponseBody(baseUrl string, expiresAt time.Time, token string, ) *MintTokenResponseBody`

NewMintTokenResponseBody instantiates a new MintTokenResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMintTokenResponseBodyWithDefaults

`func NewMintTokenResponseBodyWithDefaults() *MintTokenResponseBody`

NewMintTokenResponseBodyWithDefaults instantiates a new MintTokenResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *MintTokenResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *MintTokenResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *MintTokenResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *MintTokenResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetBaseUrl

`func (o *MintTokenResponseBody) GetBaseUrl() string`

GetBaseUrl returns the BaseUrl field if non-nil, zero value otherwise.

### GetBaseUrlOk

`func (o *MintTokenResponseBody) GetBaseUrlOk() (*string, bool)`

GetBaseUrlOk returns a tuple with the BaseUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBaseUrl

`func (o *MintTokenResponseBody) SetBaseUrl(v string)`

SetBaseUrl sets BaseUrl field to given value.


### GetExpiresAt

`func (o *MintTokenResponseBody) GetExpiresAt() time.Time`

GetExpiresAt returns the ExpiresAt field if non-nil, zero value otherwise.

### GetExpiresAtOk

`func (o *MintTokenResponseBody) GetExpiresAtOk() (*time.Time, bool)`

GetExpiresAtOk returns a tuple with the ExpiresAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExpiresAt

`func (o *MintTokenResponseBody) SetExpiresAt(v time.Time)`

SetExpiresAt sets ExpiresAt field to given value.


### GetToken

`func (o *MintTokenResponseBody) GetToken() string`

GetToken returns the Token field if non-nil, zero value otherwise.

### GetTokenOk

`func (o *MintTokenResponseBody) GetTokenOk() (*string, bool)`

GetTokenOk returns a tuple with the Token field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetToken

`func (o *MintTokenResponseBody) SetToken(v string)`

SetToken sets Token field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


