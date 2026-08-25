
# APIKeyView


## Properties

Name | Type
------------ | -------------
`createdAt` | Date
`expiresAt` | Date
`id` | string
`lastUsedAt` | Date
`name` | string
`prefix` | string
`revoked` | boolean
`scope` | string

## Example

```typescript
import type { APIKeyView } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "createdAt": null,
  "expiresAt": null,
  "id": null,
  "lastUsedAt": null,
  "name": null,
  "prefix": null,
  "revoked": null,
  "scope": null,
} satisfies APIKeyView

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as APIKeyView
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


