
# StoredTaskLogLine


## Properties

Name | Type
------------ | -------------
`loggedAt` | Date
`message` | string
`seq` | number
`stream` | string

## Example

```typescript
import type { StoredTaskLogLine } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "loggedAt": null,
  "message": null,
  "seq": null,
  "stream": null,
} satisfies StoredTaskLogLine

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as StoredTaskLogLine
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


