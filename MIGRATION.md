# Миграции (Ent + Atlas)

## Архитектура

| Что | Где |
|---|---|
| Схемы сущностей | `db/ent/schema/*.go` |
| Генерация ent-кода | `go generate ./db/ent/...` |
| Atlas конфиг | `db/atlas.hcl` |
| SQL-миграции | `db/migrations/` |
| Dev-база для Atlas | `db/docker-compose.yml` (postgres:16, port 54320) |

Один `db/ent/schema/` — единый источник истины для всех сущностей.  
Один `db/migrations/` — все SQL-миграции для всех сервисов.

---

## Формат файлов миграций

Atlas генерирует пары файлов в формате golang-migrate:

```
20260416154539_initial.up.sql    ← Encore применяет этот файл при старте
20260416154539_initial.down.sql  ← rollback (применяется вручную при необходимости)
```

Encore читает только `.up.sql` файлы, в порядке timestamp.

---

## Сценарий А — Просто запустить бэк

Ничего делать не нужно. Миграции применяются автоматически:

```bash
encore run
```

Encore сам находит все `*.up.sql` в `db/migrations/` и применяет те, которые ещё не применены.

---

## Сценарий Б — Добавить поле или таблицу (3 шага)

### 1. Отредактировать схему

Схемы живут в `db/ent/schema/`. Добавить поле или создать новый файл:

```go
// db/ent/schema/organization.go
func (Organization) Fields() []ent.Field {
    return []ent.Field{
        field.String("new_field").Optional(),
        // ...
    }
}
```

### 2. Перегенерировать Ent-код

```bash
go generate ./db/ent/...
```

Обновит все файлы в `db/ent/` автоматически.

### 3. Сгенерировать SQL-миграцию

```bash
# Поднять Atlas dev DB (нужна только для генерации, не для encore run)
cd db && docker compose up -d

# Сгенерировать миграцию
atlas migrate diff <описание_изменения> --env local

# Можно остановить dev DB
docker compose down
```

Новые файлы `*.up.sql` и `*.down.sql` появятся в `db/migrations/`. Закоммить их.

### 4. Применить

```bash
encore run
```

---

## Сценарий В — Добавить новый сервис с новой сущностью

1. Создать файл схемы: `db/ent/schema/<entity>.go`
2. `go generate ./db/ent/...`
3. `cd db && docker compose up -d && atlas migrate diff add_<entity> --env local && docker compose down`
4. Реализовать сервис, импортируя ent из `encore.app/db/ent`

Пример импорта в сервисе:

```go
import (
    "encore.app/db/ent"
    "encore.app/db/ent/organization"
)
```

---

## Сброс локальной БД (если нужно применить миграции заново)

> Используй только локально. Удаляет все данные в твоей БД.

**Не удалять файлы миграций** — сбрасывают только данные в БД:

```bash
encore db shell lms
```

```sql
DROP SCHEMA public CASCADE;
CREATE SCHEMA public;
\q
```

```bash
encore run
```

Encore применит все `.up.sql` с нуля.

---

## Правила

- **Никогда не редактировать** уже существующие файлы миграций — создавай новый `diff`
- **Никогда не удалять** файлы миграций вручную — ломает `atlas.sum`
  - Если удалил случайно: `cd db && atlas migrate hash` восстановит контрольные суммы
- **Файлы `db/ent/`** (кроме `db/ent/schema/`) — автогенерируемые, не трогай руками
- **Схемы сущностей** — только в `db/ent/schema/`, не в папках сервисов
- **`db/migrations/atlas.sum`** — коммитить вместе с миграциями
