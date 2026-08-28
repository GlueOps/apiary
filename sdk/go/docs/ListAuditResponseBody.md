# ListAuditResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Events** | [**[]AuditEventView**](AuditEventView.md) |  | 

## Methods

### NewListAuditResponseBody

`func NewListAuditResponseBody(events []AuditEventView, ) *ListAuditResponseBody`

NewListAuditResponseBody instantiates a new ListAuditResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListAuditResponseBodyWithDefaults

`func NewListAuditResponseBodyWithDefaults() *ListAuditResponseBody`

NewListAuditResponseBodyWithDefaults instantiates a new ListAuditResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ListAuditResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ListAuditResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ListAuditResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ListAuditResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetEvents

`func (o *ListAuditResponseBody) GetEvents() []AuditEventView`

GetEvents returns the Events field if non-nil, zero value otherwise.

### GetEventsOk

`func (o *ListAuditResponseBody) GetEventsOk() (*[]AuditEventView, bool)`

GetEventsOk returns a tuple with the Events field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEvents

`func (o *ListAuditResponseBody) SetEvents(v []AuditEventView)`

SetEvents sets Events field to given value.


### SetEventsNil

`func (o *ListAuditResponseBody) SetEventsNil(b bool)`

 SetEventsNil sets the value for Events to be an explicit nil

### UnsetEvents
`func (o *ListAuditResponseBody) UnsetEvents()`

UnsetEvents ensures that no value is present for Events, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


