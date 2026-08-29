# MembershipMemberView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Address** | **string** |  | 
**Fingerprint** | Pointer to **string** | This node&#39;s API certificate fingerprint, as its OWN agent reported it. A member without one is not a usable join target. | [optional] 
**HostKeys** | Pointer to **[]string** | This node&#39;s SSH host public keys. A joining host pins these before pvecm add runs ssh-copy-id. | [optional] 
**HypervisorId** | **string** | Identifies the machine across a node name it does not have yet. | 
**Node** | Pointer to **string** | The Proxmox node name, absent for a machine that has never been in a cluster. | [optional] 
**RootKey** | Pointer to **string** | This node&#39;s root SSH public key. An existing member installs it into the cluster&#39;s replicated authorized_keys so a desired-but-absent node can authenticate its join. The joiner cannot do this itself: the file is cluster state it has no access to until it is a member. | [optional] 

## Methods

### NewMembershipMemberView

`func NewMembershipMemberView(address string, hypervisorId string, ) *MembershipMemberView`

NewMembershipMemberView instantiates a new MembershipMemberView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMembershipMemberViewWithDefaults

`func NewMembershipMemberViewWithDefaults() *MembershipMemberView`

NewMembershipMemberViewWithDefaults instantiates a new MembershipMemberView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAddress

`func (o *MembershipMemberView) GetAddress() string`

GetAddress returns the Address field if non-nil, zero value otherwise.

### GetAddressOk

`func (o *MembershipMemberView) GetAddressOk() (*string, bool)`

GetAddressOk returns a tuple with the Address field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAddress

`func (o *MembershipMemberView) SetAddress(v string)`

SetAddress sets Address field to given value.


### GetFingerprint

`func (o *MembershipMemberView) GetFingerprint() string`

GetFingerprint returns the Fingerprint field if non-nil, zero value otherwise.

### GetFingerprintOk

`func (o *MembershipMemberView) GetFingerprintOk() (*string, bool)`

GetFingerprintOk returns a tuple with the Fingerprint field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFingerprint

`func (o *MembershipMemberView) SetFingerprint(v string)`

SetFingerprint sets Fingerprint field to given value.

### HasFingerprint

`func (o *MembershipMemberView) HasFingerprint() bool`

HasFingerprint returns a boolean if a field has been set.

### GetHostKeys

`func (o *MembershipMemberView) GetHostKeys() []string`

GetHostKeys returns the HostKeys field if non-nil, zero value otherwise.

### GetHostKeysOk

`func (o *MembershipMemberView) GetHostKeysOk() (*[]string, bool)`

GetHostKeysOk returns a tuple with the HostKeys field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHostKeys

`func (o *MembershipMemberView) SetHostKeys(v []string)`

SetHostKeys sets HostKeys field to given value.

### HasHostKeys

`func (o *MembershipMemberView) HasHostKeys() bool`

HasHostKeys returns a boolean if a field has been set.

### SetHostKeysNil

`func (o *MembershipMemberView) SetHostKeysNil(b bool)`

 SetHostKeysNil sets the value for HostKeys to be an explicit nil

### UnsetHostKeys
`func (o *MembershipMemberView) UnsetHostKeys()`

UnsetHostKeys ensures that no value is present for HostKeys, not even an explicit nil
### GetHypervisorId

`func (o *MembershipMemberView) GetHypervisorId() string`

GetHypervisorId returns the HypervisorId field if non-nil, zero value otherwise.

### GetHypervisorIdOk

`func (o *MembershipMemberView) GetHypervisorIdOk() (*string, bool)`

GetHypervisorIdOk returns a tuple with the HypervisorId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHypervisorId

`func (o *MembershipMemberView) SetHypervisorId(v string)`

SetHypervisorId sets HypervisorId field to given value.


### GetNode

`func (o *MembershipMemberView) GetNode() string`

GetNode returns the Node field if non-nil, zero value otherwise.

### GetNodeOk

`func (o *MembershipMemberView) GetNodeOk() (*string, bool)`

GetNodeOk returns a tuple with the Node field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNode

`func (o *MembershipMemberView) SetNode(v string)`

SetNode sets Node field to given value.

### HasNode

`func (o *MembershipMemberView) HasNode() bool`

HasNode returns a boolean if a field has been set.

### GetRootKey

`func (o *MembershipMemberView) GetRootKey() string`

GetRootKey returns the RootKey field if non-nil, zero value otherwise.

### GetRootKeyOk

`func (o *MembershipMemberView) GetRootKeyOk() (*string, bool)`

GetRootKeyOk returns a tuple with the RootKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRootKey

`func (o *MembershipMemberView) SetRootKey(v string)`

SetRootKey sets RootKey field to given value.

### HasRootKey

`func (o *MembershipMemberView) HasRootKey() bool`

HasRootKey returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


