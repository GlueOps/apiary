# OrganizationSummaryView

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CreatedAt** | **time.Time** |  | 
**Deleted** | **bool** |  | 
**Domain** | Pointer to **string** |  | [optional] 
**Id** | **string** |  | 
**Name** | **string** |  | 
**Status** | **string** |  | 

## Methods

### NewOrganizationSummaryView

`func NewOrganizationSummaryView(createdAt time.Time, deleted bool, id string, name string, status string, ) *OrganizationSummaryView`

NewOrganizationSummaryView instantiates a new OrganizationSummaryView object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewOrganizationSummaryViewWithDefaults

`func NewOrganizationSummaryViewWithDefaults() *OrganizationSummaryView`

NewOrganizationSummaryViewWithDefaults instantiates a new OrganizationSummaryView object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCreatedAt

`func (o *OrganizationSummaryView) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *OrganizationSummaryView) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *OrganizationSummaryView) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.


### GetDeleted

`func (o *OrganizationSummaryView) GetDeleted() bool`

GetDeleted returns the Deleted field if non-nil, zero value otherwise.

### GetDeletedOk

`func (o *OrganizationSummaryView) GetDeletedOk() (*bool, bool)`

GetDeletedOk returns a tuple with the Deleted field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeleted

`func (o *OrganizationSummaryView) SetDeleted(v bool)`

SetDeleted sets Deleted field to given value.


### GetDomain

`func (o *OrganizationSummaryView) GetDomain() string`

GetDomain returns the Domain field if non-nil, zero value otherwise.

### GetDomainOk

`func (o *OrganizationSummaryView) GetDomainOk() (*string, bool)`

GetDomainOk returns a tuple with the Domain field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomain

`func (o *OrganizationSummaryView) SetDomain(v string)`

SetDomain sets Domain field to given value.

### HasDomain

`func (o *OrganizationSummaryView) HasDomain() bool`

HasDomain returns a boolean if a field has been set.

### GetId

`func (o *OrganizationSummaryView) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *OrganizationSummaryView) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *OrganizationSummaryView) SetId(v string)`

SetId sets Id field to given value.


### GetName

`func (o *OrganizationSummaryView) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *OrganizationSummaryView) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *OrganizationSummaryView) SetName(v string)`

SetName sets Name field to given value.


### GetStatus

`func (o *OrganizationSummaryView) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *OrganizationSummaryView) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *OrganizationSummaryView) SetStatus(v string)`

SetStatus sets Status field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


