
# ListHivesResponseBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`hives` | [Array&lt;HiveDetailView&gt;](HiveDetailView.md)

## Example

```typescript
import type { ListHivesResponseBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "hives": null,
} satisfies ListHivesResponseBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as ListHivesResponseBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


