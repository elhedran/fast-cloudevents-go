package jsonce

import (
	"encoding/json"
	"fmt"
	"time"
	"testing"
)

type Scenarios struct { // TODO: Make interface
	T      *testing.T
	Name   string
	Input  string
	Data   CloudEvent
	Result error

	Ok  func() CloudEvent
	Err func(string) error
}

func (s Scenarios) Failf(msg string, arg ...interface{}) {
	s.T.Errorf("Unmarshal:%s:\n%s\n\n%s\n", s.Name, s.Input, fmt.Sprintf(msg, arg...))
}
func (s Scenarios) Logf(msg string, arg ...interface{}) {
	s.T.Logf("Unmarshal:%s:\n\t%s\n", s.Name, fmt.Sprintf(msg, arg...))
}
func TestUnmarshal(t *testing.T) {
	unmarshal := func(name, data string) Scenarios {
		ce := CloudEvent{}
		err := ce.UnmarshalJSON([]byte(data))
		s := Scenarios{
			T:      t,
			Name:   name,
			Input:  data,
			Data:   ce,
			Result: err,
		}
		s.Ok = func() CloudEvent {
			if err != nil {
				s.Failf("Want no error\nHave: %s", err.Error())
			}
			return ce
		}
		s.Err = func(expect string) error {
			if err == nil || err.Error() != expect {
				s.Failf("Want: %s\nHave: %s", expect, err.Error())
			}
			return err
		}
		return s
	}

	// Required

	data := `{}`
	unmarshal("Need id", data).Err(errRead("ID", "nonempty string"))
	data = `{"id":""}`
	unmarshal("Need id len", data).Err(errRead("ID", "nonempty string"))
	data = `{"id":"a"}`
	unmarshal("Need source", data).Err(errRead("Source", "nonempty string"))
	data = `{"id":"a","source":"b"}`
	unmarshal("Need version", data).Err(errRead("Spec Version", "nonempty string"))
	data = `{"id":"a","source":"b","specversion":"c"}`
	unmarshal("Need type", data).Err(errRead("Type", "nonempty string"))
	data = `{"id":"a","source":"b","specversion":"c","type":"d"}`
	unmarshal("Minimum", data).Ok()

	// Optional

	data = `{
		"id":"a","source":"b","specversion":"c","type":"d",
		"datacontenttype":"e","dataschema":"f","subject":"g","time":"2020-02-02T06:06:06+08:00"
	}`
	unmarshal("Complete", data).Ok()

	data = `{
		"id":"a","source":"b","specversion":"c","type":"d",
		"time":"2020-02-02T06:06:06.366090001+10:00"
	}`
	unmarshal("NanoTime", data).Ok()

	data = `{
		"id":"a","source":"b","specversion":"c","type":"d",
		"time":"2020-02-02T06:06:06.366090+12:00"
	}`
	unmarshal("MicroTime", data).Ok()

	data = `{
		"id":"a","source":"b","specversion":"c","type":"d",
		"time":"2020-02-02T06:06:06.366+14:00"
	}`
	unmarshal("MilliTime", data).Ok()

	data = `{
		"id":"a","source":"b","specversion":"c","type":"d",
		"time":"2020-02-02T06:06:60+25:00"
	}`
	unmarshal("InvalidTime", data).Err(`Could not read Time as time: parsing time "2020-02-02T06:06:60+25:00": second out of range`)

	// Additional - Extensions

	{ // Scoped
		data = `{
			"id":"a","source":"b","specversion":"c","type":"d",
			"x":3,"y":null,"z":0.1,"a":[{}],
			"any other string":true,
			"extensions":"Even this"
		}`
		s := unmarshal("Extensions", data)
		ex := s.Ok().Extensions
		{
			prop := "x"
			want := float64(3)
			if got, ok := ex[prop].(float64); !ok {
				s.Failf("Want: %#v\nHave bad cast of %s with type %T\n\t%#v", want, prop, ex[prop], ex[prop])
			} else if got != want {
				s.Failf("Want: %#v\nHave: %#v", want, got)
			}
		}
		{
			prop := "y"
			if ex[prop] != nil {
				s.Failf("Have bad nil of %s\n\t%#v", prop, ex[prop])
			}
		}
		{
			prop := "z"
			want := 0.1
			if got, ok := ex[prop].(float64); !ok {
				s.Failf("Have bad cast of %s\n\t%#v", prop, ex[prop])
			} else if got != want {
				s.Failf("Want: %#v\nHave: %#v", want, got)
			}
		}
		{
			prop := "a"
			if _, ok := ex[prop].([]interface{}); !ok {
				s.Failf("Have bad cast of %s\n\t%#v", prop, ex[prop])
			}
		}
		{
			prop := "any other string"
			want := true
			if got, ok := ex[prop].(bool); !ok {
				s.Failf("Want: %#v\nHave bad cast of %s\n\t%#v", want, prop, ex[prop])
			} else if got != want {
				s.Failf("Want: %#v\nHave: %#v", want, got)
			}
		}
		{
			prop := "extensions"
			want := "Even this"
			if got, ok := ex[prop].(string); !ok {
				s.Failf("Want: %#v\nHave bad cast of %s\n\t%#v", want, prop, ex[prop])
			} else if got != want {
				s.Failf("Want: %#v\nHave: %#v", want, got)
			}
		}
	}

	// Additional - JSON

	data = `{
		"id":"a","source":"b","specversion":"c","type":"d",
		"data":{"x":[1,2,"3"]}
	}`
	s := unmarshal("DataJSON", data)
	js, err := json.RawMessage(s.Ok().Data).MarshalJSON()
	if err != nil {
		s.Failf("Failed to parse data: %s", err.Error())
	}
	if want := `{"x":[1,2,"3"]}`; string(js) != want {
		s.Failf("Want: %s\nHave: %s", want, js)
	}
	s.Logf("%s", js)

	// Additional - Raw

	data = `{
		"id":"a","source":"b","specversion":"c","type":"d",
		"data":123
	}`
	s = unmarshal("DataInt", data)
	js, err = json.RawMessage(s.Ok().Data).MarshalJSON()
	if err != nil {
		s.Failf("Failed to parse data: %s", err.Error())
	}
	if want := "123"; string(js) != want {
		s.Failf("Want: %s\nHave: %s", want, js)
	}
	s.Logf("%s", js)

	// Additional - Base64

	data = `{
		"id":"a","source":"b","specversion":"c","type":"d",
		"data_base64":"aGVsbG8gd29ybGQ="
	}`
	s = unmarshal("Data64", data)
	raw := s.Ok().Data
	if want := "hello world"; string(raw) != want {
		s.Failf("Want: %s\nHave: %s", want, string(raw))
	}
	s.Logf("%s", raw)
}
func TestMarshal(t *testing.T) {
        input := CloudEvent{
                Id:          "test",
                Source:      "test",
                SpecVersion: "test",
                Type:        "test",
                Data:        []byte(`{"a":2}`),
		Extensions:  map[string]interface{}{ "hi":"test" },
        }

	// Data Normal
	want := `{"data":{"a":2},"hi":"test","id":"test","source":"test","specversion":"test","type":"test"}`
	js, err := input.MarshalJSON()
	if err != nil {
		t.Errorf("Marshal:\n%#v\n\nWant:%s\nError:%s\n", input, want, err.Error())
	}
	if have := string(js); want != have {
		t.Errorf("Marshal:\n%#v\n\nWant:%s\nHave:%s\n", input, want, have)
	}

	// Data Base64
	input.Data = []byte(`not "valid" json`)
	want = `{"data_base64":"bm90ICJ2YWxpZCIganNvbg==","hi":"test","id":"test","source":"test","specversion":"test","type":"test"}`
	js, err = input.MarshalJSON()
	if err != nil {
		t.Errorf("Marshal:\n%#v\n\nWant:%s\nError:%s\n", input, want, err.Error())
	}
	if have := string(js); want != have {
		t.Errorf("Marshal:\n%#v\n\nWant:%s\nHave:%s\n", input, want, have)
	}

	// All fields
	input.Data = []byte(`x`)
	input.DataContentType = "a"
	input.DataSchema = "b"
	input.Subject = "c"
	input.Time, err = time.Parse(time.RFC3339, "2020-02-02T06:06:06Z")
	if err != nil {
		t.Errorf("Test error parsing time: %s", err.Error())
	}

	want = `{"data_base64":"eA==","datacontenttype":"a","dataschema":"b","hi":"test","id":"test","source":"test","specversion":"test","subject":"c","time":"2020-02-02T06:06:06Z","type":"test"}`
	js, err = input.MarshalJSON()
	if err != nil {
		t.Errorf("Marshal:\n%#v\n\nWant:%s\nError:%s\n", input, want, err.Error())
	}
	if have := string(js); want != have {
		t.Errorf("Marshal:\n%#v\n\nWant:%s\nHave:%s\n", input, want, have)
	}
}
