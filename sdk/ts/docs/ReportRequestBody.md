
# ReportRequestBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`error` | string
`phase` | string
`status` | string

## Example

```typescript
import type { ReportRequestBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "error": null,
  "phase": null,
  "status": null,
} satisfies ReportRequestBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as ReportRequestBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


