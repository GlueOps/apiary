# MembershipDesiredView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Digest** | **string** | Of the rendered composition. A revision bumped by an edit that changed no member leaves this identical, and no host touches corosync. | 
**Revision** | **int64** | Only ever increases, including when a node is added back after being removed. An agent comparing revisions can use !&#x3D; or &lt; and behave identically. | 

## Methods

### NewMembershipDesiredView

`func NewMembershipDesiredView(digest string, revision int64, ) *MembershipDesiredView`

NewMembershipDesiredView instantiates a new MembershipDesiredView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMembershipDesiredViewWithDefaults

`func NewMembershipDesiredViewWithDefaults() *MembershipDesiredView`

NewMembershipDesiredViewWithDefaults instantiates a new MembershipDesiredView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDigest

`func (o *MembershipDesiredView) GetDigest() string`

GetDigest returns the Digest field if non-nil, zero value otherwise.

### GetDigestOk

`func (o *MembershipDesiredView) GetDigestOk() (*string, bool)`

GetDigestOk returns a tuple with the Digest field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDigest

`func (o *MembershipDesiredView) SetDigest(v string)`

SetDigest sets Digest field to given value.


### GetRevision

`func (o *MembershipDesiredView) GetRevision() int64`

GetRevision returns the Revision field if non-nil, zero value otherwise.

### GetRevisionOk

`func (o *MembershipDesiredView) GetRevisionOk() (*int64, bool)`

GetRevisionOk returns a tuple with the Revision field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRevision

`func (o *MembershipDesiredView) SetRevision(v int64)`

SetRevision sets Revision field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


