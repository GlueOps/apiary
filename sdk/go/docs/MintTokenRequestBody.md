# MintTokenRequestBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Schema** | Pointer to **string** | A URL to the JSON Schema for this object. | [optional] [readonly] 
**HiveId** | Pointer to **string** | Ignored, and removed in a future release. A token admits a host into the ORGANIZATION; it never names a cluster. | [optional] 
**MaxUses** | Pointer to **int64** | 0 means unlimited within the TTL. A rack is enrolled in a batch, so single-use tokens become a reason to script around the control. | [optional] 
**TtlMinutes** | Pointer to **int64** |  | [optional] [default to 60]

## Methods

### NewMintTokenRequestBody

`func NewMintTokenRequestBody() *MintTokenRequestBody`

NewMintTokenRequestBody instantiates a new MintTokenRequestBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMintTokenRequestBodyWithDefaults

`func NewMintTokenRequestBodyWithDefaults() *MintTokenRequestBody`

NewMintTokenRequestBodyWithDefaults instantiates a new MintTokenRequestBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSchema

`func (o *MintTokenRequestBody) GetSchema() string`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *MintTokenRequestBody) GetSchemaOk() (*string, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *MintTokenRequestBody) SetSchema(v string)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *MintTokenRequestBody) HasSchema() bool`

HasSchema returns a boolean if a field has been set.

### GetHiveId

`func (o *MintTokenRequestBody) GetHiveId() string`

GetHiveId returns the HiveId field if non-nil, zero value otherwise.

### GetHiveIdOk

`func (o *MintTokenRequestBody) GetHiveIdOk() (*string, bool)`

GetHiveIdOk returns a tuple with the HiveId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHiveId

`func (o *MintTokenRequestBody) SetHiveId(v string)`

SetHiveId sets HiveId field to given value.

### HasHiveId

`func (o *MintTokenRequestBody) HasHiveId() bool`

HasHiveId returns a boolean if a field has been set.

### GetMaxUses

`func (o *MintTokenRequestBody) GetMaxUses() int64`

GetMaxUses returns the MaxUses field if non-nil, zero value otherwise.

### GetMaxUsesOk

`func (o *MintTokenRequestBody) GetMaxUsesOk() (*int64, bool)`

GetMaxUsesOk returns a tuple with the MaxUses field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMaxUses

`func (o *MintTokenRequestBody) SetMaxUses(v int64)`

SetMaxUses sets MaxUses field to given value.

### HasMaxUses

`func (o *MintTokenRequestBody) HasMaxUses() bool`

HasMaxUses returns a boolean if a field has been set.

### GetTtlMinutes

`func (o *MintTokenRequestBody) GetTtlMinutes() int64`

GetTtlMinutes returns the TtlMinutes field if non-nil, zero value otherwise.

### GetTtlMinutesOk

`func (o *MintTokenRequestBody) GetTtlMinutesOk() (*int64, bool)`

GetTtlMinutesOk returns a tuple with the TtlMinutes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTtlMinutes

`func (o *MintTokenRequestBody) SetTtlMinutes(v int64)`

SetTtlMinutes sets TtlMinutes field to given value.

### HasTtlMinutes

`func (o *MintTokenRequestBody) HasTtlMinutes() bool`

HasTtlMinutes returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


