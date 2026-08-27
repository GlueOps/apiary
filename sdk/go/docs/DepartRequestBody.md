# DepartRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**CredentialDisposition** | **string** | What to do about the cluster secrets this host has seen. \&quot;rotate\&quot; if the host is not trusted: it can otherwise mint auth tickets for the cluster it left. \&quot;accept-exposure\&quot; if it is trusted and being redeployed. | 

## Methods

### NewDepartRequestBody

`func NewDepartRequestBody(credentialDisposition string, ) *DepartRequestBody`

NewDepartRequestBody instantiates a new DepartRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDepartRequestBodyWithDefaults

`func NewDepartRequestBodyWithDefaults() *DepartRequestBody`

NewDepartRequestBodyWithDefaults instantiates a new DepartRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *DepartRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *DepartRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *DepartRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *DepartRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetCredentialDisposition

`func (o *DepartRequestBody) GetCredentialDisposition() string`

GetCredentialDisposition returns the CredentialDisposition field if non-nil, zero value otherwise.

### GetCredentialDispositionOk

`func (o *DepartRequestBody) GetCredentialDispositionOk() (*string, bool)`

GetCredentialDispositionOk returns a tuple with the CredentialDisposition field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCredentialDisposition

`func (o *DepartRequestBody) SetCredentialDisposition(v string)`

SetCredentialDisposition sets CredentialDisposition field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


