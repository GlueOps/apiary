
# BeeView


## Properties

Name | Type
------------ | -------------
`address` | string
`agentVersion` | string
`hiveId` | string
`hypervisorId` | string
`id` | string
`lastSeenAt` | Date
`status` | string
`unavailableReason` | string
`unavailableUntil` | Date

## Example

```typescript
import type { BeeView } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "address": null,
  "agentVersion": null,
  "hiveId": null,
  "hypervisorId": null,
  "id": null,
  "lastSeenAt": null,
  "status": null,
  "unavailableReason": null,
  "unavailableUntil": null,
} satisfies BeeView

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as BeeView
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


