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

// Numeric validation tests

func TestMinValidator(t *testing.T) {
	// Test int
	err := Min(5)(10)
	assert.NoError(t, err)

	err = Min(5)(5)
	assert.NoError(t, err)

	err = Min(5)(3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be at least 5")

	// Test int64
	err = Min(int64(100))(int64(200))
	assert.NoError(t, err)

	err = Min(int64(100))(int64(50))
	assert.Error(t, err)

	// Test float64
	err = Min(5.5)(10.2)
	assert.NoError(t, err)

	err = Min(5.5)(3.2)
	assert.Error(t, err)
}

func TestMaxValidator(t *testing.T) {
	// Test int
	err := Max(10)(5)
	assert.NoError(t, err)

	err = Max(10)(10)
	assert.NoError(t, err)

	err = Max(10)(15)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be at most 10")

	// Test int64
	err = Max(int64(100))(int64(50))
	assert.NoError(t, err)

	err = Max(int64(100))(int64(150))
	assert.Error(t, err)

	// Test float64
	err = Max(10.5)(5.2)
	assert.NoError(t, err)

	err = Max(10.5)(15.8)
	assert.Error(t, err)
}

func TestRangeValidator(t *testing.T) {
	// Test int
	err := Range(5, 10)(7)
	assert.NoError(t, err)

	err = Range(5, 10)(5)
	assert.NoError(t, err)

	err = Range(5, 10)(10)
	assert.NoError(t, err)

	err = Range(5, 10)(3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be between 5 and 10")

	err = Range(5, 10)(15)
	assert.Error(t, err)

	// Test int64
	err = Range(int64(0), int64(100))(int64(50))
	assert.NoError(t, err)

	err = Range(int64(0), int64(100))(int64(150))
	assert.Error(t, err)

	// Test float64
	err = Range(0.0, 1.0)(0.5)
	assert.NoError(t, err)

	err = Range(0.0, 1.0)(1.5)
	assert.Error(t, err)
}

// Slice/Array validation tests

func TestRequiredWithSlices(t *testing.T) {
	// Non-empty slice should pass
	err := Required()([]string{"a", "b"})
	assert.NoError(t, err)

	// Empty slice should fail
	err = Required()([]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is required")

	// Nil slice should fail
	var nilSlice []string
	err = Required()(nilSlice)
	assert.Error(t, err)
}

func TestRequiredWithMaps(t *testing.T) {
	// Non-empty map should pass
	err := Required()(map[string]string{"key": "value"})
	assert.NoError(t, err)

	// Empty map should fail
	err = Required()(map[string]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is required")

	// Nil map should fail
	var nilMap map[string]string
	err = Required()(nilMap)
	assert.Error(t, err)
}

func TestMinLengthValidator(t *testing.T) {
	// Test with slice
	err := MinItems(2)([]string{"a", "b", "c"})
	assert.NoError(t, err)

	err = MinItems(2)([]string{"a"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must have at least 2 element(s)")

	// Test with map
	err = MinItems(1)(map[string]int{"a": 1, "b": 2})
	assert.NoError(t, err)

	err = MinItems(3)(map[string]int{"a": 1})
	assert.Error(t, err)

	// Test with AnyLength
	err = MinItems(AnyLength)([]string{})
	assert.NoError(t, err)
}

func TestMaxLengthValidator(t *testing.T) {
	// Test with slice
	err := MaxItems(5)([]string{"a", "b", "c"})
	assert.NoError(t, err)

	err = MaxItems(2)([]string{"a", "b", "c"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must have at most 2 element(s)")

	// Test with map
	err = MaxItems(3)(map[string]int{"a": 1, "b": 2})
	assert.NoError(t, err)

	err = MaxItems(1)(map[string]int{"a": 1, "b": 2})
	assert.Error(t, err)

	// Test with AnyLength
	err = MaxItems(AnyLength)([]string{"a", "b", "c", "d", "e"})
	assert.NoError(t, err)
}

func TestEachValidator(t *testing.T) {
	// Test with valid slice elements
	err := Each(Length(2, 5))([]string{"ab", "abc", "abcd"})
	assert.NoError(t, err)

	// Test with invalid element
	err = Each(Length(2, 5))([]string{"ab", "a", "abc"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "element at index 1")
	assert.Contains(t, err.Error(), "must be at least 2 characters long")

	// Test with multiple rules
	err = Each(Required(), Length(3, 10))([]string{"abc", "defg"})
	assert.NoError(t, err)

	err = Each(Required(), Length(3, 10))([]string{"abc", ""})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "element at index 1")

	// Test with numeric slice
	err = Each(Min(0), Max(100))([]int{10, 50, 99})
	assert.NoError(t, err)

	err = Each(Min(0), Max(100))([]int{10, 150, 99})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "element at index 1")
}

func TestKeysValidator(t *testing.T) {
	// Test with valid keys
	m := map[string]int{"ab": 1, "abc": 2, "abcd": 3}
	err := Keys(Length(2, 5))(m)
	assert.NoError(t, err)

	// Test with invalid key
	m = map[string]int{"ab": 1, "a": 2}
	err = Keys(Length(2, 5))(m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key 'a'")

	// Test with multiple rules
	m = map[string]int{"test1": 1, "test2": 2}
	err = Keys(Required(), Length(3, 10))(m)
	assert.NoError(t, err)
}

func TestValuesValidator(t *testing.T) {
	// Test with valid values
	m := map[string]int{"a": 10, "b": 50, "c": 99}
	err := Values(Min(0), Max(100))(m)
	assert.NoError(t, err)

	// Test with invalid value
	m = map[string]int{"a": 10, "b": 150}
	err = Values(Min(0), Max(100))(m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "value for key")

	// Test with string values
	ms := map[string]string{"a": "test", "b": "hello"}
	err = Values(Length(3, 10))(ms)
	assert.NoError(t, err)

	ms = map[string]string{"a": "test", "b": "ab"}
	err = Values(Length(3, 10))(ms)
	assert.Error(t, err)
}

// Integration tests with ValidateRequestBody

func TestValidateRequestBodyWithNumericFields(t *testing.T) {
	type CreateUserRequest struct {
		Age    int     `json:"age"`
		Score  float64 `json:"score"`
		Points int64   `json:"points"`
	}

	body := `{"age":25,"score":85.5,"points":1000}`
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))

	var result CreateUserRequest
	err := ValidateRequestBody(req, &result,
		Field(&result.Age, Min(18), Max(120)),
		Field(&result.Score, Range(0.0, 100.0)),
		Field(&result.Points, Min(int64(0))),
	)

	assert.NoError(t, err)
	assert.Equal(t, 25, result.Age)
	assert.Equal(t, 85.5, result.Score)
	assert.Equal(t, int64(1000), result.Points)
}

func TestValidateRequestBodyWithNumericFieldsFailure(t *testing.T) {
	type CreateUserRequest struct {
		Age   int     `json:"age"`
		Score float64 `json:"score"`
	}

	body := `{"age":15,"score":105.5}`
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))

	var result CreateUserRequest
	err := ValidateRequestBody(req, &result,
		Field(&result.Age, Min(18), Max(120)),
		Field(&result.Score, Range(0.0, 100.0)),
	)

	assert.Error(t, err)

	structErr, ok := err.(StructValidationError)
	assert.True(t, ok)
	assert.Len(t, structErr.Errors, 2)
	assert.Contains(t, structErr.Errors, "age")
	assert.Contains(t, structErr.Errors, "score")
}

func TestValidateRequestBodyWithSliceFields(t *testing.T) {
	type TagsRequest struct {
		Tags []string `json:"tags"`
	}

	body := `{"tags":["security","compliance","audit"]}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))

	var result TagsRequest
	err := ValidateRequestBody(req, &result,
		Field(&result.Tags, Required(), MinItems(1), MaxItems(10), Each(Length(3, 20))),
	)

	assert.NoError(t, err)
	assert.Len(t, result.Tags, 3)
}

func TestValidateRequestBodyWithSliceFieldsFailure(t *testing.T) {
	type TagsRequest struct {
		Tags []string `json:"tags"`
	}

	body := `{"tags":["ab","security"]}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))

	var result TagsRequest
	err := ValidateRequestBody(req, &result,
		Field(&result.Tags, Each(Length(3, 20))),
	)

	assert.Error(t, err)

	structErr, ok := err.(StructValidationError)
	assert.True(t, ok)
	assert.Contains(t, structErr.Errors, "tags")
	assert.Contains(t, structErr.Errors["tags"].Error(), "element at index 0")
}

func TestValidateRequestBodyWithMapFields(t *testing.T) {
	type MetadataRequest struct {
		Metadata map[string]string `json:"metadata"`
	}

	body := `{"metadata":{"env":"production","region":"us-east-1"}}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))

	var result MetadataRequest
	err := ValidateRequestBody(req, &result,
		Field(&result.Metadata, Required(), MinItems(1), Keys(Length(2, 20)), Values(Length(1, 50))),
	)

	assert.NoError(t, err)
	assert.Len(t, result.Metadata, 2)
}

func TestValidateRequestBodyWithMapFieldsFailure(t *testing.T) {
	type MetadataRequest struct {
		Metadata map[string]string `json:"metadata"`
	}

	body := `{"metadata":{"e":"production","region":"us-east-1"}}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))

	var result MetadataRequest
	err := ValidateRequestBody(req, &result,
		Field(&result.Metadata, Keys(Length(2, 20))),
	)

	assert.Error(t, err)

	structErr, ok := err.(StructValidationError)
	assert.True(t, ok)
	assert.Contains(t, structErr.Errors, "metadata")
	assert.Contains(t, structErr.Errors["metadata"].Error(), "key 'e'")
}

func TestValidateRequestBodyComplexExample(t *testing.T) {
	type CreateScanConfigRequest struct {
		Name        string            `json:"name"`
		Description string            `json:"description"`
		MaxThreads  int               `json:"max_threads"`
		Timeout     float64           `json:"timeout"`
		Tags        []string          `json:"tags"`
		Metadata    map[string]string `json:"metadata"`
	}

	body := `{
		"name":"Security Scan",
		"description":"Full security scan configuration",
		"max_threads":10,
		"timeout":300.5,
		"tags":["security","compliance"],
		"metadata":{"env":"prod","region":"us-east-1"}
	}`
	req := httptest.NewRequest(http.MethodPost, "/scan-configs", strings.NewReader(body))

	var result CreateScanConfigRequest
	err := ValidateRequestBody(req, &result,
		Field(&result.Name, Required(), Length(3, 100)),
		Field(&result.Description, Length(0, 500)),
		Field(&result.MaxThreads, Min(1), Max(100)),
		Field(&result.Timeout, Range(0.0, 3600.0)),
		Field(&result.Tags, MinItems(1), MaxItems(20), Each(Length(2, 50))),
		Field(&result.Metadata, MinItems(0), MaxItems(50), Keys(Length(1, 100)), Values(Length(1, 500))),
	)

	assert.NoError(t, err)
	assert.Equal(t, "Security Scan", result.Name)
	assert.Equal(t, 10, result.MaxThreads)
	assert.Equal(t, 300.5, result.Timeout)
	assert.Len(t, result.Tags, 2)
	assert.Len(t, result.Metadata, 2)
}
