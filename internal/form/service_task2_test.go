//go:build task2

package form_test

import (
	"context"
	"testing"

	"sdclab/internal/form"
	"sdclab/internal/form/mocks"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

// TestService_List 的重點是分頁換算。
// Handler 給的是「第幾頁、每頁幾筆」，Querier 只認得 limit 與 offset，
// OFFSET = (page - 1) * size 這個換算屬於 Service。
func TestService_List(t *testing.T) {
	testCases := []struct {
		name           string
		page           int
		size           int
		expectedParams form.ListParams
	}{
		{
			name:           "First page starts at offset 0",
			page:           1,
			size:           20,
			expectedParams: form.ListParams{Limit: 20, Offset: 0},
		},
		{
			name:           "Third page with size 10 starts at offset 20",
			page:           3,
			size:           10,
			expectedParams: form.ListParams{Limit: 10, Offset: 20},
		},
		{
			name:           "Second page with size 100 starts at offset 100",
			page:           2,
			size:           100,
			expectedParams: form.ListParams{Limit: 100, Offset: 100},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			items := []form.Form{{ID: uuid.New(), Title: "問卷調查"}}

			querier := mocks.NewQuerier(t)
			querier.On("List", mock.Anything, tc.expectedParams).Return(items, nil).Once()
			querier.On("Count", mock.Anything).Return(int64(42), nil).Once()

			service := form.NewService(zaptest.NewLogger(t), querier)

			result, err := service.List(context.Background(), tc.page, tc.size)

			assert.NoError(t, err)
			assert.Len(t, result.Items, 1)
			// total 是符合條件的總筆數，不是本頁筆數。
			assert.Equal(t, int64(42), result.Total)
		})
	}
}

func TestService_List_Errors(t *testing.T) {
	t.Run("Should propagate error when List fails", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		querier.On("List", mock.Anything, mock.Anything).Return(nil, assert.AnError).Once()

		service := form.NewService(zaptest.NewLogger(t), querier)

		_, err := service.List(context.Background(), 1, 20)
		assert.Error(t, err)
	})

	t.Run("Should propagate error when Count fails", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		querier.On("List", mock.Anything, mock.Anything).Return([]form.Form{}, nil).Once()
		querier.On("Count", mock.Anything).Return(int64(0), assert.AnError).Once()

		service := form.NewService(zaptest.NewLogger(t), querier)

		_, err := service.List(context.Background(), 1, 20)
		assert.Error(t, err)
	})
}

// TestService_GetByID 的重點：pgx 的錯誤不可以穿透到 Handler。
func TestService_GetByID(t *testing.T) {
	id := uuid.New()

	t.Run("Should return form when it exists", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		querier.On("GetByID", mock.Anything, id).Return(form.Form{ID: id, Title: "問卷調查"}, nil).Once()

		service := form.NewService(zaptest.NewLogger(t), querier)

		f, err := service.GetByID(context.Background(), id)
		assert.NoError(t, err)
		assert.Equal(t, id, f.ID)
	})

	t.Run("Should translate pgx.ErrNoRows into ErrFormNotFound", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		querier.On("GetByID", mock.Anything, id).Return(form.Form{}, pgx.ErrNoRows).Once()

		service := form.NewService(zaptest.NewLogger(t), querier)

		_, err := service.GetByID(context.Background(), id)
		assert.ErrorIs(t, err, form.ErrFormNotFound)
		// Service 要用自己的詞彙，不讓 pgx 的錯誤穿透出去。
		assert.NotErrorIs(t, err, pgx.ErrNoRows)
	})

	t.Run("Should propagate other errors", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		querier.On("GetByID", mock.Anything, id).Return(form.Form{}, assert.AnError).Once()

		service := form.NewService(zaptest.NewLogger(t), querier)

		_, err := service.GetByID(context.Background(), id)
		assert.Error(t, err)
		assert.NotErrorIs(t, err, form.ErrFormNotFound)
	})
}

func TestService_Update(t *testing.T) {
	id := uuid.New()
	current := form.Form{
		ID:          id,
		Title:       "原標題",
		Description: pgtype.Text{String: "原說明", Valid: true},
	}

	t.Run("Should return ErrFormNotFound when form does not exist", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		querier.On("GetByID", mock.Anything, id).Return(form.Form{}, pgx.ErrNoRows).Once()

		service := form.NewService(zaptest.NewLogger(t), querier)

		// Update 沒有設期望，被呼叫就會 fail。
		_, err := service.Update(context.Background(), id, ptr("新標題"), nil)
		assert.ErrorIs(t, err, form.ErrFormNotFound)
	})

	// 改成跟自己現在一樣的值，不算重複。
	t.Run("Should not conflict when title is unchanged", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		querier.On("GetByID", mock.Anything, id).Return(current, nil).Once()
		querier.On("Update", mock.Anything, form.UpdateParams{
			Title:       pgtype.Text{String: "原標題", Valid: true},
			Description: pgtype.Text{},
			ID:          id,
		}).Return(current, nil).Once()

		service := form.NewService(zaptest.NewLogger(t), querier)

		_, err := service.Update(context.Background(), id, ptr("原標題"), nil)
		assert.NoError(t, err)
	})

	t.Run("Should return ErrTitleConflict when new title belongs to another form", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		querier.On("GetByID", mock.Anything, id).Return(current, nil).Once()
		querier.On("ExistsByTitle", mock.Anything, "別人的標題").Return(true, nil).Once()

		service := form.NewService(zaptest.NewLogger(t), querier)

		_, err := service.Update(context.Background(), id, ptr("別人的標題"), nil)
		assert.ErrorIs(t, err, form.ErrTitleConflict)
	})

	// 只帶 title 時 description 必須維持不變：pgtype.Text{} 的 Valid 為 false，
	// 在 SQL 的 COALESCE 裡代表「這個欄位不變更」。
	t.Run("Should leave description untouched when only title is given", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		querier.On("GetByID", mock.Anything, id).Return(current, nil).Once()
		querier.On("ExistsByTitle", mock.Anything, "新標題").Return(false, nil).Once()
		querier.On("Update", mock.Anything, form.UpdateParams{
			Title:       pgtype.Text{String: "新標題", Valid: true},
			Description: pgtype.Text{Valid: false},
			ID:          id,
		}).Return(current, nil).Once()

		service := form.NewService(zaptest.NewLogger(t), querier)

		_, err := service.Update(context.Background(), id, ptr("新標題"), nil)
		assert.NoError(t, err)
	})

	t.Run("Should leave title untouched when only description is given", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		querier.On("GetByID", mock.Anything, id).Return(current, nil).Once()
		querier.On("Update", mock.Anything, form.UpdateParams{
			Title:       pgtype.Text{Valid: false},
			Description: pgtype.Text{String: "新說明", Valid: true},
			ID:          id,
		}).Return(current, nil).Once()

		service := form.NewService(zaptest.NewLogger(t), querier)

		_, err := service.Update(context.Background(), id, nil, ptr("新說明"))
		assert.NoError(t, err)
	})
}

// TestService_Delete：Delete 的 sqlc 註解是 :execrows，沒刪到任何一列時
// pgx 不會給 error，影響列數是唯一的線索。
func TestService_Delete(t *testing.T) {
	id := uuid.New()

	t.Run("Should delete existing form", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		// Task 3 之後 Delete 會先讀出表單檢查封存狀態，Task 2 階段還不會。
		querier.On("GetByID", mock.Anything, id).Return(form.Form{ID: id}, nil).Maybe()
		querier.On("Delete", mock.Anything, id).Return(int64(1), nil).Once()

		service := form.NewService(zaptest.NewLogger(t), querier)

		assert.NoError(t, service.Delete(context.Background(), id))
	})

	t.Run("Should return ErrFormNotFound when nothing was deleted", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		querier.On("GetByID", mock.Anything, id).Return(form.Form{}, pgx.ErrNoRows).Maybe()
		querier.On("Delete", mock.Anything, id).Return(int64(0), nil).Maybe()

		service := form.NewService(zaptest.NewLogger(t), querier)

		err := service.Delete(context.Background(), id)
		assert.ErrorIs(t, err, form.ErrFormNotFound)
		assert.NotErrorIs(t, err, pgx.ErrNoRows)
	})

	t.Run("Should propagate error when Delete fails", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		querier.On("GetByID", mock.Anything, id).Return(form.Form{ID: id}, nil).Maybe()
		querier.On("Delete", mock.Anything, id).Return(int64(0), assert.AnError).Once()

		service := form.NewService(zaptest.NewLogger(t), querier)

		err := service.Delete(context.Background(), id)
		assert.Error(t, err)
		assert.NotErrorIs(t, err, form.ErrFormNotFound)
	})
}
