# CredentialDescriptor

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **string** |  | 
**Transports** | Pointer to **[]string** |  | [optional] 
**Type** | **string** |  | 

## Methods

### NewCredentialDescriptor

`func NewCredentialDescriptor(id string, type_ string, ) *CredentialDescriptor`

NewCredentialDescriptor instantiates a new CredentialDescriptor object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCredentialDescriptorWithDefaults

`func NewCredentialDescriptorWithDefaults() *CredentialDescriptor`

NewCredentialDescriptorWithDefaults instantiates a new CredentialDescriptor object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *CredentialDescriptor) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *CredentialDescriptor) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *CredentialDescriptor) SetId(v string)`

SetId sets Id field to given value.


### GetTransports

`func (o *CredentialDescriptor) GetTransports() []string`

GetTransports returns the Transports field if non-nil, zero value otherwise.

### GetTransportsOk

`func (o *CredentialDescriptor) GetTransportsOk() (*[]string, bool)`

GetTransportsOk returns a tuple with the Transports field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTransports

`func (o *CredentialDescriptor) SetTransports(v []string)`

SetTransports sets Transports field to given value.

### HasTransports

`func (o *CredentialDescriptor) HasTransports() bool`

HasTransports returns a boolean if a field has been set.

### SetTransportsNil

`func (o *CredentialDescriptor) SetTransportsNil(b bool)`

 SetTransportsNil sets the value for Transports to be an explicit nil

### UnsetTransports
`func (o *CredentialDescriptor) UnsetTransports()`

UnsetTransports ensures that no value is present for Transports, not even an explicit nil
### GetType

`func (o *CredentialDescriptor) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *CredentialDescriptor) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *CredentialDescriptor) SetType(v string)`

SetType sets Type field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


