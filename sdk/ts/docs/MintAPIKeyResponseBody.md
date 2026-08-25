
# MintAPIKeyResponseBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`credential` | [APIKeyView](APIKeyView.md)
`key` | string
`secret` | string

## Example

```typescript
import type { MintAPIKeyResponseBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "credential": null,
  "key": null,
  "secret": null,
} satisfies MintAPIKeyResponseBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as MintAPIKeyResponseBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


