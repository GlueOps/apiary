# SoleOwnershipsResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**BlocksAccountDeletion** | **bool** |  | 
**Organizations** | [**[]Item**](Item.md) |  | 

## Methods

### NewSoleOwnershipsResponseBody

`func NewSoleOwnershipsResponseBody(blocksAccountDeletion bool, organizations []Item, ) *SoleOwnershipsResponseBody`

NewSoleOwnershipsResponseBody instantiates a new SoleOwnershipsResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSoleOwnershipsResponseBodyWithDefaults

`func NewSoleOwnershipsResponseBodyWithDefaults() *SoleOwnershipsResponseBody`

NewSoleOwnershipsResponseBodyWithDefaults instantiates a new SoleOwnershipsResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *SoleOwnershipsResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *SoleOwnershipsResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *SoleOwnershipsResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *SoleOwnershipsResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetBlocksAccountDeletion

`func (o *SoleOwnershipsResponseBody) GetBlocksAccountDeletion() bool`

GetBlocksAccountDeletion returns the BlocksAccountDeletion field if non-nil, zero value otherwise.

### GetBlocksAccountDeletionOk

`func (o *SoleOwnershipsResponseBody) GetBlocksAccountDeletionOk() (*bool, bool)`

GetBlocksAccountDeletionOk returns a tuple with the BlocksAccountDeletion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBlocksAccountDeletion

`func (o *SoleOwnershipsResponseBody) SetBlocksAccountDeletion(v bool)`

SetBlocksAccountDeletion sets BlocksAccountDeletion field to given value.


### GetOrganizations

`func (o *SoleOwnershipsResponseBody) GetOrganizations() []Item`

GetOrganizations returns the Organizations field if non-nil, zero value otherwise.

### GetOrganizationsOk

`func (o *SoleOwnershipsResponseBody) GetOrganizationsOk() (*[]Item, bool)`

GetOrganizationsOk returns a tuple with the Organizations field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrganizations

`func (o *SoleOwnershipsResponseBody) SetOrganizations(v []Item)`

SetOrganizations sets Organizations field to given value.


### SetOrganizationsNil

`func (o *SoleOwnershipsResponseBody) SetOrganizationsNil(b bool)`

 SetOrganizationsNil sets the value for Organizations to be an explicit nil

### UnsetOrganizations
`func (o *SoleOwnershipsResponseBody) UnsetOrganizations()`

UnsetOrganizations ensures that no value is present for Organizations, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


