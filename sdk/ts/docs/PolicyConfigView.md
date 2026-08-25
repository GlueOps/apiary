
# PolicyConfigView


## Properties

Name | Type
------------ | -------------
`maxLifetimeMinutes` | number
`minLength` | number
`requireDigit` | boolean
`requireLowercase` | boolean
`requireSpecial` | boolean
`requireUppercase` | boolean

## Example

```typescript
import type { PolicyConfigView } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "maxLifetimeMinutes": null,
  "minLength": null,
  "requireDigit": null,
  "requireLowercase": null,
  "requireSpecial": null,
  "requireUppercase": null,
} satisfies PolicyConfigView

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as PolicyConfigView
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


