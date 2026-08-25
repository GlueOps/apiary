# AccessRuleView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**CreatedAt** | **time.Time** |  | 
**Id** | **string** |  | 
**Kind** | **string** | An email rule matches one address; a domain rule matches every address at that domain. Email beats domain. | 
**Policy** | **string** |  | 
**Purpose** | **string** | Which question this rule answers: authenticate gates signing in, create_org gates founding an organization, join_org gates accepting an invitation. A rule of one purpose can never decide another. | 
**Reason** | Pointer to **string** |  | [optional] 
**SeedOverridden** | **bool** | A seeded public-domain deny that an operator has turned into an allow. Permanent — the seed pass never updates an existing row. | 
**SeedRestoresOnRestart** | **bool** | Deleting this rule does not last: the next boot re-inserts it as a create_org deny. | 
**Seeded** | **bool** | Written by the boot-time public-domain seed, not by an operator. | 
**UpdatedAt** | **time.Time** |  | 
**Value** | **string** |  | 

## Methods

### NewAccessRuleView

`func NewAccessRuleView(createdAt time.Time, id string, kind string, policy string, purpose string, seedOverridden bool, seedRestoresOnRestart bool, seeded bool, updatedAt time.Time, value string, ) *AccessRuleView`

NewAccessRuleView instantiates a new AccessRuleView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAccessRuleViewWithDefaults

`func NewAccessRuleViewWithDefaults() *AccessRuleView`

NewAccessRuleViewWithDefaults instantiates a new AccessRuleView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *AccessRuleView) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *AccessRuleView) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *AccessRuleView) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *AccessRuleView) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetCreatedAt

`func (o *AccessRuleView) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *AccessRuleView) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *AccessRuleView) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.


### GetId

`func (o *AccessRuleView) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *AccessRuleView) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *AccessRuleView) SetId(v string)`

SetId sets Id field to given value.


### GetKind

`func (o *AccessRuleView) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *AccessRuleView) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *AccessRuleView) SetKind(v string)`

SetKind sets Kind field to given value.


### GetPolicy

`func (o *AccessRuleView) GetPolicy() string`

GetPolicy returns the Policy field if non-nil, zero value otherwise.

### GetPolicyOk

`func (o *AccessRuleView) GetPolicyOk() (*string, bool)`

GetPolicyOk returns a tuple with the Policy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPolicy

`func (o *AccessRuleView) SetPolicy(v string)`

SetPolicy sets Policy field to given value.


### GetPurpose

`func (o *AccessRuleView) GetPurpose() string`

GetPurpose returns the Purpose field if non-nil, zero value otherwise.

### GetPurposeOk

`func (o *AccessRuleView) GetPurposeOk() (*string, bool)`

GetPurposeOk returns a tuple with the Purpose field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPurpose

`func (o *AccessRuleView) SetPurpose(v string)`

SetPurpose sets Purpose field to given value.


### GetReason

`func (o *AccessRuleView) GetReason() string`

GetReason returns the Reason field if non-nil, zero value otherwise.

### GetReasonOk

`func (o *AccessRuleView) GetReasonOk() (*string, bool)`

GetReasonOk returns a tuple with the Reason field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReason

`func (o *AccessRuleView) SetReason(v string)`

SetReason sets Reason field to given value.

### HasReason

`func (o *AccessRuleView) HasReason() bool`

HasReason returns a boolean if a field has been set.

### GetSeedOverridden

`func (o *AccessRuleView) GetSeedOverridden() bool`

GetSeedOverridden returns the SeedOverridden field if non-nil, zero value otherwise.

### GetSeedOverriddenOk

`func (o *AccessRuleView) GetSeedOverriddenOk() (*bool, bool)`

GetSeedOverriddenOk returns a tuple with the SeedOverridden field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSeedOverridden

`func (o *AccessRuleView) SetSeedOverridden(v bool)`

SetSeedOverridden sets SeedOverridden field to given value.


### GetSeedRestoresOnRestart

`func (o *AccessRuleView) GetSeedRestoresOnRestart() bool`

GetSeedRestoresOnRestart returns the SeedRestoresOnRestart field if non-nil, zero value otherwise.

### GetSeedRestoresOnRestartOk

`func (o *AccessRuleView) GetSeedRestoresOnRestartOk() (*bool, bool)`

GetSeedRestoresOnRestartOk returns a tuple with the SeedRestoresOnRestart field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSeedRestoresOnRestart

`func (o *AccessRuleView) SetSeedRestoresOnRestart(v bool)`

SetSeedRestoresOnRestart sets SeedRestoresOnRestart field to given value.


### GetSeeded

`func (o *AccessRuleView) GetSeeded() bool`

GetSeeded returns the Seeded field if non-nil, zero value otherwise.

### GetSeededOk

`func (o *AccessRuleView) GetSeededOk() (*bool, bool)`

GetSeededOk returns a tuple with the Seeded field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSeeded

`func (o *AccessRuleView) SetSeeded(v bool)`

SetSeeded sets Seeded field to given value.


### GetUpdatedAt

`func (o *AccessRuleView) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *AccessRuleView) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *AccessRuleView) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.


### GetValue

`func (o *AccessRuleView) GetValue() string`

GetValue returns the Value field if non-nil, zero value otherwise.

### GetValueOk

`func (o *AccessRuleView) GetValueOk() (*string, bool)`

GetValueOk returns a tuple with the Value field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetValue

`func (o *AccessRuleView) SetValue(v string)`

SetValue sets Value field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


