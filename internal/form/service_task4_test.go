//go:build task4

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

// TestService_Duplicate 驗證「一個流程呼叫多個 Querier」。
//
// 三次呼叫都用 .Once() 精確約束：讀來源、檢查撞名、建立新表單。
// 三個都是既有的 Querier 方法，沒有為了這支 API 新增任何 SQL。
func TestService_Duplicate(t *testing.T) {
	sourceID := uuid.New()
	source := form.Form{
		ID:          sourceID,
		Title:       "原表單",
		Description: pgtype.Text{String: "來源說明", Valid: true},
	}

	t.Run("Should duplicate form with exactly three querier calls", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		querier.On("GetByID", mock.Anything, sourceID).Return(source, nil).Once()
		querier.On("ExistsByTitle", mock.Anything, "新表單").Return(false, nil).Once()
		// description 沿用來源表單，新標題由請求指定。
		querier.On("Create", mock.Anything, form.CreateParams{
			Title:       "新表單",
			Description: pgtype.Text{String: "來源說明", Valid: true},
		}).Return(form.Form{ID: uuid.New(), Title: "新表單"}, nil).Once()

		service := form.NewService(zaptest.NewLogger(t), querier)

		created, err := service.Duplicate(context.Background(), sourceID, "新表單")
		assert.NoError(t, err)
		assert.Equal(t, "新表單", created.Title)
	})

	t.Run("Should carry over NULL description", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		querier.On("GetByID", mock.Anything, sourceID).Return(form.Form{
			ID:          sourceID,
			Title:       "原表單",
			Description: pgtype.Text{Valid: false},
		}, nil).Once()
		querier.On("ExistsByTitle", mock.Anything, "新表單").Return(false, nil).Once()
		querier.On("Create", mock.Anything, form.CreateParams{
			Title:       "新表單",
			Description: pgtype.Text{Valid: false},
		}).Return(form.Form{ID: uuid.New(), Title: "新表單"}, nil).Once()

		service := form.NewService(zaptest.NewLogger(t), querier)

		_, err := service.Duplicate(context.Background(), sourceID, "新表單")
		assert.NoError(t, err)
	})

	// 複製一份已封存的表單會成功，且新表單沒有被封存。
	// CreateParams 只有 Title 與 Description，archived 靠資料庫預設值為 false。
	t.Run("Should duplicate an archived form into a non-archived one", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		querier.On("GetByID", mock.Anything, sourceID).Return(form.Form{
			ID:          sourceID,
			Title:       "已封存的表單",
			Description: pgtype.Text{String: "來源說明", Valid: true},
			Archived:    true,
		}, nil).Once()
		querier.On("ExistsByTitle", mock.Anything, "新表單").Return(false, nil).Once()
		querier.On("Create", mock.Anything, form.CreateParams{
			Title:       "新表單",
			Description: pgtype.Text{String: "來源說明", Valid: true},
		}).Return(form.Form{ID: uuid.New(), Title: "新表單", Archived: false}, nil).Once()

		service := form.NewService(zaptest.NewLogger(t), querier)

		created, err := service.Duplicate(context.Background(), sourceID, "新表單")
		assert.NoError(t, err)
		assert.False(t, created.Archived)
	})

	// 來源不存在時，後面兩次查詢都不該發生。
	t.Run("Should return ErrFormNotFound when source does not exist", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		querier.On("GetByID", mock.Anything, sourceID).Return(form.Form{}, pgx.ErrNoRows).Once()

		service := form.NewService(zaptest.NewLogger(t), querier)

		_, err := service.Duplicate(context.Background(), sourceID, "新表單")
		assert.ErrorIs(t, err, form.ErrFormNotFound)
		assert.NotErrorIs(t, err, form.ErrTitleConflict)
	})

	// 標題撞名時不該建立任何東西。
	t.Run("Should return ErrTitleConflict when new title already exists", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		querier.On("GetByID", mock.Anything, sourceID).Return(source, nil).Once()
		querier.On("ExistsByTitle", mock.Anything, "撞名的標題").Return(true, nil).Once()

		service := form.NewService(zaptest.NewLogger(t), querier)

		_, err := service.Duplicate(context.Background(), sourceID, "撞名的標題")
		assert.ErrorIs(t, err, form.ErrTitleConflict)
		assert.NotErrorIs(t, err, form.ErrFormNotFound)
	})
}
