# API Spec

這份文件是驗收的唯一依據。教材裡的程式碼片段若與此處不一致，以此處為準。

Base URL：`http://localhost:8080`
所有請求與回應的 `Content-Type` 皆為 `application/json`。

## 共用型別

### FormResponse

```json
{
  "id": "dfdf7650-9d8c-4be6-bff2-6c89a70ddccd",
  "title": "問卷調查",
  "description": "說明文字",
  "createdAt": "2025-10-07T17:22:32.88211+08:00"
}
```

| 欄位 | 型別 | 說明 |
| --- | --- | --- |
| `id` | string (UUID) | |
| `title` | string | |
| `description` | string | 資料庫為 NULL 時回傳空字串 |
| `createdAt` | string (RFC 3339) | |

`archived` 不對外公開。

### ErrorResponse

```json
{ "error": "form not found" }
```

所有 4xx / 5xx 都使用這個格式。`500` 一律回傳 `internal server error`，不得洩漏資料庫錯誤內容。

---

## POST /api/forms

建立表單。（Task 1）

Request：

| 欄位 | 型別 | 必填 | 限制 |
| --- | --- | --- | --- |
| `title` | string | 是 | 去除頭尾空白後不可為空，長度 ≤ 255 |
| `description` | string | 否 | 長度 ≤ 1000 |

Response：`201 Created`，body 為 FormResponse。

| 情境 | Status | `error` |
| --- | --- | --- |
| body 不是合法 JSON | 400 | `invalid request body` |
| `title` 為空 | 400 | `title is required` |
| `title` 超過 255 字 | 400 | `title must be at most 255 characters` |
| `title` 已存在 | 409 | `form title already exists` |
| 其他錯誤 | 500 | `internal server error` |

---

## GET /api/forms

列出表單，依 `created_at` 由新到舊排序。（Task 2）

Query 參數：

| 參數 | 型別 | 預設 | 限制 |
| --- | --- | --- | --- |
| `page` | int | 1 | ≥ 1 |
| `size` | int | 20 | 1 ≤ size ≤ 100 |

Response：`200 OK`

```json
{
  "items": [ { "id": "...", "title": "...", "description": "...", "createdAt": "..." } ],
  "page": 1,
  "size": 20,
  "total": 42
}
```

`items` 在沒有資料時必須是 `[]`，不可以是 `null`。
`total` 是符合條件的總筆數，不是本頁筆數。

| 情境 | Status | `error` |
| --- | --- | --- |
| `page` 不是數字或 < 1 | 400 | `invalid page` |
| `size` 不是數字或 < 1 | 400 | `invalid size` |
| `size` > 100 | 400 | `size must be at most 100` |

---

## GET /api/forms/{id}

取得單一表單。（Task 2）

Response：`200 OK`，body 為 FormResponse。

| 情境 | Status | `error` |
| --- | --- | --- |
| `id` 不是合法 UUID | 400 | `invalid id` |
| 表單不存在 | 404 | `form not found` |

---

## PATCH /api/forms/{id}

部分更新。兩個欄位都是選填，未帶的欄位維持原值。（Task 2）

Request：

| 欄位 | 型別 | 說明 |
| --- | --- | --- |
| `title` | string \| null | 省略代表不變更 |
| `description` | string \| null | 省略代表不變更 |

Response：`200 OK`，body 為更新後的 FormResponse。

| 情境 | Status | `error` |
| --- | --- | --- |
| body 不是合法 JSON | 400 | `invalid request body` |
| `id` 不是合法 UUID | 400 | `invalid id` |
| 兩個欄位都沒帶 | 400 | `no fields to update` |
| `title` 帶了但為空字串 | 400 | `title is required` |
| `title` 超過 255 字 | 400 | `title must be at most 255 characters` |
| 表單不存在 | 404 | `form not found` |
| 新 `title` 與其他表單重複 | 409 | `form title already exists` |

把 `title` 改成與自己現在相同的值，不算重複，回 `200`。

---

## DELETE /api/forms/{id}

刪除表單。（Task 2，封存規則在 Task 3）

Response：`204 No Content`，無 body。

| 情境 | Status | `error` |
| --- | --- | --- |
| `id` 不是合法 UUID | 400 | `invalid id` |
| 表單不存在 | 404 | `form not found` |
| 表單已封存 | 409 | `form is archived` |

---

## POST /api/forms/{id}/duplicate

以既有表單為範本建立一份新的。`description` 沿用來源表單，新標題由請求指定。（Task 4）

Request：

| 欄位 | 型別 | 必填 | 限制 |
| --- | --- | --- | --- |
| `title` | string | 是 | 同 POST /api/forms |

Response：`201 Created`，body 為新表單的 FormResponse。

| 情境 | Status | `error` |
| --- | --- | --- |
| body 不是合法 JSON | 400 | `invalid request body` |
| `id` 不是合法 UUID | 400 | `invalid id` |
| `title` 為空 | 400 | `title is required` |
| 來源表單不存在 | 404 | `form not found` |
| 新 `title` 已存在 | 409 | `form title already exists` |

來源表單即使已封存也可以複製，複製出來的新表單 `archived` 為 `false`。
