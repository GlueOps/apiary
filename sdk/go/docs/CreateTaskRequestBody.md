# CreateTaskRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**BeeId** | **string** |  | 
**Kind** | **string** | What the agent should do. An agent handed a kind it does not implement fails the task naming its version, rather than silently doing nothing. | 
**Spec** | Pointer to **interface{}** |  | [optional] 

## Methods

### NewCreateTaskRequestBody

`func NewCreateTaskRequestBody(beeId string, kind string, ) *CreateTaskRequestBody`

NewCreateTaskRequestBody instantiates a new CreateTaskRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCreateTaskRequestBodyWithDefaults

`func NewCreateTaskRequestBodyWithDefaults() *CreateTaskRequestBody`

NewCreateTaskRequestBodyWithDefaults instantiates a new CreateTaskRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *CreateTaskRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *CreateTaskRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *CreateTaskRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *CreateTaskRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetBeeId

`func (o *CreateTaskRequestBody) GetBeeId() string`

GetBeeId returns the BeeId field if non-nil, zero value otherwise.

### GetBeeIdOk

`func (o *CreateTaskRequestBody) GetBeeIdOk() (*string, bool)`

GetBeeIdOk returns a tuple with the BeeId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBeeId

`func (o *CreateTaskRequestBody) SetBeeId(v string)`

SetBeeId sets BeeId field to given value.


### GetKind

`func (o *CreateTaskRequestBody) GetKind() string`

GetKind returns the Kind field if non-nil, zero value otherwise.

### GetKindOk

`func (o *CreateTaskRequestBody) GetKindOk() (*string, bool)`

GetKindOk returns a tuple with the Kind field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKind

`func (o *CreateTaskRequestBody) SetKind(v string)`

SetKind sets Kind field to given value.


### GetSpec

`func (o *CreateTaskRequestBody) GetSpec() interface{}`

GetSpec returns the Spec field if non-nil, zero value otherwise.

### GetSpecOk

`func (o *CreateTaskRequestBody) GetSpecOk() (*interface{}, bool)`

GetSpecOk returns a tuple with the Spec field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSpec

`func (o *CreateTaskRequestBody) SetSpec(v interface{})`

SetSpec sets Spec field to given value.

### HasSpec

`func (o *CreateTaskRequestBody) HasSpec() bool`

HasSpec returns a boolean if a field has been set.

### SetSpecNil

`func (o *CreateTaskRequestBody) SetSpecNil(b bool)`

 SetSpecNil sets the value for Spec to be an explicit nil

### UnsetSpec
`func (o *CreateTaskRequestBody) UnsetSpec()`

UnsetSpec ensures that no value is present for Spec, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


