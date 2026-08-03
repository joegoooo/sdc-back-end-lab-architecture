# LAB: System Architecture (Service, Handler)

搭配教材《System Architecture (Service, Handler)》。

教材最後留了兩個問題：

1. 這段程式是在處理 HTTP，還是在處理 Business Logic？
2. 如果未來另一支 API 也需要相同功能，這段程式是否應該放在 Service 中重複使用？

這個 LAB 就是把這兩個問題實際做一遍。**重點不是把功能做出來，而是每一段程式碼你都要能說出它為什麼放在那一層。**

## 四個 Task

依序做，不要跳。Task 1 建立模式，Task 2 用重複把模式練熟，Task 3、4 才是真正需要判斷的部分。

| Task | 內容 | 對應教材 |
| --- | --- | --- |
| 1 | 把 `CreateForm` 拆成三層 | 「不分層會怎麼樣？」 |
| 2 | 補完 CRUD：List / Get / Update / Delete | 分層設計、單向呼叫流程 |
| 3 | 加上業務規則：已封存的表單不能刪 | 「Handler 不應該有長篇判斷流程」 |
| 4 | 一個流程呼叫多個 Querier | 「呼叫多個 Querier 完成一個流程」 |

詳細要求見 [TASKS.md](TASKS.md)，API 規格見 [API_SPEC.md](API_SPEC.md)。
**API_SPEC.md 是驗收的唯一依據**，教材裡的程式碼片段若與它不一致，以它為準。

## Branch

```
main            起點。可以跑，但沒有分層
hint/task-1     Task 1 的骨架（不是答案）
hint/task-2     Task 2 的骨架
hint/task-3     Task 3 的骨架
hint/task-4     Task 4 的骨架
solution        四個 Task 的完整實作
```

hint branch 給的是**骨架不是答案**：檔案已經建好、interface 已經宣告、函式簽名已經寫好，但內容是 `panic("TODO")`。

這是刻意的。分層的難點從來不是「怎麼寫」，而是「這段要放哪」，而 hint 給出的正是那個決定。看完 hint，剩下的填空反而是簡單的部分。

**不要 merge hint branch。** 它會蓋掉你寫到一半的東西，而且 `panic("TODO")` 會讓程式跑不起來。要看就用：

```bash
git show hint/task-1:internal/form/service.go
```

## 開始

```bash
git checkout -b my-work        # 從 main 開自己的分支

docker compose up -d           # 啟動 PostgreSQL
go mod tidy                    # 下載相依套件，產生 go.sum
go run ./cmd/backend           # migration 會自動跑
```

另開一個終端機：

```bash
curl -i -X POST http://localhost:8080/api/forms \
  -H "Content-Type: application/json" \
  -d '{"title":"問卷調查","description":"說明文字"}'
```

拿到 `201` 就代表環境沒問題，可以開始 Task 1。

`api/` 底下有 Yaak 設定，匯入後所有 Task 的請求都已經建好。

## 專案結構

```
cmd/backend/main.go                     進入點，負責把各層接起來
databaseutil/migration.go               migration，不用動
internal/database/migrations/           schema 版本
internal/form/
  queries.sql                           SQL 來源
  db.go  models.go  queries.sql.go      sqlc 生成，不要手改
  form.go                               ⚠️ 沒有分層的起點，Task 1 要把它拆掉
```

**Querier 已經全部給你了。** 四個 Task 需要的資料庫操作 `queries.sql` 裡都有，你不需要寫任何 SQL，也不需要跑 `sqlc generate`。這個 LAB 練的是 Service 與 Handler 的分工。

## 自我檢查

做完之後跑一遍，三個指令都應該沒有輸出：

```bash
grep -rn "net/http" internal/form/service.go
grep -rn "http.Status" internal/form/service.go
grep -rn "SELECT\|INSERT\|UPDATE\|DELETE" internal/form/handler.go
```

然後回答教材最後那兩個問題，對著你自己寫的每一個函式各回答一次。
