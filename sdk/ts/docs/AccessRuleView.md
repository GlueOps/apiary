
# AccessRuleView


## Properties

Name | Type
------------ | -------------
`$schema` | string
`createdAt` | Date
`id` | string
`kind` | string
`policy` | string
`purpose` | string
`reason` | string
`seedOverridden` | boolean
`seedRestoresOnRestart` | boolean
`seeded` | boolean
`updatedAt` | Date
`value` | string

## Example

```typescript
import type { AccessRuleView } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "createdAt": null,
  "id": null,
  "kind": null,
  "policy": null,
  "purpose": null,
  "reason": null,
  "seedOverridden": null,
  "seedRestoresOnRestart": null,
  "seeded": null,
  "updatedAt": null,
  "value": null,
} satisfies AccessRuleView

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as AccessRuleView
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


