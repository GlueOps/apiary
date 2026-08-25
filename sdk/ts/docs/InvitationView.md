
# InvitationView


## Properties

Name | Type
------------ | -------------
`$schema` | string
`createdAt` | Date
`email` | string
`expiresAt` | Date
`id` | string
`role` | string
`status` | string

## Example

```typescript
import type { InvitationView } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "createdAt": null,
  "email": null,
  "expiresAt": null,
  "id": null,
  "role": null,
  "status": null,
} satisfies InvitationView

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as InvitationView
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


