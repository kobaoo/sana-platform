# Руководство по деплою — Sana Platform Backend

Этот документ описывает всё необходимое для развёртывания бэкенда Sana Platform в среде Kubernetes. Предназначен для DevOps-инженеров.

---

## Содержание

1. [Обзор приложения](#1-обзор-приложения)
2. [Требования](#2-требования)
3. [Переменные окружения и секреты](#3-переменные-окружения-и-секреты)
4. [Конфигурация инфраструктуры](#4-конфигурация-инфраструктуры)
5. [Передача секретов в Pod](#5-передача-секретов-в-pod)
6. [Сборка Docker-образа](#6-сборка-docker-образа)
7. [CI/CD Pipeline (GitLab)](#7-cicd-pipeline-gitlab)
8. [Миграции базы данных](#8-миграции-базы-данных)
9. [Проверка работоспособности](#9-проверка-работоспособности)
10. [Откат](#10-откат)

---

## 1. Обзор приложения

**Стек:**
- Язык: Go 1.26
- Фреймворк: [Encore.go](https://encore.dev) v1.52.1
- База данных: PostgreSQL 16 (единая БД `lms` для всех сервисов)
- Авторизация: Keycloak (JWT / OpenID Connect)
- ORM / миграции: Ent + Atlas

**Сервисы приложения** (все запускаются в одном Docker-образе):

| Сервис | Путь | Назначение |
|---|---|---|
| `auth` | `auth/authhandler` | Проверка JWT-токенов Keycloak |
| `organizations` | `orgstructure/organizations` | Управление организациями |
| `users` | `orgstructure/users` | Управление пользователями, синхронизация с Keycloak |
| `employees` | `orgstructure/employees` | Управление сотрудниками, синхронизация с Keycloak |
| `clients` | `orgstructure/clients` | Управление клиентами |
| `dzo` | `orgstructure/dzo` | Управление ДЗО-организациями |
| `requests` | `orgstructure/requests` | Управление заявками |

**Порт приложения:** `8080` (по умолчанию в Encore, переопределяется через `PORT`)

---

## 2. Требования

### На машине, выполняющей сборку образа

| Инструмент | Версия | Зачем |
|---|---|---|
| [Encore CLI](https://encore.dev/docs/install) | ≥ 1.52.1 | Сборка Docker-образа (`encore build docker`) |
| Go | ≥ 1.26 | Компиляция приложения |
| Docker | любая актуальная | Сборка и пуш образа |

Установка Encore CLI:
```bash
curl -L https://encore.dev/install.sh | bash
```

Проверка установки:
```bash
encore version
# Ожидаемый вывод: encore version 1.52.x
```

### Внешние зависимости (должны быть запущены до деплоя приложения)

| Зависимость | Версия | Описание |
|---|---|---|
| PostgreSQL | 16 | Единая БД приложения. Имя базы: `lms` |
| Keycloak | ≥ 24.0 | Сервер авторизации. Realm: `sana-lms` |

### Требования к PostgreSQL

- Должна существовать база данных с именем **`lms`**
- Пользователь должен иметь права: `CREATE TABLE`, `CREATE INDEX`, `INSERT`, `UPDATE`, `SELECT`, `DELETE` в схеме `public`

Создание БД и пользователя (пример):
```sql
CREATE DATABASE lms;
CREATE USER sana_app WITH PASSWORD 'your-strong-password';
GRANT ALL PRIVILEGES ON DATABASE lms TO sana_app;
```

### Требования к Keycloak

Перед деплоем в Keycloak должны быть настроены:

- Realm с именем **`sana-lms`**
- Client для Admin API с именем **`sana-lms-admin-api`** (тип: `confidential`, Service Accounts Enabled: `true`)
- У сервисного аккаунта клиента должны быть роли в realm `sana-lms`: `manage-users`, `view-users`, `query-users`

---

## 3. Переменные окружения и секреты

Приложение читает секреты через механизм Encore Secrets. При self-hosted деплое секреты передаются как переменные окружения, на которые ссылается `infra.config.json` (см. раздел 4).

### Полный список секретов

| Переменная окружения | Encore-секрет | Сервис | Описание | Пример значения |
|---|---|---|---|---|
| `DB_PASSWORD` | — | все | Пароль пользователя PostgreSQL | `s3cr3t-db-pass` |
| `KEYCLOAK_ISSUER_URL` | `KeycloakIssuerURL` | `auth`, `users` | URL realm в Keycloak | `http://keycloak:8080/realms/sana-lms` |
| `KEYCLOAK_AUDIENCE` | `KeycloakAudience` | `auth` | Audience JWT-токена | `account` |
| `KEYCLOAK_ADMIN_USER` | `KeycloakAdminUser` | `users` | Логин admin-пользователя Keycloak | `admin` |
| `KEYCLOAK_ADMIN_PASSWORD` | `KeycloakAdminPassword` | `users` | Пароль admin-пользователя Keycloak | `admin-pass` |
| `KEYCLOAK_ADMIN_CLIENT_ID` | `KeycloakAdminClientID` | `employees` | ID клиента Admin API | `sana-lms-admin-api` |
| `KEYCLOAK_ADMIN_CLIENT_SECRET` | `KeycloakAdminClientSecret` | `employees` | Секрет клиента Admin API | `abc123secret` |

> ⚠️ Все значения из колонки «Переменная окружения» должны быть установлены в окружении Pod'а (через Kubernetes Secret). Значения из колонки «Encore-секрет» — это внутренние имена в коде приложения, они прописаны в `infra.config.json`.

---

## 4. Конфигурация инфраструктуры

Encore требует файл `infra.config.json` — он передаётся при сборке образа и описывает подключения к БД, секреты и метаданные окружения.

**Создай файл `infra.config.json` в корне репозитория:**

```json
{
  "$schema": "https://encore.dev/schemas/infra.schema.json",
  "metadata": {
    "app_id": "sana-platform",
    "env_name": "production",
    "env_type": "production",
    "cloud": "self_hosted",
    "base_url": "http://<ВНЕШНИЙ_АДРЕС_ИЛИ_INGRESS>"
  },
  "sql_servers": [
    {
      "host": "<POSTGRES_HOST>:<POSTGRES_PORT>",
      "databases": {
        "lms": {
          "username": "<POSTGRES_USER>",
          "password": { "$env": "DB_PASSWORD" }
        }
      }
    }
  ],
  "secrets": {
    "KeycloakIssuerURL":        { "$env": "KEYCLOAK_ISSUER_URL" },
    "KeycloakAudience":         { "$env": "KEYCLOAK_AUDIENCE" },
    "KeycloakAdminUser":        { "$env": "KEYCLOAK_ADMIN_USER" },
    "KeycloakAdminPassword":    { "$env": "KEYCLOAK_ADMIN_PASSWORD" },
    "KeycloakAdminClientID":    { "$env": "KEYCLOAK_ADMIN_CLIENT_ID" },
    "KeycloakAdminClientSecret":{ "$env": "KEYCLOAK_ADMIN_CLIENT_SECRET" }
  },
  "graceful_shutdown": {
    "total": 30
  }
}
```

**Что подставить:**

| Плейсхолдер | Что указать |
|---|---|
| `<ВНЕШНИЙ_АДРЕС_ИЛИ_INGRESS>` | Публичный URL приложения (например, `https://api.sana.example.com`) |
| `<POSTGRES_HOST>` | Хост PostgreSQL внутри кластера (например, `postgres-service`) |
| `<POSTGRES_PORT>` | Порт PostgreSQL (обычно `5432`) |
| `<POSTGRES_USER>` | Имя пользователя БД |

> ⚠️ Файл `infra.config.json` **не должен содержать реальных паролей** — только ссылки `{ "$env": "..." }`. Реальные значения передаются через переменные окружения Pod'а.

> ⚠️ Имя базы данных `"lms"` в этом файле должно совпадать с именем, объявленным в коде (`sqldb.Named("lms")` в `db/db.go`). Не менять.

---

## 5. Передача секретов в Pod

Все переменные окружения из раздела 3 должны попасть в Pod через **Kubernetes Secret**. Это стандартный способ безопасной передачи паролей в кластере — значения хранятся в зашифрованном виде и не попадают в код или логи.

### Создание Kubernetes Secret

```bash
kubectl create secret generic sana-secrets \
  --namespace=<NAMESPACE> \
  --from-literal=DB_PASSWORD='ваш-пароль-бд' \
  --from-literal=KEYCLOAK_ISSUER_URL='http://keycloak:8080/realms/sana-lms' \
  --from-literal=KEYCLOAK_AUDIENCE='account' \
  --from-literal=KEYCLOAK_ADMIN_USER='admin' \
  --from-literal=KEYCLOAK_ADMIN_PASSWORD='ваш-пароль-keycloak' \
  --from-literal=KEYCLOAK_ADMIN_CLIENT_ID='sana-lms-admin-api' \
  --from-literal=KEYCLOAK_ADMIN_CLIENT_SECRET='ваш-client-secret'
```

Проверить что Secret создался (значения будут скрыты — это нормально):
```bash
kubectl get secret sana-secrets -n <NAMESPACE>
```

### Подключение Secret к Deployment

В манифесте Deployment Pod'а с приложением должна быть секция `envFrom`, которая подхватывает **все** ключи из Secret как переменные окружения:

```yaml
spec:
  containers:
    - name: sana-backend
      image: registry.gitlab.com/<GITLAB_GROUP>/<GITLAB_PROJECT>/sana-platform:latest
      ports:
        - containerPort: 8080
      envFrom:
        - secretRef:
            name: sana-secrets
```

> ⚠️ Secret должен быть создан **до** применения Deployment, иначе Pod не запустится.

> ⚠️ При изменении любого секрета (например, смене пароля) нужно пересоздать Secret и перезапустить Pod:
> ```bash
> kubectl rollout restart -n <NAMESPACE> deployment/sana-backend
> ```

---

## 6. Сборка Docker-образа

Encore не использует стандартный `Dockerfile`. Образ собирается командой Encore CLI, которая компилирует приложение и упаковывает его в минимальный образ на базе `scratch`.

### Шаг 1. Авторизоваться в GitLab Container Registry

```bash
docker login registry.gitlab.com
# Введи логин и Personal Access Token с правами read_registry / write_registry
```

### Шаг 2. Собрать образ

Выполни в корне репозитория (там, где лежит `encore.app` и `infra.config.json`):

```bash
encore build docker --config infra.config.json sana-platform:latest
```

Флаги команды:

| Флаг | Описание |
|---|---|
| `--config infra.config.json` | Путь к файлу конфигурации инфраструктуры |
| `sana-platform:latest` | Имя и тег создаваемого образа |

Дополнительные флаги (при необходимости):

```bash
# Указать конкретный тег (рекомендуется для прода — не latest)
encore build docker --config infra.config.json sana-platform:v1.2.3

# Собрать только конкретные сервисы
encore build docker --config infra.config.json --services=auth,users sana-platform:latest
```

### Шаг 3. Запушить образ в GitLab Registry

```bash
# Тегируем образ для GitLab Registry
docker tag sana-platform:latest registry.gitlab.com/<GITLAB_GROUP>/<GITLAB_PROJECT>/sana-platform:latest

# Пушим
docker push registry.gitlab.com/<GITLAB_GROUP>/<GITLAB_PROJECT>/sana-platform:latest
```

Замени `<GITLAB_GROUP>` и `<GITLAB_PROJECT>` на реальные значения из URL репозитория.

**Проверка:** После пуша образ должен появиться в GitLab → ваш проект → **Deploy → Container Registry**.

---

## 7. CI/CD Pipeline (GitLab)

Чтобы сборка и пуш образа происходили автоматически при каждом пуше в ветку `main`, добавь в корень репозитория файл `.gitlab-ci.yml`.

### Что нужно настроить в GitLab перед использованием

В GitLab → твой проект → **Settings → CI/CD → Variables** добавь следующие переменные (тип: `Variable`, Protected: `true`, Masked: `true`):

| Переменная | Значение | Зачем |
|---|---|---|
| `INFRA_DB_PASSWORD` | пароль PostgreSQL | Подставляется в `infra.config.json` при сборке |
| `INFRA_KEYCLOAK_ISSUER_URL` | URL Keycloak realm | То же |
| `INFRA_KEYCLOAK_AUDIENCE` | `account` | То же |
| `INFRA_KEYCLOAK_ADMIN_USER` | логин admin Keycloak | То же |
| `INFRA_KEYCLOAK_ADMIN_PASSWORD` | пароль admin Keycloak | То же |
| `INFRA_KEYCLOAK_ADMIN_CLIENT_ID` | `sana-lms-admin-api` | То же |
| `INFRA_KEYCLOAK_ADMIN_CLIENT_SECRET` | client secret | То же |
| `INFRA_BASE_URL` | публичный URL бэкенда | То же |
| `INFRA_POSTGRES_HOST` | хост PostgreSQL в кластере | То же |
| `INFRA_POSTGRES_USER` | пользователь PostgreSQL | То же |

> ⚠️ Эти переменные нужны только для сборки образа — в сам образ пароли **не попадают** (только ссылки `$env`). Реальные значения в Pod приходят из Kubernetes Secret (см. раздел 5).

### Файл `.gitlab-ci.yml`

Создай в корне репозитория файл `.gitlab-ci.yml`:

```yaml
stages:
  - build

variables:
  IMAGE_NAME: $CI_REGISTRY_IMAGE/sana-platform
  IMAGE_TAG: $CI_COMMIT_SHORT_SHA  # уникальный тег = короткий хэш коммита

build-and-push:
  stage: build
  # Запускается только при пуше в ветку main
  rules:
    - if: $CI_COMMIT_BRANCH == "main"

  before_script:
    # Устанавливаем Encore CLI
    - curl -L https://encore.dev/install.sh | bash
    - export PATH="$HOME/.encore/bin:$PATH"
    - encore version

    # Авторизуемся в GitLab Container Registry
    # CI_REGISTRY_USER и CI_REGISTRY_PASSWORD — встроенные переменные GitLab, настраивать не нужно
    - docker login -u $CI_REGISTRY_USER -p $CI_REGISTRY_PASSWORD $CI_REGISTRY

    # Генерируем infra.config.json из переменных окружения GitLab CI
    - |
      cat > infra.config.json << EOF
      {
        "$schema": "https://encore.dev/schemas/infra.schema.json",
        "metadata": {
          "app_id": "sana-platform",
          "env_name": "production",
          "env_type": "production",
          "cloud": "self_hosted",
          "base_url": "$INFRA_BASE_URL"
        },
        "sql_servers": [
          {
            "host": "$INFRA_POSTGRES_HOST:5432",
            "databases": {
              "lms": {
                "username": "$INFRA_POSTGRES_USER",
                "password": { "\$env": "DB_PASSWORD" }
              }
            }
          }
        ],
        "secrets": {
          "KeycloakIssuerURL":         { "\$env": "KEYCLOAK_ISSUER_URL" },
          "KeycloakAudience":          { "\$env": "KEYCLOAK_AUDIENCE" },
          "KeycloakAdminUser":         { "\$env": "KEYCLOAK_ADMIN_USER" },
          "KeycloakAdminPassword":     { "\$env": "KEYCLOAK_ADMIN_PASSWORD" },
          "KeycloakAdminClientID":     { "\$env": "KEYCLOAK_ADMIN_CLIENT_ID" },
          "KeycloakAdminClientSecret": { "\$env": "KEYCLOAK_ADMIN_CLIENT_SECRET" }
        },
        "graceful_shutdown": {
          "total": 30
        }
      }
      EOF

  script:
    # Собираем образ через Encore CLI
    - encore build docker --config infra.config.json sana-platform:$IMAGE_TAG

    # Тегируем и пушим в GitLab Registry
    - docker tag sana-platform:$IMAGE_TAG $IMAGE_NAME:$IMAGE_TAG
    - docker tag sana-platform:$IMAGE_TAG $IMAGE_NAME:latest
    - docker push $IMAGE_NAME:$IMAGE_TAG
    - docker push $IMAGE_NAME:latest

    - echo "Образ запушен -> $IMAGE_NAME:$IMAGE_TAG"
```

### Как это работает

```
Пуш в main
    │
    ▼
GitLab CI запускает job build-and-push
    │
    ├── Устанавливает Encore CLI
    ├── Генерирует infra.config.json из переменных GitLab
    ├── encore build docker → собирает образ
    └── docker push → образ в GitLab Container Registry
                          │
                          ▼
                  DevOps деплоит образ в Kubernetes
                  (kubectl set image или Helm upgrade)
```

### Проверка pipeline

После пуша в `main` перейди в GitLab → **CI/CD → Pipelines** — должен появиться запущенный pipeline. Нажми на него чтобы увидеть логи сборки.

---

## 8. Миграции базы данных

Миграции лежат в `db/migrations/` в формате `*.up.sql` / `*.down.sql`.

### Как применяются миграции: два сценария

**В локальной и dev-среде (`encore run`)** — миграции применяются **автоматически**. Encore-демон при старте сам находит все `*.up.sql` в `db/migrations/` и накатывает те, которые ещё не применены. Ничего делать не нужно.

**В production (Kubernetes)** — миграции **не применяются автоматически**. Encore self-hosted Docker-образ не содержит логики применения миграций при старте Pod'а (миграционный раннер — часть `encore run`, а не production runtime).

Применение миграций в production выполняется отдельным шагом CI через **Atlas CLI**:

- Стадия `build:migrate` в `.gitlab-ci.yml` собирает мини-образ `${CI_REGISTRY_IMAGE}/migrate:latest` (`Dockerfile.migrate`) с Atlas + `db/migrations/`.
- Стадия `migrate` запускает Kubernetes Job `sana-lms-migrate` (`k8s/migrate-job.yaml`) перед стадией `deploy` — Job применяет pending миграции через `atlas migrate apply` и завершает работу.
- Деплой backend'а проходит только если Job успешно завершился.

> Atlas CLI используется и разработчиками для **генерации** новых миграций при изменении схемы БД (см. `MIGRATION.md`), и в CI для **применения** миграций в production.

### Текущие миграции (в порядке применения)

| Файл | Что делает |
|---|---|
| `20260416154539_initial.up.sql` | Создаёт основные таблицы: `organizations`, `users`, `requests`, `training_events`, `training_participants` |
| `20260416133254_add_dzo_organizations_and_employee.up.sql` | Добавляет таблицы `dzo_organizations`, `employees` |
| `20260416170000_seed_users.up.sql` | Начальные данные: тестовые пользователи |
| `20260416170500_seed_training_and_requests.up.sql` | Начальные данные: тренинги и заявки |
| `20260417152635_add_requests.up.sql` | Расширяет таблицу `requests` |
| `20260419000000_clients.up.sql` | Добавляет таблицу `clients` |
| `20260420121445_add_is_deleted_to_employees.up.sql` | Добавляет поле `is_deleted` в `employees` |
| `20260421184237_dzo_add_created_at_updated_at.up.sql` | Добавляет `created_at`, `updated_at` в `dzo_organizations` |

### Что требуется от DevOps

Перед первым запуском Pod'а убедиться что база данных `lms` **существует** и пользователь имеет права `CREATE TABLE`, `CREATE INDEX`, `INSERT`, `UPDATE`, `SELECT`, `DELETE` (см. раздел 2). Таблицы создаст Job `sana-lms-migrate` через Atlas.

### Проверка что миграции применились

Логи Job'а: `kubectl logs -n sana-lms job/sana-lms-migrate`. Они также печатаются в job-логе GitLab CI на стадии `migrate`. Для ручной проверки состояния через port-forward:

```bash
# Пробросить порт PostgreSQL
kubectl port-forward -n <NAMESPACE> svc/<POSTGRES_SERVICE> 5432:5432

# Проверить список таблиц
psql -h localhost -U <POSTGRES_USER> -d lms -c "\dt"
```

Ожидаемый результат — в базе присутствуют таблицы: `organizations`, `users`, `employees`, `dzo_organizations`, `clients`, `requests`, `training_events`, `training_participants`.

---

## 9. Проверка работоспособности

После деплоя Pod'а проверить что приложение запустилось корректно.

### Проверка логов Pod'а

```bash
kubectl logs -n <NAMESPACE> deploy/sana-backend --follow
```

Признак успешного старта в логах — отсутствие `ERROR` при инициализации и строки вида:
```
encore: starting service auth
encore: starting service organizations
encore: starting service users
...
encore: all services started
```

### Проверка доступности API

Приложение запускается на порту `8080`. Для проверки из кластера:

```bash
# Пробросить порт для локальной проверки
kubectl port-forward -n <NAMESPACE> deploy/sana-backend 8080:8080

# Проверить публичный эндпоинт (Encore генерирует его автоматически)
curl -i http://localhost:8080/
```

Ожидаемый ответ — HTTP `200` или `404` (не `502` / `connection refused`). Ответ `404` на корень — нормален, это означает что приложение работает, просто нет эндпоинта на `/`.

### Проверка подключения к БД

Если приложение не может подключиться к PostgreSQL — оно упадёт при старте с ошибкой вида:
```
failed to connect to database "lms": ...
```

В этом случае проверь:
- Правильность хоста и порта PostgreSQL в `infra.config.json`
- Переменную окружения `DB_PASSWORD` в Pod'е
- Сетевую доступность PostgreSQL из Pod'а приложения

### Проверка авторизации

Сделай запрос к любому `auth`-защищённому эндпоинту без токена — должен вернуться `401`:

```bash
curl -i http://localhost:8080/organizations
# Ожидаемый ответ: HTTP/1.1 401 Unauthorized
```

---

## 10. Откат

### Откат приложения

Откатить деплой к предыдущей версии образа:

```bash
kubectl set image -n <NAMESPACE> deployment/sana-backend \
  sana-backend=registry.gitlab.com/<GITLAB_GROUP>/<GITLAB_PROJECT>/sana-platform:<ПРЕДЫДУЩИЙ_ТЕГ>
```

Или через `kubectl rollout`:

```bash
kubectl rollout undo -n <NAMESPACE> deployment/sana-backend

# Проверить статус отката
kubectl rollout status -n <NAMESPACE> deployment/sana-backend
```

### Откат миграций

Encore применяет миграции автоматически при старте — отдельной команды для отката в production нет. Если новая миграция сломала что-то в БД, алгоритм такой:

1. Откатить образ приложения к предыдущей версии (см. выше)
2. Вручную применить `.down.sql` файл для нужной миграции через psql:

```bash
# Пробросить порт PostgreSQL
kubectl port-forward -n <NAMESPACE> svc/<POSTGRES_SERVICE> 5432:5432

# Применить down-миграцию вручную
psql -h localhost -U <POSTGRES_USER> -d lms \
  -f db/migrations/<TIMESTAMP>_<NAME>.down.sql
```

> ⚠️ **Важно:** seed-миграции (`seed_users`, `seed_training_and_requests`) не имеют `.down.sql` файлов — они не откатываются. Перед откатом схемы убедись что это необходимо и согласуй с командой бэкенда.

---

*Документ актуален для версии приложения на `encore.dev v1.52.1` и Go 1.26.*