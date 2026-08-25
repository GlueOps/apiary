
# ListHypervisorsResponseBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`hypervisors` | [Array&lt;HypervisorView&gt;](HypervisorView.md)

## Example

```typescript
import type { ListHypervisorsResponseBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "hypervisors": null,
} satisfies ListHypervisorsResponseBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as ListHypervisorsResponseBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


