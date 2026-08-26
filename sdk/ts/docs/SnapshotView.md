
# SnapshotView


## Properties

Name | Type
------------ | -------------
`bytes` | number
`createdAt` | Date
`current` | boolean
`digest` | string
`reason` | string
`revertOf` | number
`revision` | number

## Example

```typescript
import type { SnapshotView } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "bytes": null,
  "createdAt": null,
  "current": null,
  "digest": null,
  "reason": null,
  "revertOf": null,
  "revision": null,
} satisfies SnapshotView

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as SnapshotView
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


