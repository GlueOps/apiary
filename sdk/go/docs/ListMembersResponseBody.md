# ListMembersResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Members** | [**[]MemberView**](MemberView.md) |  | 

## Methods

### NewListMembersResponseBody

`func NewListMembersResponseBody(members []MemberView, ) *ListMembersResponseBody`

NewListMembersResponseBody instantiates a new ListMembersResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListMembersResponseBodyWithDefaults

`func NewListMembersResponseBodyWithDefaults() *ListMembersResponseBody`

NewListMembersResponseBodyWithDefaults instantiates a new ListMembersResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListMembersResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListMembersResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListMembersResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListMembersResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetMembers

`func (o *ListMembersResponseBody) GetMembers() []MemberView`

GetMembers returns the Members field if non-nil, zero value otherwise.

### GetMembersOk

`func (o *ListMembersResponseBody) GetMembersOk() (*[]MemberView, bool)`

GetMembersOk returns a tuple with the Members field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMembers

`func (o *ListMembersResponseBody) SetMembers(v []MemberView)`

SetMembers sets Members field to given value.


### SetMembersNil

`func (o *ListMembersResponseBody) SetMembersNil(b bool)`

 SetMembersNil sets the value for Members to be an explicit nil

### UnsetMembers
`func (o *ListMembersResponseBody) UnsetMembers()`

UnsetMembers ensures that no value is present for Members, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


