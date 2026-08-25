# HeartbeatResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Firewall** | Pointer to [**FirewallDesiredView**](FirewallDesiredView.md) |  | [optional] 
**Task** | Pointer to [**TaskView**](TaskView.md) | This bee&#39;s one outstanding task, or absent. A bee that already holds a task is handed the SAME one back. | [optional] 

## Methods

### NewHeartbeatResponseBody

`func NewHeartbeatResponseBody() *HeartbeatResponseBody`

NewHeartbeatResponseBody instantiates a new HeartbeatResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewHeartbeatResponseBodyWithDefaults

`func NewHeartbeatResponseBodyWithDefaults() *HeartbeatResponseBody`

NewHeartbeatResponseBodyWithDefaults instantiates a new HeartbeatResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *HeartbeatResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *HeartbeatResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *HeartbeatResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *HeartbeatResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetFirewall

`func (o *HeartbeatResponseBody) GetFirewall() FirewallDesiredView`

GetFirewall returns the Firewall field if non-nil, zero value otherwise.

### GetFirewallOk

`func (o *HeartbeatResponseBody) GetFirewallOk() (*FirewallDesiredView, bool)`

GetFirewallOk returns a tuple with the Firewall field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFirewall

`func (o *HeartbeatResponseBody) SetFirewall(v FirewallDesiredView)`

SetFirewall sets Firewall field to given value.

### HasFirewall

`func (o *HeartbeatResponseBody) HasFirewall() bool`

HasFirewall returns a boolean if a field has been set.

### GetTask

`func (o *HeartbeatResponseBody) GetTask() TaskView`

GetTask returns the Task field if non-nil, zero value otherwise.

### GetTaskOk

`func (o *HeartbeatResponseBody) GetTaskOk() (*TaskView, bool)`

GetTaskOk returns a tuple with the Task field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTask

`func (o *HeartbeatResponseBody) SetTask(v TaskView)`

SetTask sets Task field to given value.

### HasTask

`func (o *HeartbeatResponseBody) HasTask() bool`

HasTask returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


