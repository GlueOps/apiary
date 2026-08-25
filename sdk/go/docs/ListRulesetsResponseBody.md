# ListRulesetsResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Rulesets** | [**[]RulesetView**](RulesetView.md) |  | 

## Methods

### NewListRulesetsResponseBody

`func NewListRulesetsResponseBody(rulesets []RulesetView, ) *ListRulesetsResponseBody`

NewListRulesetsResponseBody instantiates a new ListRulesetsResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListRulesetsResponseBodyWithDefaults

`func NewListRulesetsResponseBodyWithDefaults() *ListRulesetsResponseBody`

NewListRulesetsResponseBodyWithDefaults instantiates a new ListRulesetsResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListRulesetsResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListRulesetsResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListRulesetsResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListRulesetsResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetRulesets

`func (o *ListRulesetsResponseBody) GetRulesets() []RulesetView`

GetRulesets returns the Rulesets field if non-nil, zero value otherwise.

### GetRulesetsOk

`func (o *ListRulesetsResponseBody) GetRulesetsOk() (*[]RulesetView, bool)`

GetRulesetsOk returns a tuple with the Rulesets field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRulesets

`func (o *ListRulesetsResponseBody) SetRulesets(v []RulesetView)`

SetRulesets sets Rulesets field to given value.


### SetRulesetsNil

`func (o *ListRulesetsResponseBody) SetRulesetsNil(b bool)`

 SetRulesetsNil sets the value for Rulesets to be an explicit nil

### UnsetRulesets
`func (o *ListRulesetsResponseBody) UnsetRulesets()`

UnsetRulesets ensures that no value is present for Rulesets, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


