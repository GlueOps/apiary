
# ListRulesResponseBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`rules` | [Array&lt;RuleView&gt;](RuleView.md)

## Example

```typescript
import type { ListRulesResponseBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "rules": null,
} satisfies ListRulesResponseBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as ListRulesResponseBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


