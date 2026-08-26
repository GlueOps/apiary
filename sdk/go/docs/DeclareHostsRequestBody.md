# DeclareHostsRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Addresses** | Pointer to **[]string** | Hostnames or IPs. Not URLs and not host:port — the control plane connects over SSH. | [optional] 
**Hosts** | Pointer to **string** | The same thing as pasted text: one per line, or separated by commas or spaces. Blank lines and #-comments are ignored. | [optional] 
**User** | Pointer to **string** | SSH user. Empty means root, which is what a Proxmox node has. | [optional] 

## Methods

### NewDeclareHostsRequestBody

`func NewDeclareHostsRequestBody() *DeclareHostsRequestBody`

NewDeclareHostsRequestBody instantiates a new DeclareHostsRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDeclareHostsRequestBodyWithDefaults

`func NewDeclareHostsRequestBodyWithDefaults() *DeclareHostsRequestBody`

NewDeclareHostsRequestBodyWithDefaults instantiates a new DeclareHostsRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *DeclareHostsRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *DeclareHostsRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *DeclareHostsRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *DeclareHostsRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetAddresses

`func (o *DeclareHostsRequestBody) GetAddresses() []string`

GetAddresses returns the Addresses field if non-nil, zero value otherwise.

### GetAddressesOk

`func (o *DeclareHostsRequestBody) GetAddressesOk() (*[]string, bool)`

GetAddressesOk returns a tuple with the Addresses field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAddresses

`func (o *DeclareHostsRequestBody) SetAddresses(v []string)`

SetAddresses sets Addresses field to given value.

### HasAddresses

`func (o *DeclareHostsRequestBody) HasAddresses() bool`

HasAddresses returns a boolean if a field has been set.

### SetAddressesNil

`func (o *DeclareHostsRequestBody) SetAddressesNil(b bool)`

 SetAddressesNil sets the value for Addresses to be an explicit nil

### UnsetAddresses
`func (o *DeclareHostsRequestBody) UnsetAddresses()`

UnsetAddresses ensures that no value is present for Addresses, not even an explicit nil
### GetHosts

`func (o *DeclareHostsRequestBody) GetHosts() string`

GetHosts returns the Hosts field if non-nil, zero value otherwise.

### GetHostsOk

`func (o *DeclareHostsRequestBody) GetHostsOk() (*string, bool)`

GetHostsOk returns a tuple with the Hosts field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHosts

`func (o *DeclareHostsRequestBody) SetHosts(v string)`

SetHosts sets Hosts field to given value.

### HasHosts

`func (o *DeclareHostsRequestBody) HasHosts() bool`

HasHosts returns a boolean if a field has been set.

### GetUser

`func (o *DeclareHostsRequestBody) GetUser() string`

GetUser returns the User field if non-nil, zero value otherwise.

### GetUserOk

`func (o *DeclareHostsRequestBody) GetUserOk() (*string, bool)`

GetUserOk returns a tuple with the User field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUser

`func (o *DeclareHostsRequestBody) SetUser(v string)`

SetUser sets User field to given value.

### HasUser

`func (o *DeclareHostsRequestBody) HasUser() bool`

HasUser returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


