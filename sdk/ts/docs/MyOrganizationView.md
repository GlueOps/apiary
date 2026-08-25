
# MyOrganizationView


## Properties

Name | Type
------------ | -------------
`id` | string
`isDefault` | boolean
`name` | string
`role` | string
`status` | string
`supportEnabled` | boolean

## Example

```typescript
import type { MyOrganizationView } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "id": null,
  "isDefault": null,
  "name": null,
  "role": null,
  "status": null,
  "supportEnabled": null,
} satisfies MyOrganizationView

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as MyOrganizationView
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


