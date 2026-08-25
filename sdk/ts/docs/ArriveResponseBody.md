
# ArriveResponseBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`detail` | string
`queued` | boolean

## Example

```typescript
import type { ArriveResponseBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "detail": null,
  "queued": null,
} satisfies ArriveResponseBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as ArriveResponseBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


