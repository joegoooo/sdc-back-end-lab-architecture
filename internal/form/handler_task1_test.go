//go:build task1

package form_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sdclab/internal/form"
	"sdclab/internal/form/mocks"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

// dbLeakError 模擬一個帶著資料表 / 索引名稱的資料庫錯誤。
// 這種內容不可以出現在回應裡。
var dbLeakError = errors.New(`ERROR: duplicate key value violates unique constraint "forms_title_key" (SQLSTATE 23505)`)

func TestHandler_Create(t *testing.T) {
	createdForm := form.Form{
		ID:          uuid.New(),
		Title:       "問卷調查",
		Description: pgtype.Text{String: "說明文字", Valid: true},
	}

	testCases := []struct {
		name           string
		body           form.CreateRequest
		customBody     []byte // 給不合法 JSON 用
		setupMock      func(store *mocks.Store)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Should create form",
			body: form.CreateRequest{Title: "問卷調查", Description: "說明文字"},
			setupMock: func(store *mocks.Store) {
				store.On("Create", mock.Anything, "問卷調查", "說明文字").Return(createdForm, nil).Once()
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Should return error when request body is invalid JSON",
			customBody:     []byte(`{"title": "問卷調查",}`),
			setupMock:      func(store *mocks.Store) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request body",
		},
		{
			name:           "Should return error when title is empty",
			body:           form.CreateRequest{Description: "說明文字"},
			setupMock:      func(store *mocks.Store) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "title is required",
		},
		{
			name:           "Should return error when title is only whitespace",
			body:           form.CreateRequest{Title: "   ", Description: "說明文字"},
			setupMock:      func(store *mocks.Store) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "title is required",
		},
		{
			name: "Should accept title with exactly 255 characters",
			body: form.CreateRequest{Title: strings.Repeat("a", 255)},
			setupMock: func(store *mocks.Store) {
				store.On("Create", mock.Anything, strings.Repeat("a", 255), "").Return(createdForm, nil).Once()
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Should return error when title exceeds 255 characters",
			body:           form.CreateRequest{Title: strings.Repeat("a", 256)},
			setupMock:      func(store *mocks.Store) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "title must be at most 255 characters",
		},
		{
			name:           "Should return error when description exceeds 1000 characters",
			body:           form.CreateRequest{Title: "問卷調查", Description: strings.Repeat("a", 1001)},
			setupMock:      func(store *mocks.Store) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Should return conflict when title already exists",
			body: form.CreateRequest{Title: "問卷調查", Description: "說明文字"},
			setupMock: func(store *mocks.Store) {
				store.On("Create", mock.Anything, "問卷調查", "說明文字").
					Return(form.Form{}, form.ErrTitleConflict).Once()
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "form title already exists",
		},
		{
			name: "Should return internal server error when store fails",
			body: form.CreateRequest{Title: "問卷調查", Description: "說明文字"},
			setupMock: func(store *mocks.Store) {
				store.On("Create", mock.Anything, "問卷調查", "說明文字").
					Return(form.Form{}, dbLeakError).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "internal server error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewStore(t)
			if tc.setupMock != nil {
				tc.setupMock(store)
			}

			h := form.NewHandler(zaptest.NewLogger(t), store)

			requestBody := tc.customBody
			if requestBody == nil {
				requestBody = mustMarshal(t, tc.body)
			}

			r := httptest.NewRequest(http.MethodPost, "/api/forms", bytes.NewReader(requestBody))
			w := httptest.NewRecorder()

			h.Create(w, r)

			assert.Equal(t, tc.expectedStatus, w.Code, tc.name)
			if tc.expectedError != "" {
				assertErrorBody(t, w, tc.expectedError)
			}
		})
	}
}

// TestHandler_Create_ResponseShape 檢查成功回應的形狀：
// 欄位齊全、archived 不外洩、Content-Type 正確。
func TestHandler_Create_ResponseShape(t *testing.T) {
	createdForm := form.Form{
		ID:          uuid.MustParse("dfdf7650-9d8c-4be6-bff2-6c89a70ddccd"),
		Title:       "問卷調查",
		Description: pgtype.Text{String: "說明文字", Valid: true},
		Archived:    true, // 就算資料層是 true，也不該出現在回應裡
	}

	store := mocks.NewStore(t)
	store.On("Create", mock.Anything, "問卷調查", "說明文字").Return(createdForm, nil).Once()

	h := form.NewHandler(zaptest.NewLogger(t), store)

	body := mustMarshal(t, form.CreateRequest{Title: "問卷調查", Description: "說明文字"})
	r := httptest.NewRequest(http.MethodPost, "/api/forms", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Create(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	decoded := decodeBody(t, w)
	assert.Equal(t, "dfdf7650-9d8c-4be6-bff2-6c89a70ddccd", decoded["id"])
	assert.Equal(t, "問卷調查", decoded["title"])
	assert.Equal(t, "說明文字", decoded["description"])
	assert.Contains(t, decoded, "createdAt")
	assert.NotContains(t, decoded, "archived")
}

// TestHandler_Create_NullDescription：資料庫是 NULL 時要回空字串，不是 null。
func TestHandler_Create_NullDescription(t *testing.T) {
	store := mocks.NewStore(t)
	store.On("Create", mock.Anything, "問卷調查", "").Return(form.Form{
		ID:          uuid.New(),
		Title:       "問卷調查",
		Description: pgtype.Text{Valid: false},
	}, nil).Once()

	h := form.NewHandler(zaptest.NewLogger(t), store)

	body := mustMarshal(t, form.CreateRequest{Title: "問卷調查"})
	r := httptest.NewRequest(http.MethodPost, "/api/forms", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Create(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "", decodeBody(t, w)["description"])
}

// TestHandler_Create_DoesNotLeakDatabaseError：500 只回通用訊息，
// 不可以把資料表 / 索引名稱吐出去。
func TestHandler_Create_DoesNotLeakDatabaseError(t *testing.T) {
	store := mocks.NewStore(t)
	store.On("Create", mock.Anything, "問卷調查", "").Return(form.Form{}, dbLeakError).Once()

	h := form.NewHandler(zaptest.NewLogger(t), store)

	body := mustMarshal(t, form.CreateRequest{Title: "問卷調查"})
	r := httptest.NewRequest(http.MethodPost, "/api/forms", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Create(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assertErrorBody(t, w, "internal server error")
	assert.NotContains(t, w.Body.String(), "forms_title_key")
	assert.NotContains(t, w.Body.String(), "SQLSTATE")
}
