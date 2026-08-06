//go:build task3

package form_test

import (
	"context"
	"testing"

	"sdclab/internal/form"
	"sdclab/internal/form/mocks"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

// TestService_DeleteArchived 驗證 Task 3 的業務規則：
// 已封存的表單不能刪除。
//
// 這條規則屬於 Service —— 換成 CLI 批次刪除、換成排程清理，它一樣要成立。
func TestService_DeleteArchived(t *testing.T) {
	id := uuid.New()

	t.Run("Should refuse to delete an archived form", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		querier.On("GetByID", mock.Anything, id).Return(form.Form{
			ID:       id,
			Title:    "已封存的表單",
			Archived: true,
		}, nil).Once()
		// Delete 沒有設期望，被呼叫就會 fail —— 封存的表單根本不該進到刪除這一步。

		service := form.NewService(zaptest.NewLogger(t), querier)

		err := service.Delete(context.Background(), id)
		assert.ErrorIs(t, err, form.ErrFormArchived)
	})

	t.Run("Should delete a form that is not archived", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		querier.On("GetByID", mock.Anything, id).Return(form.Form{
			ID:       id,
			Title:    "一般表單",
			Archived: false,
		}, nil).Once()
		querier.On("Delete", mock.Anything, id).Return(int64(1), nil).Once()

		service := form.NewService(zaptest.NewLogger(t), querier)

		assert.NoError(t, service.Delete(context.Background(), id))
	})

	// 「不存在」與「已封存」是兩件事，不可以混淆。
	t.Run("Should return ErrFormNotFound instead of pgx.ErrNoRows", func(t *testing.T) {
		querier := mocks.NewQuerier(t)
		querier.On("GetByID", mock.Anything, id).Return(form.Form{}, pgx.ErrNoRows).Once()

		service := form.NewService(zaptest.NewLogger(t), querier)

		err := service.Delete(context.Background(), id)
		assert.ErrorIs(t, err, form.ErrFormNotFound)
		assert.NotErrorIs(t, err, pgx.ErrNoRows)
		assert.NotErrorIs(t, err, form.ErrFormArchived)
	})
}
