package test

import (
	"cortex/handler"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

const expectedAPIVersion = 1

func AssertSingleAPIResponse[T any](result *Result, expectedContent T) {
	var responseData handler.SingleDataResponse[T]
	err := json.NewDecoder(result.RR.Body).Decode(&responseData)

	assert.NoError(result.t, err)
	assert.Equal(result.t, expectedAPIVersion, responseData.APIVersion)
	assert.Equal(result.t, expectedContent, responseData.Data)
}

func AssertJSON[T any](t *testing.T, jsonStr string, expected T) {
	var actual T
	err := json.Unmarshal([]byte(jsonStr), &actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}
