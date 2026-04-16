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

## Требования

- Docker Desktop запущен
- Atlas dev-база запущена (один раз, держи в фоне):
  ```bash
  cd db && docker compose up -d
  ```

---

## Добавить/изменить поле или таблицу — 3 шага

### 1. Отредактировать схему

Схемы живут в `db/ent/schema/`. Добавить поле или создать новый файл:

```go
// db/ent/schema/organization.go
func (Organization) Fields() []ent.Field {
    return []ent.Field{
        field.String("new_field")...,
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
cd db && atlas migrate diff <описание_изменения> --env local
```

Новый файл появится в `db/migrations/`. Закоммить его.

### Применить

```bash
encore run
```

Encore автоматически применяет все миграции из `db/migrations/` при старте.

---

## Добавить новый сервис с новой сущностью

1. Создать файл схемы: `db/ent/schema/<entity>.go`
2. `go generate ./db/ent/...`
3. `cd db && atlas migrate diff add_<entity> --env local`
4. Реализовать сервис, импортируя ent из `encore.app/db/ent`

Пример импорта в сервисе:

```go
import (
    "encore.app/db/ent"
    "encore.app/db/ent/organization"
)
```

---

## Правила

- **Никогда не редактировать** уже существующие файлы миграций — создавай новый `diff`
- **Никогда не удалять** файлы миграций вручную через `rm` — ломает `atlas.sum`. Если удалил случайно: `cd db && atlas migrate hash` восстановит контрольные суммы
- **Файлы `db/ent/`** (кроме `db/ent/schema/`) — автогенерируемые, не трогай руками
- **Схемы сущностей** — только в `db/ent/schema/`, не в папках сервисов
- **`db/migrations/atlas.sum`** — коммитить, Atlas использует его для проверки целостности

---

## Снос локальной БД (если нужно применить миграции заново)

```bash
encore db shell lms
```

```sql
DROP SCHEMA public CASCADE;
CREATE SCHEMA public;
\q
```

После этого `encore run` создаст всё заново.
