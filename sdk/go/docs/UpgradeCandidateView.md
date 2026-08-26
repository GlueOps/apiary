# UpgradeCandidateView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AgentVersion** | Pointer to **string** | What this agent last reported running. | [optional] 
**BeeId** | **string** |  | 
**Queued** | **bool** |  | 
**Skipped** | Pointer to **string** | Why this agent was left alone. Per agent rather than a count, because \&quot;14 of 20 queued\&quot; leaves the operator to work out which six. | [optional] 

## Methods

### NewUpgradeCandidateView

`func NewUpgradeCandidateView(beeId string, queued bool, ) *UpgradeCandidateView`

NewUpgradeCandidateView instantiates a new UpgradeCandidateView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewUpgradeCandidateViewWithDefaults

`func NewUpgradeCandidateViewWithDefaults() *UpgradeCandidateView`

NewUpgradeCandidateViewWithDefaults instantiates a new UpgradeCandidateView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAgentVersion

`func (o *UpgradeCandidateView) GetAgentVersion() string`

GetAgentVersion returns the AgentVersion field if non-nil, zero value otherwise.

### GetAgentVersionOk

`func (o *UpgradeCandidateView) GetAgentVersionOk() (*string, bool)`

GetAgentVersionOk returns a tuple with the AgentVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgentVersion

`func (o *UpgradeCandidateView) SetAgentVersion(v string)`

SetAgentVersion sets AgentVersion field to given value.

### HasAgentVersion

`func (o *UpgradeCandidateView) HasAgentVersion() bool`

HasAgentVersion returns a boolean if a field has been set.

### GetBeeId

`func (o *UpgradeCandidateView) GetBeeId() string`

GetBeeId returns the BeeId field if non-nil, zero value otherwise.

### GetBeeIdOk

`func (o *UpgradeCandidateView) GetBeeIdOk() (*string, bool)`

GetBeeIdOk returns a tuple with the BeeId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBeeId

`func (o *UpgradeCandidateView) SetBeeId(v string)`

SetBeeId sets BeeId field to given value.


### GetQueued

`func (o *UpgradeCandidateView) GetQueued() bool`

GetQueued returns the Queued field if non-nil, zero value otherwise.

### GetQueuedOk

`func (o *UpgradeCandidateView) GetQueuedOk() (*bool, bool)`

GetQueuedOk returns a tuple with the Queued field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetQueued

`func (o *UpgradeCandidateView) SetQueued(v bool)`

SetQueued sets Queued field to given value.


### GetSkipped

`func (o *UpgradeCandidateView) GetSkipped() string`

GetSkipped returns the Skipped field if non-nil, zero value otherwise.

### GetSkippedOk

`func (o *UpgradeCandidateView) GetSkippedOk() (*string, bool)`

GetSkippedOk returns a tuple with the Skipped field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSkipped

`func (o *UpgradeCandidateView) SetSkipped(v string)`

SetSkipped sets Skipped field to given value.

### HasSkipped

`func (o *UpgradeCandidateView) HasSkipped() bool`

HasSkipped returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


