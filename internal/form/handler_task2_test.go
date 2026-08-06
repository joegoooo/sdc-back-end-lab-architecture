//go:build task2

package form_test

import (
	"fmt"
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

func TestHandler_Get(t *testing.T) {
	id := uuid.New()
	existing := form.Form{
		ID:          id,
		Title:       "問卷調查",
		Description: pgtype.Text{String: "說明文字", Valid: true},
		Archived:    true,
	}

	testCases := []struct {
		name           string
		id             string
		setupMock      func(store *mocks.Store)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Should return form",
			id:   id.String(),
			setupMock: func(store *mocks.Store) {
				store.On("GetByID", mock.Anything, id).Return(existing, nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Should return error when id is not a valid UUID",
			id:             "not-a-uuid",
			setupMock:      func(store *mocks.Store) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid id",
		},
		{
			name: "Should return not found when form does not exist",
			id:   id.String(),
			setupMock: func(store *mocks.Store) {
				store.On("GetByID", mock.Anything, id).Return(form.Form{}, form.ErrFormNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "form not found",
		},
		{
			name: "Should return internal server error when store fails",
			id:   id.String(),
			setupMock: func(store *mocks.Store) {
				store.On("GetByID", mock.Anything, id).Return(form.Form{}, assert.AnError).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "internal server error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewStore(t)
			tc.setupMock(store)

			h := form.NewHandler(zaptest.NewLogger(t), store)

			r := newRequestWithID(http.MethodGet, "/api/forms/"+tc.id, tc.id, nil)
			w := httptest.NewRecorder()

			h.Get(w, r)

			assert.Equal(t, tc.expectedStatus, w.Code, tc.name)
			if tc.expectedError != "" {
				assertErrorBody(t, w, tc.expectedError)
				return
			}

			// archived 不對外公開。
			assert.NotContains(t, decodeBody(t, w), "archived")
		})
	}
}

func TestHandler_List(t *testing.T) {
	testCases := []struct {
		name           string
		query          string
		expectedPage   int
		expectedSize   int
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Should use default page and size",
			query:          "",
			expectedPage:   1,
			expectedSize:   20,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Should accept explicit page and size",
			query:          "?page=3&size=10",
			expectedPage:   3,
			expectedSize:   10,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Should accept size of exactly 100",
			query:          "?size=100",
			expectedPage:   1,
			expectedSize:   100,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Should reject page 0",
			query:          "?page=0",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid page",
		},
		{
			name:           "Should reject negative page",
			query:          "?page=-1",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid page",
		},
		{
			name:           "Should reject non-numeric page",
			query:          "?page=abc",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid page",
		},
		{
			name:           "Should reject size 0",
			query:          "?size=0",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid size",
		},
		{
			name:           "Should reject non-numeric size",
			query:          "?size=abc",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid size",
		},
		{
			name:           "Should reject size 101",
			query:          "?size=101",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "size must be at most 100",
		},
		{
			name:           "Should reject size 999",
			query:          "?size=999",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "size must be at most 100",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewStore(t)
			if tc.expectedStatus == http.StatusOK {
				store.On("List", mock.Anything, tc.expectedPage, tc.expectedSize).
					Return(form.ListResult{Items: []form.Form{}, Total: 0}, nil).Once()
			}

			h := form.NewHandler(zaptest.NewLogger(t), store)

			r := httptest.NewRequest(http.MethodGet, "/api/forms"+tc.query, nil)
			w := httptest.NewRecorder()

			h.List(w, r)

			assert.Equal(t, tc.expectedStatus, w.Code, tc.name)
			if tc.expectedError != "" {
				assertErrorBody(t, w, tc.expectedError)
			}
		})
	}
}

// TestHandler_List_EmptyItemsIsArray：Go 的 nil slice encode 出來是 null，
// API_SPEC 要求沒有資料時 items 必須是 []。
func TestHandler_List_EmptyItemsIsArray(t *testing.T) {
	store := mocks.NewStore(t)
	store.On("List", mock.Anything, 1, 20).Return(form.ListResult{Items: nil, Total: 0}, nil).Once()

	h := form.NewHandler(zaptest.NewLogger(t), store)

	r := httptest.NewRequest(http.MethodGet, "/api/forms", nil)
	w := httptest.NewRecorder()

	h.List(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"items":[]`)
	assert.NotContains(t, w.Body.String(), `"items":null`)
}

func TestHandler_List_ResponseShape(t *testing.T) {
	items := []form.Form{
		{ID: uuid.New(), Title: "第二份", Archived: true},
		{ID: uuid.New(), Title: "第一份"},
	}

	store := mocks.NewStore(t)
	store.On("List", mock.Anything, 3, 10).Return(form.ListResult{Items: items, Total: 42}, nil).Once()

	h := form.NewHandler(zaptest.NewLogger(t), store)

	r := httptest.NewRequest(http.MethodGet, "/api/forms?page=3&size=10", nil)
	w := httptest.NewRecorder()

	h.List(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	body := decodeBody(t, w)
	assert.Equal(t, float64(3), body["page"])
	assert.Equal(t, float64(10), body["size"])
	assert.Equal(t, float64(42), body["total"])
	assert.Len(t, body["items"], 2)
	assert.NotContains(t, body["items"].([]any)[0], "archived")
}

func TestHandler_Update(t *testing.T) {
	id := uuid.New()
	updated := form.Form{ID: id, Title: "新標題"}

	testCases := []struct {
		name           string
		id             string
		body           string
		setupMock      func(store *mocks.Store)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Should update form",
			id:   id.String(),
			body: `{"title":"新標題","description":"新說明"}`,
			setupMock: func(store *mocks.Store) {
				store.On("Update", mock.Anything, id, mock.Anything, mock.Anything).Return(updated, nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Should return error when id is not a valid UUID",
			id:             "not-a-uuid",
			body:           `{"title":"新標題"}`,
			setupMock:      func(store *mocks.Store) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid id",
		},
		{
			name:           "Should return error when body is invalid JSON",
			id:             id.String(),
			body:           `{"title":"新標題",}`,
			setupMock:      func(store *mocks.Store) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request body",
		},
		{
			name:           "Should return error when no field is given",
			id:             id.String(),
			body:           `{}`,
			setupMock:      func(store *mocks.Store) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "no fields to update",
		},
		{
			name:           "Should return error when title is an empty string",
			id:             id.String(),
			body:           `{"title":""}`,
			setupMock:      func(store *mocks.Store) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "title is required",
		},
		{
			name:           "Should return error when title exceeds 255 characters",
			id:             id.String(),
			body:           fmt.Sprintf(`{"title":%q}`, strings.Repeat("a", 256)),
			setupMock:      func(store *mocks.Store) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "title must be at most 255 characters",
		},
		{
			name: "Should return not found when form does not exist",
			id:   id.String(),
			body: `{"title":"新標題"}`,
			setupMock: func(store *mocks.Store) {
				store.On("Update", mock.Anything, id, mock.Anything, mock.Anything).
					Return(form.Form{}, form.ErrFormNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "form not found",
		},
		{
			name: "Should return conflict when new title belongs to another form",
			id:   id.String(),
			body: `{"title":"別人的標題"}`,
			setupMock: func(store *mocks.Store) {
				store.On("Update", mock.Anything, id, mock.Anything, mock.Anything).
					Return(form.Form{}, form.ErrTitleConflict).Once()
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "form title already exists",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewStore(t)
			tc.setupMock(store)

			h := form.NewHandler(zaptest.NewLogger(t), store)

			r := newRequestWithID(http.MethodPatch, "/api/forms/"+tc.id, tc.id, []byte(tc.body))
			w := httptest.NewRecorder()

			h.Update(w, r)

			assert.Equal(t, tc.expectedStatus, w.Code, tc.name)
			if tc.expectedError != "" {
				assertErrorBody(t, w, tc.expectedError)
			}
		})
	}
}

// TestHandler_Update_OmittedFieldIsNil：只帶 title 時，description 要傳 nil pointer
// 給 Service，而不是指向空字串 —— 否則說明文字會被清空。
func TestHandler_Update_OmittedFieldIsNil(t *testing.T) {
	id := uuid.New()

	store := mocks.NewStore(t)
	store.On("Update", mock.Anything, id,
		mock.MatchedBy(func(title *string) bool { return title != nil && *title == "新標題" }),
		mock.MatchedBy(func(description *string) bool { return description == nil }),
	).Return(form.Form{ID: id, Title: "新標題"}, nil).Once()

	h := form.NewHandler(zaptest.NewLogger(t), store)

	r := newRequestWithID(http.MethodPatch, "/api/forms/"+id.String(), id.String(), []byte(`{"title":"新標題"}`))
	w := httptest.NewRecorder()

	h.Update(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestHandler_Update_EmptyDescriptionIsNotNil：帶了空字串代表「清空說明」，
// 跟「沒帶這個欄位」是兩件事。
func TestHandler_Update_EmptyDescriptionIsNotNil(t *testing.T) {
	id := uuid.New()

	store := mocks.NewStore(t)
	store.On("Update", mock.Anything, id,
		mock.MatchedBy(func(title *string) bool { return title == nil }),
		mock.MatchedBy(func(description *string) bool { return description != nil && *description == "" }),
	).Return(form.Form{ID: id}, nil).Once()

	h := form.NewHandler(zaptest.NewLogger(t), store)

	r := newRequestWithID(http.MethodPatch, "/api/forms/"+id.String(), id.String(), []byte(`{"description":""}`))
	w := httptest.NewRecorder()

	h.Update(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Delete(t *testing.T) {
	id := uuid.New()

	testCases := []struct {
		name           string
		id             string
		setupMock      func(store *mocks.Store)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Should delete form",
			id:   id.String(),
			setupMock: func(store *mocks.Store) {
				store.On("Delete", mock.Anything, id).Return(nil).Once()
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Should return error when id is not a valid UUID",
			id:             "not-a-uuid",
			setupMock:      func(store *mocks.Store) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid id",
		},
		{
			// 對不存在的 id 發 DELETE，要回 404 而不是 204 或 500。
			name: "Should return not found when form does not exist",
			id:   id.String(),
			setupMock: func(store *mocks.Store) {
				store.On("Delete", mock.Anything, id).Return(form.ErrFormNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "form not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewStore(t)
			tc.setupMock(store)

			h := form.NewHandler(zaptest.NewLogger(t), store)

			r := newRequestWithID(http.MethodDelete, "/api/forms/"+tc.id, tc.id, nil)
			w := httptest.NewRecorder()

			h.Delete(w, r)

			assert.Equal(t, tc.expectedStatus, w.Code, tc.name)
			if tc.expectedError != "" {
				assertErrorBody(t, w, tc.expectedError)
				return
			}

			// 204 No Content 不可以有 body。
			assert.Empty(t, w.Body.String())
		})
	}
}
