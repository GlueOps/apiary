# EnrolRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**AgentVersion** | Pointer to **string** |  | [optional] 
**EnrolmentToken** | **string** |  | 
**Hostname** | **string** | How the control plane should reach this machine. Recorded against the MACHINE, not the agent: reinstalling an agent does not move a box. | 
**MachineId** | **string** | The host&#39;s own stable identifier, typically /etc/machine-id. Hashed before storage; re-enrolling with the same value repoints this host&#39;s existing machine record rather than creating a second one. | 

## Methods

### NewEnrolRequestBody

`func NewEnrolRequestBody(enrolmentToken string, hostname string, machineId string, ) *EnrolRequestBody`

NewEnrolRequestBody instantiates a new EnrolRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewEnrolRequestBodyWithDefaults

`func NewEnrolRequestBodyWithDefaults() *EnrolRequestBody`

NewEnrolRequestBodyWithDefaults instantiates a new EnrolRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *EnrolRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *EnrolRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *EnrolRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *EnrolRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetAgentVersion

`func (o *EnrolRequestBody) GetAgentVersion() string`

GetAgentVersion returns the AgentVersion field if non-nil, zero value otherwise.

### GetAgentVersionOk

`func (o *EnrolRequestBody) GetAgentVersionOk() (*string, bool)`

GetAgentVersionOk returns a tuple with the AgentVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgentVersion

`func (o *EnrolRequestBody) SetAgentVersion(v string)`

SetAgentVersion sets AgentVersion field to given value.

### HasAgentVersion

`func (o *EnrolRequestBody) HasAgentVersion() bool`

HasAgentVersion returns a boolean if a field has been set.

### GetEnrolmentToken

`func (o *EnrolRequestBody) GetEnrolmentToken() string`

GetEnrolmentToken returns the EnrolmentToken field if non-nil, zero value otherwise.

### GetEnrolmentTokenOk

`func (o *EnrolRequestBody) GetEnrolmentTokenOk() (*string, bool)`

GetEnrolmentTokenOk returns a tuple with the EnrolmentToken field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnrolmentToken

`func (o *EnrolRequestBody) SetEnrolmentToken(v string)`

SetEnrolmentToken sets EnrolmentToken field to given value.


### GetHostname

`func (o *EnrolRequestBody) GetHostname() string`

GetHostname returns the Hostname field if non-nil, zero value otherwise.

### GetHostnameOk

`func (o *EnrolRequestBody) GetHostnameOk() (*string, bool)`

GetHostnameOk returns a tuple with the Hostname field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHostname

`func (o *EnrolRequestBody) SetHostname(v string)`

SetHostname sets Hostname field to given value.


### GetMachineId

`func (o *EnrolRequestBody) GetMachineId() string`

GetMachineId returns the MachineId field if non-nil, zero value otherwise.

### GetMachineIdOk

`func (o *EnrolRequestBody) GetMachineIdOk() (*string, bool)`

GetMachineIdOk returns a tuple with the MachineId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMachineId

`func (o *EnrolRequestBody) SetMachineId(v string)`

SetMachineId sets MachineId field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


