package messages

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v3"
)

// SQL is usedto unmalshall JSON Payload
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

// ToJSON export the configuration in JSON format
func (c *SQL) ToJSON() string {
	out, err := json.Marshal(c)
	if err != nil {
		log.Panic(err)
	}
	return string(out)
}

// ToYAML export the configuration in YAML format
func (c *SQL) ToYAML() string {
	out, err := yaml.Marshal(c)
	if err != nil {
		log.Panic(err)
	}
	return string(out)
}
