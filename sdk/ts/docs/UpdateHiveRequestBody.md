
# UpdateHiveRequestBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`apiUrl` | string
`cpuOvercommitRatio` | number
`credentialId` | string
`description` | string
`insecureSkipVerify` | boolean
`minimumNodes` | number

## Example

```typescript
import type { UpdateHiveRequestBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "apiUrl": null,
  "cpuOvercommitRatio": null,
  "credentialId": null,
  "description": null,
  "insecureSkipVerify": null,
  "minimumNodes": null,
} satisfies UpdateHiveRequestBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as UpdateHiveRequestBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


