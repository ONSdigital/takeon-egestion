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

func TestGetFileName_GivenOneSurveyPeriod_OutputCorrectFileName(t *testing.T){
	var expected = "snapshot-023_201902-fwwekfnsdn"
	var snapshotID = "fwwekfnsdn"
	var singleSurveyPeriod = []SurveyPeriods {SurveyPeriods{Survey:"023", Period:"201902"}}
	var actual, _ = getFileName(snapshotID, singleSurveyPeriod)

	assert.EqualValues(t, expected, actual)
}


func TestGetFileName_GivenTwoSurveyPeriod_OutputCorrectFileName(t *testing.T){
	var expected = "snapshot-066_201902-023_201902-fwwekfnsdn"
	var snapshotID = "fwwekfnsdn"
	var singleSurveyPeriod = []SurveyPeriods {SurveyPeriods{Survey:"066", Period:"201902"}, SurveyPeriods{Survey:"023", Period:"201902"} }
	var actual, _ = getFileName(snapshotID, singleSurveyPeriod)

	assert.EqualValues(t, expected, actual)
}


func TestGetFileName_GivenMultipleSurveyPeriod_OutputCorrectFileName(t *testing.T){
	var expected = "snapshot-066_201902-023_201902-023_201904-fwwekfnsdn"
	var snapshotID = "fwwekfnsdn"
	var singleSurveyPeriod = []SurveyPeriods {SurveyPeriods{Survey:"066", Period:"201902"}, SurveyPeriods{Survey:"023", Period:"201902"}, SurveyPeriods{Survey:"023", Period:"201904"} }
	var actual, _ = getFileName(snapshotID, singleSurveyPeriod)

	assert.EqualValues(t, expected, actual)
}


func TestGetFileName_GivenBlankeSurveyPeriod_OutputError(t *testing.T){
	var snapshotID = "fwwekfnsdn"
	var singleSurveyPeriod = []SurveyPeriods {}
	var actual, err = getFileName(snapshotID, singleSurveyPeriod)

	assert.EqualValues(t, "", actual)

	assert.Error(t, err)
}


func TestGetFileName_GivenNullSurveyPeriod_OutputError(t *testing.T){
	var snapshotID = "fwwekfnsdn"
	var actual, err = getFileName(snapshotID, nil)

	assert.EqualValues(t, "", actual)

	assert.Error(t, err)
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
	  },
	  {
		"survey": "066",
		"period": "201903"
	  },
	  {
		"survey": "067",
		"period": "201904"
	  }
	]
  }`)
