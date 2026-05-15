# Стиль кода

## Структура файлов

Каждый сервисный файл `<service>.go` делится ровно на три секции через баннер-комментарии:

```go
// ════ DATABASE ════
// ════ ENDPOINTS ════
// ════ INTERNAL ════
```

Порядок секций фиксирован. Не менять местами и не добавлять промежуточные секции.

## Эндпоинты

Каждая функция-эндпоинт строго следует порядку:

```go
// 1. Проверка роли
ud, ok := auth.Data().(*AuthData)
if !ok || ud.Role != "admin" {
    return nil, errs.B().Code(errs.PermissionDenied).Msg("...").Err()
}

// 2. Валидация входных данных
if strings.TrimSpace(req.Name) == "" {
    return nil, errs.B().Code(errs.InvalidArgument).Msg("...").Err()
}

// 3. Вызов DB-хелпера
result, err := insertSomething(ctx, req)
if err != nil {
    return nil, err
}

// 4. Возврат ответа
return &Response{...}, nil
```

Не смешивать SQL и бизнес-логику внутри тела эндпоинта.

## Обработка ошибок

Все ошибки оборачиваются через билдер:

```go
// Ошибка с причиной (непредвиденные ошибки БД)
return nil, errs.B().Code(errs.Internal).Msg("failed to create org").Cause(err).Err()

// Ошибка без причины (ожидаемые случаи)
return nil, errs.B().Code(errs.NotFound).Msg("organization not found").Err()
```

| Ситуация | Код |
|---|---|
| Пустой / некорректный ввод | `errs.InvalidArgument` |
| Запись не найдена | `errs.NotFound` |
| Нарушение уникальности | `errs.AlreadyExists` |
| Недостаточно прав | `errs.PermissionDenied` |
| Неожиданная ошибка БД | `errs.Internal` + `.Cause(err)` |

Никогда не пробрасывать голый `err` — всегда оборачивать.

## SQL

```go
// Правильно — явные колонки, плейсхолдеры
db.QueryRow(ctx, `
    SELECT id, name, code, parent_id, type, is_active, created_at, updated_at
    FROM organizations
    WHERE id = $1
`, id)

// Неправильно — SELECT *, конкатенация строк
db.QueryRow(ctx, "SELECT * FROM organizations WHERE id = '" + id + "'")
```

Порядок колонок в `SELECT` должен совпадать с порядком аргументов в `Scan`.

Частичное обновление — только через `COALESCE`:

```go
UPDATE organizations
SET name = COALESCE($2, name), code = COALESCE($3, code), updated_at = NOW()
WHERE id = $1
```

## Типы и константы

Типизированные строки для перечислений:

```go
type OrgType string

const (
    OrgTypeCompany    OrgType = "company"
    OrgTypeSubsidiary OrgType = "subsidiary"
)

func (o OrgType) IsValid() bool { ... }
```

Опциональные поля в запросах — указатели с `omitempty`:

```go
type UpdateOrgRequest struct {
    Name *string  `json:"name,omitempty"`
    Code *string  `json:"code,omitempty"`
}
```

## Коллекции

Пустой срез всегда инициализируется явно:

```go
// Правильно
orgs := []Organization{}

// Неправильно
var orgs []Organization
```

## Тесты

- Вспомогательные функции помечаются `t.Helper()`.
- Контекстные хелперы: `adminCtx()`, `hrCtx()`, `employeeCtx()`.
- Фабричные хелперы: `makeOrg(t, name, code, type)`.
- Проверка кода ошибки через `errs.Code(err)`, не через строку.

```go
if errs.Code(err) != errs.PermissionDenied {
    t.Errorf("expected PermissionDenied, got %v", errs.Code(err))
}
```

## Именование

- Файлы: `<service>_types.go`, `<service>.go`, `<service>_test.go`.
- DB-хелперы: `insertX`, `queryXByID`, `queryActiveXs`, `updateX`, `softDeleteX`.
- Ответы: `GetXResponse`, `ListXsResponse`, `DeleteXResponse`.
- Запросы: `CreateXRequest`, `UpdateXRequest`.
