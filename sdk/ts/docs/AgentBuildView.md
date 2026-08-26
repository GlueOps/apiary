
# AgentBuildView


## Properties

Name | Type
------------ | -------------
`arch` | string
`os` | string
`sha256` | string
`size` | number

## Example

```typescript
import type { AgentBuildView } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "arch": null,
  "os": null,
  "sha256": null,
  "size": null,
} satisfies AgentBuildView

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as AgentBuildView
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


