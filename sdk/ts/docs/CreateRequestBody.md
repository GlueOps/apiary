
# CreateRequestBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`credentialKind` | string
`credentialName` | string
`identity` | string
`name` | string
`provider` | string
`scope` | { [key: string]: string; }
`secret` | string

## Example

```typescript
import type { CreateRequestBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "credentialKind": null,
  "credentialName": null,
  "identity": null,
  "name": null,
  "provider": null,
  "scope": null,
  "secret": null,
} satisfies CreateRequestBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as CreateRequestBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


