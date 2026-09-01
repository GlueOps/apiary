# FleetActivityView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Address** | Pointer to **string** |  | [optional] 
**Attempt** | **int64** |  | 
**BeeId** | **string** |  | 
**CreatedAt** | **time.Time** |  | 
**Error** | Pointer to **string** | The agent&#39;s own account of what stopped it. Empty unless it failed. | [optional] 
**FinishedAt** | Pointer to **time.Time** |  | [optional] 
**HiveName** | Pointer to **string** | The cluster the task ACTED AGAINST, which during a migration is not the one the machine is in. | [optional] 
**Id** | **string** |  | 
**Kind** | **string** |  | 
**NodeName** | Pointer to **string** |  | [optional] 
**Operation** | Pointer to **string** | The kind as an operator reads it. Empty for a kind with no label yet. | [optional] 
**Phase** | Pointer to **string** |  | [optional] 
**StartedAt** | Pointer to **time.Time** |  | [optional] 
**Status** | **string** |  | 

## Methods

### NewFleetActivityView

`func NewFleetActivityView(attempt int64, beeId string, createdAt time.Time, id string, kind string, status string, ) *FleetActivityView`

NewFleetActivityView instantiates a new FleetActivityView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewFleetActivityViewWithDefaults

`func NewFleetActivityViewWithDefaults() *FleetActivityView`

NewFleetActivityViewWithDefaults instantiates a new FleetActivityView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAddress

`func (o *FleetActivityView) GetAddress() string`

GetAddress returns the Address field if non-nil, zero value otherwise.

### GetAddressOk

`func (o *FleetActivityView) GetAddressOk() (*string, bool)`

GetAddressOk returns a tuple with the Address field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAddress

`func (o *FleetActivityView) SetAddress(v string)`

SetAddress sets Address field to given value.

### HasAddress

`func (o *FleetActivityView) HasAddress() bool`

HasAddress returns a boolean if a field has been set.

### GetAttempt

`func (o *FleetActivityView) GetAttempt() int64`

GetAttempt returns the Attempt field if non-nil, zero value otherwise.

### GetAttemptOk

`func (o *FleetActivityView) GetAttemptOk() (*int64, bool)`

GetAttemptOk returns a tuple with the Attempt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAttempt

`func (o *FleetActivityView) SetAttempt(v int64)`

SetAttempt sets Attempt field to given value.


### GetBeeId

`func (o *FleetActivityView) GetBeeId() string`

GetBeeId returns the BeeId field if non-nil, zero value otherwise.

### GetBeeIdOk

`func (o *FleetActivityView) GetBeeIdOk() (*string, bool)`

GetBeeIdOk returns a tuple with the BeeId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBeeId

`func (o *FleetActivityView) SetBeeId(v string)`

SetBeeId sets BeeId field to given value.


### GetCreatedAt

`func (o *FleetActivityView) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *FleetActivityView) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *FleetActivityView) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.


### GetError

`func (o *FleetActivityView) GetError() string`

GetError returns the Error field if non-nil, zero value otherwise.

### GetErrorOk

`func (o *FleetActivityView) GetErrorOk() (*string, bool)`

GetErrorOk returns a tuple with the Error field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetError

`func (o *FleetActivityView) SetError(v string)`

SetError sets Error field to given value.

### HasError

`func (o *FleetActivityView) HasError() bool`

HasError returns a boolean if a field has been set.

### GetFinishedAt

`func (o *FleetActivityView) GetFinishedAt() time.Time`

GetFinishedAt returns the FinishedAt field if non-nil, zero value otherwise.

### GetFinishedAtOk

`func (o *FleetActivityView) GetFinishedAtOk() (*time.Time, bool)`

GetFinishedAtOk returns a tuple with the FinishedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFinishedAt

`func (o *FleetActivityView) SetFinishedAt(v time.Time)`

SetFinishedAt sets FinishedAt field to given value.

### HasFinishedAt

`func (o *FleetActivityView) HasFinishedAt() bool`

HasFinishedAt returns a boolean if a field has been set.

### GetHiveName

`func (o *FleetActivityView) GetHiveName() string`

GetHiveName returns the HiveName field if non-nil, zero value otherwise.

### GetHiveNameOk

`func (o *FleetActivityView) GetHiveNameOk() (*string, bool)`

GetHiveNameOk returns a tuple with the HiveName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHiveName

`func (o *FleetActivityView) SetHiveName(v string)`

SetHiveName sets HiveName field to given value.

### HasHiveName

`func (o *FleetActivityView) HasHiveName() bool`

HasHiveName returns a boolean if a field has been set.

### GetId

`func (o *FleetActivityView) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *FleetActivityView) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *FleetActivityView) SetId(v string)`

SetId sets Id field to given value.


### GetKind

`func (o *FleetActivityView) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *FleetActivityView) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *FleetActivityView) SetKind(v string)`

SetKind sets Kind field to given value.


### GetNodeName

`func (o *FleetActivityView) GetNodeName() string`

GetNodeName returns the NodeName field if non-nil, zero value otherwise.

### GetNodeNameOk

`func (o *FleetActivityView) GetNodeNameOk() (*string, bool)`

GetNodeNameOk returns a tuple with the NodeName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNodeName

`func (o *FleetActivityView) SetNodeName(v string)`

SetNodeName sets NodeName field to given value.

### HasNodeName

`func (o *FleetActivityView) HasNodeName() bool`

HasNodeName returns a boolean if a field has been set.

### GetOperation

`func (o *FleetActivityView) GetOperation() string`

GetOperation returns the Operation field if non-nil, zero value otherwise.

### GetOperationOk

`func (o *FleetActivityView) GetOperationOk() (*string, bool)`

GetOperationOk returns a tuple with the Operation field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOperation

`func (o *FleetActivityView) SetOperation(v string)`

SetOperation sets Operation field to given value.

### HasOperation

`func (o *FleetActivityView) HasOperation() bool`

HasOperation returns a boolean if a field has been set.

### GetPhase

`func (o *FleetActivityView) GetPhase() string`

GetPhase returns the Phase field if non-nil, zero value otherwise.

### GetPhaseOk

`func (o *FleetActivityView) GetPhaseOk() (*string, bool)`

GetPhaseOk returns a tuple with the Phase field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPhase

`func (o *FleetActivityView) SetPhase(v string)`

SetPhase sets Phase field to given value.

### HasPhase

`func (o *FleetActivityView) HasPhase() bool`

HasPhase returns a boolean if a field has been set.

### GetStartedAt

`func (o *FleetActivityView) GetStartedAt() time.Time`

GetStartedAt returns the StartedAt field if non-nil, zero value otherwise.

### GetStartedAtOk

`func (o *FleetActivityView) GetStartedAtOk() (*time.Time, bool)`

GetStartedAtOk returns a tuple with the StartedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStartedAt

`func (o *FleetActivityView) SetStartedAt(v time.Time)`

SetStartedAt sets StartedAt field to given value.

### HasStartedAt

`func (o *FleetActivityView) HasStartedAt() bool`

HasStartedAt returns a boolean if a field has been set.

### GetStatus

`func (o *FleetActivityView) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *FleetActivityView) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *FleetActivityView) SetStatus(v string)`

SetStatus sets Status field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


