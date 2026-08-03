# Yaak

匯入 `yaak.sdc-lab.json`（Yaak → Import），選 `local` 環境。

`formId` 這個環境變數要自己填：先跑 Task 1 的 POST，把回傳的 `id` 貼進去，
後面所有帶 `{id}` 的請求就都能用了。

Task 3 的「已封存」情境需要先手動把資料改成封存狀態：

```sql
UPDATE forms SET archived = true WHERE id = '<formId>';
```
