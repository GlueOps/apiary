
# BindingView


## Properties

Name | Type
------------ | -------------
`$schema` | string
`enabled` | boolean
`hiveId` | string
`id` | string
`priority` | number
`rulesetId` | string
`rulesetName` | string
`swarmId` | string

## Example

```typescript
import type { BindingView } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "enabled": null,
  "hiveId": null,
  "id": null,
  "priority": null,
  "rulesetId": null,
  "rulesetName": null,
  "swarmId": null,
} satisfies BindingView

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as BindingView
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


