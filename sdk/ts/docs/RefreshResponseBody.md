
# RefreshResponseBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`accessToken` | string
`expiresIn` | number
`organization` | string
`organizationId` | string
`refreshExpiresIn` | number
`refreshToken` | string
`tokenType` | string

## Example

```typescript
import type { RefreshResponseBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "accessToken": null,
  "expiresIn": null,
  "organization": null,
  "organizationId": null,
  "refreshExpiresIn": null,
  "refreshToken": null,
  "tokenType": null,
} satisfies RefreshResponseBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as RefreshResponseBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


