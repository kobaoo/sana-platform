# Интеграция: аудит изменений договоров

## Импорт

```go
import csh "encore.app/orgstructure/contracts_suppliers_history"
```

## Паттерн

1. Получить состояние **до** мутации
2. Выполнить мутацию
3. Получить состояние **после** мутации
4. Конвертировать обе строки через `csh.EntToContract()`
5. Вызвать `csh.InsertAuditRecord(ctx, id, opType, old, new)`

## Соответствие эндпоинтов

| Эндпоинт | Операция |
|---|---|
| `POST /suppliers/{id}/contracts` | `csh.OpCreate` |
| `PATCH /contracts-suppliers/{id}` | `csh.OpUpdate` |
| `POST /contracts-suppliers/{id}/amendment` | `csh.OpUpdate` |
| `POST /contracts-suppliers/{id}/upload-file` | `csh.OpUpdate` |
| `DELETE /contracts-suppliers/{id}` | `csh.OpDelete` |

## Пример: CREATE

Старого состояния нет — передаём `nil`.

```go
newContract := csh.EntToContract(insertedRow)
csh.InsertAuditRecord(ctx, newContract.ID, csh.OpCreate, nil, newContract)
```

## Пример: UPDATE

```go
oldContract := csh.EntToContract(rowBefore)
// ... мутация ...
newContract := csh.EntToContract(rowAfter)
csh.InsertAuditRecord(ctx, id, csh.OpUpdate, oldContract, newContract)
```

## Пример: DELETE

```go
oldContract := csh.EntToContract(rowBefore)
// ... soft delete ...
newContract := *oldContract
newContract.IsActive = false
csh.InsertAuditRecord(ctx, id, csh.OpDelete, oldContract, &newContract)
```

## Что записывается автоматически

- `changed_by` — из контекста авторизации (Keycloak ID)
- `snapshot` — полное состояние после мутации
- `diff` — только изменённые поля: `{ "field": { "old": X, "new": Y } }`
