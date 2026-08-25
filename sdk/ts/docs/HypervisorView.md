
# HypervisorView


## Properties

Name | Type
------------ | -------------
`$schema` | string
`address` | string
`allocatable` | [CapacityView](CapacityView.md)
`beeId` | string
`beeStatus` | string
`departureRefusals` | [Array&lt;RefusalView&gt;](RefusalView.md)
`drainState` | string
`heldPlacements` | number
`hiveId` | string
`hiveName` | string
`id` | string
`lastSyncedAt` | Date
`name` | string
`observationStale` | boolean
`observed` | [CapacityView](CapacityView.md)
`observedGuests` | number
`provisioningAttempt` | number
`provisioningError` | string
`provisioningState` | string
`reserved` | [CapacityView](CapacityView.md)
`schedulable` | boolean

## Example

```typescript
import type { HypervisorView } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "address": null,
  "allocatable": null,
  "beeId": null,
  "beeStatus": null,
  "departureRefusals": null,
  "drainState": null,
  "heldPlacements": null,
  "hiveId": null,
  "hiveName": null,
  "id": null,
  "lastSyncedAt": null,
  "name": null,
  "observationStale": null,
  "observed": null,
  "observedGuests": null,
  "provisioningAttempt": null,
  "provisioningError": null,
  "provisioningState": null,
  "reserved": null,
  "schedulable": null,
} satisfies HypervisorView

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as HypervisorView
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


