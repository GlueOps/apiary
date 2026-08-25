
# AcceptRequestBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`firstName` | string
`lastName` | string
`password` | string
`token` | string

## Example

```typescript
import type { AcceptRequestBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "firstName": null,
  "lastName": null,
  "password": null,
  "token": null,
} satisfies AcceptRequestBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as AcceptRequestBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


