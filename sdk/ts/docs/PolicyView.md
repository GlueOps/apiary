
# PolicyView


## Properties

Name | Type
------------ | -------------
`$schema` | string
`config` | [PolicyConfigView](PolicyConfigView.md)
`effectiveConfig` | [PolicyConfigView](PolicyConfigView.md)
`effectiveEnabled` | boolean
`kind` | string
`state` | string

## Example

```typescript
import type { PolicyView } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "config": null,
  "effectiveConfig": null,
  "effectiveEnabled": null,
  "kind": null,
  "state": null,
} satisfies PolicyView

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as PolicyView
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


