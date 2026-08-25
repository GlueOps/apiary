
# EstateView


## Properties

Name | Type
------------ | -------------
`$schema` | string
`credentialId` | string
`credentialKind` | string
`credentialName` | string
`fingerprint` | string
`id` | string
`name` | string
`provider` | string
`scope` | { [key: string]: string; }
`scopeVersion` | number

## Example

```typescript
import type { EstateView } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "credentialId": null,
  "credentialKind": null,
  "credentialName": null,
  "fingerprint": null,
  "id": null,
  "name": null,
  "provider": null,
  "scope": null,
  "scopeVersion": null,
} satisfies EstateView

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as EstateView
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


