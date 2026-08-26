
# DeclareHostsRequestBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`addresses` | Array&lt;string&gt;
`hosts` | string
`user` | string

## Example

```typescript
import type { DeclareHostsRequestBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "addresses": null,
  "hosts": null,
  "user": null,
} satisfies DeclareHostsRequestBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as DeclareHostsRequestBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


