# Миграции (Ent + Atlas)

## Что это
- Ent — описываем таблицы Go-кодом в `<сервис>/ent/schema/`
- Atlas — собирает ent-схемы **всех** сервисов из `db/atlas.hcl` и генерирует SQL-миграцию
- Все миграции лежат в `db/migrations/` — единая БД

## Требования
- Docker Desktop должен быть запущен

## Добавить/изменить таблицу — 4 шага

### 1. Отредактировать схему
Схема живёт внутри сервиса: `orgstructure/organizations/ent/schema/organization.go`

### 2. Перегенерировать Ent-код
```
cd orgstructure/organizations
go generate ./ent/...
```

### 3. Сгенерировать SQL-миграцию через Atlas
```
cd db
atlas migrate diff --env local
```
Новый файл появится в `db/migrations/`

### 4. Применить
```
encore run
```
Encore автоматически применяет миграции из `db/migrations/`

## Добавить новый сервис с таблицами

1. Создать ent-схему: `<группа>/<сервис>/ent/schema/<entity>.go`
2. Добавить путь в `db/atlas.hcl`:
   ```hcl
   src = [
     "ent://../orgstructure/organizations/ent/schema",
     "ent://../<группа>/<сервис>/ent/schema",
   ]
   ```
3. Сгенерировать миграцию: `cd db && atlas migrate diff --env local`

## Важно
- Никогда не редактировать уже существующие файлы миграций
- Файлы внутри `ent/` кроме `ent/schema/` — генерируются автоматически, не трогать руками
- Миграции создаются только в `db/migrations/`, не в сервисах
