package main

import (
	"encoding/json"
	"testing"
	"fmt"
)
type Scenarios struct {
	T *testing.T
	Name string
	Input string
	Data CloudEvent
	Result error

	Ok func()CloudEvent
	Err func(string)error
}
func (s Scenarios) Failf(msg string, arg ...interface{}) {
        s.T.Errorf("Unmarshal:%s:\n%s\n\n%s\n", s.Name, s.Input, fmt.Sprintf(msg, arg...))
}
func (s Scenarios) Logf(msg string, arg ...interface{}) {
        s.T.Logf("Unmarshal:%s:\n\t%s\n", s.Name, fmt.Sprintf(msg, arg...))
}
func TestUnmarshal(t *testing.T){
        unmarshal := func(name, data string) Scenarios {
                ce := CloudEvent{}
		err := ce.UnmarshalJSON([]byte(data))
		s:= Scenarios {
			T: t,
			Name: name,
			Input: data,
			Data: ce,
			Result: err,
		}
		s.Ok=func()CloudEvent{
			if err!=nil{
				s.Failf("Want no error\nHave: %s", err.Error())
			}
			return ce
		}
		s.Err=func(expect string)error{
			if err==nil || err.Error() != expect {
			        s.Failf("Want: %s\nHave: %s", expect, err.Error())
			}
			return err
		}
		return s
        }

	// Required

	data := `{}`
        unmarshal("Need id", data).Err(errRead("ID","nonempty string"))
	data = `{"id":""}`
        unmarshal("Need id len", data).Err(errRead("ID","nonempty string"))
	data = `{"id":"a"}`
        unmarshal("Need source", data).Err(errRead("Source","nonempty string"))
	data = `{"id":"a","source":"b"}`
        unmarshal("Need version", data).Err(errRead("Spec Version","nonempty string"))
	data = `{"id":"a","source":"b","specversion":"c"}`
        unmarshal("Need type", data).Err(errRead("Type","nonempty string"))
	data = `{"id":"a","source":"b","specversion":"c","type":"d"}`
        unmarshal("Minimum", data).Ok()

	// Optional
	//
	// Time nano vs invalid vs ms
	//
	// Time nano vs invalid vs ms
	//
	// Time nano vs invalid vs ms
	//
	// Time nano vs invalid vs ms
	//
	// Time nano vs invalid vs ms

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
        unmarshal("IvalidTime", data).Err(`Could not read Time as time: parsing time "2020-02-02T06:06:60+25:00": second out of range`)

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
	if want := `{"x":[1,2,"3"]}` ; string(js) != want {
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
	if want := "123" ; string(js) != want {
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
	if want := "hello world" ; string(raw) != want {
		s.Failf("Want: %s\nHave: %s", want, string(raw))
	}
        s.Logf("%s", raw)
}

