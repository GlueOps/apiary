# ImpersonateStartRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**OrganizationId** | **string** |  | 
**Reason** | **string** | Recorded in the audit trail. Required: an impersonation with no stated reason is not reviewable. | 
**UserId** | **string** |  | 

## Methods

### NewImpersonateStartRequestBody

`func NewImpersonateStartRequestBody(organizationId string, reason string, userId string, ) *ImpersonateStartRequestBody`

NewImpersonateStartRequestBody instantiates a new ImpersonateStartRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewImpersonateStartRequestBodyWithDefaults

`func NewImpersonateStartRequestBodyWithDefaults() *ImpersonateStartRequestBody`

NewImpersonateStartRequestBodyWithDefaults instantiates a new ImpersonateStartRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ImpersonateStartRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ImpersonateStartRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ImpersonateStartRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ImpersonateStartRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetOrganizationId

`func (o *ImpersonateStartRequestBody) GetOrganizationId() string`

GetOrganizationId returns the OrganizationId field if non-nil, zero value otherwise.

### GetOrganizationIdOk

`func (o *ImpersonateStartRequestBody) GetOrganizationIdOk() (*string, bool)`

GetOrganizationIdOk returns a tuple with the OrganizationId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrganizationId

`func (o *ImpersonateStartRequestBody) SetOrganizationId(v string)`

SetOrganizationId sets OrganizationId field to given value.


### GetReason

`func (o *ImpersonateStartRequestBody) GetReason() string`

GetReason returns the Reason field if non-nil, zero value otherwise.

### GetReasonOk

`func (o *ImpersonateStartRequestBody) GetReasonOk() (*string, bool)`

GetReasonOk returns a tuple with the Reason field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReason

`func (o *ImpersonateStartRequestBody) SetReason(v string)`

SetReason sets Reason field to given value.


### GetUserId

`func (o *ImpersonateStartRequestBody) GetUserId() string`

GetUserId returns the UserId field if non-nil, zero value otherwise.

### GetUserIdOk

`func (o *ImpersonateStartRequestBody) GetUserIdOk() (*string, bool)`

GetUserIdOk returns a tuple with the UserId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUserId

`func (o *ImpersonateStartRequestBody) SetUserId(v string)`

SetUserId sets UserId field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


