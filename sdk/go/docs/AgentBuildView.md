# AgentBuildView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Arch** | **string** |  | 
**Os** | **string** |  | 
**Sha256** | **string** | Verified by the agent before it installs anything. Without it an upgrade is &#39;execute whatever arrived&#39;. | 
**Size** | **int64** |  | 

## Methods

### NewAgentBuildView

`func NewAgentBuildView(arch string, os string, sha256 string, size int64, ) *AgentBuildView`

NewAgentBuildView instantiates a new AgentBuildView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAgentBuildViewWithDefaults

`func NewAgentBuildViewWithDefaults() *AgentBuildView`

NewAgentBuildViewWithDefaults instantiates a new AgentBuildView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetArch

`func (o *AgentBuildView) GetArch() string`

GetArch returns the Arch field if non-nil, zero value otherwise.

### GetArchOk

`func (o *AgentBuildView) GetArchOk() (*string, bool)`

GetArchOk returns a tuple with the Arch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArch

`func (o *AgentBuildView) SetArch(v string)`

SetArch sets Arch field to given value.


### GetOs

`func (o *AgentBuildView) GetOs() string`

GetOs returns the Os field if non-nil, zero value otherwise.

### GetOsOk

`func (o *AgentBuildView) GetOsOk() (*string, bool)`

GetOsOk returns a tuple with the Os field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOs

`func (o *AgentBuildView) SetOs(v string)`

SetOs sets Os field to given value.


### GetSha256

`func (o *AgentBuildView) GetSha256() string`

GetSha256 returns the Sha256 field if non-nil, zero value otherwise.

### GetSha256Ok

`func (o *AgentBuildView) GetSha256Ok() (*string, bool)`

GetSha256Ok returns a tuple with the Sha256 field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSha256

`func (o *AgentBuildView) SetSha256(v string)`

SetSha256 sets Sha256 field to given value.


### GetSize

`func (o *AgentBuildView) GetSize() int64`

GetSize returns the Size field if non-nil, zero value otherwise.

### GetSizeOk

`func (o *AgentBuildView) GetSizeOk() (*int64, bool)`

GetSizeOk returns a tuple with the Size field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSize

`func (o *AgentBuildView) SetSize(v int64)`

SetSize sets Size field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


