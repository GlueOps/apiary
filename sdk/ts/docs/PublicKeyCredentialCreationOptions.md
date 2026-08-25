
# PublicKeyCredentialCreationOptions


## Properties

Name | Type
------------ | -------------
`attestation` | string
`attestationFormats` | Array&lt;string&gt;
`authenticatorSelection` | [AuthenticatorSelection](AuthenticatorSelection.md)
`challenge` | string
`excludeCredentials` | [Array&lt;CredentialDescriptor&gt;](CredentialDescriptor.md)
`extensions` | { [key: string]: any | null; }
`hints` | Array&lt;string&gt;
`pubKeyCredParams` | [Array&lt;CredentialParameter&gt;](CredentialParameter.md)
`rp` | [RelyingPartyEntity](RelyingPartyEntity.md)
`timeout` | number
`user` | [UserEntity](UserEntity.md)

## Example

```typescript
import type { PublicKeyCredentialCreationOptions } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "attestation": null,
  "attestationFormats": null,
  "authenticatorSelection": null,
  "challenge": null,
  "excludeCredentials": null,
  "extensions": null,
  "hints": null,
  "pubKeyCredParams": null,
  "rp": null,
  "timeout": null,
  "user": null,
} satisfies PublicKeyCredentialCreationOptions

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as PublicKeyCredentialCreationOptions
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


