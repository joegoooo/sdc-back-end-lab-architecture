package form

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

func NewService(logger *zap.Logger, querier Querier) *Service {
	return &Service{
		logger:  logger,
		queries: querier,
	}
}

func (s *Service) Create(ctx context.Context, title, description string) (Form, error) {
	if err := s.ensureTitleAvailable(ctx, title); err != nil {
		return Form{}, err
	}

	created, err := s.queries.Create(ctx, CreateParams{
		Title:       title,
		Description: textOrNull(description),
	})
	if err != nil {
		return Form{}, fmt.Errorf("create form: %w", err)
	}

	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (Form, error) {
	f, err := s.queries.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Form{}, ErrFormNotFound
		}
		return Form{}, fmt.Errorf("get form %s: %w", id, err)
	}

	return f, nil
}

// List 收到的是「第幾頁、每頁幾筆」，Querier 認得的是 limit 與 offset。
// 這個換算就是 Service 的工作。
func (s *Service) List(ctx context.Context, page, size int) (ListResult, error) {
	items, err := s.queries.List(ctx, ListParams{
		Limit:  int32(size),
		Offset: int32((page - 1) * size),
	})
	if err != nil {
		return ListResult{}, fmt.Errorf("list forms: %w", err)
	}

	total, err := s.queries.Count(ctx)
	if err != nil {
		return ListResult{}, fmt.Errorf("count forms: %w", err)
	}

	return ListResult{Items: items, Total: total}, nil
}

// Update 部分更新。nil 代表「這個欄位不變更」。
func (s *Service) Update(ctx context.Context, id uuid.UUID, title, description *string) (Form, error) {
	current, err := s.GetByID(ctx, id)
	if err != nil {
		return Form{}, err
	}

	// 只有在標題真的變動時才檢查撞名。
	// 改成跟自己現在一樣的值，不算衝突。
	if title != nil && *title != current.Title {
		if err := s.ensureTitleAvailable(ctx, *title); err != nil {
			return Form{}, err
		}
	}

	updated, err := s.queries.Update(ctx, UpdateParams{
		Title:       optionalText(title),
		Description: optionalText(description),
		ID:          id,
	})
	if err != nil {
		return Form{}, fmt.Errorf("update form %s: %w", id, err)
	}

	return updated, nil
}

// Delete 刪除表單。已封存的表單不能刪除。
//
// 這條規則在這裡，不是在 Handler——換成 CLI 批次刪除、
// 換成排程清理，它一樣要成立。
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	f, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if f.Archived {
		return ErrFormArchived
	}

	rows, err := s.queries.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("delete form %s: %w", id, err)
	}
	if rows == 0 {
		// GetByID 到 Delete 之間有人先刪掉了。
		return ErrFormNotFound
	}

	return nil
}

// Duplicate 以既有表單為範本建立一份新的。
//
// 三次 Querier 呼叫（讀來源、檢查撞名、建立），但 Handler 只呼叫一次 Service，
// 而且完全不知道裡面做了幾次查詢。三個 Querier 方法都是既有的，
// 沒有為了這支 API 新增任何 SQL。
func (s *Service) Duplicate(ctx context.Context, sourceID uuid.UUID, title string) (Form, error) {
	source, err := s.GetByID(ctx, sourceID)
	if err != nil {
		return Form{}, err
	}

	if err := s.ensureTitleAvailable(ctx, title); err != nil {
		return Form{}, err
	}

	created, err := s.queries.Create(ctx, CreateParams{
		Title:       title,
		Description: source.Description,
	})
	if err != nil {
		return Form{}, fmt.Errorf("duplicate form %s: %w", sourceID, err)
	}

	return created, nil
}

// ensureTitleAvailable 是 Create、Update、Duplicate 共用的業務規則。
// 抽出來之後，「標題不可重複」這件事在整個 Backend 裡只寫了一次。
func (s *Service) ensureTitleAvailable(ctx context.Context, title string) error {
	exists, err := s.queries.ExistsByTitle(ctx, title)
	if err != nil {
		return fmt.Errorf("check title duplication: %w", err)
	}
	if exists {
		return ErrTitleConflict
	}

	return nil
}

// textOrNull 用在建立時：沒填描述就存 NULL，而不是空字串。
// 這兩件事在 SQL 裡的意義不同（WHERE description IS NULL 查不到空字串）。
func textOrNull(v string) pgtype.Text {
	return pgtype.Text{String: v, Valid: v != ""}
}

// optionalText 用在部分更新：nil 產生一個 Valid 為 false 的值，
// 在 SQL 的 COALESCE 裡代表「這個欄位不變更」。
func optionalText(v *string) pgtype.Text {
	if v == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{String: *v, Valid: true}
}
