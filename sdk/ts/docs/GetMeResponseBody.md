
# GetMeResponseBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`createdAt` | Date
`email` | string
`firstName` | string
`id` | string
`isActive` | boolean
`lastLoginAt` | Date
`lastName` | string
`mfa` | [MFAStruct](MFAStruct.md)
`organization` | [OrganizationStruct](OrganizationStruct.md)
`role` | string

## Example

```typescript
import type { GetMeResponseBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "createdAt": null,
  "email": null,
  "firstName": null,
  "id": null,
  "isActive": null,
  "lastLoginAt": null,
  "lastName": null,
  "mfa": null,
  "organization": null,
  "role": null,
} satisfies GetMeResponseBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as GetMeResponseBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


