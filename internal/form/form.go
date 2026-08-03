package form

// ⚠️ 這是「不分層」的版本，也就是教材裡那段「能動，但問題在哪？」的程式碼。
//
// Task 1 的工作就是把這個檔案拆成 handler.go / service.go，
// 並且開始使用底下已經生成好的 Querier（queries.sql.go）。
//
// 拆完之後這個檔案應該就不存在了。

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Request struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Response struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

// CreateForm 同時做了四件事：解析 JSON、檢查業務規則、操作資料庫、組裝 Response。
func CreateForm(dbPool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req Request
		_ = json.NewDecoder(r.Body).Decode(&req)

		var count int
		_ = dbPool.QueryRow(r.Context(),
			"SELECT count(*) FROM forms WHERE title = $1", req.Title).Scan(&count)
		if count > 0 {
			http.Error(w, "title already exists", http.StatusConflict)
			return
		}

		var id uuid.UUID
		var createdAt time.Time
		_ = dbPool.QueryRow(r.Context(),
			"INSERT INTO forms (title, description) VALUES ($1, $2) RETURNING id, created_at",
			req.Title, req.Description).Scan(&id, &createdAt)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Response{
			ID:          id.String(),
			Title:       req.Title,
			Description: req.Description,
			CreatedAt:   createdAt,
		})
	}
}

// TODO(Task 2): GET /api/forms, GET /api/forms/{id}, PATCH /api/forms/{id}, DELETE /api/forms/{id}
// TODO(Task 3): 已封存的表單不能刪除
// TODO(Task 4): POST /api/forms/{id}/duplicate
