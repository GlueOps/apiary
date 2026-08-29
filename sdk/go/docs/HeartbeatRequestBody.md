# HeartbeatRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**AgentDigest** | Pointer to **string** |  | [optional] 
**AgentVersion** | Pointer to **string** |  | [optional] 
**AppliedFirewallDigest** | Pointer to **string** | Digest of the body actually loaded, so a host that applied the right revision from a stale body is still detected as diverged. | [optional] 
**AppliedFirewallRevision** | Pointer to **int64** | The firewall revision this host has actually loaded. Omit or zero if it has applied none. | [optional] 
**AppliedMembershipDigest** | Pointer to **string** |  | [optional] 
**AppliedMembershipRevision** | Pointer to **int64** | The cluster composition this host has converged on. Omit or zero if it has converged on none. | [optional] 
**Back** | Pointer to **bool** | Withdraws a standing declaration: the host has returned. | [optional] 
**ObservedCluster** | Pointer to **string** | The corosync cluster this host is in. EMPTY STRING means standalone, which is an observation; omitting the field means this agent did not look. | [optional] 
**ObservedFingerprint** | Pointer to **string** |  | [optional] 
**ObservedGuests** | Pointer to **int64** | VMs and containers registered to this host, stopped ones included. Counted from /etc/pve rather than the Proxmox API, so it survives a broken pveproxy. | [optional] 
**ObservedMembers** | Pointer to **[]string** | Node names corosync reports, this host included. | [optional] 
**ObservedQuorate** | Pointer to **bool** |  | [optional] 
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

### GetAppliedMembershipDigest

`func (o *HeartbeatRequestBody) GetAppliedMembershipDigest() string`

GetAppliedMembershipDigest returns the AppliedMembershipDigest field if non-nil, zero value otherwise.

### GetAppliedMembershipDigestOk

`func (o *HeartbeatRequestBody) GetAppliedMembershipDigestOk() (*string, bool)`

GetAppliedMembershipDigestOk returns a tuple with the AppliedMembershipDigest field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAppliedMembershipDigest

`func (o *HeartbeatRequestBody) SetAppliedMembershipDigest(v string)`

SetAppliedMembershipDigest sets AppliedMembershipDigest field to given value.

### HasAppliedMembershipDigest

`func (o *HeartbeatRequestBody) HasAppliedMembershipDigest() bool`

HasAppliedMembershipDigest returns a boolean if a field has been set.

### GetAppliedMembershipRevision

`func (o *HeartbeatRequestBody) GetAppliedMembershipRevision() int64`

GetAppliedMembershipRevision returns the AppliedMembershipRevision field if non-nil, zero value otherwise.

### GetAppliedMembershipRevisionOk

`func (o *HeartbeatRequestBody) GetAppliedMembershipRevisionOk() (*int64, bool)`

GetAppliedMembershipRevisionOk returns a tuple with the AppliedMembershipRevision field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAppliedMembershipRevision

`func (o *HeartbeatRequestBody) SetAppliedMembershipRevision(v int64)`

SetAppliedMembershipRevision sets AppliedMembershipRevision field to given value.

### HasAppliedMembershipRevision

`func (o *HeartbeatRequestBody) HasAppliedMembershipRevision() bool`

HasAppliedMembershipRevision returns a boolean if a field has been set.

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

### GetObservedCluster

`func (o *HeartbeatRequestBody) GetObservedCluster() string`

GetObservedCluster returns the ObservedCluster field if non-nil, zero value otherwise.

### GetObservedClusterOk

`func (o *HeartbeatRequestBody) GetObservedClusterOk() (*string, bool)`

GetObservedClusterOk returns a tuple with the ObservedCluster field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetObservedCluster

`func (o *HeartbeatRequestBody) SetObservedCluster(v string)`

SetObservedCluster sets ObservedCluster field to given value.

### HasObservedCluster

`func (o *HeartbeatRequestBody) HasObservedCluster() bool`

HasObservedCluster returns a boolean if a field has been set.

### GetObservedFingerprint

`func (o *HeartbeatRequestBody) GetObservedFingerprint() string`

GetObservedFingerprint returns the ObservedFingerprint field if non-nil, zero value otherwise.

### GetObservedFingerprintOk

`func (o *HeartbeatRequestBody) GetObservedFingerprintOk() (*string, bool)`

GetObservedFingerprintOk returns a tuple with the ObservedFingerprint field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetObservedFingerprint

`func (o *HeartbeatRequestBody) SetObservedFingerprint(v string)`

SetObservedFingerprint sets ObservedFingerprint field to given value.

### HasObservedFingerprint

`func (o *HeartbeatRequestBody) HasObservedFingerprint() bool`

HasObservedFingerprint returns a boolean if a field has been set.

### GetObservedGuests

`func (o *HeartbeatRequestBody) GetObservedGuests() int64`

GetObservedGuests returns the ObservedGuests field if non-nil, zero value otherwise.

### GetObservedGuestsOk

`func (o *HeartbeatRequestBody) GetObservedGuestsOk() (*int64, bool)`

GetObservedGuestsOk returns a tuple with the ObservedGuests field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetObservedGuests

`func (o *HeartbeatRequestBody) SetObservedGuests(v int64)`

SetObservedGuests sets ObservedGuests field to given value.

### HasObservedGuests

`func (o *HeartbeatRequestBody) HasObservedGuests() bool`

HasObservedGuests returns a boolean if a field has been set.

### GetObservedMembers

`func (o *HeartbeatRequestBody) GetObservedMembers() []string`

GetObservedMembers returns the ObservedMembers field if non-nil, zero value otherwise.

### GetObservedMembersOk

`func (o *HeartbeatRequestBody) GetObservedMembersOk() (*[]string, bool)`

GetObservedMembersOk returns a tuple with the ObservedMembers field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetObservedMembers

`func (o *HeartbeatRequestBody) SetObservedMembers(v []string)`

SetObservedMembers sets ObservedMembers field to given value.

### HasObservedMembers

`func (o *HeartbeatRequestBody) HasObservedMembers() bool`

HasObservedMembers returns a boolean if a field has been set.

### SetObservedMembersNil

`func (o *HeartbeatRequestBody) SetObservedMembersNil(b bool)`

 SetObservedMembersNil sets the value for ObservedMembers to be an explicit nil

### UnsetObservedMembers
`func (o *HeartbeatRequestBody) UnsetObservedMembers()`

UnsetObservedMembers ensures that no value is present for ObservedMembers, not even an explicit nil
### GetObservedQuorate

`func (o *HeartbeatRequestBody) GetObservedQuorate() bool`

GetObservedQuorate returns the ObservedQuorate field if non-nil, zero value otherwise.

### GetObservedQuorateOk

`func (o *HeartbeatRequestBody) GetObservedQuorateOk() (*bool, bool)`

GetObservedQuorateOk returns a tuple with the ObservedQuorate field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetObservedQuorate

`func (o *HeartbeatRequestBody) SetObservedQuorate(v bool)`

SetObservedQuorate sets ObservedQuorate field to given value.

### HasObservedQuorate

`func (o *HeartbeatRequestBody) HasObservedQuorate() bool`

HasObservedQuorate returns a boolean if a field has been set.

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


