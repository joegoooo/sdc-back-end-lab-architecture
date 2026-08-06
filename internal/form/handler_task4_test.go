//go:build task4

package form_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"sdclab/internal/form"
	"sdclab/internal/form/mocks"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

func TestHandler_Duplicate(t *testing.T) {
	sourceID := uuid.New()
	created := form.Form{
		ID:          uuid.New(),
		Title:       "新表單",
		Description: pgtype.Text{String: "來源說明", Valid: true},
	}

	testCases := []struct {
		name           string
		id             string
		body           string
		setupMock      func(store *mocks.Store)
		expectedStatus int
		expectedError  string
	}{
		{
			// Handler 只呼叫一次 Service，完全不知道裡面做了幾次資料庫查詢。
			name: "Should duplicate form with a single service call",
			id:   sourceID.String(),
			body: `{"title":"新表單"}`,
			setupMock: func(store *mocks.Store) {
				store.On("Duplicate", mock.Anything, sourceID, "新表單").Return(created, nil).Once()
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Should return error when id is not a valid UUID",
			id:             "not-a-uuid",
			body:           `{"title":"新表單"}`,
			setupMock:      func(store *mocks.Store) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid id",
		},
		{
			name:           "Should return error when body is invalid JSON",
			id:             sourceID.String(),
			body:           `{"title":"新表單",}`,
			setupMock:      func(store *mocks.Store) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request body",
		},
		{
			name:           "Should return error when title is empty",
			id:             sourceID.String(),
			body:           `{"title":""}`,
			setupMock:      func(store *mocks.Store) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "title is required",
		},
		{
			name: "Should return not found when source form does not exist",
			id:   sourceID.String(),
			body: `{"title":"新表單"}`,
			setupMock: func(store *mocks.Store) {
				store.On("Duplicate", mock.Anything, sourceID, "新表單").
					Return(form.Form{}, form.ErrFormNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "form not found",
		},
		{
			name: "Should return conflict when new title already exists",
			id:   sourceID.String(),
			body: `{"title":"撞名的標題"}`,
			setupMock: func(store *mocks.Store) {
				store.On("Duplicate", mock.Anything, sourceID, "撞名的標題").
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

			r := newRequestWithID(http.MethodPost, "/api/forms/"+tc.id+"/duplicate", tc.id, []byte(tc.body))
			w := httptest.NewRecorder()

			h.Duplicate(w, r)

			assert.Equal(t, tc.expectedStatus, w.Code, tc.name)
			if tc.expectedError != "" {
				assertErrorBody(t, w, tc.expectedError)
				return
			}

			body := decodeBody(t, w)
			assert.Equal(t, "新表單", body["title"])
			assert.Equal(t, "來源說明", body["description"])
			assert.NotContains(t, body, "archived")
		})
	}
}
