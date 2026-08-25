
# FleetlessHostView


## Properties

Name | Type
------------ | -------------
`address` | string
`arrivalRefusals` | [Array&lt;RefusalView&gt;](RefusalView.md)
`beeId` | string
`hypervisorId` | string
`lastSeenAt` | Date
`provisioningError` | string
`provisioningState` | string
`status` | string

## Example

```typescript
import type { FleetlessHostView } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "address": null,
  "arrivalRefusals": null,
  "beeId": null,
  "hypervisorId": null,
  "lastSeenAt": null,
  "provisioningError": null,
  "provisioningState": null,
  "status": null,
} satisfies FleetlessHostView

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as FleetlessHostView
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


