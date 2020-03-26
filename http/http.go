package http

import "fmt"

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
