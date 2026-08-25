# CreateHiveRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Description** | Pointer to **string** |  | [optional] 
**Name** | **string** |  | 
**Provider** | Pointer to **string** |  | [optional] [default to "proxmox"]

## Methods

### NewCreateHiveRequestBody

`func NewCreateHiveRequestBody(name string, ) *CreateHiveRequestBody`

NewCreateHiveRequestBody instantiates a new CreateHiveRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCreateHiveRequestBodyWithDefaults

`func NewCreateHiveRequestBodyWithDefaults() *CreateHiveRequestBody`

NewCreateHiveRequestBodyWithDefaults instantiates a new CreateHiveRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *CreateHiveRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *CreateHiveRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *CreateHiveRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *CreateHiveRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetDescription

`func (o *CreateHiveRequestBody) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *CreateHiveRequestBody) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *CreateHiveRequestBody) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *CreateHiveRequestBody) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetName

`func (o *CreateHiveRequestBody) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *CreateHiveRequestBody) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *CreateHiveRequestBody) SetName(v string)`

SetName sets Name field to given value.


### GetProvider

`func (o *CreateHiveRequestBody) GetProvider() string`

GetProvider returns the Provider field if non-nil, zero value otherwise.

### GetProviderOk

`func (o *CreateHiveRequestBody) GetProviderOk() (*string, bool)`

GetProviderOk returns a tuple with the Provider field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProvider

`func (o *CreateHiveRequestBody) SetProvider(v string)`

SetProvider sets Provider field to given value.

### HasProvider

`func (o *CreateHiveRequestBody) HasProvider() bool`

HasProvider returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


