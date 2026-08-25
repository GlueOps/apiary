# APIKeyView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CreatedAt** | **time.Time** |  | 
**ExpiresAt** | Pointer to **time.Time** | Absent means the key does not expire. | [optional] 
**Id** | **string** |  | 
**LastUsedAt** | Pointer to **time.Time** | Absent means the key has never authenticated a request. Best-effort: a failure to record a use never fails the request it belongs to. | [optional] 
**Name** | **string** |  | 
**Prefix** | Pointer to **string** | First characters of the identifying half, so a key can be told apart in a list. Not a credential and not enough to derive one. | [optional] 
**Revoked** | **bool** |  | 
**Scope** | **string** | &#39;user&#39; is the caller&#39;s own key; &#39;org&#39; is the organization&#39;s, visible to every member and revocable by admins. Agent keys are not listed here — they belong to the fleet surface and are managed through enrolment. | 

## Methods

### NewAPIKeyView

`func NewAPIKeyView(createdAt time.Time, id string, name string, revoked bool, scope string, ) *APIKeyView`

NewAPIKeyView instantiates a new APIKeyView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAPIKeyViewWithDefaults

`func NewAPIKeyViewWithDefaults() *APIKeyView`

NewAPIKeyViewWithDefaults instantiates a new APIKeyView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCreatedAt

`func (o *APIKeyView) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *APIKeyView) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *APIKeyView) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.


### GetExpiresAt

`func (o *APIKeyView) GetExpiresAt() time.Time`

GetExpiresAt returns the ExpiresAt field if non-nil, zero value otherwise.

### GetExpiresAtOk

`func (o *APIKeyView) GetExpiresAtOk() (*time.Time, bool)`

GetExpiresAtOk returns a tuple with the ExpiresAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExpiresAt

`func (o *APIKeyView) SetExpiresAt(v time.Time)`

SetExpiresAt sets ExpiresAt field to given value.

### HasExpiresAt

`func (o *APIKeyView) HasExpiresAt() bool`

HasExpiresAt returns a boolean if a field has been set.

### GetId

`func (o *APIKeyView) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *APIKeyView) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *APIKeyView) SetId(v string)`

SetId sets Id field to given value.


### GetLastUsedAt

`func (o *APIKeyView) GetLastUsedAt() time.Time`

GetLastUsedAt returns the LastUsedAt field if non-nil, zero value otherwise.

### GetLastUsedAtOk

`func (o *APIKeyView) GetLastUsedAtOk() (*time.Time, bool)`

GetLastUsedAtOk returns a tuple with the LastUsedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastUsedAt

`func (o *APIKeyView) SetLastUsedAt(v time.Time)`

SetLastUsedAt sets LastUsedAt field to given value.

### HasLastUsedAt

`func (o *APIKeyView) HasLastUsedAt() bool`

HasLastUsedAt returns a boolean if a field has been set.

### GetName

`func (o *APIKeyView) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *APIKeyView) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *APIKeyView) SetName(v string)`

SetName sets Name field to given value.


### GetPrefix

`func (o *APIKeyView) GetPrefix() string`

GetPrefix returns the Prefix field if non-nil, zero value otherwise.

### GetPrefixOk

`func (o *APIKeyView) GetPrefixOk() (*string, bool)`

GetPrefixOk returns a tuple with the Prefix field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPrefix

`func (o *APIKeyView) SetPrefix(v string)`

SetPrefix sets Prefix field to given value.

### HasPrefix

`func (o *APIKeyView) HasPrefix() bool`

HasPrefix returns a boolean if a field has been set.

### GetRevoked

`func (o *APIKeyView) GetRevoked() bool`

GetRevoked returns the Revoked field if non-nil, zero value otherwise.

### GetRevokedOk

`func (o *APIKeyView) GetRevokedOk() (*bool, bool)`

GetRevokedOk returns a tuple with the Revoked field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRevoked

`func (o *APIKeyView) SetRevoked(v bool)`

SetRevoked sets Revoked field to given value.


### GetScope

`func (o *APIKeyView) GetScope() string`

GetScope returns the Scope field if non-nil, zero value otherwise.

### GetScopeOk

`func (o *APIKeyView) GetScopeOk() (*string, bool)`

GetScopeOk returns a tuple with the Scope field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetScope

`func (o *APIKeyView) SetScope(v string)`

SetScope sets Scope field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


