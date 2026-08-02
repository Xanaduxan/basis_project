# Task Manager API

## Первый запуск

```powershell
Copy-Item .env.example .env
docker compose up -d --build
```

## Миграции

После запуска MySQL:

```powershell
docker run --rm --network basis_project_default -v "${PWD}/internal/migrations:/migrations" migrate/migrate:v4.18.3 -path=/migrations -database "mysql://task_user:task_password@tcp(mysql:3306)/task_manager?multiStatements=true" up
```

## Статус контейнеров

```powershell
docker compose ps
```

## Логи API

```powershell
docker compose logs -f api
```

## Тесты

```powershell
go test ./...
```

## Остановка

```powershell
docker compose down
```

Остановка с удалением данных MySQL:

```powershell
docker compose down -v
```

API: `http://localhost:8080`

Метрики: `http://localhost:8080/metrics`
