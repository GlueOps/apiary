
# ImpersonateStartRequestBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`organizationId` | string
`reason` | string
`userId` | string

## Example

```typescript
import type { ImpersonateStartRequestBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "organizationId": null,
  "reason": null,
  "userId": null,
} satisfies ImpersonateStartRequestBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as ImpersonateStartRequestBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


