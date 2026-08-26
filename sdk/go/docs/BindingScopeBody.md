# BindingScopeBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**HiveId** | Pointer to **string** | The fleet this binding applies to. Empty means every host in the organization, including hosts in no fleet. | [optional] 
**SwarmId** | Pointer to **string** | A group inside that fleet. Requires hive_id: swarm names are unique only within a fleet, so a bare swarm id names a group without saying whose. | [optional] 

## Methods

### NewBindingScopeBody

`func NewBindingScopeBody() *BindingScopeBody`

NewBindingScopeBody instantiates a new BindingScopeBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewBindingScopeBodyWithDefaults

`func NewBindingScopeBodyWithDefaults() *BindingScopeBody`

NewBindingScopeBodyWithDefaults instantiates a new BindingScopeBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetHiveId

`func (o *BindingScopeBody) GetHiveId() string`

GetHiveId returns the HiveId field if non-nil, zero value otherwise.

### GetHiveIdOk

`func (o *BindingScopeBody) GetHiveIdOk() (*string, bool)`

GetHiveIdOk returns a tuple with the HiveId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHiveId

`func (o *BindingScopeBody) SetHiveId(v string)`

SetHiveId sets HiveId field to given value.

### HasHiveId

`func (o *BindingScopeBody) HasHiveId() bool`

HasHiveId returns a boolean if a field has been set.

### GetSwarmId

`func (o *BindingScopeBody) GetSwarmId() string`

GetSwarmId returns the SwarmId field if non-nil, zero value otherwise.

### GetSwarmIdOk

`func (o *BindingScopeBody) GetSwarmIdOk() (*string, bool)`

GetSwarmIdOk returns a tuple with the SwarmId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSwarmId

`func (o *BindingScopeBody) SetSwarmId(v string)`

SetSwarmId sets SwarmId field to given value.

### HasSwarmId

`func (o *BindingScopeBody) HasSwarmId() bool`

HasSwarmId returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


