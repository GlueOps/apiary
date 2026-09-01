# FlagView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Default** | **bool** | What applies when neither level has a row. | 
**Key** | **string** |  | 
**Scope** | **string** | platform: one value for the installation. tenant: a platform default that an organization row overrides. | 
**Source** | **string** | default, platform or organization — which level decided. | 
**Value** | **bool** |  | 
**Why** | Pointer to **string** |  | [optional] 

## Methods

### NewFlagView

`func NewFlagView(default_ bool, key string, scope string, source string, value bool, ) *FlagView`

NewFlagView instantiates a new FlagView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewFlagViewWithDefaults

`func NewFlagViewWithDefaults() *FlagView`

NewFlagViewWithDefaults instantiates a new FlagView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *FlagView) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *FlagView) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *FlagView) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *FlagView) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetDefault

`func (o *FlagView) GetDefault() bool`

GetDefault returns the Default field if non-nil, zero value otherwise.

### GetDefaultOk

`func (o *FlagView) GetDefaultOk() (*bool, bool)`

GetDefaultOk returns a tuple with the Default field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDefault

`func (o *FlagView) SetDefault(v bool)`

SetDefault sets Default field to given value.


### GetKey

`func (o *FlagView) GetKey() string`

GetKey returns the Key field if non-nil, zero value otherwise.

### GetKeyOk

`func (o *FlagView) GetKeyOk() (*string, bool)`

GetKeyOk returns a tuple with the Key field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKey

`func (o *FlagView) SetKey(v string)`

SetKey sets Key field to given value.


### GetScope

`func (o *FlagView) GetScope() string`

GetScope returns the Scope field if non-nil, zero value otherwise.

### GetScopeOk

`func (o *FlagView) GetScopeOk() (*string, bool)`

GetScopeOk returns a tuple with the Scope field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetScope

`func (o *FlagView) SetScope(v string)`

SetScope sets Scope field to given value.


### GetSource

`func (o *FlagView) GetSource() string`

GetSource returns the Source field if non-nil, zero value otherwise.

### GetSourceOk

`func (o *FlagView) GetSourceOk() (*string, bool)`

GetSourceOk returns a tuple with the Source field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSource

`func (o *FlagView) SetSource(v string)`

SetSource sets Source field to given value.


### GetValue

`func (o *FlagView) GetValue() bool`

GetValue returns the Value field if non-nil, zero value otherwise.

### GetValueOk

`func (o *FlagView) GetValueOk() (*bool, bool)`

GetValueOk returns a tuple with the Value field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetValue

`func (o *FlagView) SetValue(v bool)`

SetValue sets Value field to given value.


### GetWhy

`func (o *FlagView) GetWhy() string`

GetWhy returns the Why field if non-nil, zero value otherwise.

### GetWhyOk

`func (o *FlagView) GetWhyOk() (*string, bool)`

GetWhyOk returns a tuple with the Why field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWhy

`func (o *FlagView) SetWhy(v string)`

SetWhy sets Why field to given value.

### HasWhy

`func (o *FlagView) HasWhy() bool`

HasWhy returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


