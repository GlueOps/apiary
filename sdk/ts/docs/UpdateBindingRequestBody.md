
# UpdateBindingRequestBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`enabled` | boolean
`priority` | number
`scope` | [BindingScopeBody](BindingScopeBody.md)

## Example

```typescript
import type { UpdateBindingRequestBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "enabled": null,
  "priority": null,
  "scope": null,
} satisfies UpdateBindingRequestBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as UpdateBindingRequestBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


