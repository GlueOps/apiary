
# TaskView


## Properties

Name | Type
------------ | -------------
`attempt` | number
`id` | string
`kind` | string
`spec` | any
`status` | string

## Example

```typescript
import type { TaskView } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "attempt": null,
  "id": null,
  "kind": null,
  "spec": null,
  "status": null,
} satisfies TaskView

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as TaskView
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


