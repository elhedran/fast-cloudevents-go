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

// ContentType returns the known part of the mode protocol
func (m Mode) ContentType() string {
	switch m {
	case ModeBinary:
		return ""
	case ModeStructure:
		return "application/cloudevents"
	case ModeBatch:
		return "application/cloudevents-batch"
	default:
		return ""
	}
}

// ContentTypePlus returns the known part of the mode protocol plus "+" some subtype
// For example, a structured mode request would be prefixed with `application/cloudevents`
// But to specify that it is structured in JSON, the full content-type header would read:
// `application/cloudevents+json`, which is produced by calling `ModeStructure.ContentTypePlus("json")`
// In case the ContentType returned is empty (in case of an unimplemented type, or binary), only the subtype is returned
// In case the subtype is empty, the plus "+" symbol will still be suffixed to the result
// In case both ContentType and subtype are empty, "" is returned
func (m Mode) ContentTypePlus(subtype string) string {
	ct := m.ContentType()
	if len(ct) < 1 {
		return subtype
	}
	return fmt.Sprintf("%s+%s", ct, subtype)
}

type CloudEvents []CloudEvent

// DataStruct is used for capturing certain fields conveniently from json.unmarshal
type DataStruct struct {
	Data   json.RawMessage `json:"data"`
	Data64 []byte          `json:"data_base64"`
}

// CEMap loosely represents the intermediate map form of a CloudEvent
type CEMap map[string]interface{}

// CEMaps represents a collection of maps representing CloudEvents
type CEMaps []CEMap

// MapToMap represents a function which can be called between Un/Marshall and To/FromMap calls
type MapToMap func(CEMap) (CEMap, error)

// MapToCE represents any function which can translate a map into a CloudEvent
// The default choice is DefaultMapToCE
type MapToCE func(CEMap) (CloudEvent, error)

// CEToMap represents any function which can translate a CloudEvent into a map
// The default choice is DefaultCEToMap
type CEToMap func(ces CloudEvent) (CEMap, error)

// ToCEs is a convenience wrapper for calling a mapper on an array
func (cms *CEMaps) ToCEs(mapper MapToCE) (ces CloudEvents, err error) {
	ces = CloudEvents{}
	for i, cm := range *cms {
		var ce CloudEvent
		ce, err = cm.ToCE(mapper)
		if err != nil {
			err = fmt.Errorf("%d: %s", i, err.Error())
			return
		}
		ces = append(ces, ce)
	}
	return
}

func CEMapsFromInterface(m []map[string]interface{}) CEMaps {
	cms := CEMaps{}
	for _, v := range m {
		cms = append(cms, CEMap(v))
	}
	return cms
}

// FromCEs is a convenience wrapper for calling a mapper on an array
func (cms *CEMaps) FromCEs(mapper CEToMap, ces CloudEvents) (err error) {
	for i, ce := range ces {
		cm := CEMap{}
		err = cm.FromCE(mapper, ce)
		if err != nil {
			err = fmt.Errorf("%d: %s", i, err.Error())
			return
		}
		*cms = append(*cms, cm)
	}
	return
}

// ToCE is a convenience wrapper for calling a mapper on a CEMap
func (cm *CEMap) ToCE(mapper MapToCE) (ce CloudEvent, err error) {
	ce = CloudEvent{}
	ce, err = mapper(*cm)
	if err != nil {
		err = fmt.Errorf("Mapper error: %s", err.Error())
		return
	}
	return
}

// FromCE is a convenience wrapper for calling a mapper on a CEMap
func (cm *CEMap) FromCE(mapper CEToMap, ce CloudEvent) (err error) {
	*cm, err = mapper(ce)
	if err != nil {
		err = fmt.Errorf("Mapper error: %s", err.Error())
		return
	}
	return
}

// UnmarshalJSON allows translation of []byte to CEMap
// It is called by json.Unmarshal
func (cm *CEMap) UnmarshalJSON(data []byte) (err error) {
	m := map[string]interface{}(*cm) // Avoid infinite loop
	g := DataStruct{}
	if err = json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("Could not unmarshal map: %s", err.Error())
	}
	if err = json.Unmarshal(data, &g); err != nil {
		return fmt.Errorf("Could not unmarshal map data: %s", err.Error())
	}

	*cm = CEMap(m)
	if g.Data != nil {
		(*cm)["data"] = g.Data
	}
	if len(g.Data64) > 0 {
		(*cm)["data_base64"] = []byte(g.Data64)
	} else if g.Data != nil {
		(*cm)["data_base64"] = []byte(g.Data)
	}
	return
}

// MarshalJSON allows translation of []byte to CEMap
// It is called by json.Marshal
func (cm CEMap) MarshalJSON() (data []byte, err error) {
	m := map[string]interface{}(cm) // Avoid infinite loop

	// https://github.com/cloudevents/spec/blob/v1.0/json-format.md#31-handling-of-data
	// If there was data, then we can ignore data_base64 for marshaling
	if m["data"] != nil {
		delete(m, "data_base64")
	} // Else, we will either marshal data_base64 or have no data

	data, err = json.Marshal(&m)
	if err != nil {
		err = fmt.Errorf("Could not marshal map: %s", err.Error())
		return
	}
	return
}

// UnmarshalJSON allows translation of []byte to CloudEvent
// It is called by json.Unmarshal
func (ce *CloudEvent) UnmarshalJSON(data []byte) (err error) {
	cm := CEMap{}
	err = json.Unmarshal(data, &cm)
	if err != nil {
		return fmt.Errorf("Map error: %s", err.Error())
	}
	*ce, err = DefaultMapToCE(cm)
	return err
}

// MarshalJSON allows translation of CloudEvent to []byte
// It is called by json.Marshal
func (ce CloudEvent) MarshalJSON() (data []byte, err error) {
	var cm CEMap
	cm, err = DefaultCEToMap(ce)
	if err != nil {
		err = fmt.Errorf("Map error: %s", err.Error())
		return
	}
	return json.Marshal(cm)
}

// DefaultCEToMap (formerly ToMap) produces an intermediate representation of a CloudEvent
// It is the compliment to DefaultMapToCE and is used as the default mapper function for
// calls which require a CEToMap function
func DefaultCEToMap(ce CloudEvent) (m CEMap, err error) {
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
		m["time"] = ce.Time.Format(time.RFC3339Nano)
	}

	// Additional
	for k, v := range ce.Extensions {
		m[k] = v
	}

	if len(ce.Data) > 0 {
		// https://github.com/cloudevents/spec/blob/v1.0/json-format.md#31-handling-of-data
		m["data_base64"] = []byte(ce.Data)
		if js, err := rawJSON(ce.Data); err == nil {
			m["data"] = js
		}
	}

	return
}

// DefaultMapToCE (formarly FromMap) converts the intermediate map representation back into a CloudEvent
// It has specific expectations about the types present in the provided map m.
// It does not perform validation, see .Valid()
// DefaultMapToCE makes a best attempt to put the given map into an event, but can be
// wrapped or replaced with a custom mapper (jsonce.CEToMap or jsonce.MapToCE)
func DefaultMapToCE(m CEMap) (ce CloudEvent, err error) {
	fmt.Sprintf("DefaultMapToCE: %#v\n", m)
	ce = CloudEvent{}
	// Required https://github.com/cloudevents/spec/blob/master/spec.md#required-attributes
	ok := false
	if ce.Id, ok = m["id"].(string); !ok || len(ce.Id) < 1 {
		err = fmt.Errorf(errRead("ID", "nonempty string"))
		return
	}
	if ce.Source, ok = m["source"].(string); !ok || len(ce.Source) < 1 {
		err = fmt.Errorf(errRead("Source", "nonempty string"))
		return
	}
	if ce.SpecVersion, ok = m["specversion"].(string); !ok || len(ce.Source) < 1 {
		err = fmt.Errorf(errRead("Spec Version", "nonempty string"))
		return
	}
	if ce.Type, ok = m["type"].(string); !ok || len(ce.Type) < 1 {
		err = fmt.Errorf(errRead("Type", "nonempty string"))
		return
	}

	// Optional
	if m["datacontenttype"] != nil {
		if ce.DataContentType, ok = m["datacontenttype"].(string); !ok {
			err = fmt.Errorf(errRead("Data Content Type", "string"))
			return
		}
	}
	if m["dataschema"] != nil {
		if ce.DataSchema, ok = m["dataschema"].(string); !ok {
			err = fmt.Errorf(errRead("Data Schema", "string"))
			return
		}
	}
	if m["subject"] != nil {
		if ce.Subject, ok = m["subject"].(string); !ok {
			err = fmt.Errorf(errRead("Subject", "string"))
			return
		}
	}
	if m["time"] != nil {
		ceTime, ok := m["time"].(string)
		if !ok {
			err = fmt.Errorf(errRead("Time", "string"))
			return
		}
		ce.Time, err = time.Parse(
			time.RFC3339, // allows Nano - see tests
			ceTime)
		if err != nil {
			err = fmt.Errorf("%s: %s", errRead("Time", "time"), err.Error())
			return
		}
	}

	// Additional - Extensions
	ex, err := GetMapExtensions(m)
	if err != nil {
		err = fmt.Errorf("Could not read Extensions: %s", err.Error())
		return
	}
	ce.Extensions = ex

	// Additional - Data
	if m["data"] != nil {
		mData, ok := m["data"].(json.RawMessage)
		if !ok {
			err = fmt.Errorf(errRead("Data", "string"))
			return
		}
		ceData, err := rawJSON([]byte(mData))
		if err != nil {
			err = fmt.Errorf("%s: %s", errRead("Data", "json"), err.Error())
			return ce, err
		}
		ce.Data = ceData
	} else if m["data_base64"] != nil {
		if ce.Data, ok = m["data_base64"].([]byte); !ok {
			err = fmt.Errorf("%s: %T", errRead("Data Base64", "[]byte"), m["data_base64"])
			return
		}
	}

	return
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

/*
 ██╗   ██╗████████╗██╗██╗     ███████╗
 ██║   ██║╚══██╔══╝██║██║     ██╔════╝
 ██║   ██║   ██║   ██║██║     ███████╗
 ██║   ██║   ██║   ██║██║     ╚════██║
 ╚██████╔╝   ██║   ██║███████╗███████║
  ╚═════╝    ╚═╝   ╚═╝╚══════╝╚══════╝
*/

// GenerateValidEvents provides an array of CloudEvents of a given length
func GenerateValidEvents(num uint) []CloudEvent {
	ces := []CloudEvent{}
	for i := uint(0); i < num; i++ {
		ces = append(ces, CloudEvent{
			Id:              fmt.Sprintf("Example_%s", timestamp()),
			Source:          "Example",
			SpecVersion:     "v1.0",
			Type:            "test",
			DataContentType: "text/plain",
			DataSchema:      "http://localhost/schema",
			Subject:         "test",
			Time:            time.Now(),
			Extensions: map[string]interface{}{
				"extension-1": "value",
			},
			Data: []byte("raw data"),
		})
	}
	return ces
}

// Produce a probably-unique unix nano timestamp
func timestamp() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
