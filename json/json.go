package json

import (
	"encoding/json"

	events "ce/events"
)

type JsonCloudEvent struct {
	// Required
	Id          string `json:"id"`
	Source      string `json:"source"`
	SpecVersion string `json:"specversion"`
	Type        string `json:"type"`

	// Optional
	DataContentType string          `json:"datacontenttype"`
	DataSchema      string          `json:dataschema"`
	Subject         string          `json:subject`
	Time            string          `json:"time"`
	Data            json.RawMessage `json:"data"`
	Data64          []byte          `json:"data_base64"`

	Extensions map[string]interface{} `json:"omit"`
}

func UnMarshal(data []byte, v events.CloudEvent) {

	f := Foo{}
	if err := json.Unmarshal([]byte(s), &f); err != nil {
		panic(err)
	}

	if err := json.Unmarshal([]byte(s), &f.X); err != nil {
		panic(err)
	}
	for key := range events.ContextProperties {

	}

	delete(f.X, "a")
	delete(f.X, "b")
}
