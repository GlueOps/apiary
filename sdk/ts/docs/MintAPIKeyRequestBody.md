
# MintAPIKeyRequestBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`name` | string
`scope` | string
`ttlHours` | number

## Example

```typescript
import type { MintAPIKeyRequestBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "name": null,
  "scope": null,
  "ttlHours": null,
} satisfies MintAPIKeyRequestBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as MintAPIKeyRequestBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


