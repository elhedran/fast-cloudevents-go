package events

import (
	"fmt"
	"net/url"
	"strings"
	"time"
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

var (
	ErrorIdEmpty     = fmt.Errorf("Required field Id is empty")
	ErrorSourceEmpty = fmt.Errorf("Required field Source is empty")
	ErrorSpecEmpty   = fmt.Errorf("Required field SpecVersion is empty")
	ErrorTypeEmpty   = fmt.Errorf("Required field Type is empty")
)

func isURIOrEmpty(s string) error {
	// if len(s) == 0{
	// 	return nil
	// }
	_, err := url.Parse(s) // "" is a valid URL, could be stricter otherwise
	return err
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

// Valid returns nil, nil if the CloudEvent seems to fit the spec
// Valid returns error, nil if the CloudEvent seems valid but has warnings
// For use cases which differ from the CloudEvents spec,
// a custom validator can be used instead of calling this function
func (ce CloudEvent) Valid() (warns []error, err error) {
	if len(ce.Id) == 0 {
		err = ErrorIdEmpty
		return
	}
	if len(ce.Source) == 0 {
		err = ErrorSourceEmpty
		return
	}
	if len(ce.SpecVersion) == 0 {
		err = ErrorSpecEmpty
		return
	}
	if len(ce.Type) == 0 {
		err = ErrorTypeEmpty
		return
	}
	if err = isURIOrEmpty(ce.DataSchema); err != nil {
		err = fmt.Errorf("DataSchema is not a URI: %s", err.Error())
		return
	}
	for k, v := range ce.Extensions {
		if InSlice(k, ContextProperties) {
			err = fmt.Errorf("Extension %s: not allowed", k)
			return
		}
		// Multiline headers could be a warning (deprecated under RFC 7230)
		if strings.Contains(fmt.Sprintf("%s", k), "\n") {
			warns = append(warns, fmt.Errorf("Extension %s: name contains newline character", k))
		}
		if strings.Contains(fmt.Sprintf("%+v", v), "\n") {
			warns = append(warns, fmt.Errorf("Extension %s: value contains newline character", k))
		}
	}
	if w := isURIOrEmpty(ce.Source); w != nil {
		// Inherently checks for \n
		warns = append(warns, fmt.Errorf("Source is not a URI: %s", w.Error()))
	}
	if ce.Time.IsZero() {
		warns = append(warns, fmt.Errorf("Time is zero"))
	}
	if strings.Contains(ce.Id, "\n") {
		warns = append(warns, fmt.Errorf("Id contains newline character"))
	}
	if strings.Contains(ce.SpecVersion, "\n") {
		warns = append(warns, fmt.Errorf("SpecVersion contains newline character"))
	}
	if strings.Contains(ce.Type, "\n") {
		warns = append(warns, fmt.Errorf("Type contains newline character"))
	}
	if strings.Contains(ce.DataContentType, "\n") {
		warns = append(warns, fmt.Errorf("DataContentType contains newline character"))
	}
	if strings.Contains(ce.DataSchema, "\n") {
		warns = append(warns, fmt.Errorf("DataSchema contains newline character"))
	}
	if strings.Contains(ce.Subject, "\n") {
		warns = append(warns, fmt.Errorf("Subject contains newline character"))
	}

	// We can't check if the data is compatible with the DataContentType
	return warns, err
}
