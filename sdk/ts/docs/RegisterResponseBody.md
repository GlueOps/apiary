
# RegisterResponseBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`message` | string
`organization` | string
`organizationId` | string
`verificationSent` | boolean

## Example

```typescript
import type { RegisterResponseBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "message": null,
  "organization": null,
  "organizationId": null,
  "verificationSent": null,
} satisfies RegisterResponseBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as RegisterResponseBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


