//go:build task1

package form_test

import (
	"context"
	"testing"

	"sdclab/internal/form"
	"sdclab/internal/form/mocks"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

func TestService_Create(t *testing.T) {
	createdForm := form.Form{
		ID:          uuid.New(),
		Title:       "問卷調查",
		Description: pgtype.Text{String: "說明文字", Valid: true},
	}

	testCases := []struct {
		name           string
		title          string
		description    string
		setupMock      func(querier *mocks.Querier)
		expectedHasErr bool
		expectedErr    error
	}{
		{
			name:        "Should create form when title is available",
			title:       "問卷調查",
			description: "說明文字",
			setupMock: func(querier *mocks.Querier) {
				querier.On("ExistsByTitle", mock.Anything, "問卷調查").Return(false, nil).Once()
				querier.On("Create", mock.Anything, form.CreateParams{
					Title:       "問卷調查",
					Description: pgtype.Text{String: "說明文字", Valid: true},
				}).Return(createdForm, nil).Once()
			},
			expectedHasErr: false,
		},
		{
			// 沒填描述要存 NULL，不是空字串。這兩件事在 SQL 裡意義不同。
			name:        "Should store NULL description when description is empty",
			title:       "問卷調查",
			description: "",
			setupMock: func(querier *mocks.Querier) {
				querier.On("ExistsByTitle", mock.Anything, "問卷調查").Return(false, nil).Once()
				querier.On("Create", mock.Anything, form.CreateParams{
					Title:       "問卷調查",
					Description: pgtype.Text{Valid: false},
				}).Return(createdForm, nil).Once()
			},
			expectedHasErr: false,
		},
		{
			// Create 沒有設期望，被呼叫就會 fail。
			name:        "Should return ErrTitleConflict when title already exists",
			title:       "問卷調查",
			description: "說明文字",
			setupMock: func(querier *mocks.Querier) {
				querier.On("ExistsByTitle", mock.Anything, "問卷調查").Return(true, nil).Once()
			},
			expectedHasErr: true,
			expectedErr:    form.ErrTitleConflict,
		},
		{
			name:        "Should propagate error when ExistsByTitle fails",
			title:       "問卷調查",
			description: "說明文字",
			setupMock: func(querier *mocks.Querier) {
				querier.On("ExistsByTitle", mock.Anything, "問卷調查").Return(false, assert.AnError).Once()
			},
			expectedHasErr: true,
		},
		{
			name:        "Should propagate error when Create fails",
			title:       "問卷調查",
			description: "說明文字",
			setupMock: func(querier *mocks.Querier) {
				querier.On("ExistsByTitle", mock.Anything, "問卷調查").Return(false, nil).Once()
				querier.On("Create", mock.Anything, mock.Anything).Return(form.Form{}, assert.AnError).Once()
			},
			expectedHasErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			querier := mocks.NewQuerier(t)
			if tc.setupMock != nil {
				tc.setupMock(querier)
			}

			service := form.NewService(zaptest.NewLogger(t), querier)

			created, err := service.Create(context.Background(), tc.title, tc.description)

			if tc.expectedHasErr {
				assert.Error(t, err)
				if tc.expectedErr != nil {
					assert.ErrorIs(t, err, tc.expectedErr)
				} else {
					// 一般的資料庫錯誤不可以被誤判成業務錯誤。
					assert.NotErrorIs(t, err, form.ErrTitleConflict)
				}
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, createdForm.ID, created.ID)
			assert.Equal(t, createdForm.Title, created.Title)
		})
	}
}
