package form

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const (
	maxTitleLength       = 255
	maxDescriptionLength = 1000
)

// Store 是 Handler 對 Service 的需求，同樣定義在使用端。
type Store interface {
	Create(ctx context.Context, title, description string) (Form, error)
}

type Handler struct {
	logger *zap.Logger
	store  Store
}

func NewHandler(logger *zap.Logger, store Store) *Handler {
	return &Handler{logger: logger, store: store}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/forms", h.Create)
}

// ---------- DTO ----------
//
// 不直接把 Form 丟出去，因為 Form 是資料表的形狀：
// 未來加的欄位會意外外洩、pgtype.Text 序列化出來前端看不懂、命名慣例也不同。

type CreateRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type FormResponse struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

type errorBody struct {
	Error string `json:"error"`
}

// Validate 只檢查格式，不檢查業務規則。
// 「title 不可為空」在這裡；「title 不可與現有表單重複」在 Service。
func (r CreateRequest) Validate() error {
	panic("TODO")
}

// toResponse 把資料層的形狀轉成對外的形狀。
func toResponse(f Form) FormResponse {
	panic("TODO")
}

// ---------- Handler ----------

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	// 1. 解析 JSON，失敗回 400
	// 2. Validate，失敗回 400
	// 3. 呼叫 h.store.Create
	// 4. 失敗交給 h.writeError，成功回 201 + toResponse
	//
	// 每一個錯誤分支都要 return。Go 不會因為你「回覆了錯誤」就結束函式。
	panic("TODO")
}

// ---------- 回應 ----------

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

// writeError 是這一層的核心：把 Service 的業務錯誤翻譯成 HTTP 語意。
//
// Service 說「衝突了」，這裡才決定那是 409。
// 未預期的錯誤在這裡 log 一次就好，對外只回通用訊息，
// 不要把資料庫錯誤原文丟給前端。
func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	panic("TODO")
}
