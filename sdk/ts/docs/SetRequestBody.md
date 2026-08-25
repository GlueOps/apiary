
# SetRequestBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`config` | [PolicyConfigView](PolicyConfigView.md)
`state` | string

## Example

```typescript
import type { SetRequestBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "config": null,
  "state": null,
} satisfies SetRequestBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as SetRequestBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


