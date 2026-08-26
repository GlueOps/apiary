# DeclareHostsResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Created** | **int64** |  | 
**Hosts** | [**[]DeclaredHostView**](DeclaredHostView.md) |  | 

## Methods

### NewDeclareHostsResponseBody

`func NewDeclareHostsResponseBody(created int64, hosts []DeclaredHostView, ) *DeclareHostsResponseBody`

NewDeclareHostsResponseBody instantiates a new DeclareHostsResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDeclareHostsResponseBodyWithDefaults

`func NewDeclareHostsResponseBodyWithDefaults() *DeclareHostsResponseBody`

NewDeclareHostsResponseBodyWithDefaults instantiates a new DeclareHostsResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *DeclareHostsResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *DeclareHostsResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *DeclareHostsResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *DeclareHostsResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetCreated

`func (o *DeclareHostsResponseBody) GetCreated() int64`

GetCreated returns the Created field if non-nil, zero value otherwise.

### GetCreatedOk

`func (o *DeclareHostsResponseBody) GetCreatedOk() (*int64, bool)`

GetCreatedOk returns a tuple with the Created field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreated

`func (o *DeclareHostsResponseBody) SetCreated(v int64)`

SetCreated sets Created field to given value.


### GetHosts

`func (o *DeclareHostsResponseBody) GetHosts() []DeclaredHostView`

GetHosts returns the Hosts field if non-nil, zero value otherwise.

### GetHostsOk

`func (o *DeclareHostsResponseBody) GetHostsOk() (*[]DeclaredHostView, bool)`

GetHostsOk returns a tuple with the Hosts field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHosts

`func (o *DeclareHostsResponseBody) SetHosts(v []DeclaredHostView)`

SetHosts sets Hosts field to given value.


### SetHostsNil

`func (o *DeclareHostsResponseBody) SetHostsNil(b bool)`

 SetHostsNil sets the value for Hosts to be an explicit nil

### UnsetHosts
`func (o *DeclareHostsResponseBody) UnsetHosts()`

UnsetHosts ensures that no value is present for Hosts, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


