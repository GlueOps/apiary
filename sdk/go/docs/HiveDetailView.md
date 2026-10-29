# HiveDetailView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**ApiUrl** | Pointer to **string** |  | [optional] 
**Configured** | **bool** |  | 
**CpuOvercommitRatio** | **float64** |  | 
**CredentialId** | Pointer to **string** |  | [optional] 
**Description** | Pointer to **string** |  | [optional] 
**EffectiveMinimumNodes** | **int64** | What minimum_nodes resolves to, so a client does not reimplement the zero-means-default rule. | 
**Id** | **string** |  | 
**InsecureSkipVerify** | **bool** |  | 
**MinimumNodes** | **int64** | Fewest nodes a departure may LEAVE the cluster with, counting departures already under way as gone. Zero means the default of 3. A cluster left with two loses quorum the moment either fails, and a departure from two strands the survivor read-only. | 
**Name** | **string** |  | 
**NodeCount** | **int64** |  | 
**Provider** | **string** |  | 

## Methods

### NewHiveDetailView

`func NewHiveDetailView(configured bool, cpuOvercommitRatio float64, effectiveMinimumNodes int64, id string, insecureSkipVerify bool, minimumNodes int64, name string, nodeCount int64, provider string, ) *HiveDetailView`

NewHiveDetailView instantiates a new HiveDetailView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewHiveDetailViewWithDefaults

`func NewHiveDetailViewWithDefaults() *HiveDetailView`

NewHiveDetailViewWithDefaults instantiates a new HiveDetailView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *HiveDetailView) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *HiveDetailView) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *HiveDetailView) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *HiveDetailView) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetApiUrl

`func (o *HiveDetailView) GetApiUrl() string`

GetApiUrl returns the ApiUrl field if non-nil, zero value otherwise.

### GetApiUrlOk

`func (o *HiveDetailView) GetApiUrlOk() (*string, bool)`

GetApiUrlOk returns a tuple with the ApiUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApiUrl

`func (o *HiveDetailView) SetApiUrl(v string)`

SetApiUrl sets ApiUrl field to given value.

### HasApiUrl

`func (o *HiveDetailView) HasApiUrl() bool`

HasApiUrl returns a boolean if a field has been set.

### GetConfigured

`func (o *HiveDetailView) GetConfigured() bool`

GetConfigured returns the Configured field if non-nil, zero value otherwise.

### GetConfiguredOk

`func (o *HiveDetailView) GetConfiguredOk() (*bool, bool)`

GetConfiguredOk returns a tuple with the Configured field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConfigured

`func (o *HiveDetailView) SetConfigured(v bool)`

SetConfigured sets Configured field to given value.


### GetCpuOvercommitRatio

`func (o *HiveDetailView) GetCpuOvercommitRatio() float64`

GetCpuOvercommitRatio returns the CpuOvercommitRatio field if non-nil, zero value otherwise.

### GetCpuOvercommitRatioOk

`func (o *HiveDetailView) GetCpuOvercommitRatioOk() (*float64, bool)`

GetCpuOvercommitRatioOk returns a tuple with the CpuOvercommitRatio field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCpuOvercommitRatio

`func (o *HiveDetailView) SetCpuOvercommitRatio(v float64)`

SetCpuOvercommitRatio sets CpuOvercommitRatio field to given value.


### GetCredentialId

`func (o *HiveDetailView) GetCredentialId() string`

GetCredentialId returns the CredentialId field if non-nil, zero value otherwise.

### GetCredentialIdOk

`func (o *HiveDetailView) GetCredentialIdOk() (*string, bool)`

GetCredentialIdOk returns a tuple with the CredentialId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCredentialId

`func (o *HiveDetailView) SetCredentialId(v string)`

SetCredentialId sets CredentialId field to given value.

### HasCredentialId

`func (o *HiveDetailView) HasCredentialId() bool`

HasCredentialId returns a boolean if a field has been set.

### GetDescription

`func (o *HiveDetailView) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *HiveDetailView) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *HiveDetailView) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *HiveDetailView) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetEffectiveMinimumNodes

`func (o *HiveDetailView) GetEffectiveMinimumNodes() int64`

GetEffectiveMinimumNodes returns the EffectiveMinimumNodes field if non-nil, zero value otherwise.

### GetEffectiveMinimumNodesOk

`func (o *HiveDetailView) GetEffectiveMinimumNodesOk() (*int64, bool)`

GetEffectiveMinimumNodesOk returns a tuple with the EffectiveMinimumNodes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEffectiveMinimumNodes

`func (o *HiveDetailView) SetEffectiveMinimumNodes(v int64)`

SetEffectiveMinimumNodes sets EffectiveMinimumNodes field to given value.


### GetId

`func (o *HiveDetailView) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *HiveDetailView) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *HiveDetailView) SetId(v string)`

SetId sets Id field to given value.


### GetInsecureSkipVerify

`func (o *HiveDetailView) GetInsecureSkipVerify() bool`

GetInsecureSkipVerify returns the InsecureSkipVerify field if non-nil, zero value otherwise.

### GetInsecureSkipVerifyOk

`func (o *HiveDetailView) GetInsecureSkipVerifyOk() (*bool, bool)`

GetInsecureSkipVerifyOk returns a tuple with the InsecureSkipVerify field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInsecureSkipVerify

`func (o *HiveDetailView) SetInsecureSkipVerify(v bool)`

SetInsecureSkipVerify sets InsecureSkipVerify field to given value.


### GetMinimumNodes

`func (o *HiveDetailView) GetMinimumNodes() int64`

GetMinimumNodes returns the MinimumNodes field if non-nil, zero value otherwise.

### GetMinimumNodesOk

`func (o *HiveDetailView) GetMinimumNodesOk() (*int64, bool)`

GetMinimumNodesOk returns a tuple with the MinimumNodes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMinimumNodes

`func (o *HiveDetailView) SetMinimumNodes(v int64)`

SetMinimumNodes sets MinimumNodes field to given value.


### GetName

`func (o *HiveDetailView) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *HiveDetailView) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *HiveDetailView) SetName(v string)`

SetName sets Name field to given value.


### GetNodeCount

`func (o *HiveDetailView) GetNodeCount() int64`

GetNodeCount returns the NodeCount field if non-nil, zero value otherwise.

### GetNodeCountOk

`func (o *HiveDetailView) GetNodeCountOk() (*int64, bool)`

GetNodeCountOk returns a tuple with the NodeCount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNodeCount

`func (o *HiveDetailView) SetNodeCount(v int64)`

SetNodeCount sets NodeCount field to given value.


### GetProvider

`func (o *HiveDetailView) GetProvider() string`

GetProvider returns the Provider field if non-nil, zero value otherwise.

### GetProviderOk

`func (o *HiveDetailView) GetProviderOk() (*string, bool)`

GetProviderOk returns a tuple with the Provider field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProvider

`func (o *HiveDetailView) SetProvider(v string)`

SetProvider sets Provider field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


