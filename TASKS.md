# Tasks

API 規格見 [API_SPEC.md](API_SPEC.md)。這裡只寫「要做什麼」與「怎麼算做完」。

---

## Task 1：把 `CreateForm` 拆成三層

`internal/form/form.go` 裡的 `CreateForm` 能正常運作，但它同時做了四件事：解析 JSON、檢查業務規則、操作資料庫、組裝 Response。

把它拆成 `handler.go` 與 `service.go`，並開始使用已經生成好的 Querier。**外部行為完全不變**——重構前後 curl 的輸出應該一模一樣。

拆完之後 `form.go` 應該就不存在了。

### 驗收條件

- [ ] `service.go` 裡沒有 `net/http` 的 import
- [ ] `service.go` 裡沒有出現任何 HTTP status code
- [ ] `handler.go` 裡沒有任何 SQL 字串
- [ ] `handler.go` 裡沒有 `pgxpool` 或 `dbPool`
- [ ] 「標題重複」這個判斷寫在 Service
- [ ] 「標題重複要回 409」這個決定寫在 Handler
- [ ] 重構前後 `POST /api/forms` 的回應一致

最後兩項容易搞混，值得停下來想：**規則本身**屬於 Service，**規則違反時要回什麼 HTTP 狀態**屬於 Handler。Service 只說「衝突了」，Handler 才說「那就是 409」。

### hint/task-1 提供

`service.go`（`Querier` interface、`Service` struct、`NewService`、`Create` 簽名）、`handler.go`（`Store` interface、`Handler` struct、`NewHandler`、`Create` 骨架、`RegisterRoutes`）、`errors.go`（`ErrTitleConflict`）。

---

## Task 2：補完 CRUD

四支 API：`GET /api/forms`、`GET /api/forms/{id}`、`PATCH /api/forms/{id}`、`DELETE /api/forms/{id}`。

Task 1 已經把模式立起來了，這個 Task 是把它重複四次直到變成肌肉記憶。四支裡只有 PATCH 需要額外思考，其餘三支照抄 Create 的形狀就好。

### 分層考點：分頁

`page` 與 `size` 是 HTTP query string，解析與範圍檢查屬於 Handler。
但 `OFFSET = (page - 1) * size` 這個換算屬於 Service，因為 Querier 只認得 limit 與 offset，不認得「第幾頁」。

### 驗收條件

- [ ] 對不存在的 id 發 `DELETE`，回 404 而不是 204 或 500
- [ ] 對不存在的 id 發 `PATCH`，回 404
- [ ] `GET /api/forms?size=999` 回 400
- [ ] 資料表為空時 `items` 是 `[]`
- [ ] `PATCH` 只帶 `title` 時，`description` 沒有被清空
- [ ] `PATCH` 把 `title` 改成跟自己一樣的值，回 200 而不是 409
- [ ] offset 的計算在 Service，不在 Handler
- [ ] 四個 Service 方法都不接收 `*http.Request`

### 提示

- `Delete` 的 sqlc 註解是 `:execrows`，它回傳「影響了幾列」。沒刪到任何一列時 pgx 不會給你 error，這個數字是你唯一的線索。
- 部分更新：Request struct 的欄位用 `*string`，才能區分「沒帶這個欄位」與「帶了空字串」。
- Go 的 nil slice encode 出來是 `null`。用 `make([]FormResponse, 0, len(items))` 初始化就會是 `[]`。
- 判斷「新標題有沒有跟別人撞名」時，先想清楚跟自己撞名要不要算。

### hint/task-2 提供

`service.go` 與 `handler.go` 四組方法的簽名（內容 `panic("TODO")`）、`errors.go` 的 `ErrFormNotFound`、`ListResult` 型別。

---

## Task 3：加上一條業務規則

**已封存（`archived = true`）的表單不能刪除，回 409。**

`archived` 欄位在 migration `2_add_archived` 就存在了，只是到目前為止沒有人用它。

這條規則很容易被寫進 Handler——反正就是一個 if——但它是業務規則。換成 CLI 批次刪除、換成排程任務清理，它一樣要成立。這個 Task 就是練「認出它不該在 Handler」。

測試資料可以直接在資料庫改：

```sql
UPDATE forms SET archived = true WHERE id = '<某個 id>';
```

### 驗收條件

- [ ] 刪除已封存的表單回 409，`error` 是 `form is archived`
- [ ] Handler 裡沒有 `if form.Archived` 這類判斷
- [ ] **Handler 的 `Delete` 沒有因為這條規則變長**
- [ ] Service 回傳的是自己定義的錯誤，不是 `pgx.ErrNoRows`

第三項是這個 Task 真正的驗收點。加了一條業務規則之後 Handler 跟著變長，就是放錯層了。

### hint/task-3 提供

`errors.go` 的 `ErrFormArchived`。

---

## Task 4：一個流程，多個 Querier

`POST /api/forms/{id}/duplicate`

以既有表單為範本建立一份新的。`description` 沿用來源表單，新標題由請求指定。

這對應教材裡明確列為「不應該寫在 Handler」的一項：**呼叫多個 Querier 完成一個流程**。

Service 內部會做三件事：讀出來源表單、檢查新標題有沒有撞名、建立新表單。三次 Querier 呼叫，但 Handler 只呼叫一次 Service，而且完全不知道裡面做了幾次查詢。

### 驗收條件

- [ ] Handler 只呼叫一次 Service
- [ ] Handler 完全不知道這個操作內部做了幾次資料庫查詢
- [ ] **沒有為了這支 API 新增任何 Querier 方法**（三個都是既有的）
- [ ] 來源不存在時回 404，標題撞名時回 409，兩者不會混淆
- [ ] 複製一份已封存的表單會成功，且新表單沒有被封存

第三項是重點。Querier 是可重用的積木，組合積木是 Service 的工作。如果你發現自己想在 `queries.sql` 裡寫一個 `DuplicateForm`，先問問是不是把 Service 的工作推給 Querier 了。

### hint/task-4 提供

`service.go` 的 `Duplicate` 簽名、`handler.go` 的 `Duplicate` 骨架與路由。
