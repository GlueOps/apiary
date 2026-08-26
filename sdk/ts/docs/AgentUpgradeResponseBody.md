
# AgentUpgradeResponseBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`agents` | [Array&lt;UpgradeCandidateView&gt;](UpgradeCandidateView.md)
`queued` | number
`version` | string

## Example

```typescript
import type { AgentUpgradeResponseBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "agents": null,
  "queued": null,
  "version": null,
} satisfies AgentUpgradeResponseBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as AgentUpgradeResponseBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


