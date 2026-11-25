# AcknowledgeMaintenanceResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**AcknowledgedAt** | **time.Time** |  | 
**Cleared** | **int64** | How many strikes the acknowledgement cleared. | 
**HypervisorId** | **string** |  | 

## Methods

### NewAcknowledgeMaintenanceResponseBody

`func NewAcknowledgeMaintenanceResponseBody(acknowledgedAt time.Time, cleared int64, hypervisorId string, ) *AcknowledgeMaintenanceResponseBody`

NewAcknowledgeMaintenanceResponseBody instantiates a new AcknowledgeMaintenanceResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAcknowledgeMaintenanceResponseBodyWithDefaults

`func NewAcknowledgeMaintenanceResponseBodyWithDefaults() *AcknowledgeMaintenanceResponseBody`

NewAcknowledgeMaintenanceResponseBodyWithDefaults instantiates a new AcknowledgeMaintenanceResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *AcknowledgeMaintenanceResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *AcknowledgeMaintenanceResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *AcknowledgeMaintenanceResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *AcknowledgeMaintenanceResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetAcknowledgedAt

`func (o *AcknowledgeMaintenanceResponseBody) GetAcknowledgedAt() time.Time`

GetAcknowledgedAt returns the AcknowledgedAt field if non-nil, zero value otherwise.

### GetAcknowledgedAtOk

`func (o *AcknowledgeMaintenanceResponseBody) GetAcknowledgedAtOk() (*time.Time, bool)`

GetAcknowledgedAtOk returns a tuple with the AcknowledgedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAcknowledgedAt

`func (o *AcknowledgeMaintenanceResponseBody) SetAcknowledgedAt(v time.Time)`

SetAcknowledgedAt sets AcknowledgedAt field to given value.


### GetCleared

`func (o *AcknowledgeMaintenanceResponseBody) GetCleared() int64`

GetCleared returns the Cleared field if non-nil, zero value otherwise.

### GetClearedOk

`func (o *AcknowledgeMaintenanceResponseBody) GetClearedOk() (*int64, bool)`

GetClearedOk returns a tuple with the Cleared field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCleared

`func (o *AcknowledgeMaintenanceResponseBody) SetCleared(v int64)`

SetCleared sets Cleared field to given value.


### GetHypervisorId

`func (o *AcknowledgeMaintenanceResponseBody) GetHypervisorId() string`

GetHypervisorId returns the HypervisorId field if non-nil, zero value otherwise.

### GetHypervisorIdOk

`func (o *AcknowledgeMaintenanceResponseBody) GetHypervisorIdOk() (*string, bool)`

GetHypervisorIdOk returns a tuple with the HypervisorId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHypervisorId

`func (o *AcknowledgeMaintenanceResponseBody) SetHypervisorId(v string)`

SetHypervisorId sets HypervisorId field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


