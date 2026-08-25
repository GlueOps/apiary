# RuleView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Action** | **string** |  | 
**ChainName** | **string** |  | 
**Comment** | **string** |  | 
**DstCidr** | **string** |  | 
**DstPort** | **int64** |  | 
**DstPortEnd** | **int64** |  | 
**Enabled** | **bool** |  | 
**Family** | **string** |  | 
**Id** | **string** |  | 
**InIface** | **string** |  | 
**Name** | **string** |  | 
**OutIface** | **string** |  | 
**Position** | **int64** | Order within this ruleset. Order is the policy: an accept above a drop is a different rule from the reverse. | 
**Protocol** | **string** |  | 
**RulesetId** | **string** |  | 
**SrcCidr** | **string** |  | 
**SrcPort** | **int64** |  | 
**SrcPortEnd** | **int64** |  | 
**TableName** | **string** |  | 

## Methods

### NewRuleView

`func NewRuleView(action string, chainName string, comment string, dstCidr string, dstPort int64, dstPortEnd int64, enabled bool, family string, id string, inIface string, name string, outIface string, position int64, protocol string, rulesetId string, srcCidr string, srcPort int64, srcPortEnd int64, tableName string, ) *RuleView`

NewRuleView instantiates a new RuleView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRuleViewWithDefaults

`func NewRuleViewWithDefaults() *RuleView`

NewRuleViewWithDefaults instantiates a new RuleView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *RuleView) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *RuleView) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *RuleView) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *RuleView) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetAction

`func (o *RuleView) GetAction() string`

GetAction returns the Action field if non-nil, zero value otherwise.

### GetActionOk

`func (o *RuleView) GetActionOk() (*string, bool)`

GetActionOk returns a tuple with the Action field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAction

`func (o *RuleView) SetAction(v string)`

SetAction sets Action field to given value.


### GetChainName

`func (o *RuleView) GetChainName() string`

GetChainName returns the ChainName field if non-nil, zero value otherwise.

### GetChainNameOk

`func (o *RuleView) GetChainNameOk() (*string, bool)`

GetChainNameOk returns a tuple with the ChainName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetChainName

`func (o *RuleView) SetChainName(v string)`

SetChainName sets ChainName field to given value.


### GetComment

`func (o *RuleView) GetComment() string`

GetComment returns the Comment field if non-nil, zero value otherwise.

### GetCommentOk

`func (o *RuleView) GetCommentOk() (*string, bool)`

GetCommentOk returns a tuple with the Comment field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetComment

`func (o *RuleView) SetComment(v string)`

SetComment sets Comment field to given value.


### GetDstCidr

`func (o *RuleView) GetDstCidr() string`

GetDstCidr returns the DstCidr field if non-nil, zero value otherwise.

### GetDstCidrOk

`func (o *RuleView) GetDstCidrOk() (*string, bool)`

GetDstCidrOk returns a tuple with the DstCidr field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDstCidr

`func (o *RuleView) SetDstCidr(v string)`

SetDstCidr sets DstCidr field to given value.


### GetDstPort

`func (o *RuleView) GetDstPort() int64`

GetDstPort returns the DstPort field if non-nil, zero value otherwise.

### GetDstPortOk

`func (o *RuleView) GetDstPortOk() (*int64, bool)`

GetDstPortOk returns a tuple with the DstPort field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDstPort

`func (o *RuleView) SetDstPort(v int64)`

SetDstPort sets DstPort field to given value.


### GetDstPortEnd

`func (o *RuleView) GetDstPortEnd() int64`

GetDstPortEnd returns the DstPortEnd field if non-nil, zero value otherwise.

### GetDstPortEndOk

`func (o *RuleView) GetDstPortEndOk() (*int64, bool)`

GetDstPortEndOk returns a tuple with the DstPortEnd field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDstPortEnd

`func (o *RuleView) SetDstPortEnd(v int64)`

SetDstPortEnd sets DstPortEnd field to given value.


### GetEnabled

`func (o *RuleView) GetEnabled() bool`

GetEnabled returns the Enabled field if non-nil, zero value otherwise.

### GetEnabledOk

`func (o *RuleView) GetEnabledOk() (*bool, bool)`

GetEnabledOk returns a tuple with the Enabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnabled

`func (o *RuleView) SetEnabled(v bool)`

SetEnabled sets Enabled field to given value.


### GetFamily

`func (o *RuleView) GetFamily() string`

GetFamily returns the Family field if non-nil, zero value otherwise.

### GetFamilyOk

`func (o *RuleView) GetFamilyOk() (*string, bool)`

GetFamilyOk returns a tuple with the Family field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFamily

`func (o *RuleView) SetFamily(v string)`

SetFamily sets Family field to given value.


### GetId

`func (o *RuleView) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *RuleView) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *RuleView) SetId(v string)`

SetId sets Id field to given value.


### GetInIface

`func (o *RuleView) GetInIface() string`

GetInIface returns the InIface field if non-nil, zero value otherwise.

### GetInIfaceOk

`func (o *RuleView) GetInIfaceOk() (*string, bool)`

GetInIfaceOk returns a tuple with the InIface field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInIface

`func (o *RuleView) SetInIface(v string)`

SetInIface sets InIface field to given value.


### GetName

`func (o *RuleView) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *RuleView) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *RuleView) SetName(v string)`

SetName sets Name field to given value.


### GetOutIface

`func (o *RuleView) GetOutIface() string`

GetOutIface returns the OutIface field if non-nil, zero value otherwise.

### GetOutIfaceOk

`func (o *RuleView) GetOutIfaceOk() (*string, bool)`

GetOutIfaceOk returns a tuple with the OutIface field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOutIface

`func (o *RuleView) SetOutIface(v string)`

SetOutIface sets OutIface field to given value.


### GetPosition

`func (o *RuleView) GetPosition() int64`

GetPosition returns the Position field if non-nil, zero value otherwise.

### GetPositionOk

`func (o *RuleView) GetPositionOk() (*int64, bool)`

GetPositionOk returns a tuple with the Position field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPosition

`func (o *RuleView) SetPosition(v int64)`

SetPosition sets Position field to given value.


### GetProtocol

`func (o *RuleView) GetProtocol() string`

GetProtocol returns the Protocol field if non-nil, zero value otherwise.

### GetProtocolOk

`func (o *RuleView) GetProtocolOk() (*string, bool)`

GetProtocolOk returns a tuple with the Protocol field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProtocol

`func (o *RuleView) SetProtocol(v string)`

SetProtocol sets Protocol field to given value.


### GetRulesetId

`func (o *RuleView) GetRulesetId() string`

GetRulesetId returns the RulesetId field if non-nil, zero value otherwise.

### GetRulesetIdOk

`func (o *RuleView) GetRulesetIdOk() (*string, bool)`

GetRulesetIdOk returns a tuple with the RulesetId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRulesetId

`func (o *RuleView) SetRulesetId(v string)`

SetRulesetId sets RulesetId field to given value.


### GetSrcCidr

`func (o *RuleView) GetSrcCidr() string`

GetSrcCidr returns the SrcCidr field if non-nil, zero value otherwise.

### GetSrcCidrOk

`func (o *RuleView) GetSrcCidrOk() (*string, bool)`

GetSrcCidrOk returns a tuple with the SrcCidr field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSrcCidr

`func (o *RuleView) SetSrcCidr(v string)`

SetSrcCidr sets SrcCidr field to given value.


### GetSrcPort

`func (o *RuleView) GetSrcPort() int64`

GetSrcPort returns the SrcPort field if non-nil, zero value otherwise.

### GetSrcPortOk

`func (o *RuleView) GetSrcPortOk() (*int64, bool)`

GetSrcPortOk returns a tuple with the SrcPort field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSrcPort

`func (o *RuleView) SetSrcPort(v int64)`

SetSrcPort sets SrcPort field to given value.


### GetSrcPortEnd

`func (o *RuleView) GetSrcPortEnd() int64`

GetSrcPortEnd returns the SrcPortEnd field if non-nil, zero value otherwise.

### GetSrcPortEndOk

`func (o *RuleView) GetSrcPortEndOk() (*int64, bool)`

GetSrcPortEndOk returns a tuple with the SrcPortEnd field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSrcPortEnd

`func (o *RuleView) SetSrcPortEnd(v int64)`

SetSrcPortEnd sets SrcPortEnd field to given value.


### GetTableName

`func (o *RuleView) GetTableName() string`

GetTableName returns the TableName field if non-nil, zero value otherwise.

### GetTableNameOk

`func (o *RuleView) GetTableNameOk() (*string, bool)`

GetTableNameOk returns a tuple with the TableName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTableName

`func (o *RuleView) SetTableName(v string)`

SetTableName sets TableName field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


