
# MintTokenRequestBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`hiveId` | string
`maxUses` | number
`ttlMinutes` | number

## Example

```typescript
import type { MintTokenRequestBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "hiveId": null,
  "maxUses": null,
  "ttlMinutes": null,
} satisfies MintTokenRequestBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as MintTokenRequestBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


