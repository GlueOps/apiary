
# ListRulesetsResponseBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`rulesets` | [Array&lt;RulesetView&gt;](RulesetView.md)

## Example

```typescript
import type { ListRulesetsResponseBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "rulesets": null,
} satisfies ListRulesetsResponseBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as ListRulesetsResponseBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


