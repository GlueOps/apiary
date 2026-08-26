
# AgentDistResponseBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`builds` | [Array&lt;AgentBuildView&gt;](AgentBuildView.md)
`version` | string

## Example

```typescript
import type { AgentDistResponseBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "builds": null,
  "version": null,
} satisfies AgentDistResponseBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as AgentDistResponseBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


