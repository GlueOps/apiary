# FirewallDesiredView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Digest** | **string** | Of the rendered body. Compared as well as the revision, so a host that loaded the right revision from a corrupted or hand-edited file is still seen as diverged. | 
**Revision** | **int64** | Only ever increases. A backout is a NEW revision carrying an older body, so an agent comparing revisions can use !&#x3D; or &lt; and behave identically. | 

## Methods

### NewFirewallDesiredView

`func NewFirewallDesiredView(digest string, revision int64, ) *FirewallDesiredView`

NewFirewallDesiredView instantiates a new FirewallDesiredView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewFirewallDesiredViewWithDefaults

`func NewFirewallDesiredViewWithDefaults() *FirewallDesiredView`

NewFirewallDesiredViewWithDefaults instantiates a new FirewallDesiredView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDigest

`func (o *FirewallDesiredView) GetDigest() string`

GetDigest returns the Digest field if non-nil, zero value otherwise.

### GetDigestOk

`func (o *FirewallDesiredView) GetDigestOk() (*string, bool)`

GetDigestOk returns a tuple with the Digest field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDigest

`func (o *FirewallDesiredView) SetDigest(v string)`

SetDigest sets Digest field to given value.


### GetRevision

`func (o *FirewallDesiredView) GetRevision() int64`

GetRevision returns the Revision field if non-nil, zero value otherwise.

### GetRevisionOk

`func (o *FirewallDesiredView) GetRevisionOk() (*int64, bool)`

GetRevisionOk returns a tuple with the Revision field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRevision

`func (o *FirewallDesiredView) SetRevision(v int64)`

SetRevision sets Revision field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


