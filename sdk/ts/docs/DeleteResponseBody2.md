
# DeleteResponseBody2


## Properties

Name | Type
------------ | -------------
`$schema` | string
`deleted` | boolean
`seedRestoresOnRestart` | boolean
`warning` | string

## Example

```typescript
import type { DeleteResponseBody2 } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "deleted": null,
  "seedRestoresOnRestart": null,
  "warning": null,
} satisfies DeleteResponseBody2

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as DeleteResponseBody2
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


