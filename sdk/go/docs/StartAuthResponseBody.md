# StartAuthResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**AuthUrl** | **string** |  | 

## Methods

### NewStartAuthResponseBody

`func NewStartAuthResponseBody(authUrl string, ) *StartAuthResponseBody`

NewStartAuthResponseBody instantiates a new StartAuthResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewStartAuthResponseBodyWithDefaults

`func NewStartAuthResponseBodyWithDefaults() *StartAuthResponseBody`

NewStartAuthResponseBodyWithDefaults instantiates a new StartAuthResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *StartAuthResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *StartAuthResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *StartAuthResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *StartAuthResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetAuthUrl

`func (o *StartAuthResponseBody) GetAuthUrl() string`

GetAuthUrl returns the AuthUrl field if non-nil, zero value otherwise.

### GetAuthUrlOk

`func (o *StartAuthResponseBody) GetAuthUrlOk() (*string, bool)`

GetAuthUrlOk returns a tuple with the AuthUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAuthUrl

`func (o *StartAuthResponseBody) SetAuthUrl(v string)`

SetAuthUrl sets AuthUrl field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


