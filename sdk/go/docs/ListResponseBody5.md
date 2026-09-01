# ListResponseBody5

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Rules** | [**[]AccessRuleView**](AccessRuleView.md) |  | 
**SeedingEnabled** | **bool** | Whether auth.seed_public_domain_rules re-applies the public-domain corpus at every boot. | 

## Methods

### NewListResponseBody5

`func NewListResponseBody5(rules []AccessRuleView, seedingEnabled bool, ) *ListResponseBody5`

NewListResponseBody5 instantiates a new ListResponseBody5 object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListResponseBody5WithDefaults

`func NewListResponseBody5WithDefaults() *ListResponseBody5`

NewListResponseBody5WithDefaults instantiates a new ListResponseBody5 object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListResponseBody5) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListResponseBody5) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListResponseBody5) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListResponseBody5) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetRules

`func (o *ListResponseBody5) GetRules() []AccessRuleView`

GetRules returns the Rules field if non-nil, zero value otherwise.

### GetRulesOk

`func (o *ListResponseBody5) GetRulesOk() (*[]AccessRuleView, bool)`

GetRulesOk returns a tuple with the Rules field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRules

`func (o *ListResponseBody5) SetRules(v []AccessRuleView)`

SetRules sets Rules field to given value.


### SetRulesNil

`func (o *ListResponseBody5) SetRulesNil(b bool)`

 SetRulesNil sets the value for Rules to be an explicit nil

### UnsetRules
`func (o *ListResponseBody5) UnsetRules()`

UnsetRules ensures that no value is present for Rules, not even an explicit nil
### GetSeedingEnabled

`func (o *ListResponseBody5) GetSeedingEnabled() bool`

GetSeedingEnabled returns the SeedingEnabled field if non-nil, zero value otherwise.

### GetSeedingEnabledOk

`func (o *ListResponseBody5) GetSeedingEnabledOk() (*bool, bool)`

GetSeedingEnabledOk returns a tuple with the SeedingEnabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSeedingEnabled

`func (o *ListResponseBody5) SetSeedingEnabled(v bool)`

SetSeedingEnabled sets SeedingEnabled field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


