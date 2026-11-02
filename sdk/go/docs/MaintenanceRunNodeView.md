# MaintenanceRunNodeView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Deadline** | Pointer to **time.Time** |  | [optional] 
**FinishedAt** | Pointer to **time.Time** |  | [optional] 
**HypervisorId** | **string** |  | 
**LeftCordoned** | **bool** | The run cordoned this node and has not handed the cordon back: it failed, or somebody cordoned it by hand meanwhile. | 
**Name** | **string** |  | 
**Reason** | Pointer to **string** | Why it was skipped or failed, or why it was left cordoned. | [optional] 
**StartedAt** | Pointer to **time.Time** |  | [optional] 
**Status** | **string** |  | 
**TaskId** | Pointer to **string** |  | [optional] 

## Methods

### NewMaintenanceRunNodeView

`func NewMaintenanceRunNodeView(hypervisorId string, leftCordoned bool, name string, status string, ) *MaintenanceRunNodeView`

NewMaintenanceRunNodeView instantiates a new MaintenanceRunNodeView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMaintenanceRunNodeViewWithDefaults

`func NewMaintenanceRunNodeViewWithDefaults() *MaintenanceRunNodeView`

NewMaintenanceRunNodeViewWithDefaults instantiates a new MaintenanceRunNodeView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDeadline

`func (o *MaintenanceRunNodeView) GetDeadline() time.Time`

GetDeadline returns the Deadline field if non-nil, zero value otherwise.

### GetDeadlineOk

`func (o *MaintenanceRunNodeView) GetDeadlineOk() (*time.Time, bool)`

GetDeadlineOk returns a tuple with the Deadline field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeadline

`func (o *MaintenanceRunNodeView) SetDeadline(v time.Time)`

SetDeadline sets Deadline field to given value.

### HasDeadline

`func (o *MaintenanceRunNodeView) HasDeadline() bool`

HasDeadline returns a boolean if a field has been set.

### GetFinishedAt

`func (o *MaintenanceRunNodeView) GetFinishedAt() time.Time`

GetFinishedAt returns the FinishedAt field if non-nil, zero value otherwise.

### GetFinishedAtOk

`func (o *MaintenanceRunNodeView) GetFinishedAtOk() (*time.Time, bool)`

GetFinishedAtOk returns a tuple with the FinishedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFinishedAt

`func (o *MaintenanceRunNodeView) SetFinishedAt(v time.Time)`

SetFinishedAt sets FinishedAt field to given value.

### HasFinishedAt

`func (o *MaintenanceRunNodeView) HasFinishedAt() bool`

HasFinishedAt returns a boolean if a field has been set.

### GetHypervisorId

`func (o *MaintenanceRunNodeView) GetHypervisorId() string`

GetHypervisorId returns the HypervisorId field if non-nil, zero value otherwise.

### GetHypervisorIdOk

`func (o *MaintenanceRunNodeView) GetHypervisorIdOk() (*string, bool)`

GetHypervisorIdOk returns a tuple with the HypervisorId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHypervisorId

`func (o *MaintenanceRunNodeView) SetHypervisorId(v string)`

SetHypervisorId sets HypervisorId field to given value.


### GetLeftCordoned

`func (o *MaintenanceRunNodeView) GetLeftCordoned() bool`

GetLeftCordoned returns the LeftCordoned field if non-nil, zero value otherwise.

### GetLeftCordonedOk

`func (o *MaintenanceRunNodeView) GetLeftCordonedOk() (*bool, bool)`

GetLeftCordonedOk returns a tuple with the LeftCordoned field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLeftCordoned

`func (o *MaintenanceRunNodeView) SetLeftCordoned(v bool)`

SetLeftCordoned sets LeftCordoned field to given value.


### GetName

`func (o *MaintenanceRunNodeView) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *MaintenanceRunNodeView) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *MaintenanceRunNodeView) SetName(v string)`

SetName sets Name field to given value.


### GetReason

`func (o *MaintenanceRunNodeView) GetReason() string`

GetReason returns the Reason field if non-nil, zero value otherwise.

### GetReasonOk

`func (o *MaintenanceRunNodeView) GetReasonOk() (*string, bool)`

GetReasonOk returns a tuple with the Reason field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReason

`func (o *MaintenanceRunNodeView) SetReason(v string)`

SetReason sets Reason field to given value.

### HasReason

`func (o *MaintenanceRunNodeView) HasReason() bool`

HasReason returns a boolean if a field has been set.

### GetStartedAt

`func (o *MaintenanceRunNodeView) GetStartedAt() time.Time`

GetStartedAt returns the StartedAt field if non-nil, zero value otherwise.

### GetStartedAtOk

`func (o *MaintenanceRunNodeView) GetStartedAtOk() (*time.Time, bool)`

GetStartedAtOk returns a tuple with the StartedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStartedAt

`func (o *MaintenanceRunNodeView) SetStartedAt(v time.Time)`

SetStartedAt sets StartedAt field to given value.

### HasStartedAt

`func (o *MaintenanceRunNodeView) HasStartedAt() bool`

HasStartedAt returns a boolean if a field has been set.

### GetStatus

`func (o *MaintenanceRunNodeView) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *MaintenanceRunNodeView) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *MaintenanceRunNodeView) SetStatus(v string)`

SetStatus sets Status field to given value.


### GetTaskId

`func (o *MaintenanceRunNodeView) GetTaskId() string`

GetTaskId returns the TaskId field if non-nil, zero value otherwise.

### GetTaskIdOk

`func (o *MaintenanceRunNodeView) GetTaskIdOk() (*string, bool)`

GetTaskIdOk returns a tuple with the TaskId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTaskId

`func (o *MaintenanceRunNodeView) SetTaskId(v string)`

SetTaskId sets TaskId field to given value.

### HasTaskId

`func (o *MaintenanceRunNodeView) HasTaskId() bool`

HasTaskId returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


