# TaskView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Attempt** | **int64** |  | 
**Id** | **string** |  | 
**Kind** | **string** |  | 
**Spec** | **interface{}** |  | 
**Status** | **string** |  | 

## Methods

### NewTaskView

`func NewTaskView(attempt int64, id string, kind string, spec interface{}, status string, ) *TaskView`

NewTaskView instantiates a new TaskView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewTaskViewWithDefaults

`func NewTaskViewWithDefaults() *TaskView`

NewTaskViewWithDefaults instantiates a new TaskView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAttempt

`func (o *TaskView) GetAttempt() int64`

GetAttempt returns the Attempt field if non-nil, zero value otherwise.

### GetAttemptOk

`func (o *TaskView) GetAttemptOk() (*int64, bool)`

GetAttemptOk returns a tuple with the Attempt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAttempt

`func (o *TaskView) SetAttempt(v int64)`

SetAttempt sets Attempt field to given value.


### GetId

`func (o *TaskView) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *TaskView) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *TaskView) SetId(v string)`

SetId sets Id field to given value.


### GetKind

`func (o *TaskView) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *TaskView) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *TaskView) SetKind(v string)`

SetKind sets Kind field to given value.


### GetSpec

`func (o *TaskView) GetSpec() interface{}`

GetSpec returns the Spec field if non-nil, zero value otherwise.

### GetSpecOk

`func (o *TaskView) GetSpecOk() (*interface{}, bool)`

GetSpecOk returns a tuple with the Spec field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSpec

`func (o *TaskView) SetSpec(v interface{})`

SetSpec sets Spec field to given value.


### SetSpecNil

`func (o *TaskView) SetSpecNil(b bool)`

 SetSpecNil sets the value for Spec to be an explicit nil

### UnsetSpec
`func (o *TaskView) UnsetSpec()`

UnsetSpec ensures that no value is present for Spec, not even an explicit nil
### GetStatus

`func (o *TaskView) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *TaskView) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *TaskView) SetStatus(v string)`

SetStatus sets Status field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


