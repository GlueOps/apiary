
# DeclaredHostView


## Properties

Name | Type
------------ | -------------
`address` | string
`created` | boolean
`detail` | string
`id` | string
`queued` | boolean

## Example

```typescript
import type { DeclaredHostView } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "address": null,
  "created": null,
  "detail": null,
  "id": null,
  "queued": null,
} satisfies DeclaredHostView

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as DeclaredHostView
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


