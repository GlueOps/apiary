# ListFleetlessResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Hosts** | [**[]FleetlessHostView**](FleetlessHostView.md) |  | 

## Methods

### NewListFleetlessResponseBody

`func NewListFleetlessResponseBody(hosts []FleetlessHostView, ) *ListFleetlessResponseBody`

NewListFleetlessResponseBody instantiates a new ListFleetlessResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListFleetlessResponseBodyWithDefaults

`func NewListFleetlessResponseBodyWithDefaults() *ListFleetlessResponseBody`

NewListFleetlessResponseBodyWithDefaults instantiates a new ListFleetlessResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListFleetlessResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListFleetlessResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListFleetlessResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListFleetlessResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetHosts

`func (o *ListFleetlessResponseBody) GetHosts() []FleetlessHostView`

GetHosts returns the Hosts field if non-nil, zero value otherwise.

### GetHostsOk

`func (o *ListFleetlessResponseBody) GetHostsOk() (*[]FleetlessHostView, bool)`

GetHostsOk returns a tuple with the Hosts field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHosts

`func (o *ListFleetlessResponseBody) SetHosts(v []FleetlessHostView)`

SetHosts sets Hosts field to given value.


### SetHostsNil

`func (o *ListFleetlessResponseBody) SetHostsNil(b bool)`

 SetHostsNil sets the value for Hosts to be an explicit nil

### UnsetHosts
`func (o *ListFleetlessResponseBody) UnsetHosts()`

UnsetHosts ensures that no value is present for Hosts, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


