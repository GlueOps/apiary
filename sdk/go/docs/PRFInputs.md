# PRFInputs

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Eval** | Pointer to [**PRFValues**](PRFValues.md) |  | [optional] 
**EvalByCredential** | Pointer to [**map[string]PRFValues**](PRFValues.md) |  | [optional] 

## Methods

### NewPRFInputs

`func NewPRFInputs() *PRFInputs`

NewPRFInputs instantiates a new PRFInputs object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPRFInputsWithDefaults

`func NewPRFInputsWithDefaults() *PRFInputs`

NewPRFInputsWithDefaults instantiates a new PRFInputs object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetEval

`func (o *PRFInputs) GetEval() PRFValues`

GetEval returns the Eval field if non-nil, zero value otherwise.

### GetEvalOk

`func (o *PRFInputs) GetEvalOk() (*PRFValues, bool)`

GetEvalOk returns a tuple with the Eval field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEval

`func (o *PRFInputs) SetEval(v PRFValues)`

SetEval sets Eval field to given value.

### HasEval

`func (o *PRFInputs) HasEval() bool`

HasEval returns a boolean if a field has been set.

### GetEvalByCredential

`func (o *PRFInputs) GetEvalByCredential() map[string]PRFValues`

GetEvalByCredential returns the EvalByCredential field if non-nil, zero value otherwise.

### GetEvalByCredentialOk

`func (o *PRFInputs) GetEvalByCredentialOk() (*map[string]PRFValues, bool)`

GetEvalByCredentialOk returns a tuple with the EvalByCredential field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEvalByCredential

`func (o *PRFInputs) SetEvalByCredential(v map[string]PRFValues)`

SetEvalByCredential sets EvalByCredential field to given value.

### HasEvalByCredential

`func (o *PRFInputs) HasEvalByCredential() bool`

HasEvalByCredential returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


