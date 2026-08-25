# ListResponseBody1

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Invitations** | [**[]InvitationView**](InvitationView.md) |  | 

## Methods

### NewListResponseBody1

`func NewListResponseBody1(invitations []InvitationView, ) *ListResponseBody1`

NewListResponseBody1 instantiates a new ListResponseBody1 object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListResponseBody1WithDefaults

`func NewListResponseBody1WithDefaults() *ListResponseBody1`

NewListResponseBody1WithDefaults instantiates a new ListResponseBody1 object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListResponseBody1) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListResponseBody1) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListResponseBody1) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListResponseBody1) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetInvitations

`func (o *ListResponseBody1) GetInvitations() []InvitationView`

GetInvitations returns the Invitations field if non-nil, zero value otherwise.

### GetInvitationsOk

`func (o *ListResponseBody1) GetInvitationsOk() (*[]InvitationView, bool)`

GetInvitationsOk returns a tuple with the Invitations field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInvitations

`func (o *ListResponseBody1) SetInvitations(v []InvitationView)`

SetInvitations sets Invitations field to given value.


### SetInvitationsNil

`func (o *ListResponseBody1) SetInvitationsNil(b bool)`

 SetInvitationsNil sets the value for Invitations to be an explicit nil

### UnsetInvitations
`func (o *ListResponseBody1) UnsetInvitations()`

UnsetInvitations ensures that no value is present for Invitations, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


