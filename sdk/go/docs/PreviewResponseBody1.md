# PreviewResponseBody1

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Count** | **int64** |  | 
**Rules** | [**[]RuleView**](RuleView.md) |  | 

## Methods

### NewPreviewResponseBody1

`func NewPreviewResponseBody1(count int64, rules []RuleView, ) *PreviewResponseBody1`

NewPreviewResponseBody1 instantiates a new PreviewResponseBody1 object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPreviewResponseBody1WithDefaults

`func NewPreviewResponseBody1WithDefaults() *PreviewResponseBody1`

NewPreviewResponseBody1WithDefaults instantiates a new PreviewResponseBody1 object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *PreviewResponseBody1) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *PreviewResponseBody1) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *PreviewResponseBody1) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *PreviewResponseBody1) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetCount

`func (o *PreviewResponseBody1) GetCount() int64`

GetCount returns the Count field if non-nil, zero value otherwise.

### GetCountOk

`func (o *PreviewResponseBody1) GetCountOk() (*int64, bool)`

GetCountOk returns a tuple with the Count field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCount

`func (o *PreviewResponseBody1) SetCount(v int64)`

SetCount sets Count field to given value.


### GetRules

`func (o *PreviewResponseBody1) GetRules() []RuleView`

GetRules returns the Rules field if non-nil, zero value otherwise.

### GetRulesOk

`func (o *PreviewResponseBody1) GetRulesOk() (*[]RuleView, bool)`

GetRulesOk returns a tuple with the Rules field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRules

`func (o *PreviewResponseBody1) SetRules(v []RuleView)`

SetRules sets Rules field to given value.


### SetRulesNil

`func (o *PreviewResponseBody1) SetRulesNil(b bool)`

 SetRulesNil sets the value for Rules to be an explicit nil

### UnsetRules
`func (o *PreviewResponseBody1) UnsetRules()`

UnsetRules ensures that no value is present for Rules, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


