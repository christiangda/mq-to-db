package messages

import (
	"encoding/json"

	"github.com/christiangda/mq-to-db/internal/consumer"

	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v3"
)

// SQL is used to Unmarshal JSON Payload
type SQL struct {
	Kind    string `json:"TYPE" yaml:"TYPE"`
	Content struct {
		Server   string `json:"SERVER" yaml:"SERVER"`
		DB       string `json:"DB" yaml:"DB"`
		User     string `json:"USER" yaml:"USER"`
		Pass     string `json:"PASS" yaml:"PASS"`
		Sentence string `json:"SENTENCE" yaml:"SENTENCE"`
	} `json:"CONTENT" yaml:"CONTENT"`
	Date       string `json:"DATE" yaml:"DATE"`
	AppID      string `json:"APPID" yaml:"APPID"`
	Additional string `json:"ADITIONAL" yaml:"ADITIONAL"`
	ACK        string `json:"ACK" yaml:"ACK"`
	Response   string `json:"RESPONSE" yaml:"RESPONSE"`
}

// NewSQL create a new SQL message type
func NewSQL(m *consumer.Messages) (*SQL, error) {
	out := &SQL{}
	err := json.Unmarshal(m.Payload, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ToJSON export the SQL in JSON format
func (m *SQL) ToJSON() string {
	out, err := json.Marshal(m)
	if err != nil {
		log.Panic(err)
	}
	return string(out)
}

// ToYAML export the SQL in YAML format
func (m *SQL) ToYAML() string {
	out, err := yaml.Marshal(m)
	if err != nil {
		log.Panic(err)
	}
	return string(out)
}
