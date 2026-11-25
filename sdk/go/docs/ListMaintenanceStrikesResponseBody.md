# ListMaintenanceStrikesResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Hypervisors** | [**[]MaintenanceStrikesView**](MaintenanceStrikesView.md) |  | 

## Methods

### NewListMaintenanceStrikesResponseBody

`func NewListMaintenanceStrikesResponseBody(hypervisors []MaintenanceStrikesView, ) *ListMaintenanceStrikesResponseBody`

NewListMaintenanceStrikesResponseBody instantiates a new ListMaintenanceStrikesResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListMaintenanceStrikesResponseBodyWithDefaults

`func NewListMaintenanceStrikesResponseBodyWithDefaults() *ListMaintenanceStrikesResponseBody`

NewListMaintenanceStrikesResponseBodyWithDefaults instantiates a new ListMaintenanceStrikesResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListMaintenanceStrikesResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListMaintenanceStrikesResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListMaintenanceStrikesResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListMaintenanceStrikesResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetHypervisors

`func (o *ListMaintenanceStrikesResponseBody) GetHypervisors() []MaintenanceStrikesView`

GetHypervisors returns the Hypervisors field if non-nil, zero value otherwise.

### GetHypervisorsOk

`func (o *ListMaintenanceStrikesResponseBody) GetHypervisorsOk() (*[]MaintenanceStrikesView, bool)`

GetHypervisorsOk returns a tuple with the Hypervisors field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHypervisors

`func (o *ListMaintenanceStrikesResponseBody) SetHypervisors(v []MaintenanceStrikesView)`

SetHypervisors sets Hypervisors field to given value.


### SetHypervisorsNil

`func (o *ListMaintenanceStrikesResponseBody) SetHypervisorsNil(b bool)`

 SetHypervisorsNil sets the value for Hypervisors to be an explicit nil

### UnsetHypervisors
`func (o *ListMaintenanceStrikesResponseBody) UnsetHypervisors()`

UnsetHypervisors ensures that no value is present for Hypervisors, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


