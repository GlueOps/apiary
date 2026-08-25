# ImpersonateEndResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**Ended** | **bool** |  | 

## Methods

### NewImpersonateEndResponseBody

`func NewImpersonateEndResponseBody(ended bool, ) *ImpersonateEndResponseBody`

NewImpersonateEndResponseBody instantiates a new ImpersonateEndResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewImpersonateEndResponseBodyWithDefaults

`func NewImpersonateEndResponseBodyWithDefaults() *ImpersonateEndResponseBody`

NewImpersonateEndResponseBodyWithDefaults instantiates a new ImpersonateEndResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *ImpersonateEndResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ImpersonateEndResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ImpersonateEndResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ImpersonateEndResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetEnded

`func (o *ImpersonateEndResponseBody) GetEnded() bool`

GetEnded returns the Ended field if non-nil, zero value otherwise.

### GetEndedOk

`func (o *ImpersonateEndResponseBody) GetEndedOk() (*bool, bool)`

GetEndedOk returns a tuple with the Ended field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnded

`func (o *ImpersonateEndResponseBody) SetEnded(v bool)`

SetEnded sets Ended field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


