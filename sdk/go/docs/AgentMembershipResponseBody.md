# AgentMembershipResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Cluster** | Pointer to **string** | The corosync cluster name this hive should form. | [optional] 
**Departing** | Pointer to **[]string** | Nodes the control plane records as removed. A member may evict one of these and nothing else; anything present and unrecorded is reported as a discrepancy instead. | [optional] 
**Digest** | Pointer to **string** |  | [optional] 
**Members** | Pointer to [**[]MembershipMemberView**](MembershipMemberView.md) | The complete desired composition, not a delta. | [optional] 
**Revision** | Pointer to **int64** |  | [optional] 
**Self** | Pointer to **string** | This host&#39;s node name as the control plane knows it. Empty for a machine that has never joined a cluster. | [optional] 
**SelfId** | Pointer to **string** | This machine&#39;s hypervisor id, which is how a host with no node name yet locates itself in the list — the state every arrival starts from. | [optional] 

## Methods

### NewAgentMembershipResponseBody

`func NewAgentMembershipResponseBody() *AgentMembershipResponseBody`

NewAgentMembershipResponseBody instantiates a new AgentMembershipResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAgentMembershipResponseBodyWithDefaults

`func NewAgentMembershipResponseBodyWithDefaults() *AgentMembershipResponseBody`

NewAgentMembershipResponseBodyWithDefaults instantiates a new AgentMembershipResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *AgentMembershipResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *AgentMembershipResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *AgentMembershipResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *AgentMembershipResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetCluster

`func (o *AgentMembershipResponseBody) GetCluster() string`

GetCluster returns the Cluster field if non-nil, zero value otherwise.

### GetClusterOk

`func (o *AgentMembershipResponseBody) GetClusterOk() (*string, bool)`

GetClusterOk returns a tuple with the Cluster field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCluster

`func (o *AgentMembershipResponseBody) SetCluster(v string)`

SetCluster sets Cluster field to given value.

### HasCluster

`func (o *AgentMembershipResponseBody) HasCluster() bool`

HasCluster returns a boolean if a field has been set.

### GetDeparting

`func (o *AgentMembershipResponseBody) GetDeparting() []string`

GetDeparting returns the Departing field if non-nil, zero value otherwise.

### GetDepartingOk

`func (o *AgentMembershipResponseBody) GetDepartingOk() (*[]string, bool)`

GetDepartingOk returns a tuple with the Departing field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeparting

`func (o *AgentMembershipResponseBody) SetDeparting(v []string)`

SetDeparting sets Departing field to given value.

### HasDeparting

`func (o *AgentMembershipResponseBody) HasDeparting() bool`

HasDeparting returns a boolean if a field has been set.

### SetDepartingNil

`func (o *AgentMembershipResponseBody) SetDepartingNil(b bool)`

 SetDepartingNil sets the value for Departing to be an explicit nil

### UnsetDeparting
`func (o *AgentMembershipResponseBody) UnsetDeparting()`

UnsetDeparting ensures that no value is present for Departing, not even an explicit nil
### GetDigest

`func (o *AgentMembershipResponseBody) GetDigest() string`

GetDigest returns the Digest field if non-nil, zero value otherwise.

### GetDigestOk

`func (o *AgentMembershipResponseBody) GetDigestOk() (*string, bool)`

GetDigestOk returns a tuple with the Digest field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDigest

`func (o *AgentMembershipResponseBody) SetDigest(v string)`

SetDigest sets Digest field to given value.

### HasDigest

`func (o *AgentMembershipResponseBody) HasDigest() bool`

HasDigest returns a boolean if a field has been set.

### GetMembers

`func (o *AgentMembershipResponseBody) GetMembers() []MembershipMemberView`

GetMembers returns the Members field if non-nil, zero value otherwise.

### GetMembersOk

`func (o *AgentMembershipResponseBody) GetMembersOk() (*[]MembershipMemberView, bool)`

GetMembersOk returns a tuple with the Members field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMembers

`func (o *AgentMembershipResponseBody) SetMembers(v []MembershipMemberView)`

SetMembers sets Members field to given value.

### HasMembers

`func (o *AgentMembershipResponseBody) HasMembers() bool`

HasMembers returns a boolean if a field has been set.

### SetMembersNil

`func (o *AgentMembershipResponseBody) SetMembersNil(b bool)`

 SetMembersNil sets the value for Members to be an explicit nil

### UnsetMembers
`func (o *AgentMembershipResponseBody) UnsetMembers()`

UnsetMembers ensures that no value is present for Members, not even an explicit nil
### GetRevision

`func (o *AgentMembershipResponseBody) GetRevision() int64`

GetRevision returns the Revision field if non-nil, zero value otherwise.

### GetRevisionOk

`func (o *AgentMembershipResponseBody) GetRevisionOk() (*int64, bool)`

GetRevisionOk returns a tuple with the Revision field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRevision

`func (o *AgentMembershipResponseBody) SetRevision(v int64)`

SetRevision sets Revision field to given value.

### HasRevision

`func (o *AgentMembershipResponseBody) HasRevision() bool`

HasRevision returns a boolean if a field has been set.

### GetSelf

`func (o *AgentMembershipResponseBody) GetSelf() string`

GetSelf returns the Self field if non-nil, zero value otherwise.

### GetSelfOk

`func (o *AgentMembershipResponseBody) GetSelfOk() (*string, bool)`

GetSelfOk returns a tuple with the Self field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSelf

`func (o *AgentMembershipResponseBody) SetSelf(v string)`

SetSelf sets Self field to given value.

### HasSelf

`func (o *AgentMembershipResponseBody) HasSelf() bool`

HasSelf returns a boolean if a field has been set.

### GetSelfId

`func (o *AgentMembershipResponseBody) GetSelfId() string`

GetSelfId returns the SelfId field if non-nil, zero value otherwise.

### GetSelfIdOk

`func (o *AgentMembershipResponseBody) GetSelfIdOk() (*string, bool)`

GetSelfIdOk returns a tuple with the SelfId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSelfId

`func (o *AgentMembershipResponseBody) SetSelfId(v string)`

SetSelfId sets SelfId field to given value.

### HasSelfId

`func (o *AgentMembershipResponseBody) HasSelfId() bool`

HasSelfId returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


