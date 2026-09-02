# ListOrganizationsResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Organizations** | [**[]OrganizationSummaryView**](OrganizationSummaryView.md) |  | 

## Methods

### NewListOrganizationsResponseBody

`func NewListOrganizationsResponseBody(organizations []OrganizationSummaryView, ) *ListOrganizationsResponseBody`

NewListOrganizationsResponseBody instantiates a new ListOrganizationsResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListOrganizationsResponseBodyWithDefaults

`func NewListOrganizationsResponseBodyWithDefaults() *ListOrganizationsResponseBody`

NewListOrganizationsResponseBodyWithDefaults instantiates a new ListOrganizationsResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListOrganizationsResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListOrganizationsResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListOrganizationsResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListOrganizationsResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetOrganizations

`func (o *ListOrganizationsResponseBody) GetOrganizations() []OrganizationSummaryView`

GetOrganizations returns the Organizations field if non-nil, zero value otherwise.

### GetOrganizationsOk

`func (o *ListOrganizationsResponseBody) GetOrganizationsOk() (*[]OrganizationSummaryView, bool)`

GetOrganizationsOk returns a tuple with the Organizations field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrganizations

`func (o *ListOrganizationsResponseBody) SetOrganizations(v []OrganizationSummaryView)`

SetOrganizations sets Organizations field to given value.


### SetOrganizationsNil

`func (o *ListOrganizationsResponseBody) SetOrganizationsNil(b bool)`

 SetOrganizationsNil sets the value for Organizations to be an explicit nil

### UnsetOrganizations
`func (o *ListOrganizationsResponseBody) UnsetOrganizations()`

UnsetOrganizations ensures that no value is present for Organizations, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


