# UpdateHiveRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**ApiUrl** | Pointer to **string** | e.g. https://10.0.0.1:8006/api2/json | [optional] 
**CpuOvercommitRatio** | Pointer to **float64** |  | [optional] 
**CredentialId** | Pointer to **string** | A credential id to bind, or \&quot;\&quot; to unbind. Omit the field to leave the binding unchanged. | [optional] 
**Description** | Pointer to **string** |  | [optional] 
**InsecureSkipVerify** | Pointer to **bool** | Accepts a self-signed API certificate. The sealed token is handed to whatever answers on that address, so set it deliberately and per hive. | [optional] 
**MinimumNodes** | Pointer to **int64** | Lower this only for a test fleet. Below three, a departure strands the nodes that remain. | [optional] 
**Name** | Pointer to **string** | Also the Proxmox cluster name when this cluster is created on its first node, so it must be a legal one: letters, digits and hyphens, starting and ending alphanumeric. | [optional] 

## Methods

### NewUpdateHiveRequestBody

`func NewUpdateHiveRequestBody() *UpdateHiveRequestBody`

NewUpdateHiveRequestBody instantiates a new UpdateHiveRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewUpdateHiveRequestBodyWithDefaults

`func NewUpdateHiveRequestBodyWithDefaults() *UpdateHiveRequestBody`

NewUpdateHiveRequestBodyWithDefaults instantiates a new UpdateHiveRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *UpdateHiveRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *UpdateHiveRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *UpdateHiveRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *UpdateHiveRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetApiUrl

`func (o *UpdateHiveRequestBody) GetApiUrl() string`

GetApiUrl returns the ApiUrl field if non-nil, zero value otherwise.

### GetApiUrlOk

`func (o *UpdateHiveRequestBody) GetApiUrlOk() (*string, bool)`

GetApiUrlOk returns a tuple with the ApiUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApiUrl

`func (o *UpdateHiveRequestBody) SetApiUrl(v string)`

SetApiUrl sets ApiUrl field to given value.

### HasApiUrl

`func (o *UpdateHiveRequestBody) HasApiUrl() bool`

HasApiUrl returns a boolean if a field has been set.

### GetCpuOvercommitRatio

`func (o *UpdateHiveRequestBody) GetCpuOvercommitRatio() float64`

GetCpuOvercommitRatio returns the CpuOvercommitRatio field if non-nil, zero value otherwise.

### GetCpuOvercommitRatioOk

`func (o *UpdateHiveRequestBody) GetCpuOvercommitRatioOk() (*float64, bool)`

GetCpuOvercommitRatioOk returns a tuple with the CpuOvercommitRatio field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCpuOvercommitRatio

`func (o *UpdateHiveRequestBody) SetCpuOvercommitRatio(v float64)`

SetCpuOvercommitRatio sets CpuOvercommitRatio field to given value.

### HasCpuOvercommitRatio

`func (o *UpdateHiveRequestBody) HasCpuOvercommitRatio() bool`

HasCpuOvercommitRatio returns a boolean if a field has been set.

### GetCredentialId

`func (o *UpdateHiveRequestBody) GetCredentialId() string`

GetCredentialId returns the CredentialId field if non-nil, zero value otherwise.

### GetCredentialIdOk

`func (o *UpdateHiveRequestBody) GetCredentialIdOk() (*string, bool)`

GetCredentialIdOk returns a tuple with the CredentialId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCredentialId

`func (o *UpdateHiveRequestBody) SetCredentialId(v string)`

SetCredentialId sets CredentialId field to given value.

### HasCredentialId

`func (o *UpdateHiveRequestBody) HasCredentialId() bool`

HasCredentialId returns a boolean if a field has been set.

### GetDescription

`func (o *UpdateHiveRequestBody) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *UpdateHiveRequestBody) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *UpdateHiveRequestBody) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *UpdateHiveRequestBody) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetInsecureSkipVerify

`func (o *UpdateHiveRequestBody) GetInsecureSkipVerify() bool`

GetInsecureSkipVerify returns the InsecureSkipVerify field if non-nil, zero value otherwise.

### GetInsecureSkipVerifyOk

`func (o *UpdateHiveRequestBody) GetInsecureSkipVerifyOk() (*bool, bool)`

GetInsecureSkipVerifyOk returns a tuple with the InsecureSkipVerify field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInsecureSkipVerify

`func (o *UpdateHiveRequestBody) SetInsecureSkipVerify(v bool)`

SetInsecureSkipVerify sets InsecureSkipVerify field to given value.

### HasInsecureSkipVerify

`func (o *UpdateHiveRequestBody) HasInsecureSkipVerify() bool`

HasInsecureSkipVerify returns a boolean if a field has been set.

### GetMinimumNodes

`func (o *UpdateHiveRequestBody) GetMinimumNodes() int64`

GetMinimumNodes returns the MinimumNodes field if non-nil, zero value otherwise.

### GetMinimumNodesOk

`func (o *UpdateHiveRequestBody) GetMinimumNodesOk() (*int64, bool)`

GetMinimumNodesOk returns a tuple with the MinimumNodes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMinimumNodes

`func (o *UpdateHiveRequestBody) SetMinimumNodes(v int64)`

SetMinimumNodes sets MinimumNodes field to given value.

### HasMinimumNodes

`func (o *UpdateHiveRequestBody) HasMinimumNodes() bool`

HasMinimumNodes returns a boolean if a field has been set.

### GetName

`func (o *UpdateHiveRequestBody) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *UpdateHiveRequestBody) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *UpdateHiveRequestBody) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *UpdateHiveRequestBody) HasName() bool`

HasName returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


