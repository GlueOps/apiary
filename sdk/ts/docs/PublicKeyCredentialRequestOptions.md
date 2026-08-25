
# PublicKeyCredentialRequestOptions


## Properties

Name | Type
------------ | -------------
`allowCredentials` | [Array&lt;CredentialDescriptor&gt;](CredentialDescriptor.md)
`challenge` | string
`extensions` | { [key: string]: any | null; }
`hints` | Array&lt;string&gt;
`rpId` | string
`timeout` | number
`userVerification` | string

## Example

```typescript
import type { PublicKeyCredentialRequestOptions } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "allowCredentials": null,
  "challenge": null,
  "extensions": null,
  "hints": null,
  "rpId": null,
  "timeout": null,
  "userVerification": null,
} satisfies PublicKeyCredentialRequestOptions

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as PublicKeyCredentialRequestOptions
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


