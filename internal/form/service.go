package form

import (
	"context"

	"go.uber.org/zap"
)

// Querier 是 Service 對資料層的需求。
// 定義在這裡而不是 queries.sql.go 旁邊，是因為「誰要用，誰定義需求」。
// sqlc 生成的 *Queries 剛好滿足它，不需要寫 implements。
type Querier interface {
	Create(ctx context.Context, arg CreateParams) (Form, error)
	ExistsByTitle(ctx context.Context, title string) (bool, error)
}

type Service struct {
	logger  *zap.Logger
	queries Querier
}

func NewService(logger *zap.Logger, querier Querier) *Service {
	return &Service{
		logger:  logger,
		queries: querier,
	}
}

// Create 建立一份表單。
//
// 注意簽名裡沒有任何 HTTP 的東西：沒有 http.Request，沒有 status code。
// Service 只認得 title 與 description，以及「衝突了」這件事本身。
//
// 步驟：
//  1. 用 ExistsByTitle 檢查標題是否重複，重複就回傳 ErrTitleConflict
//  2. 用 Create 寫入
//  3. 包裝底層錯誤時用 %w，上層才能用 errors.Is 追回去
func (s *Service) Create(ctx context.Context, title, description string) (Form, error) {
	panic("TODO")
}
