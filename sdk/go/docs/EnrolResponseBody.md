# EnrolResponseBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**BeeId** | **string** |  | 
**Key** | **string** | The public half of this host&#39;s agent credential. Presented as the HTTP Basic username. | 
**OrganizationId** | **string** | The organization this host was admitted into. It joins a fleet afterwards, as a task — the same path a host migrating between fleets takes. | 
**Secret** | **string** | Returned ONCE. Presented as the HTTP Basic password. There is no endpoint that reissues it, because such an endpoint would let anyone holding a bee id take the bee over. A host that loses it enrols again. | 

## Methods

### NewEnrolResponseBody

`func NewEnrolResponseBody(beeId string, key string, organizationId string, secret string, ) *EnrolResponseBody`

NewEnrolResponseBody instantiates a new EnrolResponseBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewEnrolResponseBodyWithDefaults

`func NewEnrolResponseBodyWithDefaults() *EnrolResponseBody`

NewEnrolResponseBodyWithDefaults instantiates a new EnrolResponseBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *EnrolResponseBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *EnrolResponseBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *EnrolResponseBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *EnrolResponseBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetBeeId

`func (o *EnrolResponseBody) GetBeeId() string`

GetBeeId returns the BeeId field if non-nil, zero value otherwise.

### GetBeeIdOk

`func (o *EnrolResponseBody) GetBeeIdOk() (*string, bool)`

GetBeeIdOk returns a tuple with the BeeId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBeeId

`func (o *EnrolResponseBody) SetBeeId(v string)`

SetBeeId sets BeeId field to given value.


### GetKey

`func (o *EnrolResponseBody) GetKey() string`

GetKey returns the Key field if non-nil, zero value otherwise.

### GetKeyOk

`func (o *EnrolResponseBody) GetKeyOk() (*string, bool)`

GetKeyOk returns a tuple with the Key field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKey

`func (o *EnrolResponseBody) SetKey(v string)`

SetKey sets Key field to given value.


### GetOrganizationId

`func (o *EnrolResponseBody) GetOrganizationId() string`

GetOrganizationId returns the OrganizationId field if non-nil, zero value otherwise.

### GetOrganizationIdOk

`func (o *EnrolResponseBody) GetOrganizationIdOk() (*string, bool)`

GetOrganizationIdOk returns a tuple with the OrganizationId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrganizationId

`func (o *EnrolResponseBody) SetOrganizationId(v string)`

SetOrganizationId sets OrganizationId field to given value.


### GetSecret

`func (o *EnrolResponseBody) GetSecret() string`

GetSecret returns the Secret field if non-nil, zero value otherwise.

### GetSecretOk

`func (o *EnrolResponseBody) GetSecretOk() (*string, bool)`

GetSecretOk returns a tuple with the Secret field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecret

`func (o *EnrolResponseBody) SetSecret(v string)`

SetSecret sets Secret field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


