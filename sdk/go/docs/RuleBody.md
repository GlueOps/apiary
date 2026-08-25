# RuleBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Action** | Pointer to **string** |  | [optional] [default to "accept"]
**ChainName** | Pointer to **string** |  | [optional] [default to "input"]
**Comment** | Pointer to **string** |  | [optional] 
**DstCidr** | Pointer to **string** |  | [optional] 
**DstPort** | Pointer to **int64** |  | [optional] 
**DstPortEnd** | Pointer to **int64** |  | [optional] 
**Enabled** | **bool** |  | [default to true]
**Family** | Pointer to **string** | inet covers both address families and is almost always what is wanted. | [optional] [default to "inet"]
**InIface** | Pointer to **string** | Empty matches any interface. | [optional] 
**Name** | **string** |  | 
**OutIface** | Pointer to **string** |  | [optional] 
**Position** | Pointer to **int64** | Space these out (10, 20, 30) so a rule can be inserted between two others without renumbering. | [optional] [default to 100]
**Protocol** | Pointer to **string** |  | [optional] [default to "tcp"]
**SrcCidr** | Pointer to **string** | CIDR. Must match the rule&#39;s family — nft rejects the WHOLE ruleset over one mismatch, taking the host&#39;s entire firewall with it. | [optional] 
**SrcPort** | Pointer to **int64** |  | [optional] 
**SrcPortEnd** | Pointer to **int64** | Zero means just the start port. | [optional] 
**TableName** | Pointer to **string** |  | [optional] [default to "filter"]

## Methods

### NewRuleBody

`func NewRuleBody(enabled bool, name string, ) *RuleBody`

NewRuleBody instantiates a new RuleBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRuleBodyWithDefaults

`func NewRuleBodyWithDefaults() *RuleBody`

NewRuleBodyWithDefaults instantiates a new RuleBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *RuleBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *RuleBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *RuleBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *RuleBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetAction

`func (o *RuleBody) GetAction() string`

GetAction returns the Action field if non-nil, zero value otherwise.

### GetActionOk

`func (o *RuleBody) GetActionOk() (*string, bool)`

GetActionOk returns a tuple with the Action field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAction

`func (o *RuleBody) SetAction(v string)`

SetAction sets Action field to given value.

### HasAction

`func (o *RuleBody) HasAction() bool`

HasAction returns a boolean if a field has been set.

### GetChainName

`func (o *RuleBody) GetChainName() string`

GetChainName returns the ChainName field if non-nil, zero value otherwise.

### GetChainNameOk

`func (o *RuleBody) GetChainNameOk() (*string, bool)`

GetChainNameOk returns a tuple with the ChainName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetChainName

`func (o *RuleBody) SetChainName(v string)`

SetChainName sets ChainName field to given value.

### HasChainName

`func (o *RuleBody) HasChainName() bool`

HasChainName returns a boolean if a field has been set.

### GetComment

`func (o *RuleBody) GetComment() string`

GetComment returns the Comment field if non-nil, zero value otherwise.

### GetCommentOk

`func (o *RuleBody) GetCommentOk() (*string, bool)`

GetCommentOk returns a tuple with the Comment field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetComment

`func (o *RuleBody) SetComment(v string)`

SetComment sets Comment field to given value.

### HasComment

`func (o *RuleBody) HasComment() bool`

HasComment returns a boolean if a field has been set.

### GetDstCidr

`func (o *RuleBody) GetDstCidr() string`

GetDstCidr returns the DstCidr field if non-nil, zero value otherwise.

### GetDstCidrOk

`func (o *RuleBody) GetDstCidrOk() (*string, bool)`

GetDstCidrOk returns a tuple with the DstCidr field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDstCidr

`func (o *RuleBody) SetDstCidr(v string)`

SetDstCidr sets DstCidr field to given value.

### HasDstCidr

`func (o *RuleBody) HasDstCidr() bool`

HasDstCidr returns a boolean if a field has been set.

### GetDstPort

`func (o *RuleBody) GetDstPort() int64`

GetDstPort returns the DstPort field if non-nil, zero value otherwise.

### GetDstPortOk

`func (o *RuleBody) GetDstPortOk() (*int64, bool)`

GetDstPortOk returns a tuple with the DstPort field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDstPort

`func (o *RuleBody) SetDstPort(v int64)`

SetDstPort sets DstPort field to given value.

### HasDstPort

`func (o *RuleBody) HasDstPort() bool`

HasDstPort returns a boolean if a field has been set.

### GetDstPortEnd

`func (o *RuleBody) GetDstPortEnd() int64`

GetDstPortEnd returns the DstPortEnd field if non-nil, zero value otherwise.

### GetDstPortEndOk

`func (o *RuleBody) GetDstPortEndOk() (*int64, bool)`

GetDstPortEndOk returns a tuple with the DstPortEnd field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDstPortEnd

`func (o *RuleBody) SetDstPortEnd(v int64)`

SetDstPortEnd sets DstPortEnd field to given value.

### HasDstPortEnd

`func (o *RuleBody) HasDstPortEnd() bool`

HasDstPortEnd returns a boolean if a field has been set.

### GetEnabled

`func (o *RuleBody) GetEnabled() bool`

GetEnabled returns the Enabled field if non-nil, zero value otherwise.

### GetEnabledOk

`func (o *RuleBody) GetEnabledOk() (*bool, bool)`

GetEnabledOk returns a tuple with the Enabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnabled

`func (o *RuleBody) SetEnabled(v bool)`

SetEnabled sets Enabled field to given value.


### GetFamily

`func (o *RuleBody) GetFamily() string`

GetFamily returns the Family field if non-nil, zero value otherwise.

### GetFamilyOk

`func (o *RuleBody) GetFamilyOk() (*string, bool)`

GetFamilyOk returns a tuple with the Family field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFamily

`func (o *RuleBody) SetFamily(v string)`

SetFamily sets Family field to given value.

### HasFamily

`func (o *RuleBody) HasFamily() bool`

HasFamily returns a boolean if a field has been set.

### GetInIface

`func (o *RuleBody) GetInIface() string`

GetInIface returns the InIface field if non-nil, zero value otherwise.

### GetInIfaceOk

`func (o *RuleBody) GetInIfaceOk() (*string, bool)`

GetInIfaceOk returns a tuple with the InIface field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInIface

`func (o *RuleBody) SetInIface(v string)`

SetInIface sets InIface field to given value.

### HasInIface

`func (o *RuleBody) HasInIface() bool`

HasInIface returns a boolean if a field has been set.

### GetName

`func (o *RuleBody) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *RuleBody) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *RuleBody) SetName(v string)`

SetName sets Name field to given value.


### GetOutIface

`func (o *RuleBody) GetOutIface() string`

GetOutIface returns the OutIface field if non-nil, zero value otherwise.

### GetOutIfaceOk

`func (o *RuleBody) GetOutIfaceOk() (*string, bool)`

GetOutIfaceOk returns a tuple with the OutIface field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOutIface

`func (o *RuleBody) SetOutIface(v string)`

SetOutIface sets OutIface field to given value.

### HasOutIface

`func (o *RuleBody) HasOutIface() bool`

HasOutIface returns a boolean if a field has been set.

### GetPosition

`func (o *RuleBody) GetPosition() int64`

GetPosition returns the Position field if non-nil, zero value otherwise.

### GetPositionOk

`func (o *RuleBody) GetPositionOk() (*int64, bool)`

GetPositionOk returns a tuple with the Position field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPosition

`func (o *RuleBody) SetPosition(v int64)`

SetPosition sets Position field to given value.

### HasPosition

`func (o *RuleBody) HasPosition() bool`

HasPosition returns a boolean if a field has been set.

### GetProtocol

`func (o *RuleBody) GetProtocol() string`

GetProtocol returns the Protocol field if non-nil, zero value otherwise.

### GetProtocolOk

`func (o *RuleBody) GetProtocolOk() (*string, bool)`

GetProtocolOk returns a tuple with the Protocol field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProtocol

`func (o *RuleBody) SetProtocol(v string)`

SetProtocol sets Protocol field to given value.

### HasProtocol

`func (o *RuleBody) HasProtocol() bool`

HasProtocol returns a boolean if a field has been set.

### GetSrcCidr

`func (o *RuleBody) GetSrcCidr() string`

GetSrcCidr returns the SrcCidr field if non-nil, zero value otherwise.

### GetSrcCidrOk

`func (o *RuleBody) GetSrcCidrOk() (*string, bool)`

GetSrcCidrOk returns a tuple with the SrcCidr field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSrcCidr

`func (o *RuleBody) SetSrcCidr(v string)`

SetSrcCidr sets SrcCidr field to given value.

### HasSrcCidr

`func (o *RuleBody) HasSrcCidr() bool`

HasSrcCidr returns a boolean if a field has been set.

### GetSrcPort

`func (o *RuleBody) GetSrcPort() int64`

GetSrcPort returns the SrcPort field if non-nil, zero value otherwise.

### GetSrcPortOk

`func (o *RuleBody) GetSrcPortOk() (*int64, bool)`

GetSrcPortOk returns a tuple with the SrcPort field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSrcPort

`func (o *RuleBody) SetSrcPort(v int64)`

SetSrcPort sets SrcPort field to given value.

### HasSrcPort

`func (o *RuleBody) HasSrcPort() bool`

HasSrcPort returns a boolean if a field has been set.

### GetSrcPortEnd

`func (o *RuleBody) GetSrcPortEnd() int64`

GetSrcPortEnd returns the SrcPortEnd field if non-nil, zero value otherwise.

### GetSrcPortEndOk

`func (o *RuleBody) GetSrcPortEndOk() (*int64, bool)`

GetSrcPortEndOk returns a tuple with the SrcPortEnd field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSrcPortEnd

`func (o *RuleBody) SetSrcPortEnd(v int64)`

SetSrcPortEnd sets SrcPortEnd field to given value.

### HasSrcPortEnd

`func (o *RuleBody) HasSrcPortEnd() bool`

HasSrcPortEnd returns a boolean if a field has been set.

### GetTableName

`func (o *RuleBody) GetTableName() string`

GetTableName returns the TableName field if non-nil, zero value otherwise.

### GetTableNameOk

`func (o *RuleBody) GetTableNameOk() (*string, bool)`

GetTableNameOk returns a tuple with the TableName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTableName

`func (o *RuleBody) SetTableName(v string)`

SetTableName sets TableName field to given value.

### HasTableName

`func (o *RuleBody) HasTableName() bool`

HasTableName returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


