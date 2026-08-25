
# LogsRequestBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`lines` | [Array&lt;TaskLogLine&gt;](TaskLogLine.md)

## Example

```typescript
import type { LogsRequestBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "lines": null,
} satisfies LogsRequestBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as LogsRequestBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


