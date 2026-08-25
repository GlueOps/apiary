
# PublishResponseBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`bytes` | number
`digest` | string
`revision` | number
`unchanged` | boolean

## Example

```typescript
import type { PublishResponseBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "bytes": null,
  "digest": null,
  "revision": null,
  "unchanged": null,
} satisfies PublishResponseBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as PublishResponseBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


