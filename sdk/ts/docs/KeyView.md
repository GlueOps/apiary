
# KeyView


## Properties

Name | Type
------------ | -------------
`$schema` | string
`createdAt` | Date
`expiresAt` | Date
`fingerprint` | string
`id` | string
`name` | string
`publicKey` | string
`purpose` | string
`revoked` | boolean
`usable` | boolean

## Example

```typescript
import type { KeyView } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "createdAt": null,
  "expiresAt": null,
  "fingerprint": null,
  "id": null,
  "name": null,
  "publicKey": null,
  "purpose": null,
  "revoked": null,
  "usable": null,
} satisfies KeyView

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as KeyView
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


