
# MemberView


## Properties

Name | Type
------------ | -------------
`email` | string
`firstName` | string
`lastName` | string
`membershipId` | string
`role` | string
`supportEnabled` | boolean
`userId` | string

## Example

```typescript
import type { MemberView } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "email": null,
  "firstName": null,
  "lastName": null,
  "membershipId": null,
  "role": null,
  "supportEnabled": null,
  "userId": null,
} satisfies MemberView

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as MemberView
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


