package json

import (
	"encoding/json"
	"fmt"

	events "github.com/elhedran/fast-cloudevents-go/events"
)

type jsonCloudEventBase struct {
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
}

type JsonCloudEvent struct {
	jsonCloudEventBase

	Extensions map[string]interface{} `json:"omit"`
}

type JsonCloudEventBatch []JsonCloudEvent

func (v *JsonCloudEvent) UnMarshal(data []byte) {
	base := jsonCloudEventBase{}
	extensions := make(map[string]interface{})

	if err := json.Unmarshal(data, &base); err != nil {
		panic(err)
	}

	if err := json.Unmarshal(data, &extensions); err != nil {
		panic(err)
	}
	fmt.Printf("got extensions %q\n", extensions)
	for _, prop := range events.ContextProperties {
		delete(extensions, prop)
	}

	*v = JsonCloudEvent{
		jsonCloudEventBase: base,
		Extensions:         extensions,
	}
	v.Extensions = extensions
}
