package jsonce

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

type Mode int

const (
	ModeBinary Mode = iota
	ModeStructure
	ModeBatch
)

// CloudEvent is the primary format for events
// https://github.com/cloudevents/spec/blob/master/spec.md
type CloudEvent struct {
	// Required
	Id          string
	Source      string // URI-reference
	SpecVersion string
	Type        string

	// Optional
	DataContentType string // RFC 2046
	DataSchema      string // URI
	Subject         string
	Time            time.Time // RFC3339

	// Additional
	// https://github.com/cloudevents/spec/blob/master/spec.md#type-system
	Extensions map[string]interface{} // This type is subject to change to be more specific to CE
	Data       []byte
}

// DataStruct is used for capturing certain fields conveniently from json.unmarshal
type DataStruct struct {
	Data   json.RawMessage `json:"data"`
	Data64 []byte          `json:"data_base64"`
}

// Valid returns true if the CloudEvent seems to fit the spec
func (ce CloudEvent) Valid() bool {
	panic("Stubbed function")
	// Multiline headers could be a warning (deprecated under RFC 7230)
	// If extensions contains a Context Attribute name, that's bad
	// IF an extension has data that does not fit into the CE type system
	// If the data does not seem to be compatible with the contenttype
	// If URI fields are out of spec
	return true
}

// UnmarshalJSON allows translation of []byte to CloudEvent
func (ce *CloudEvent) UnmarshalJSON(data []byte) (err error) {
	m := map[string]interface{}{}
	g := DataStruct{}
	if err = json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("Could not unmarshal event: %s", err.Error())
	}
	if err = json.Unmarshal(data, &g); err != nil {
		return fmt.Errorf("Could not unmarshal event data: %s", err.Error())
	}
	if len(g.Data) > 0 {
		m["data"] = g.Data
	}
	if len(g.Data64) > 0 {
		m["data_base64"] = g.Data64
	}
	return ce.FromMap(m)
}

// MarshalJSON allows translation of CloudEvent to []byte
func (ce CloudEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(ce.ToMap())
}

// ToMap produces an intermediate representation of a CloudEvent
func (ce CloudEvent) ToMap() (m map[string]interface{}) {
	// Required
	m = map[string]interface{}{}
	m["id"] = ce.Id
	m["source"] = ce.Source
	m["specversion"] = ce.SpecVersion
	m["type"] = ce.Type

	// Optional
	if len(ce.DataContentType) > 0 {
		m["datacontenttype"] = ce.DataContentType
	}
	if len(ce.DataSchema) > 0 {
		m["dataschema"] = ce.DataSchema
	}
	if len(ce.Subject) > 0 {
		m["subject"] = ce.Subject
	}
	if !ce.Time.IsZero() {
		m["time"] = ce.Time.Format(time.RFC3339)
	}

	// Additional
	for k, v := range ce.Extensions {
		m[k] = v
	}

	if len(ce.Data) > 0 {
		// https://github.com/cloudevents/spec/blob/v1.0/json-format.md#31-handling-of-data
		if js, err := rawJSON(ce.Data); err == nil {
			m["data"] = js
		} else {
			m["data_base64"] = ce.Data
		}
	}

	return
}

// FromMap converts the intermediate map representation back into a CloudEvent
func (ce *CloudEvent) FromMap(m map[string]interface{}) (err error) {
	// Required https://github.com/cloudevents/spec/blob/master/spec.md#required-attributes
	ok := false
	if ce.Id, ok = m["id"].(string); !ok || len(ce.Id) < 1 {
		return fmt.Errorf(errRead("ID", "nonempty string"))
	}
	if ce.Source, ok = m["source"].(string); !ok || len(ce.Source) < 1 {
		return fmt.Errorf(errRead("Source", "nonempty string"))
	}
	if ce.SpecVersion, ok = m["specversion"].(string); !ok || len(ce.Source) < 1 {
		return fmt.Errorf(errRead("Spec Version", "nonempty string"))
	}
	if ce.Type, ok = m["type"].(string); !ok || len(ce.Type) < 1 {
		return fmt.Errorf(errRead("Type", "nonempty string"))
	}

	// Optional
	if m["datacontenttype"] != nil {
		if ce.DataContentType, ok = m["datacontenttype"].(string); !ok {
			return fmt.Errorf(errRead("Data Content Type", "string"))
		}
	}
	if m["dataschema"] != nil {
		if ce.DataSchema, ok = m["dataschema"].(string); !ok {
			return fmt.Errorf(errRead("Data Schema", "string"))
		}
	}
	if m["subject"] != nil {
		if ce.DataSchema, ok = m["subject"].(string); !ok {
			return fmt.Errorf(errRead("Subject", "string"))
		}
	}
	if m["time"] != nil {
		ceTime, ok := m["time"].(string)
		if !ok {
			return fmt.Errorf(errRead("Time", "string"))
		}
		ce.Time, err = time.Parse(
			time.RFC3339, // allows Nano - see tests
			ceTime)
		if err != nil {
			return fmt.Errorf("%s: %s", errRead("Time", "time"), err.Error())
		}
	}

	// Additional - Extensions
	ex, err := GetMapExtensions(m)
	if err != nil {
		return fmt.Errorf("Could not read Extensions: %s", err.Error())
	}
	if len(ex) > 0 {
		ce.Extensions = ex
	}

	// Additional - Data
	if m["data_base64"] != nil {
		if ce.Data, ok = m["data_base64"].([]byte); !ok {
			return fmt.Errorf(errRead("Data Base64", "[]byte"))
		}
	} else if m["data"] != nil {
		mData, ok := m["data"].(json.RawMessage)
		if !ok {
			return fmt.Errorf(errRead("Data", "string"))
		}
		if len(mData) < 1 {
			return nil
		}
		ceData, err := rawJSON([]byte(mData))
		if err != nil {
			return fmt.Errorf("%s: %s", errRead("Data", "json"), err.Error())
		}
		ce.Data = ceData
	}

	return nil
}

// ContextProperties is a list of default context properties which cannot be extensions
var ContextProperties = []string{
	"id",
	"source",
	"specversion",
	"type",
	"datacontenttype",
	"dataschema",
	"subject",
	"time",
	"data",
	"data_base64",
}

// GetMapExtensions is used to extract extension properties from the intermediate map representation
func GetMapExtensions(m map[string]interface{}) (ex map[string]interface{}, err error) {
	ex = map[string]interface{}{}
	for k, v := range m {
		if InSlice(k, ContextProperties) {
			continue
		}
		ex[k] = v
		// We might want to parse these into something more specific:
		// https://www.json.org/json-en.html
		// https://github.com/cloudevents/spec/blob/master/spec.md#type-system
	}
	return
}

// SetData is a utility field for setting binary data on data_base64 on a map without encoding
func SetData(m map[string]interface{}, data []byte) {
	// Could use some optimisation if we know len(src)
	m["data_base64"] = []byte(base64.StdEncoding.EncodeToString(data))
}

// InSlice is useful for checking the presence of an element in a slice
func InSlice(e string, list []string) bool {
	for _, v := range list {
		if v == e {
			return true
		}
	}
	return false
}

// rawJSON is useful for Unmarshalling json data types
func rawJSON(data []byte) (js json.RawMessage, err error) {
	return js, json.Unmarshal(data, &js)
}

// nonempty produces a predictable error string when needed
func errRead(prop string, as string) string {
	return fmt.Sprintf("Could not read %s as %s", prop, as)
}
