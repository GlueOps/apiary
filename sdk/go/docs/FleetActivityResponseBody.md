# FleetActivityResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Activity** | [**[]FleetActivityView**](FleetActivityView.md) |  | 

## Methods

### NewFleetActivityResponseBody

`func NewFleetActivityResponseBody(activity []FleetActivityView, ) *FleetActivityResponseBody`

NewFleetActivityResponseBody instantiates a new FleetActivityResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewFleetActivityResponseBodyWithDefaults

`func NewFleetActivityResponseBodyWithDefaults() *FleetActivityResponseBody`

NewFleetActivityResponseBodyWithDefaults instantiates a new FleetActivityResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *FleetActivityResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *FleetActivityResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *FleetActivityResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *FleetActivityResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetActivity

`func (o *FleetActivityResponseBody) GetActivity() []FleetActivityView`

GetActivity returns the Activity field if non-nil, zero value otherwise.

### GetActivityOk

`func (o *FleetActivityResponseBody) GetActivityOk() (*[]FleetActivityView, bool)`

GetActivityOk returns a tuple with the Activity field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetActivity

`func (o *FleetActivityResponseBody) SetActivity(v []FleetActivityView)`

SetActivity sets Activity field to given value.


### SetActivityNil

`func (o *FleetActivityResponseBody) SetActivityNil(b bool)`

 SetActivityNil sets the value for Activity to be an explicit nil

### UnsetActivity
`func (o *FleetActivityResponseBody) UnsetActivity()`

UnsetActivity ensures that no value is present for Activity, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


