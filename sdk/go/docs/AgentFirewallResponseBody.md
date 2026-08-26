# AgentFirewallResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Content** | Pointer to **string** | The complete nft ruleset, to be loaded with nft -f. It begins with a flush because it is desired state rather than a patch: anything already loaded that it does not declare must go. | [optional] 
**Digest** | Pointer to **string** |  | [optional] 
**Revision** | Pointer to **int64** |  | [optional] 

## Methods

### NewAgentFirewallResponseBody

`func NewAgentFirewallResponseBody() *AgentFirewallResponseBody`

NewAgentFirewallResponseBody instantiates a new AgentFirewallResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAgentFirewallResponseBodyWithDefaults

`func NewAgentFirewallResponseBodyWithDefaults() *AgentFirewallResponseBody`

NewAgentFirewallResponseBodyWithDefaults instantiates a new AgentFirewallResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *AgentFirewallResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *AgentFirewallResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *AgentFirewallResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *AgentFirewallResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetContent

`func (o *AgentFirewallResponseBody) GetContent() string`

GetContent returns the Content field if non-nil, zero value otherwise.

### GetContentOk

`func (o *AgentFirewallResponseBody) GetContentOk() (*string, bool)`

GetContentOk returns a tuple with the Content field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetContent

`func (o *AgentFirewallResponseBody) SetContent(v string)`

SetContent sets Content field to given value.

### HasContent

`func (o *AgentFirewallResponseBody) HasContent() bool`

HasContent returns a boolean if a field has been set.

### GetDigest

`func (o *AgentFirewallResponseBody) GetDigest() string`

GetDigest returns the Digest field if non-nil, zero value otherwise.

### GetDigestOk

`func (o *AgentFirewallResponseBody) GetDigestOk() (*string, bool)`

GetDigestOk returns a tuple with the Digest field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDigest

`func (o *AgentFirewallResponseBody) SetDigest(v string)`

SetDigest sets Digest field to given value.

### HasDigest

`func (o *AgentFirewallResponseBody) HasDigest() bool`

HasDigest returns a boolean if a field has been set.

### GetRevision

`func (o *AgentFirewallResponseBody) GetRevision() int64`

GetRevision returns the Revision field if non-nil, zero value otherwise.

### GetRevisionOk

`func (o *AgentFirewallResponseBody) GetRevisionOk() (*int64, bool)`

GetRevisionOk returns a tuple with the Revision field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRevision

`func (o *AgentFirewallResponseBody) SetRevision(v int64)`

SetRevision sets Revision field to given value.

### HasRevision

`func (o *AgentFirewallResponseBody) HasRevision() bool`

HasRevision returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


