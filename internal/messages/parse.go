package messages

import (
	"encoding/json"

	"github.com/christiangda/mq-to-db/internal/consumer"
	"golang.org/x/xerrors"
)

var (
	// ErrUnableUnmarshalJSONMessageAsRaw is returned when
	ErrUnableUnmarshalJSONMessageAsRaw = xerrors.New("Unable to Unmarshal JSON message as raw")
)

func GetType(m consumer.Messages) (string, error) {

	raw := RAW{}
	err := json.Unmarshal(m.Payload, &raw)
	if err != nil {
		return "", ErrUnableUnmarshalJSONMessageAsRaw
	}

	if val, ok := raw["TYPE"]; ok {

		switch val {
		case "SQL":
			return "SQL", nil
		default:
			return "RAW", nil
		}

	} else {
		return "UNKNOW", nil
	}
}
