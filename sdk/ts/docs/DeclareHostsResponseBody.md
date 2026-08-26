
# DeclareHostsResponseBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`created` | number
`hosts` | [Array&lt;DeclaredHostView&gt;](DeclaredHostView.md)

## Example

```typescript
import type { DeclareHostsResponseBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "created": null,
  "hosts": null,
} satisfies DeclareHostsResponseBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as DeclareHostsResponseBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


