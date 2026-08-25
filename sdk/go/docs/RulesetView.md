# RulesetView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Description** | **string** |  | 
**Id** | **string** |  | 
**Name** | **string** |  | 

## Methods

### NewRulesetView

`func NewRulesetView(description string, id string, name string, ) *RulesetView`

NewRulesetView instantiates a new RulesetView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRulesetViewWithDefaults

`func NewRulesetViewWithDefaults() *RulesetView`

NewRulesetViewWithDefaults instantiates a new RulesetView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *RulesetView) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *RulesetView) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *RulesetView) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *RulesetView) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetDescription

`func (o *RulesetView) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *RulesetView) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *RulesetView) SetDescription(v string)`

SetDescription sets Description field to given value.


### GetId

`func (o *RulesetView) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *RulesetView) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *RulesetView) SetId(v string)`

SetId sets Id field to given value.


### GetName

`func (o *RulesetView) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *RulesetView) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *RulesetView) SetName(v string)`

SetName sets Name field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


