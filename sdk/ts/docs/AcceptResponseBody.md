
# AcceptResponseBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`accepted` | boolean
`accountCreated` | boolean
`email` | string
`message` | string
`organization` | string

## Example

```typescript
import type { AcceptResponseBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "accepted": null,
  "accountCreated": null,
  "email": null,
  "message": null,
  "organization": null,
} satisfies AcceptResponseBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as AcceptResponseBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


