# DisarmRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**CredentialDisposition** | Pointer to **string** | Omit to reuse the disposition recorded when this host was departed. | [optional] 

## Methods

### NewDisarmRequestBody

`func NewDisarmRequestBody() *DisarmRequestBody`

NewDisarmRequestBody instantiates a new DisarmRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDisarmRequestBodyWithDefaults

`func NewDisarmRequestBodyWithDefaults() *DisarmRequestBody`

NewDisarmRequestBodyWithDefaults instantiates a new DisarmRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *DisarmRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *DisarmRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *DisarmRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *DisarmRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetCredentialDisposition

`func (o *DisarmRequestBody) GetCredentialDisposition() string`

GetCredentialDisposition returns the CredentialDisposition field if non-nil, zero value otherwise.

### GetCredentialDispositionOk

`func (o *DisarmRequestBody) GetCredentialDispositionOk() (*string, bool)`

GetCredentialDispositionOk returns a tuple with the CredentialDisposition field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCredentialDisposition

`func (o *DisarmRequestBody) SetCredentialDisposition(v string)`

SetCredentialDisposition sets CredentialDisposition field to given value.

### HasCredentialDisposition

`func (o *DisarmRequestBody) HasCredentialDisposition() bool`

HasCredentialDisposition returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


