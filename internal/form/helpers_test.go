//go:build task1 || task2 || task3 || task4

package form_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ptr 用來組出「有帶這個欄位」的部分更新參數。
func ptr(s string) *string { return &s }

// newRequestWithID 建一個帶 path value 的 request。
// 直接呼叫 handler 方法時 ServeMux 不在中間，要自己 SetPathValue。
func newRequestWithID(method, target, id string, body []byte) *http.Request {
	var r *http.Request
	if body == nil {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, bytes.NewReader(body))
	}
	r.SetPathValue("id", id)

	return r
}

// assertErrorBody 檢查 ErrorResponse 的 error 欄位。
//
// Clustron 的 handler 測試多半只比 w.Code，但本 LAB 的 API_SPEC.md
// 對每個錯誤情境都指定了 error 字串，所以這裡一併比對。
func assertErrorBody(t *testing.T, w *httptest.ResponseRecorder, expected string) {
	t.Helper()

	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode error body %q: %v", w.Body.String(), err)
	}

	assert.Equal(t, expected, body.Error)
}

// decodeBody 把回應 body 解成 map，用來檢查欄位有沒有多出來（例如 archived）。
func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode body %q: %v", w.Body.String(), err)
	}

	return body
}

// mustMarshal 把 request struct 轉成 body bytes。
func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()

	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	return b
}
