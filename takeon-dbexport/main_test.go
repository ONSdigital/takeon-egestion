package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInputMessageReturnsExpected(t *testing.T) {
	tests := []struct {
		jsonMessage     []byte
		expectedMessage []byte
	}{
		{validJSONMessage, validJSONMessage},
	}

	var testInput InputJSON
	var expected InputJSON

	for _, test := range tests {
		json.Unmarshal(test.jsonMessage, &testInput)
		json.Unmarshal(test.expectedMessage, &expected)
		var actual, _ = validateInputMessage(testInput)
		assert.EqualValues(t, actual, expected)
	}
}

var validJSONMessage = []byte(`
{
	"snapshot_id": "14e0fb27-d450-44d4-8452-9f6996b00e27",
	"surveyperiods": [
	  {
		"survey": "023",
		"period": "201904"
	  },
	  {
		"survey": "023",
		"period": "201903"
	  }
	]
  }`)
