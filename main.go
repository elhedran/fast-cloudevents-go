package main

import (
	"encoding/json"
	"fmt"

	cejson "github.com/elhedran/fast-cloudevents-go/json"
)

func main() {
	example := "{ \"id\": \"123\", \"ce-exten\": 123 }"

	singleEvent := cejson.JsonCloudEvent{}
	//	multEvent := []cejson.JsonCloudEvent{}

	err := json.Unmarshal([]byte(example), &singleEvent)
	if err != nil {
		fmt.Printf("Error %q\n", err)
	}
	fmt.Printf("event %q, %q\n", singleEvent, singleEvent.Extensions["ce-exten"])

}
