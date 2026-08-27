# AuditEventView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Action** | **string** |  | 
**ActorId** | Pointer to **string** | The platform administrator behind an impersonated session, when there was one. | [optional] 
**CreatedAt** | **time.Time** |  | 
**Detail** | Pointer to **string** | Why it was refused. Empty when it was not. | [optional] 
**Id** | **string** |  | 
**Metadata** | Pointer to **map[string]interface{}** |  | [optional] 
**Outcome** | **string** | ok, refused or failed. | 
**PrincipalId** | **string** |  | 
**PrincipalKind** | **string** | user, org_key, agent or system. | 
**TargetId** | Pointer to **string** |  | [optional] 
**TargetKind** | **string** |  | 

## Methods

### NewAuditEventView

`func NewAuditEventView(action string, createdAt time.Time, id string, outcome string, principalId string, principalKind string, targetKind string, ) *AuditEventView`

NewAuditEventView instantiates a new AuditEventView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAuditEventViewWithDefaults

`func NewAuditEventViewWithDefaults() *AuditEventView`

NewAuditEventViewWithDefaults instantiates a new AuditEventView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAction

`func (o *AuditEventView) GetAction() string`

GetAction returns the Action field if non-nil, zero value otherwise.

### GetActionOk

`func (o *AuditEventView) GetActionOk() (*string, bool)`

GetActionOk returns a tuple with the Action field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAction

`func (o *AuditEventView) SetAction(v string)`

SetAction sets Action field to given value.


### GetActorId

`func (o *AuditEventView) GetActorId() string`

GetActorId returns the ActorId field if non-nil, zero value otherwise.

### GetActorIdOk

`func (o *AuditEventView) GetActorIdOk() (*string, bool)`

GetActorIdOk returns a tuple with the ActorId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetActorId

`func (o *AuditEventView) SetActorId(v string)`

SetActorId sets ActorId field to given value.

### HasActorId

`func (o *AuditEventView) HasActorId() bool`

HasActorId returns a boolean if a field has been set.

### GetCreatedAt

`func (o *AuditEventView) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *AuditEventView) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *AuditEventView) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.


### GetDetail

`func (o *AuditEventView) GetDetail() string`

GetDetail returns the Detail field if non-nil, zero value otherwise.

### GetDetailOk

`func (o *AuditEventView) GetDetailOk() (*string, bool)`

GetDetailOk returns a tuple with the Detail field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDetail

`func (o *AuditEventView) SetDetail(v string)`

SetDetail sets Detail field to given value.

### HasDetail

`func (o *AuditEventView) HasDetail() bool`

HasDetail returns a boolean if a field has been set.

### GetId

`func (o *AuditEventView) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *AuditEventView) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *AuditEventView) SetId(v string)`

SetId sets Id field to given value.


### GetMetadata

`func (o *AuditEventView) GetMetadata() map[string]*interface{}`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *AuditEventView) GetMetadataOk() (*map[string]*interface{}, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *AuditEventView) SetMetadata(v map[string]*interface{})`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *AuditEventView) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.

### GetOutcome

`func (o *AuditEventView) GetOutcome() string`

GetOutcome returns the Outcome field if non-nil, zero value otherwise.

### GetOutcomeOk

`func (o *AuditEventView) GetOutcomeOk() (*string, bool)`

GetOutcomeOk returns a tuple with the Outcome field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOutcome

`func (o *AuditEventView) SetOutcome(v string)`

SetOutcome sets Outcome field to given value.


### GetPrincipalId

`func (o *AuditEventView) GetPrincipalId() string`

GetPrincipalId returns the PrincipalId field if non-nil, zero value otherwise.

### GetPrincipalIdOk

`func (o *AuditEventView) GetPrincipalIdOk() (*string, bool)`

GetPrincipalIdOk returns a tuple with the PrincipalId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPrincipalId

`func (o *AuditEventView) SetPrincipalId(v string)`

SetPrincipalId sets PrincipalId field to given value.


### GetPrincipalKind

`func (o *AuditEventView) GetPrincipalKind() string`

GetPrincipalKind returns the PrincipalKind field if non-nil, zero value otherwise.

### GetPrincipalKindOk

`func (o *AuditEventView) GetPrincipalKindOk() (*string, bool)`

GetPrincipalKindOk returns a tuple with the PrincipalKind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPrincipalKind

`func (o *AuditEventView) SetPrincipalKind(v string)`

SetPrincipalKind sets PrincipalKind field to given value.


### GetTargetId

`func (o *AuditEventView) GetTargetId() string`

GetTargetId returns the TargetId field if non-nil, zero value otherwise.

### GetTargetIdOk

`func (o *AuditEventView) GetTargetIdOk() (*string, bool)`

GetTargetIdOk returns a tuple with the TargetId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTargetId

`func (o *AuditEventView) SetTargetId(v string)`

SetTargetId sets TargetId field to given value.

### HasTargetId

`func (o *AuditEventView) HasTargetId() bool`

HasTargetId returns a boolean if a field has been set.

### GetTargetKind

`func (o *AuditEventView) GetTargetKind() string`

GetTargetKind returns the TargetKind field if non-nil, zero value otherwise.

### GetTargetKindOk

`func (o *AuditEventView) GetTargetKindOk() (*string, bool)`

GetTargetKindOk returns a tuple with the TargetKind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTargetKind

`func (o *AuditEventView) SetTargetKind(v string)`

SetTargetKind sets TargetKind field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


