package jsonce

import (
	"fmt"
	"testing"
	"time"
)

type ValidResult struct {
	Err   error
	Warns []error
}
type ValidateScenarios struct { // TODO: Make interface
	T      *testing.T
	Name   string
	Input  CloudEvent
	Result ValidResult

	Warns func(...string) ValidateScenarios
	Err   func(string) ValidateScenarios
}

func (s ValidateScenarios) Failf(msg string, arg ...interface{}) {
	s.T.Fatalf("%s\nValidate:%s:\n%s\n", s.Input, s.Name, fmt.Sprintf(msg, arg...))
}
func (s ValidateScenarios) Logf(msg string, arg ...interface{}) {
	s.T.Logf("Validate:%s:\n\t%s\n", s.Name, fmt.Sprintf(msg, arg...))
}

func excludeFrom(A, B []string) (C []string) {
	for _, a := range A {
		if !InSlice(a, B) {
			C = append(C, a)
		}
	}
	return
}

// TestValidator ensures that the validator works as expected
func TestValidator(t *testing.T) {
	validate := func(name string, ce CloudEvent) ValidateScenarios {
		warns, err := ce.Valid()
		warnings := []string{}
		for _, W := range warns {
			// Assuming != nil because that would also be an error.
			// If a stacktrace brought you here, fix .Valid() instead.
			warnings = append(warnings, W.Error())
		}

		s := ValidateScenarios{
			T:     t,
			Name:  name,
			Input: ce,
			Result: ValidResult{
				Err:   err,
				Warns: warns,
			},
		}

		s.Warns = func(expect ...string) ValidateScenarios {
			errors := ""
			unexpected := excludeFrom(warnings, expect)
			missing := excludeFrom(expect, warnings)
			s.Logf("%d/%d expected warnings present, %d unexpected, %d missing",
				len(warns),
				len(expect),
				len(unexpected),
				len(missing),
			)

			for _, w := range unexpected {
				errors = fmt.Sprintf("%s\n\tUnexpected warning: %s", errors, w)
			}
			for _, e := range missing {
				errors = fmt.Sprintf("%s\n\tExpected warning not found: %s", errors, e)
			}
			if len(errors) > 0 {
				s.Failf("Warnings did not match expectation:%s", errors)
			}
			return s
		}
		s.Err = func(expect string) ValidateScenarios {
			if len(expect) == 0 {
				if err == nil {
					return s
				}
				s.Failf("Want no error\nHave: %s", err.Error())
			}
			if err == nil {
				s.Failf("Want: %s\nHave: no error", expect)
			}
			if err.Error() != expect {
				s.Failf("Want: %s\nHave: %s", expect, err.Error())
			}
			return s
		}
		return s
	}

	var ce CloudEvent

	// Valid
	ce = GenerateValidEvents(1)[0]
	validate("Valid", ce).Warns().Err("")

	// Id
	ce = GenerateValidEvents(1)[0]
	ce.Id = ""
	validate("No Id", ce).Warns().Err("Required field Id is empty")
	ce.Id = "te\nst"
	validate("Newline Id", ce).Warns("Id contains newline character").Err("")

	// Source
	ce = GenerateValidEvents(1)[0]
	ce.Source = ""
	validate("No Source", ce).Warns().Err("Required field Source is empty")
	ce.Source = "te\nst"
	validate("Newline Source", ce).Warns(
		"Source is not a URI: parse te\nst: net/url: invalid control character in URL",
	).Err("")

	ce.Id = "te\nst"
	validate("Newline Id and Source", ce).Warns(
		"Source is not a URI: parse te\nst: net/url: invalid control character in URL",
		"Id contains newline character",
	).Err("")

	// SpecVersion
	ce = GenerateValidEvents(1)[0]
	ce.SpecVersion = ""
	validate("No SpecVersion", ce).Warns().Err("Required field SpecVersion is empty")
	ce.SpecVersion = "te\nst"
	validate("Newline Id but Newline SpecVersion", ce).Warns("SpecVersion contains newline character").Err("")

	// Type
	ce = GenerateValidEvents(1)[0]
	ce.Type = ""
	validate("No Type", ce).Warns().Err("Required field Type is empty")
	ce.Type = "\n"
	validate("Newline Type", ce).Warns("Type contains newline character").Err("")

	// DataContentType
	ce = GenerateValidEvents(1)[0]
	ce.DataContentType = ""
	validate("No DataContentType", ce).Warns().Err("")
	ce.DataContentType = ";@;';$&{)_"
	validate("Ugly DataContentType", ce).Warns().Err("")
	ce.DataContentType = "\n"
	validate("Newline DataContentType", ce).Warns("DataContentType contains newline character").Err("")

	// DataSchema
	ce = GenerateValidEvents(1)[0]
	ce.DataSchema = ""
	validate("No DataSchema", ce).Warns().Err("")
	ce.DataSchema = "@;@;';$&{)_#"
	validate("Ugly DataSchema", ce).Warns().Err("")
	ce.DataSchema = "\n"
	validate("Newline DataSchema", ce).Warns().Err("DataSchema is not a URI: parse \n: net/url: invalid control character in URL")
	ce.DataSchema = "\v"
	validate("Control character DataSchema", ce).Warns().Err("DataSchema is not a URI: parse \v: net/url: invalid control character in URL")
	ce.DataSchema = ":"
	validate("No scheme DataSchema", ce).Warns().Err("DataSchema is not a URI: parse :: missing protocol scheme")

	// Subject
	ce = GenerateValidEvents(1)[0]
	ce.Subject = ""
	validate("No Subject", ce).Warns().Err("")
	ce.Subject = "te\nst"
	validate("Newline Subject", ce).Warns("Subject contains newline character").Err("")

	// Time
	ce = GenerateValidEvents(1)[0]
	ce.Time = time.Time{}
	validate("No Time", ce).Warns("Time is zero").Err("")

	// Extensions
	ce = GenerateValidEvents(1)[0]
	ce.Extensions = map[string]interface{}{}
	validate("No Extensions", ce).Warns().Err("")
	ce.Extensions = map[string]interface{}{"id": "test"}
	validate("Invalid Extensions", ce).Warns().Err("Extension id: not allowed")
	ce.Extensions = map[string]interface{}{"test\n": "test\n"}
	validate("Multiline Extensions", ce).Warns(
		"Extension test\n: name contains newline character",
		"Extension test\n: value contains newline character",
	).Err("")

	// Data
	ce = GenerateValidEvents(1)[0]
	ce.Data = []byte("")
	validate("No Data", ce).Warns().Err("")

	// All Warnings
	ce = GenerateValidEvents(1)[0]
	ce.Id = "te\nst"
	ce.Source = "%||"
	ce.SpecVersion = "te\nst"
	ce.Type = "\n"
	ce.DataContentType = "\n"
	ce.Subject = "\n"
	ce.Time = time.Time{}
	ce.Extensions = map[string]interface{}{
		"test\n": "test\n",
	}
	validate("All warnings", ce).Warns(
		"Id contains newline character",
		"Source is not a URI: parse %||: invalid URL escape \"%||\"",
		"SpecVersion contains newline character",
		"Type contains newline character",
		"DataContentType contains newline character",
		"Subject contains newline character",
		"Time is zero",
		"Extension test\n: name contains newline character",
		"Extension test\n: value contains newline character",
	).Err("")

	// Extension name Errors
	ce = GenerateValidEvents(1)[0]
	for _, ext := range ContextProperties {
		ce.Extensions = map[string]interface{}{
			ext: "\n",
		}
		validate(fmt.Sprintf("All errors: %s", ext), ce).Warns().Err(fmt.Sprintf("Extension %s: not allowed", ext))
	}
}

// TestGenerateValidEvents users the validator to test GenerateValidEvents
func TestGenerateValidEvents(t *testing.T) {
	for i, ce := range GenerateValidEvents(100) {
		warns, err := ce.Valid()
		if err != nil {
			t.Fatalf("Event %d validation error: %s", i, err.Error())
		}
		for j, w := range warns {
			t.Logf("Event %d validation warning [%d/%d]: %s", i, j, len(warns), w.Error())
		}
	}
}
