
# HiveDetailView


## Properties

Name | Type
------------ | -------------
`$schema` | string
`apiUrl` | string
`configured` | boolean
`cpuOvercommitRatio` | number
`credentialId` | string
`description` | string
`effectiveMinimumNodes` | number
`id` | string
`insecureSkipVerify` | boolean
`minimumNodes` | number
`name` | string
`nodeCount` | number
`provider` | string

## Example

```typescript
import type { HiveDetailView } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "apiUrl": null,
  "configured": null,
  "cpuOvercommitRatio": null,
  "credentialId": null,
  "description": null,
  "effectiveMinimumNodes": null,
  "id": null,
  "insecureSkipVerify": null,
  "minimumNodes": null,
  "name": null,
  "nodeCount": null,
  "provider": null,
} satisfies HiveDetailView

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as HiveDetailView
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


