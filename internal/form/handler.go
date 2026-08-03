package form

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	maxTitleLength       = 255
	maxDescriptionLength = 1000

	defaultPage = 1
	defaultSize = 20
	maxSize     = 100
)

// Store 是 Handler 對 Service 的需求，同樣定義在使用端。
type Store interface {
	Create(ctx context.Context, title, description string) (Form, error)
	GetByID(ctx context.Context, id uuid.UUID) (Form, error)
	List(ctx context.Context, page, size int) (ListResult, error)
	Update(ctx context.Context, id uuid.UUID, title, description *string) (Form, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Duplicate(ctx context.Context, sourceID uuid.UUID, title string) (Form, error)
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
	mux.HandleFunc("GET /api/forms", h.List)
	mux.HandleFunc("GET /api/forms/{id}", h.Get)
	mux.HandleFunc("PATCH /api/forms/{id}", h.Update)
	mux.HandleFunc("DELETE /api/forms/{id}", h.Delete)
	mux.HandleFunc("POST /api/forms/{id}/duplicate", h.Duplicate)
}

// ---------- DTO ----------
//
// 不直接把 Form 丟出去，因為 Form 是資料表的形狀：
// 未來加的欄位會意外外洩、pgtype.Text 序列化出來前端看不懂、命名慣例也不同。

type CreateRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// UpdateRequest 的欄位用 *string 才能區分
// 「沒帶這個欄位」（nil）與「帶了空字串」（指向 ""）。
type UpdateRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

type DuplicateRequest struct {
	Title string `json:"title"`
}

type FormResponse struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

type ListResponse struct {
	Items []FormResponse `json:"items"`
	Page  int            `json:"page"`
	Size  int            `json:"size"`
	Total int64          `json:"total"`
}

type errorBody struct {
	Error string `json:"error"`
}

// Validate 只檢查格式，不檢查業務規則。
// 「title 不可為空」在這裡；「title 不可與現有表單重複」在 Service。
func (r CreateRequest) Validate() error {
	panic("TODO")
}

func (r UpdateRequest) Validate() error {
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

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	// page / size 從 r.URL.Query() 拿，是字串，要自己轉。
	// 範圍檢查（size <= 100）在這裡，offset 換算不在這裡。
	panic("TODO")
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	panic("TODO")
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	panic("TODO")
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	// 成功時回 204，而且不能有 body。
	panic("TODO")
}

func (h *Handler) Duplicate(w http.ResponseWriter, r *http.Request) {
	// 這個 Handler 應該跟 Create 幾乎一樣長。
	// 如果它變長了，代表流程控制跑到錯的層去了。
	panic("TODO")
}

// parseID 把路徑參數轉成 UUID。四支 API 都會用到，抽出來就好。
func (h *Handler) parseID(r *http.Request) (uuid.UUID, error) {
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
