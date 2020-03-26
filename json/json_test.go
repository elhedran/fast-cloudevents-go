package json

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalSingle(t *testing.T) {
	example := "{ \"id\": \"123\", \"ce-exten\": 123 }"

	singleEvent := JsonCloudEvent{}

	err := json.Unmarshal([]byte(example), &singleEvent)

	if err != nil {
		t.Errorf("Could not unmarshal json: %q", err)
		t.FailNow()
	}
	if singleEvent.Id != "123" {
		t.Error("Id not set correctly")
	}

	ceExten, ok := singleEvent.Extensions["ce-exten"].(float64)

	if !ok || ceExten != float64(123) {
		t.Error("Extension field not set correctly")
	}
}

func TestUnmarshalMany(t *testing.T) {
	example := "[{ \"id\": \"123\", \"ce-exten\": 123 },{\"id\": \"345\"}]"
	manyEvents := make([]JsonCloudEvent, 0)

	err := json.Unmarshal([]byte(example), &manyEvents)
	if err != nil {
		t.Errorf("Could not unmarshal json: %q", err)
		t.FailNow()
	}
	if len(manyEvents) != 2 {
		t.Errorf("Incorrect number of events unmarshaled")
	}

	if manyEvents[0].Id != "123" {
		t.Error("First Id not set correctly")
	}

	if manyEvents[1].Id != "345" {
		t.Error("First Id not set correctly")
	}

	ceExten, ok := manyEvents[0].Extensions["ce-exten"].(float64)

	if !ok || ceExten != float64(123) {
		t.Error("Extension field not set correctly")
	}
}
