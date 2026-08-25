# MintAPIKeyRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Name** | **string** | What this key is for. It is the only thing distinguishing two keys in a list, so a name like &#39;test&#39; costs someone an outage later. | 
**Scope** | **string** | &#39;user&#39; for a key that acts as you, in whichever organization the request names; &#39;org&#39; for a key that acts as the organization itself. Agent keys are issued only by fleet enrolment and are refused here. | 
**TtlHours** | Pointer to **int64** | Lifetime in hours. Omit for a key that does not expire. | [optional] 

## Methods

### NewMintAPIKeyRequestBody

`func NewMintAPIKeyRequestBody(name string, scope string, ) *MintAPIKeyRequestBody`

NewMintAPIKeyRequestBody instantiates a new MintAPIKeyRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMintAPIKeyRequestBodyWithDefaults

`func NewMintAPIKeyRequestBodyWithDefaults() *MintAPIKeyRequestBody`

NewMintAPIKeyRequestBodyWithDefaults instantiates a new MintAPIKeyRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *MintAPIKeyRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *MintAPIKeyRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *MintAPIKeyRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *MintAPIKeyRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetName

`func (o *MintAPIKeyRequestBody) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *MintAPIKeyRequestBody) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *MintAPIKeyRequestBody) SetName(v string)`

SetName sets Name field to given value.


### GetScope

`func (o *MintAPIKeyRequestBody) GetScope() string`

GetScope returns the Scope field if non-nil, zero value otherwise.

### GetScopeOk

`func (o *MintAPIKeyRequestBody) GetScopeOk() (*string, bool)`

GetScopeOk returns a tuple with the Scope field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetScope

`func (o *MintAPIKeyRequestBody) SetScope(v string)`

SetScope sets Scope field to given value.


### GetTtlHours

`func (o *MintAPIKeyRequestBody) GetTtlHours() int64`

GetTtlHours returns the TtlHours field if non-nil, zero value otherwise.

### GetTtlHoursOk

`func (o *MintAPIKeyRequestBody) GetTtlHoursOk() (*int64, bool)`

GetTtlHoursOk returns a tuple with the TtlHours field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTtlHours

`func (o *MintAPIKeyRequestBody) SetTtlHours(v int64)`

SetTtlHours sets TtlHours field to given value.

### HasTtlHours

`func (o *MintAPIKeyRequestBody) HasTtlHours() bool`

HasTtlHours returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


