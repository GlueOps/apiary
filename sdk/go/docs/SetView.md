# SetView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Description** | Pointer to **string** |  | [optional] 
**Elements** | [**[]SetElementView**](SetElementView.md) |  | 
**Family** | **string** |  | 
**Id** | **string** |  | 
**Managed** | **bool** | Contents maintained by the control plane. Edits are refused rather than silently reverted by the next convergence. | 
**Name** | **string** |  | 
**Purpose** | **string** |  | 

## Methods

### NewSetView

`func NewSetView(elements []SetElementView, family string, id string, managed bool, name string, purpose string, ) *SetView`

NewSetView instantiates a new SetView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSetViewWithDefaults

`func NewSetViewWithDefaults() *SetView`

NewSetViewWithDefaults instantiates a new SetView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *SetView) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *SetView) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *SetView) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *SetView) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetDescription

`func (o *SetView) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *SetView) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *SetView) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *SetView) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetElements

`func (o *SetView) GetElements() []SetElementView`

GetElements returns the Elements field if non-nil, zero value otherwise.

### GetElementsOk

`func (o *SetView) GetElementsOk() (*[]SetElementView, bool)`

GetElementsOk returns a tuple with the Elements field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetElements

`func (o *SetView) SetElements(v []SetElementView)`

SetElements sets Elements field to given value.


### SetElementsNil

`func (o *SetView) SetElementsNil(b bool)`

 SetElementsNil sets the value for Elements to be an explicit nil

### UnsetElements
`func (o *SetView) UnsetElements()`

UnsetElements ensures that no value is present for Elements, not even an explicit nil
### GetFamily

`func (o *SetView) GetFamily() string`

GetFamily returns the Family field if non-nil, zero value otherwise.

### GetFamilyOk

`func (o *SetView) GetFamilyOk() (*string, bool)`

GetFamilyOk returns a tuple with the Family field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFamily

`func (o *SetView) SetFamily(v string)`

SetFamily sets Family field to given value.


### GetId

`func (o *SetView) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *SetView) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *SetView) SetId(v string)`

SetId sets Id field to given value.


### GetManaged

`func (o *SetView) GetManaged() bool`

GetManaged returns the Managed field if non-nil, zero value otherwise.

### GetManagedOk

`func (o *SetView) GetManagedOk() (*bool, bool)`

GetManagedOk returns a tuple with the Managed field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetManaged

`func (o *SetView) SetManaged(v bool)`

SetManaged sets Managed field to given value.


### GetName

`func (o *SetView) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *SetView) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *SetView) SetName(v string)`

SetName sets Name field to given value.


### GetPurpose

`func (o *SetView) GetPurpose() string`

GetPurpose returns the Purpose field if non-nil, zero value otherwise.

### GetPurposeOk

`func (o *SetView) GetPurposeOk() (*string, bool)`

GetPurposeOk returns a tuple with the Purpose field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPurpose

`func (o *SetView) SetPurpose(v string)`

SetPurpose sets Purpose field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


