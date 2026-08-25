
# PasskeyBeginRegisterResponseBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`challenge` | string
`options` | [CredentialCreation](CredentialCreation.md)

## Example

```typescript
import type { PasskeyBeginRegisterResponseBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "challenge": null,
  "options": null,
} satisfies PasskeyBeginRegisterResponseBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as PasskeyBeginRegisterResponseBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


