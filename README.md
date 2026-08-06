# LAB: System Architecture (Service, Handler)

搭配教材《System Architecture (Service, Handler)》。

這個 LAB 有四個 Task，從一份沒有分層的程式碼開始，一路把它拆成 Handler / Service / Querier 三層，並補完一組完整的表單 API。

請依序做，後面的 Task 會用到前面建立的結構。

| Task | 內容 |
| --- | --- |
| 1 | 把 `CreateForm` 拆成三層 |
| 2 | 補完 CRUD：List / Get / Update / Delete |
| 3 | 加上業務規則：已封存的表單不能刪除 |
| 4 | 複製表單：一個流程呼叫多個 Querier |

完整要求見 [TASKS.md](TASKS.md)，API 規格見 [API_SPEC.md](API_SPEC.md)。

**API_SPEC.md 是驗收的唯一依據。** 教材裡的程式碼片段若與它不一致，以 API_SPEC.md 為準。

---

## 環境需求

- Go 1.25 以上
- Docker（跑 PostgreSQL 用）

## 開始

```bash
git checkout -b my-work        # 從 main 開一個自己的 branch

docker compose up -d           # 啟動 PostgreSQL
go mod tidy                    # 下載相依套件
go run ./cmd/backend           # migration 會自動執行
```

另開一個終端機：

```bash
curl -i -X POST http://localhost:8080/api/forms \
  -H "Content-Type: application/json" \
  -d '{"title":"問卷調查","description":"說明文字"}'
```

拿到 `201` 就代表環境沒問題，可以開始 Task 1。

`api/` 底下有 Yaak 設定，匯入後四個 Task 的請求都已經建好，包含各種錯誤情境。

---

## 專案結構

```
cmd/backend/main.go                  進入點，把各層接起來
databaseutil/migration.go            migration，不需要修改
internal/database/migrations/        schema 版本
internal/form/
  queries.sql                        SQL 來源
  db.go  models.go  queries.sql.go   sqlc 生成，不要手動修改
  form.go                            起點，Task 1 要把它拆掉
api/                                 Yaak 測試設定
```

Task 1 開始之後，你會在 `internal/form/` 底下建立 `handler.go`、`service.go`、`errors.go`。

## 資料庫操作已經寫好了

四個 Task 需要的查詢在 `queries.sql` 裡都有，對應的 Go 程式碼也已經生成。

你不需要寫任何 SQL，也不需要執行 `sqlc generate`。可以使用的方法：

| 方法 | 用途 |
| --- | --- |
| `Create` | 新增一筆 |
| `GetByID` | 依 id 取一筆，查不到時回傳 `pgx.ErrNoRows` |
| `List` | 依 limit / offset 取多筆，依建立時間新到舊排序 |
| `Count` | 總筆數 |
| `ExistsByTitle` | 標題是否已存在 |
| `Update` | 部分更新，傳入 `pgtype.Text{}`（Valid 為 false）代表該欄位不變更 |
| `Delete` | 刪除，回傳影響的列數 |

---

## Branch

```
main            起點
hint/task-1     Task 1 的骨架
hint/task-2     Task 2 的骨架
hint/task-3     Task 3 的骨架
hint/task-4     Task 4 的骨架
solution        四個 Task 的完整實作
```

hint branch 裡的檔案已經建好、interface 已經宣告、函式簽名已經寫好，但函式內容是 `panic("TODO")`。它告訴你「東西放在哪裡」，不告訴你「內容怎麼寫」。

卡住時再看，看了之後邏輯仍然要自己填。

**不要 merge hint branch。** 它會蓋掉你寫到一半的程式碼，而且 `panic("TODO")` 會讓程式跑不起來。想查看提示可以用：

```bash
git show hint/task-1:internal/form/service.go
```

或切過去看完再切回來：

```bash
git checkout hint/task-1
git checkout my-work
```

---

## 完成標準

四個 Task 的個別驗收條件寫在 [TASKS.md](TASKS.md)。全部做完之後，以下三項都要成立。

**一、行為符合 API_SPEC.md。** 六支 API 的成功回應與所有錯誤情境都要對，包含 status code 與 `error` 訊息內容。

**二、分層檢查。** 這三個指令都不能有輸出：

```bash
grep -rn "net/http" internal/form/service.go
grep -rn "http.Status" internal/form/service.go
grep -rn "SELECT\|INSERT\|UPDATE\|DELETE" internal/form/handler.go
```

**三、`internal/form/form.go` 已經被刪除。**

---

## 自動化測試

每個 Task 都有對應的單元測試，用 mock 取代資料庫，不需要 Postgres。

```bash
make test TAGS=task1        # 做完 Task 1 之後
make test TAGS=task1,task2  # 做完 Task 2 之後，累加上去
make test                   # 四個 Task 全部做完
```

`make test` 會先跑 `mockery` 依照 `service.go` 的 `Querier` 與 `handler.go` 的 `Store`
生成 mock，再跑測試。所以**每次改動 interface 之後都要重新 `make test`**，
不要只跑 `go test`。

一開始測試是編譯不過的：

```
package sdclab/internal/form/mocks is not in std
```

這是預期的。`Querier` 與 `Store` 還不存在，mockery 生不出東西。
做完 Task 1、有了這兩個 interface 之後，`make test TAGS=task1` 就會跑起來。

測試檔一個 Task 一組（`service_task1_test.go` / `handler_task1_test.go`，依此類推）。
`internal/form/mocks/` 是生成物，不進版控。

