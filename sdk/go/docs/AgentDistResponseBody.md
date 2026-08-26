# AgentDistResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Builds** | [**[]AgentBuildView**](AgentBuildView.md) | Empty when this server carries no agent binaries, which is normal for a development build. An agent treats that as nothing to do rather than as a failure. | 
**Version** | **string** | The control plane&#39;s version. These binaries shipped with it, so an agent compares its own against this. | 

## Methods

### NewAgentDistResponseBody

`func NewAgentDistResponseBody(builds []AgentBuildView, version string, ) *AgentDistResponseBody`

NewAgentDistResponseBody instantiates a new AgentDistResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAgentDistResponseBodyWithDefaults

`func NewAgentDistResponseBodyWithDefaults() *AgentDistResponseBody`

NewAgentDistResponseBodyWithDefaults instantiates a new AgentDistResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *AgentDistResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *AgentDistResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *AgentDistResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *AgentDistResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetBuilds

`func (o *AgentDistResponseBody) GetBuilds() []AgentBuildView`

GetBuilds returns the Builds field if non-nil, zero value otherwise.

### GetBuildsOk

`func (o *AgentDistResponseBody) GetBuildsOk() (*[]AgentBuildView, bool)`

GetBuildsOk returns a tuple with the Builds field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBuilds

`func (o *AgentDistResponseBody) SetBuilds(v []AgentBuildView)`

SetBuilds sets Builds field to given value.


### SetBuildsNil

`func (o *AgentDistResponseBody) SetBuildsNil(b bool)`

 SetBuildsNil sets the value for Builds to be an explicit nil

### UnsetBuilds
`func (o *AgentDistResponseBody) UnsetBuilds()`

UnsetBuilds ensures that no value is present for Builds, not even an explicit nil
### GetVersion

`func (o *AgentDistResponseBody) GetVersion() string`

GetVersion returns the Version field if non-nil, zero value otherwise.

### GetVersionOk

`func (o *AgentDistResponseBody) GetVersionOk() (*string, bool)`

GetVersionOk returns a tuple with the Version field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVersion

`func (o *AgentDistResponseBody) SetVersion(v string)`

SetVersion sets Version field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


