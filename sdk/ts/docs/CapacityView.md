
# CapacityView


## Properties

Name | Type
------------ | -------------
`cpu` | number
`diskGb` | number
`ramGb` | number

## Example

```typescript
import type { CapacityView } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "cpu": null,
  "diskGb": null,
  "ramGb": null,
} satisfies CapacityView

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as CapacityView
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


