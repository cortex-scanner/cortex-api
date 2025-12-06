package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathValidationFail(t *testing.T) {
	testReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	testReq.SetPathValue("id", "a")

	_, err := ValidateString(testReq.PathValue("id"), Length(2, AnyLength)).Validate()
	assert.Error(t, err)
}

func TestPathValidationSuccess(t *testing.T) {
	testReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	testReq.SetPathValue("id", "ab")

	id, err := ValidateString(testReq.PathValue("id"), Length(2, AnyLength)).Validate()

	assert.NoError(t, err)
	assert.Equal(t, "ab", id)
}

func TestMultipleValidatorsFail(t *testing.T) {
	testReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	testReq.SetPathValue("id", "abc")

	_, err := ValidateString(
		testReq.PathValue("id"),
		Length(2, AnyLength),
		Regex("^ab$")).Validate()

	assert.Error(t, err)
}

func TestMultipleValidatorsSuccess(t *testing.T) {
	testReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	testReq.SetPathValue("id", "abc")

	id, err := ValidateString(
		testReq.PathValue("id"),
		Length(2, AnyLength),
		Regex("^abc$")).Validate()

	assert.NoError(t, err)
	assert.Equal(t, "abc", id)
}

func TestLengthValidator(t *testing.T) {
	testStr := "test"

	err := Length(2, AnyLength)(testStr)
	assert.NoError(t, err)

	err = Length(5, AnyLength)(testStr)
	assert.Error(t, err)

	err = Length(2, 4)(testStr)
	assert.NoError(t, err)

	err = Length(2, 3)(testStr)
	assert.Error(t, err)

	err = Length(AnyLength, AnyLength)(testStr)
	assert.NoError(t, err)
}

func TestRegexValidator(t *testing.T) {
	testStr := "test123"

	err := Regex("^test$")(testStr)
	assert.Error(t, err)

	err = Regex("^test.*$")(testStr)
	assert.NoError(t, err)

	err = Regex("^tx$")(testStr)
	assert.Error(t, err)
}

func TestUUIDValidator(t *testing.T) {
	validUUID := "8283d6df-3738-42ac-aaf0-6b97d509a2d8"
	err := UUID()(validUUID)
	assert.NoError(t, err)

	invalidUUID := "invalid-uuid"
	err = UUID()(invalidUUID)
	assert.Error(t, err)
}
