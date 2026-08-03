package form

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
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

// ---------- 驗證 ----------
//
// 這裡只檢查格式，不檢查業務規則。
// 「title 不可為空」在這裡；「title 不可與現有表單重複」在 Service。

func validateTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return errors.New("title is required")
	}
	if len(title) > maxTitleLength {
		return errors.New("title must be at most 255 characters")
	}

	return nil
}

func validateDescription(description string) error {
	if len(description) > maxDescriptionLength {
		return errors.New("description must be at most 1000 characters")
	}

	return nil
}

func (r CreateRequest) Validate() error {
	if err := validateTitle(r.Title); err != nil {
		return err
	}

	return validateDescription(r.Description)
}

func (r UpdateRequest) Validate() error {
	if r.Title == nil && r.Description == nil {
		return errors.New("no fields to update")
	}
	if r.Title != nil {
		if err := validateTitle(*r.Title); err != nil {
			return err
		}
	}
	if r.Description != nil {
		if err := validateDescription(*r.Description); err != nil {
			return err
		}
	}

	return nil
}

func (r DuplicateRequest) Validate() error {
	return validateTitle(r.Title)
}

// toResponse 把資料層的形狀轉成對外的形狀。
// archived 不對外公開，所以這裡沒有它。
func toResponse(f Form) FormResponse {
	return FormResponse{
		ID:          f.ID.String(),
		Title:       f.Title,
		Description: f.Description.String,
		CreatedAt:   f.CreatedAt.Time,
	}
}

// ---------- Handler ----------

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "invalid request body"})
		return
	}

	if err := req.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: err.Error()})
		return
	}

	created, err := h.store.Create(r.Context(), req.Title, req.Description)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, toResponse(created))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, err := intQuery(r, "page", defaultPage)
	if err != nil || page < 1 {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "invalid page"})
		return
	}

	size, err := intQuery(r, "size", defaultSize)
	if err != nil || size < 1 {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "invalid size"})
		return
	}
	if size > maxSize {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "size must be at most 100"})
		return
	}

	// offset 的換算不在這裡，Service 才知道怎麼分頁。
	result, err := h.store.List(r.Context(), page, size)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// nil slice 會被 encode 成 null，這裡要的是 []。
	items := make([]FormResponse, 0, len(result.Items))
	for _, f := range result.Items {
		items = append(items, toResponse(f))
	}

	writeJSON(w, http.StatusOK, ListResponse{
		Items: items,
		Page:  page,
		Size:  size,
		Total: result.Total,
	})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "invalid id"})
		return
	}

	f, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toResponse(f))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "invalid id"})
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "invalid request body"})
		return
	}

	if err := req.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: err.Error()})
		return
	}

	updated, err := h.store.Update(r.Context(), id, req.Title, req.Description)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toResponse(updated))
}

// Delete 沒有因為 Task 3 的封存規則變長，因為那條規則不在這一層。
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "invalid id"})
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

// Duplicate 跟 Create 幾乎一樣長。
// Service 內部呼叫了三次 Querier，這一層完全不知道。
func (h *Handler) Duplicate(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "invalid id"})
		return
	}

	var req DuplicateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "invalid request body"})
		return
	}

	if err := req.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: err.Error()})
		return
	}

	created, err := h.store.Duplicate(r.Context(), id, req.Title)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, toResponse(created))
}

// ---------- 小工具 ----------

func (h *Handler) parseID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(r.PathValue("id"))
}

func intQuery(r *http.Request, key string, fallback int) (int, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback, nil
	}

	return strconv.Atoi(raw)
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
// Service 只說「找不到」「衝突了」「已封存」，
// 是這裡決定那分別是 404、409、409。
func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrFormNotFound):
		writeJSON(w, http.StatusNotFound, errorBody{Error: "form not found"})
	case errors.Is(err, ErrTitleConflict):
		writeJSON(w, http.StatusConflict, errorBody{Error: "form title already exists"})
	case errors.Is(err, ErrFormArchived):
		writeJSON(w, http.StatusConflict, errorBody{Error: "form is archived"})
	default:
		// 未預期的錯誤在這裡 log 一次就好，對外只回通用訊息。
		// 資料庫錯誤常常帶著資料表名稱與欄位名稱，那是給我們看的。
		h.logger.Error("unhandled error",
			zap.Error(err),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path))
		writeJSON(w, http.StatusInternalServerError, errorBody{Error: "internal server error"})
	}
}
