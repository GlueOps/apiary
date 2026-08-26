# HeartbeatRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**AgentDigest** | Pointer to **string** |  | [optional] 
**AgentVersion** | Pointer to **string** |  | [optional] 
**AppliedFirewallDigest** | Pointer to **string** | Digest of the body actually loaded, so a host that applied the right revision from a stale body is still detected as diverged. | [optional] 
**AppliedFirewallRevision** | Pointer to **int64** | The firewall revision this host has actually loaded. Omit or zero if it has applied none. | [optional] 
**Back** | Pointer to **bool** | Withdraws a standing declaration: the host has returned. | [optional] 
**UnavailableForSeconds** | Pointer to **int64** | Declared BEFORE acting: how long this host expects to be unreachable, e.g. across a reboot. Until it elapses, silence from this bee is expected rather than a fault. | [optional] 
**UnavailableReason** | Pointer to **string** |  | [optional] 
**WaitSeconds** | Pointer to **int64** | Hold the request open for up to this long waiting for a task. The response is the same shape either way. | [optional] 

## Methods

### NewHeartbeatRequestBody

`func NewHeartbeatRequestBody() *HeartbeatRequestBody`

NewHeartbeatRequestBody instantiates a new HeartbeatRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewHeartbeatRequestBodyWithDefaults

`func NewHeartbeatRequestBodyWithDefaults() *HeartbeatRequestBody`

NewHeartbeatRequestBodyWithDefaults instantiates a new HeartbeatRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *HeartbeatRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *HeartbeatRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *HeartbeatRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *HeartbeatRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetAgentDigest

`func (o *HeartbeatRequestBody) GetAgentDigest() string`

GetAgentDigest returns the AgentDigest field if non-nil, zero value otherwise.

### GetAgentDigestOk

`func (o *HeartbeatRequestBody) GetAgentDigestOk() (*string, bool)`

GetAgentDigestOk returns a tuple with the AgentDigest field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgentDigest

`func (o *HeartbeatRequestBody) SetAgentDigest(v string)`

SetAgentDigest sets AgentDigest field to given value.

### HasAgentDigest

`func (o *HeartbeatRequestBody) HasAgentDigest() bool`

HasAgentDigest returns a boolean if a field has been set.

### GetAgentVersion

`func (o *HeartbeatRequestBody) GetAgentVersion() string`

GetAgentVersion returns the AgentVersion field if non-nil, zero value otherwise.

### GetAgentVersionOk

`func (o *HeartbeatRequestBody) GetAgentVersionOk() (*string, bool)`

GetAgentVersionOk returns a tuple with the AgentVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgentVersion

`func (o *HeartbeatRequestBody) SetAgentVersion(v string)`

SetAgentVersion sets AgentVersion field to given value.

### HasAgentVersion

`func (o *HeartbeatRequestBody) HasAgentVersion() bool`

HasAgentVersion returns a boolean if a field has been set.

### GetAppliedFirewallDigest

`func (o *HeartbeatRequestBody) GetAppliedFirewallDigest() string`

GetAppliedFirewallDigest returns the AppliedFirewallDigest field if non-nil, zero value otherwise.

### GetAppliedFirewallDigestOk

`func (o *HeartbeatRequestBody) GetAppliedFirewallDigestOk() (*string, bool)`

GetAppliedFirewallDigestOk returns a tuple with the AppliedFirewallDigest field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAppliedFirewallDigest

`func (o *HeartbeatRequestBody) SetAppliedFirewallDigest(v string)`

SetAppliedFirewallDigest sets AppliedFirewallDigest field to given value.

### HasAppliedFirewallDigest

`func (o *HeartbeatRequestBody) HasAppliedFirewallDigest() bool`

HasAppliedFirewallDigest returns a boolean if a field has been set.

### GetAppliedFirewallRevision

`func (o *HeartbeatRequestBody) GetAppliedFirewallRevision() int64`

GetAppliedFirewallRevision returns the AppliedFirewallRevision field if non-nil, zero value otherwise.

### GetAppliedFirewallRevisionOk

`func (o *HeartbeatRequestBody) GetAppliedFirewallRevisionOk() (*int64, bool)`

GetAppliedFirewallRevisionOk returns a tuple with the AppliedFirewallRevision field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAppliedFirewallRevision

`func (o *HeartbeatRequestBody) SetAppliedFirewallRevision(v int64)`

SetAppliedFirewallRevision sets AppliedFirewallRevision field to given value.

### HasAppliedFirewallRevision

`func (o *HeartbeatRequestBody) HasAppliedFirewallRevision() bool`

HasAppliedFirewallRevision returns a boolean if a field has been set.

### GetBack

`func (o *HeartbeatRequestBody) GetBack() bool`

GetBack returns the Back field if non-nil, zero value otherwise.

### GetBackOk

`func (o *HeartbeatRequestBody) GetBackOk() (*bool, bool)`

GetBackOk returns a tuple with the Back field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBack

`func (o *HeartbeatRequestBody) SetBack(v bool)`

SetBack sets Back field to given value.

### HasBack

`func (o *HeartbeatRequestBody) HasBack() bool`

HasBack returns a boolean if a field has been set.

### GetUnavailableForSeconds

`func (o *HeartbeatRequestBody) GetUnavailableForSeconds() int64`

GetUnavailableForSeconds returns the UnavailableForSeconds field if non-nil, zero value otherwise.

### GetUnavailableForSecondsOk

`func (o *HeartbeatRequestBody) GetUnavailableForSecondsOk() (*int64, bool)`

GetUnavailableForSecondsOk returns a tuple with the UnavailableForSeconds field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUnavailableForSeconds

`func (o *HeartbeatRequestBody) SetUnavailableForSeconds(v int64)`

SetUnavailableForSeconds sets UnavailableForSeconds field to given value.

### HasUnavailableForSeconds

`func (o *HeartbeatRequestBody) HasUnavailableForSeconds() bool`

HasUnavailableForSeconds returns a boolean if a field has been set.

### GetUnavailableReason

`func (o *HeartbeatRequestBody) GetUnavailableReason() string`

GetUnavailableReason returns the UnavailableReason field if non-nil, zero value otherwise.

### GetUnavailableReasonOk

`func (o *HeartbeatRequestBody) GetUnavailableReasonOk() (*string, bool)`

GetUnavailableReasonOk returns a tuple with the UnavailableReason field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUnavailableReason

`func (o *HeartbeatRequestBody) SetUnavailableReason(v string)`

SetUnavailableReason sets UnavailableReason field to given value.

### HasUnavailableReason

`func (o *HeartbeatRequestBody) HasUnavailableReason() bool`

HasUnavailableReason returns a boolean if a field has been set.

### GetWaitSeconds

`func (o *HeartbeatRequestBody) GetWaitSeconds() int64`

GetWaitSeconds returns the WaitSeconds field if non-nil, zero value otherwise.

### GetWaitSecondsOk

`func (o *HeartbeatRequestBody) GetWaitSecondsOk() (*int64, bool)`

GetWaitSecondsOk returns a tuple with the WaitSeconds field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWaitSeconds

`func (o *HeartbeatRequestBody) SetWaitSeconds(v int64)`

SetWaitSeconds sets WaitSeconds field to given value.

### HasWaitSeconds

`func (o *HeartbeatRequestBody) HasWaitSeconds() bool`

HasWaitSeconds returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


