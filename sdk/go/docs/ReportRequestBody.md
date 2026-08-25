# ReportRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Error** | Pointer to **string** |  | [optional] 
**Phase** | Pointer to **string** |  | [optional] 
**Status** | **string** |  | 

## Methods

### NewReportRequestBody

`func NewReportRequestBody(status string, ) *ReportRequestBody`

NewReportRequestBody instantiates a new ReportRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewReportRequestBodyWithDefaults

`func NewReportRequestBodyWithDefaults() *ReportRequestBody`

NewReportRequestBodyWithDefaults instantiates a new ReportRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ReportRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ReportRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ReportRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ReportRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetError

`func (o *ReportRequestBody) GetError() string`

GetError returns the Error field if non-nil, zero value otherwise.

### GetErrorOk

`func (o *ReportRequestBody) GetErrorOk() (*string, bool)`

GetErrorOk returns a tuple with the Error field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetError

`func (o *ReportRequestBody) SetError(v string)`

SetError sets Error field to given value.

### HasError

`func (o *ReportRequestBody) HasError() bool`

HasError returns a boolean if a field has been set.

### GetPhase

`func (o *ReportRequestBody) GetPhase() string`

GetPhase returns the Phase field if non-nil, zero value otherwise.

### GetPhaseOk

`func (o *ReportRequestBody) GetPhaseOk() (*string, bool)`

GetPhaseOk returns a tuple with the Phase field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPhase

`func (o *ReportRequestBody) SetPhase(v string)`

SetPhase sets Phase field to given value.

### HasPhase

`func (o *ReportRequestBody) HasPhase() bool`

HasPhase returns a boolean if a field has been set.

### GetStatus

`func (o *ReportRequestBody) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *ReportRequestBody) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *ReportRequestBody) SetStatus(v string)`

SetStatus sets Status field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


