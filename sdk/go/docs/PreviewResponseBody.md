# PreviewResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Email** | **string** |  | 
**NeedsAccount** | **bool** | True when this address has no account yet, so the page must ask for a name and password. False means sign in to accept. | 
**Organization** | **string** |  | 
**Role** | **string** |  | 

## Methods

### NewPreviewResponseBody

`func NewPreviewResponseBody(email string, needsAccount bool, organization string, role string, ) *PreviewResponseBody`

NewPreviewResponseBody instantiates a new PreviewResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPreviewResponseBodyWithDefaults

`func NewPreviewResponseBodyWithDefaults() *PreviewResponseBody`

NewPreviewResponseBodyWithDefaults instantiates a new PreviewResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *PreviewResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *PreviewResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *PreviewResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *PreviewResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetEmail

`func (o *PreviewResponseBody) GetEmail() string`

GetEmail returns the Email field if non-nil, zero value otherwise.

### GetEmailOk

`func (o *PreviewResponseBody) GetEmailOk() (*string, bool)`

GetEmailOk returns a tuple with the Email field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEmail

`func (o *PreviewResponseBody) SetEmail(v string)`

SetEmail sets Email field to given value.


### GetNeedsAccount

`func (o *PreviewResponseBody) GetNeedsAccount() bool`

GetNeedsAccount returns the NeedsAccount field if non-nil, zero value otherwise.

### GetNeedsAccountOk

`func (o *PreviewResponseBody) GetNeedsAccountOk() (*bool, bool)`

GetNeedsAccountOk returns a tuple with the NeedsAccount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNeedsAccount

`func (o *PreviewResponseBody) SetNeedsAccount(v bool)`

SetNeedsAccount sets NeedsAccount field to given value.


### GetOrganization

`func (o *PreviewResponseBody) GetOrganization() string`

GetOrganization returns the Organization field if non-nil, zero value otherwise.

### GetOrganizationOk

`func (o *PreviewResponseBody) GetOrganizationOk() (*string, bool)`

GetOrganizationOk returns a tuple with the Organization field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrganization

`func (o *PreviewResponseBody) SetOrganization(v string)`

SetOrganization sets Organization field to given value.


### GetRole

`func (o *PreviewResponseBody) GetRole() string`

GetRole returns the Role field if non-nil, zero value otherwise.

### GetRoleOk

`func (o *PreviewResponseBody) GetRoleOk() (*string, bool)`

GetRoleOk returns a tuple with the Role field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRole

`func (o *PreviewResponseBody) SetRole(v string)`

SetRole sets Role field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


