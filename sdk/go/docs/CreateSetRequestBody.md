# CreateSetRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Description** | Pointer to **string** |  | [optional] 
**Family** | Pointer to **string** | An nft set holds ONE address family. inet is a table family and is refused here; leave the rule inet and pick the family the addresses are. | [optional] [default to "ip"]
**Name** | **string** |  | 
**Purpose** | Pointer to **string** | management widens the from&#x3D; restriction on the fleet SSH key, so it defaults to general — the harmless one. | [optional] [default to "general"]

## Methods

### NewCreateSetRequestBody

`func NewCreateSetRequestBody(name string, ) *CreateSetRequestBody`

NewCreateSetRequestBody instantiates a new CreateSetRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCreateSetRequestBodyWithDefaults

`func NewCreateSetRequestBodyWithDefaults() *CreateSetRequestBody`

NewCreateSetRequestBodyWithDefaults instantiates a new CreateSetRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *CreateSetRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *CreateSetRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *CreateSetRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *CreateSetRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetDescription

`func (o *CreateSetRequestBody) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *CreateSetRequestBody) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *CreateSetRequestBody) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *CreateSetRequestBody) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetFamily

`func (o *CreateSetRequestBody) GetFamily() string`

GetFamily returns the Family field if non-nil, zero value otherwise.

### GetFamilyOk

`func (o *CreateSetRequestBody) GetFamilyOk() (*string, bool)`

GetFamilyOk returns a tuple with the Family field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFamily

`func (o *CreateSetRequestBody) SetFamily(v string)`

SetFamily sets Family field to given value.

### HasFamily

`func (o *CreateSetRequestBody) HasFamily() bool`

HasFamily returns a boolean if a field has been set.

### GetName

`func (o *CreateSetRequestBody) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *CreateSetRequestBody) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *CreateSetRequestBody) SetName(v string)`

SetName sets Name field to given value.


### GetPurpose

`func (o *CreateSetRequestBody) GetPurpose() string`

GetPurpose returns the Purpose field if non-nil, zero value otherwise.

### GetPurposeOk

`func (o *CreateSetRequestBody) GetPurposeOk() (*string, bool)`

GetPurposeOk returns a tuple with the Purpose field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPurpose

`func (o *CreateSetRequestBody) SetPurpose(v string)`

SetPurpose sets Purpose field to given value.

### HasPurpose

`func (o *CreateSetRequestBody) HasPurpose() bool`

HasPurpose returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


