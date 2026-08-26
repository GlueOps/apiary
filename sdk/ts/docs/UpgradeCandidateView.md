
# UpgradeCandidateView


## Properties

Name | Type
------------ | -------------
`agentVersion` | string
`beeId` | string
`queued` | boolean
`skipped` | string

## Example

```typescript
import type { UpgradeCandidateView } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "agentVersion": null,
  "beeId": null,
  "queued": null,
  "skipped": null,
} satisfies UpgradeCandidateView

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as UpgradeCandidateView
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


