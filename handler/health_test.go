package handler_test

import (
	"cortex/handler"
	"cortex/test"
	"net/http"
	"testing"
)

func TestHealthy(t *testing.T) {
	runner := test.NewTestRunner(handler.HandleHealth)
	res := runner.Run(t).ExpectNoError().ExpectStatusCode(http.StatusOK)
	test.AssertSingleAPIResponse(res, "OK")
}
