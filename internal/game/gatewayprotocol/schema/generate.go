package schema

import (
	"encoding/json"

	"github.com/crimsab/oneday/internal/game/gatewayprotocol"
	"github.com/invopop/jsonschema"
)

func Generate() ([]byte, error) {
	reflector := &jsonschema.Reflector{Anonymous: true}
	root := reflector.Reflect(&gatewayprotocol.SchemaRoot{})
	root.ID = jsonschema.ID("https://oneday.local/contracts/gateway-v1.schema.json")
	payload, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(payload, '\n'), nil
}
