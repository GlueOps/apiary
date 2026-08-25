
# RuleBody


## Properties

Name | Type
------------ | -------------
`$schema` | string
`action` | string
`chainName` | string
`comment` | string
`dstCidr` | string
`dstPort` | number
`dstPortEnd` | number
`enabled` | boolean
`family` | string
`inIface` | string
`name` | string
`outIface` | string
`position` | number
`protocol` | string
`srcCidr` | string
`srcPort` | number
`srcPortEnd` | number
`tableName` | string

## Example

```typescript
import type { RuleBody } from '@glueops/apiary-sdk'

// TODO: Update the object below with actual values
const example = {
  "$schema": null,
  "action": null,
  "chainName": null,
  "comment": null,
  "dstCidr": null,
  "dstPort": null,
  "dstPortEnd": null,
  "enabled": null,
  "family": null,
  "inIface": null,
  "name": null,
  "outIface": null,
  "position": null,
  "protocol": null,
  "srcCidr": null,
  "srcPort": null,
  "srcPortEnd": null,
  "tableName": null,
} satisfies RuleBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as RuleBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


