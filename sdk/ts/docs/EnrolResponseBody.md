
# EnrolResponseBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`beeId` | string
`key` | string
`organizationId` | string
`secret` | string

## Example

```typescript
import type { EnrolResponseBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "beeId": null,
  "key": null,
  "organizationId": null,
  "secret": null,
} satisfies EnrolResponseBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as EnrolResponseBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


