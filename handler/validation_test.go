package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestRequiredValidator(t *testing.T) {
	err := Required()("test")
	assert.NoError(t, err)

	err = Required()("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is required")

	err = Required()(nil)
	assert.Error(t, err)
}

func TestInValidator(t *testing.T) {
	err := In("foo", "bar", "baz")("foo")
	assert.NoError(t, err)

	err = In("foo", "bar", "baz")("qux")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be one of")
}

func TestValidateStruct(t *testing.T) {
	type User struct {
		Username string
		Password string
		Email    string
	}

	user := User{
		Username: "john",
		Password: "secret123",
		Email:    "john@example.com",
	}

	err := ValidateStruct(
		fieldRulesCompat("Username", user.Username, Required(), Length(3, 20)),
		fieldRulesCompat("Password", user.Password, Required(), Length(8, AnyLength)),
		fieldRulesCompat("Email", user.Email, Required()),
	)

	assert.NoError(t, err)
}

func TestValidateStructFailure(t *testing.T) {
	type User struct {
		Username string
		Password string
	}

	user := User{
		Username: "ab",
		Password: "",
	}

	err := ValidateStruct(
		fieldRulesCompat("Username", user.Username, Required(), Length(3, 20)),
		fieldRulesCompat("Password", user.Password, Required()),
	)

	assert.Error(t, err)

	structErr, ok := err.(StructValidationError)
	assert.True(t, ok)
	assert.Len(t, structErr.Errors, 2)
	assert.Contains(t, structErr.Errors, "Username")
	assert.Contains(t, structErr.Errors, "Password")
}

func TestValidateStructMultipleRulesStopAtFirst(t *testing.T) {
	type User struct {
		Username string
	}

	user := User{
		Username: "",
	}

	err := ValidateStruct(
		fieldRulesCompat("Username", user.Username, Required(), Length(3, 20), UUID()),
	)

	assert.Error(t, err)

	structErr, ok := err.(StructValidationError)
	assert.True(t, ok)
	assert.Len(t, structErr.Errors, 1)

	// Should fail on Required, not Length or UUID
	assert.Contains(t, structErr.Errors["Username"].Error(), "is required")
}

func TestValidateStructPartialFailure(t *testing.T) {
	type User struct {
		Username string
		Email    string
	}

	user := User{
		Username: "john",
		Email:    "",
	}

	err := ValidateStruct(
		fieldRulesCompat("Username", user.Username, Required()),
		fieldRulesCompat("Email", user.Email, Required()),
	)

	assert.Error(t, err)

	structErr, ok := err.(StructValidationError)
	assert.True(t, ok)
	assert.Len(t, structErr.Errors, 1)
	assert.Contains(t, structErr.Errors, "Email")
	assert.NotContains(t, structErr.Errors, "Username")
}

func TestFieldFunction(t *testing.T) {
	field := fieldRulesCompat("TestField", "testvalue", Required(), Length(5, 20))

	assert.Equal(t, "TestField", field.FieldName)
	assert.Equal(t, "testvalue", field.Value)
	assert.Len(t, field.Rules, 2)
}

func TestStructValidationErrorMessage(t *testing.T) {
	type User struct {
		Username string
		Password string
	}

	user := User{
		Username: "",
		Password: "short",
	}

	err := ValidateStruct(
		fieldRulesCompat("Username", user.Username, Required()),
		fieldRulesCompat("Password", user.Password, Length(8, AnyLength)),
	)

	assert.Error(t, err)
	errMsg := err.Error()
	assert.Contains(t, errMsg, "validation failed")
	assert.Contains(t, errMsg, "Username")
	assert.Contains(t, errMsg, "Password")
}

func TestValidateRequestBodySuccess(t *testing.T) {
	type LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	body := `{"username":"john","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))

	var result LoginRequest
	err := ValidateRequestBody(req, &result,
		Field(&result.Username, Required(), Length(3, 20)),
		Field(&result.Password, Required(), Length(8, AnyLength)),
	)

	assert.NoError(t, err)
	assert.Equal(t, "john", result.Username)
	assert.Equal(t, "secret123", result.Password)
}

func TestValidateRequestBodyValidationFailure(t *testing.T) {
	type LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	body := `{"username":"ab","password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))

	var result LoginRequest
	err := ValidateRequestBody(req, &result,
		Field(&result.Username, Required(), Length(3, 20)),
		Field(&result.Password, Required(), Length(8, AnyLength)),
	)

	assert.Error(t, err)

	structErr, ok := err.(StructValidationError)
	assert.True(t, ok)
	assert.Len(t, structErr.Errors, 2)
	assert.Contains(t, structErr.Errors, "username")
	assert.Contains(t, structErr.Errors, "password")
}

func TestValidateRequestBodyInvalidJSON(t *testing.T) {
	type LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	body := `{"username":"john",`
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))

	var result LoginRequest
	err := ValidateRequestBody(req, &result,
		Field(&result.Username, Required()),
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestValidateRequestBodyEmptyFields(t *testing.T) {
	type LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	body := `{"username":"","password":""}`
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))

	var result LoginRequest
	err := ValidateRequestBody(req, &result,
		Field(&result.Username, Required()),
		Field(&result.Password, Required()),
	)

	assert.Error(t, err)

	structErr, ok := err.(StructValidationError)
	assert.True(t, ok)
	assert.Len(t, structErr.Errors, 2)
}

func TestValidateRequestBodyPartialValidation(t *testing.T) {
	type LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	body := `{"username":"john","password":"secret123","email":""}`
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))

	var result LoginRequest
	err := ValidateRequestBody(req, &result,
		Field(&result.Username, Required()),
		Field(&result.Password, Required()),
		Field(&result.Email, Required()),
	)

	assert.Error(t, err)

	structErr, ok := err.(StructValidationError)
	assert.True(t, ok)
	assert.Len(t, structErr.Errors, 1)
	assert.Contains(t, structErr.Errors, "email")
	assert.NotContains(t, structErr.Errors, "username")
	assert.NotContains(t, structErr.Errors, "password")
}

func TestValidateRequestBodyAutoDerivesJSONFieldNames(t *testing.T) {
	type CreateUserRequest struct {
		UserName string `json:"user_name"`
		Email    string `json:"email_address"`
	}

	body := `{"user_name":"","email_address":"invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))

	var result CreateUserRequest
	err := ValidateRequestBody(req, &result,
		Field(&result.UserName, Required()),
		Field(&result.Email, Required(), Regex(`^[^@]+@[^@]+\.[^@]+$`)),
	)

	assert.Error(t, err)

	structErr, ok := err.(StructValidationError)
	assert.True(t, ok)
	assert.Len(t, structErr.Errors, 2)
	// Should use JSON tag names, not struct field names
	assert.Contains(t, structErr.Errors, "user_name")
	assert.Contains(t, structErr.Errors, "email_address")
}

func TestValidateRequestBodyNoJSONTag(t *testing.T) {
	type SimpleRequest struct {
		Name string // No json tag
	}

	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))

	var result SimpleRequest
	err := ValidateRequestBody(req, &result,
		Field(&result.Name, Required()),
	)

	assert.Error(t, err)

	structErr, ok := err.(StructValidationError)
	assert.True(t, ok)
	// Should use lowercase field name when no json tag
	assert.Contains(t, structErr.Errors, "name")
}

func TestValidateRequestBodyJSONTagWithOmitempty(t *testing.T) {
	type UserRequest struct {
		Name string `json:"name,omitempty"`
	}

	body := `{"name":"ab"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))

	var result UserRequest
	err := ValidateRequestBody(req, &result,
		Field(&result.Name, Length(3, 20)),
	)

	assert.Error(t, err)

	structErr, ok := err.(StructValidationError)
	assert.True(t, ok)
	// Should extract "name" from "name,omitempty"
	assert.Contains(t, structErr.Errors, "name")
}
