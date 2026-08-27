# BootstrapRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Links** | [**[]CorosyncLinkInput**](CorosyncLinkInput.md) | The first node&#39;s corosync rings. Every node that joins later inherits this ring configuration. | 
**TargetHiveId** | **string** |  | 

## Methods

### NewBootstrapRequestBody

`func NewBootstrapRequestBody(links []CorosyncLinkInput, targetHiveId string, ) *BootstrapRequestBody`

NewBootstrapRequestBody instantiates a new BootstrapRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewBootstrapRequestBodyWithDefaults

`func NewBootstrapRequestBodyWithDefaults() *BootstrapRequestBody`

NewBootstrapRequestBodyWithDefaults instantiates a new BootstrapRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *BootstrapRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *BootstrapRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *BootstrapRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *BootstrapRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetLinks

`func (o *BootstrapRequestBody) GetLinks() []CorosyncLinkInput`

GetLinks returns the Links field if non-nil, zero value otherwise.

### GetLinksOk

`func (o *BootstrapRequestBody) GetLinksOk() (*[]CorosyncLinkInput, bool)`

GetLinksOk returns a tuple with the Links field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLinks

`func (o *BootstrapRequestBody) SetLinks(v []CorosyncLinkInput)`

SetLinks sets Links field to given value.


### SetLinksNil

`func (o *BootstrapRequestBody) SetLinksNil(b bool)`

 SetLinksNil sets the value for Links to be an explicit nil

### UnsetLinks
`func (o *BootstrapRequestBody) UnsetLinks()`

UnsetLinks ensures that no value is present for Links, not even an explicit nil
### GetTargetHiveId

`func (o *BootstrapRequestBody) GetTargetHiveId() string`

GetTargetHiveId returns the TargetHiveId field if non-nil, zero value otherwise.

### GetTargetHiveIdOk

`func (o *BootstrapRequestBody) GetTargetHiveIdOk() (*string, bool)`

GetTargetHiveIdOk returns a tuple with the TargetHiveId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTargetHiveId

`func (o *BootstrapRequestBody) SetTargetHiveId(v string)`

SetTargetHiveId sets TargetHiveId field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


