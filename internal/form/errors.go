package form

import "errors"

// Service 用自己的詞彙描述失敗，不讓 pgx 的錯誤穿透到 Handler。
// Handler 再把這些錯誤映射成 HTTP status code。
var (
	ErrTitleConflict = errors.New("form title already exists")
)
