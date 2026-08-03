package form

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Querier 是 Service 對資料層的需求。
// 定義在這裡而不是 queries.sql.go 旁邊，是因為「誰要用，誰定義需求」。
// sqlc 生成的 *Queries 剛好滿足它，不需要寫 implements。
type Querier interface {
	Create(ctx context.Context, arg CreateParams) (Form, error)
	GetByID(ctx context.Context, id uuid.UUID) (Form, error)
	List(ctx context.Context, arg ListParams) ([]Form, error)
	Count(ctx context.Context) (int64, error)
	ExistsByTitle(ctx context.Context, title string) (bool, error)
	Update(ctx context.Context, arg UpdateParams) (Form, error)
	Delete(ctx context.Context, id uuid.UUID) (int64, error)
}

// ListResult 是 Service 回給 Handler 的東西。
// 它不是 JSON，也不知道 JSON——欄位命名、序列化格式是 Handler 的事。
type ListResult struct {
	Items []Form
	Total int64
}

type Service struct {
	logger  *zap.Logger
	queries Querier
}

func NewService(logger *zap.Logger, db DBTX) *Service {
	return &Service{
		logger:  logger,
		queries: New(db),
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

// GetByID 取得單一表單。
//
// pgx 在查不到資料時回傳 pgx.ErrNoRows。把它轉譯成 ErrFormNotFound，
// Handler 才不需要知道底下用的是 pgx。
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (Form, error) {
	panic("TODO")
}

// List 分頁列出表單。
//
// 收到的是「第幾頁、每頁幾筆」，Querier 認得的是 limit 與 offset。
// 這個換算就是 Service 的工作，Handler 只負責把 query string 轉成 int。
func (s *Service) List(ctx context.Context, page, size int) (ListResult, error) {
	panic("TODO")
}

// Update 部分更新。nil 代表「這個欄位不變更」。
//
// 需要先確認表單存在（否則 404 從哪來？），
// 也需要在 title 真的有變動時才檢查撞名——改成跟自己一樣的值不算衝突。
func (s *Service) Update(ctx context.Context, id uuid.UUID, title, description *string) (Form, error) {
	panic("TODO")
}

// Delete 刪除表單。
//
// Querier 的 Delete 回傳「影響了幾列」。0 代表沒有這筆資料。
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	panic("TODO")
}
