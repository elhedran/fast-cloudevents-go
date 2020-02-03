package main // jsonce

import (
	"encoding/json"
        "fmt"
        "time"
)

func main(){
	ce := CloudEvent{
		Id: "test",
		Source: "test",
		SpecVersion: "test",
		Type: "test",
		Data: []byte(`{"a":2}`),
	}
	fmt.Printf("	%#v\n", ce)
	b, e := ce.MarshalJSON()
	fmt.Printf("	json:%s\n	err:%+v\n", b, e)



	ce.From
	fmt.Printf("	%#v\n", ce)


}

// https://github.com/cloudevents/spec/blob/master/spec.md
type CloudEvent struct {
	// Required
	Id string
	Source string // URI-reference
	SpecVersion string
	Type string

	// Optional
	DataContentType string // RFC 2046
	DataSchema string // URI
	Subject string
	Time time.Time // RFC3339

	// Additional
	Extensions map[string]json.RawMessage
	Data []byte //json.RawMessage // Internal, use .GetData()
	//Data64 []byte // Internal, use .GetData()
}

func (ce CloudEvent) Valid() bool {
	// If extensions contains a Context Attribute name, that's bad
	// If the data does not seem to be compatible with the contenttype
	// If URI fields are out of spec
	return true
}

func (ce CloudEvent) UnmarshalJSON(data []byte) (err error) {
    m = map[string]interface{}{}
    if err = json.Unmarshal(data, &m); err != nil {
	return fmt.Errorf("Could not unmarshal event: %s", err.Error())
    }
    ce.FromMap(m)
    return
}
func (ce CloudEvent) MarshalJSON() ([]byte, error) {
    return json.Marshal(ce.ToMap())
}
func (ce CloudEvent) ToMap() (m map[string]interface{}) {
    // Required
    m = map[string]interface{}{}
    m["id"] = ce.Id
    m["source"] = ce.Source
    m["specversion"] = ce.SpecVersion
    m["type"] = ce.Type

    // Optional
    if len(ce.DataContentType)>0 {
        m["datacontenttype"] = ce.DataContentType
    }
    if len(ce.DataSchema)>0 {
        m["dataschema"] = ce.DataSchema
    }
    if len(ce.Subject)>0 {
        m["subject"] = ce.Subject
    }
    if !ce.Time.IsZero() {
        m["time"] = ce.Time.Format(time.RFC3339)
    }

    // Additional
    for k, v := range ce.Extensions {
	m[k] = v
    }

    if len(ce.Data)>0 {
	// https://github.com/cloudevents/spec/blob/v1.0/json-format.md#31-handling-of-data
	if js, err := RawJSON(ce.Data); err == nil {
		m["data"] = js
	} else {
		m["data_base64"] = ce.Data
	}
    }

    return
}

func RawJSON(data []byte) (js json.RawMessage, err error) {
    return js, json.Unmarshal(data, &js)
}
func (ce *CloudEvent) FromMap (m map[string]interface{}) (err error) {
	// Required
	ok := false
	if ce.Id, ok = m["id"].(string); !ok {
		return fmt.Errorf("Could not read ID as string")
	}
	if ce.Source, ok = m["source"].(string); !ok {
		return fmt.Errorf("Could not read Source as string")
	}
	if ce.SpecVersion, ok = m["specversion"].(string); !ok {
		return fmt.Errorf("Could not read Spec Version as string")
	}
	if ce.Type, ok = m["type"].(string); !ok {
		return fmt.Errorf("Could not read Type as string")
	}

	// Optional
	if m["datacontenttype"] != nil {
		if ce.DataContentType, ok = m["datacontenttype"].(string); !ok {
			return fmt.Errorf("Could not read Data Content Type as string")
		}
	}
	if m["dataschema"] != nil {
		if ce.DataSchema, ok = m["dataschema"].(string); !ok {
			return fmt.Errorf("Could not read Data Schema as string")
		}
	}
	if m["subject"] != nil {
		if ce.DataSchema, ok = m["subject"].(string); !ok {
			return fmt.Errorf("Could not read Subject as string")
		}
	}
	if m["time"] != nil {
		ceTime, ok := m["time"].(string)
		if !ok {
			return fmt.Errorf("Could not read Time as string")
		}
		ce.Time, err = time.Parse(
			time.RFC3339,//Nano
			ceTime)
		if err != nil {
			return fmt.Errorf("Could not read Time as time: %s", err.Error())
		}
	}


	// Additional
	ex, err := GetMapExtensions(m)
	if err != nil {
		return fmt.Errorf("Could not read Extensions: %s", err.Error())
	}
	if len(ex)>0 {
		ce.Extensions = ex
	}

	if m["data_base64"] != nil {
		if ce.Data, ok = m["data_base64"].([]byte) ; !ok {
			return fmt.Errorf("Could not read Data Base64 as []byte")
		}
	} else if m["data"] != nil {
		mData, ok := m["data"].([]byte)
		if !ok {
			return fmt.Errorf("Could not read Data as string")
		}
		ceData, err := RawJSON(mData)
		if err != nil {
			return fmt.Errorf("Could not read Data as json: %s", err.Error())
		}
		ce.Data = ceData
	}

	return nil
}


var contextProperties = []string{
	"id",
	"source",
	"specversion",
	"type",
	"datacontenttype",
	"dataschema",
	"subject",
	"time",
	"data",
}
func GetMapExtensions(m map[string]interface{}) (ex map[string]json.RawMessage, err error){
	for k, v := range m {
		if inSlice(k, contextProperties) {
			continue;
		}
		raw, ok := v.([]byte)
		if !ok {
			return ex, fmt.Errorf("Could not read extension %s", k)
		}
		data, err := RawJSON(raw)
		if err != nil {
			return ex, fmt.Errorf("Could not parse extension %s: %s", k, err.Error())
		}
		ex[k] = data
	}
	return
}
func inSlice(e string, list []string) bool {
    for _, v := range list {
        if v == e {
            return true
        }
    }
    return false
}

