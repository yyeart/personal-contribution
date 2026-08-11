# Личный вклад в антискам тренажер

## Автор
Смирнов Вячеслав [@yyeart](https://t.me/yyeart)

## Backend

### Фича пользователей
  #### Доменная сущность
  [backend/internal/domain/models/user.go](backend/internal/domain/models/user.go)
  - Добавил доменную модель пользователя и тип идентификатора пользователя
  - Определил ограничения и валидацию данных пользователя на уровне домена

  #### Сервисный слой
  [backend/internal/services/users](backend/internal/services/users)
  - Создание пользователя
  - Валидация входных данных при создании пользователя
  - Хеширование пароля
  - Получение доменной сущности пользователя из репозитория

  #### Репозиторный слой
  [backend/internal/storage/postgres/users](backend/internal/storage/postgres/user/)
  - Сохранение пользователя в БД
  - Получение пользователя из БД по ID и Email

### Фича прогресса
  #### Доменная сущность
  [backend/internal/domain/models/progress.go](backend/internal/domain/models/progress.go)
  - Модели прогресса по сценариям, ролям и попыткам
  - Модели баллов, динамики, рекомендаций, опыта и достижений

  #### Сервисный слой
  [backend/internal/services/progress](backend/internal/services/progress)
  - Рассчет динамики
  - Построение рекомендаций
  - Начисление достижений и опыта
  - Агрегация общего и ролевого прогресса

  #### Репозиторный слой
  [backend/internal/storage/postgres/progress](backend/internal/storage/postgres/progress)
  - Получение снапшота прогресса из БД
  - Сбор сценариев, завершенных попыток, истории попыток, исходов и активных попыток

  #### Транспортный слой
  [backend/internal/transport/http/progress.go](backend/internal/transport/http/progress.go)
  - Эндпоинт `GET /v1/progress`
  - DTO и маппинг прогресса, ролей, рекомендаций, XP и достижений в JSON

### Обработка ошибок
[backend/internal/domain/errors](backend/internal/domain/errors)
[backend/internal/transport/http/errors_handler.go](backend/internal/transportger/http/errors_handler.go)
- Корректное отображение ошибок и HTTP-статусов
- Обработка ошибок фич `training` и `progress`

## Инфраструктура
- Пул соединений через `pgx`: [backend/internal/storage/postgres/pool](backend/internal/storage/postgres/pool)
- Логирование через `zap`: [backend/internal/logger](backend/internal/logger)
- PostgreSQL и миграции: [docker-compose.yaml](./docker-compose.yaml)
- Таргеты запуска проекта, миграций, тестов, линтера, swagger'а: [Makefile](./Makefile)

# Границы атрибуции
Я не приписываю себе полную реализацию присутствующего в этом репзоитории пакета `domain/models` (а именно `scenario.go`, `session.go`), пакета `domain/errors`,  файлов `main.go`, `transport/http/server.go`
