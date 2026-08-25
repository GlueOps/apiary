# VerifyEmailResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**AlreadyVerified** | **bool** | True when this address was already confirmed — a second click on the same link. | 
**Message** | **string** |  | 
**TenantProvisioning** | **bool** | True when confirming released provisioning of the organization&#39;s database. | 
**Verified** | **bool** |  | 

## Methods

### NewVerifyEmailResponseBody

`func NewVerifyEmailResponseBody(alreadyVerified bool, message string, tenantProvisioning bool, verified bool, ) *VerifyEmailResponseBody`

NewVerifyEmailResponseBody instantiates a new VerifyEmailResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewVerifyEmailResponseBodyWithDefaults

`func NewVerifyEmailResponseBodyWithDefaults() *VerifyEmailResponseBody`

NewVerifyEmailResponseBodyWithDefaults instantiates a new VerifyEmailResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *VerifyEmailResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *VerifyEmailResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *VerifyEmailResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *VerifyEmailResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetAlreadyVerified

`func (o *VerifyEmailResponseBody) GetAlreadyVerified() bool`

GetAlreadyVerified returns the AlreadyVerified field if non-nil, zero value otherwise.

### GetAlreadyVerifiedOk

`func (o *VerifyEmailResponseBody) GetAlreadyVerifiedOk() (*bool, bool)`

GetAlreadyVerifiedOk returns a tuple with the AlreadyVerified field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAlreadyVerified

`func (o *VerifyEmailResponseBody) SetAlreadyVerified(v bool)`

SetAlreadyVerified sets AlreadyVerified field to given value.


### GetMessage

`func (o *VerifyEmailResponseBody) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *VerifyEmailResponseBody) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *VerifyEmailResponseBody) SetMessage(v string)`

SetMessage sets Message field to given value.


### GetTenantProvisioning

`func (o *VerifyEmailResponseBody) GetTenantProvisioning() bool`

GetTenantProvisioning returns the TenantProvisioning field if non-nil, zero value otherwise.

### GetTenantProvisioningOk

`func (o *VerifyEmailResponseBody) GetTenantProvisioningOk() (*bool, bool)`

GetTenantProvisioningOk returns a tuple with the TenantProvisioning field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTenantProvisioning

`func (o *VerifyEmailResponseBody) SetTenantProvisioning(v bool)`

SetTenantProvisioning sets TenantProvisioning field to given value.


### GetVerified

`func (o *VerifyEmailResponseBody) GetVerified() bool`

GetVerified returns the Verified field if non-nil, zero value otherwise.

### GetVerifiedOk

`func (o *VerifyEmailResponseBody) GetVerifiedOk() (*bool, bool)`

GetVerifiedOk returns a tuple with the Verified field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVerified

`func (o *VerifyEmailResponseBody) SetVerified(v bool)`

SetVerified sets Verified field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


