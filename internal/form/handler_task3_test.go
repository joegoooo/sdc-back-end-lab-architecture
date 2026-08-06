//go:build task3

package form_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"sdclab/internal/form"
	"sdclab/internal/form/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

// TestHandler_DeleteArchived：Handler 這一層不知道「封存」是什麼，
// 它只負責把 Service 的 ErrFormArchived 翻譯成 409。
func TestHandler_DeleteArchived(t *testing.T) {
	id := uuid.New()

	testCases := []struct {
		name           string
		storeErr       error
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Should return conflict when form is archived",
			storeErr:       form.ErrFormArchived,
			expectedStatus: http.StatusConflict,
			expectedError:  "form is archived",
		},
		{
			// 已封存與不存在不可以混淆。
			name:           "Should return not found when form does not exist",
			storeErr:       form.ErrFormNotFound,
			expectedStatus: http.StatusNotFound,
			expectedError:  "form not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := mocks.NewStore(t)
			store.On("Delete", mock.Anything, id).Return(tc.storeErr).Once()

			h := form.NewHandler(zaptest.NewLogger(t), store)

			r := newRequestWithID(http.MethodDelete, "/api/forms/"+id.String(), id.String(), nil)
			w := httptest.NewRecorder()

			h.Delete(w, r)

			assert.Equal(t, tc.expectedStatus, w.Code, tc.name)
			assertErrorBody(t, w, tc.expectedError)
		})
	}
}
