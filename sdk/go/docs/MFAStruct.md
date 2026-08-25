# MFAStruct

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Enabled** | **bool** |  | 
**TotpEnabled** | **bool** |  | 

## Methods

### NewMFAStruct

`func NewMFAStruct(enabled bool, totpEnabled bool, ) *MFAStruct`

NewMFAStruct instantiates a new MFAStruct object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMFAStructWithDefaults

`func NewMFAStructWithDefaults() *MFAStruct`

NewMFAStructWithDefaults instantiates a new MFAStruct object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetEnabled

`func (o *MFAStruct) GetEnabled() bool`

GetEnabled returns the Enabled field if non-nil, zero value otherwise.

### GetEnabledOk

`func (o *MFAStruct) GetEnabledOk() (*bool, bool)`

GetEnabledOk returns a tuple with the Enabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnabled

`func (o *MFAStruct) SetEnabled(v bool)`

SetEnabled sets Enabled field to given value.


### GetTotpEnabled

`func (o *MFAStruct) GetTotpEnabled() bool`

GetTotpEnabled returns the TotpEnabled field if non-nil, zero value otherwise.

### GetTotpEnabledOk

`func (o *MFAStruct) GetTotpEnabledOk() (*bool, bool)`

GetTotpEnabledOk returns a tuple with the TotpEnabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTotpEnabled

`func (o *MFAStruct) SetTotpEnabled(v bool)`

SetTotpEnabled sets TotpEnabled field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


