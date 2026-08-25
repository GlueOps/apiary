# CreateRequestBody2

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Kind** | **string** |  | 
**Policy** | **string** |  | 
**Purpose** | **string** | Required, with no default. A rule stored under the wrong purpose is not a weaker rule, it is a rule that never fires on the question you meant to ask. | 
**Reason** | Pointer to **string** | Free text for whoever reads this row in a year. | [optional] 
**Value** | **string** | An address for kind&#x3D;email, a bare domain for kind&#x3D;domain. Normalised on the way in — a value that does not normalise is refused rather than stored as a rule that can never match. | 

## Methods

### NewCreateRequestBody2

`func NewCreateRequestBody2(kind string, policy string, purpose string, value string, ) *CreateRequestBody2`

NewCreateRequestBody2 instantiates a new CreateRequestBody2 object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCreateRequestBody2WithDefaults

`func NewCreateRequestBody2WithDefaults() *CreateRequestBody2`

NewCreateRequestBody2WithDefaults instantiates a new CreateRequestBody2 object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *CreateRequestBody2) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *CreateRequestBody2) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *CreateRequestBody2) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *CreateRequestBody2) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetKind

`func (o *CreateRequestBody2) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *CreateRequestBody2) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *CreateRequestBody2) SetKind(v string)`

SetKind sets Kind field to given value.


### GetPolicy

`func (o *CreateRequestBody2) GetPolicy() string`

GetPolicy returns the Policy field if non-nil, zero value otherwise.

### GetPolicyOk

`func (o *CreateRequestBody2) GetPolicyOk() (*string, bool)`

GetPolicyOk returns a tuple with the Policy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPolicy

`func (o *CreateRequestBody2) SetPolicy(v string)`

SetPolicy sets Policy field to given value.


### GetPurpose

`func (o *CreateRequestBody2) GetPurpose() string`

GetPurpose returns the Purpose field if non-nil, zero value otherwise.

### GetPurposeOk

`func (o *CreateRequestBody2) GetPurposeOk() (*string, bool)`

GetPurposeOk returns a tuple with the Purpose field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPurpose

`func (o *CreateRequestBody2) SetPurpose(v string)`

SetPurpose sets Purpose field to given value.


### GetReason

`func (o *CreateRequestBody2) GetReason() string`

GetReason returns the Reason field if non-nil, zero value otherwise.

### GetReasonOk

`func (o *CreateRequestBody2) GetReasonOk() (*string, bool)`

GetReasonOk returns a tuple with the Reason field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReason

`func (o *CreateRequestBody2) SetReason(v string)`

SetReason sets Reason field to given value.

### HasReason

`func (o *CreateRequestBody2) HasReason() bool`

HasReason returns a boolean if a field has been set.

### GetValue

`func (o *CreateRequestBody2) GetValue() string`

GetValue returns the Value field if non-nil, zero value otherwise.

### GetValueOk

`func (o *CreateRequestBody2) GetValueOk() (*string, bool)`

GetValueOk returns a tuple with the Value field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetValue

`func (o *CreateRequestBody2) SetValue(v string)`

SetValue sets Value field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


