
# UpdateMeResponseBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`email` | string
`firstName` | string
`id` | string
`isActive` | boolean
`lastName` | string
`role` | string

## Example

```typescript
import type { UpdateMeResponseBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "email": null,
  "firstName": null,
  "id": null,
  "isActive": null,
  "lastName": null,
  "role": null,
} satisfies UpdateMeResponseBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as UpdateMeResponseBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


