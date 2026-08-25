# ListRulesResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Rules** | [**[]RuleView**](RuleView.md) |  | 

## Methods

### NewListRulesResponseBody

`func NewListRulesResponseBody(rules []RuleView, ) *ListRulesResponseBody`

NewListRulesResponseBody instantiates a new ListRulesResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListRulesResponseBodyWithDefaults

`func NewListRulesResponseBodyWithDefaults() *ListRulesResponseBody`

NewListRulesResponseBodyWithDefaults instantiates a new ListRulesResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListRulesResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListRulesResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListRulesResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListRulesResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetRules

`func (o *ListRulesResponseBody) GetRules() []RuleView`

GetRules returns the Rules field if non-nil, zero value otherwise.

### GetRulesOk

`func (o *ListRulesResponseBody) GetRulesOk() (*[]RuleView, bool)`

GetRulesOk returns a tuple with the Rules field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRules

`func (o *ListRulesResponseBody) SetRules(v []RuleView)`

SetRules sets Rules field to given value.


### SetRulesNil

`func (o *ListRulesResponseBody) SetRulesNil(b bool)`

 SetRulesNil sets the value for Rules to be an explicit nil

### UnsetRules
`func (o *ListRulesResponseBody) UnsetRules()`

UnsetRules ensures that no value is present for Rules, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


