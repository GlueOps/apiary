
# VerifyEmailResponseBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`alreadyVerified` | boolean
`message` | string
`tenantProvisioning` | boolean
`verified` | boolean

## Example

```typescript
import type { VerifyEmailResponseBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "alreadyVerified": null,
  "message": null,
  "tenantProvisioning": null,
  "verified": null,
} satisfies VerifyEmailResponseBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as VerifyEmailResponseBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


