package fleet

import "encoding/json"

// jsonUnmarshal keeps the encoding/json import out of the handler file, where
// its only use is rendering a stored jsonb spec back onto the wire.
func jsonUnmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }
