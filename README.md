# PactTelegramService

gRPC-сервис для управления Telegram-сессиями с использованием библиотеки gotd/td (MTProto).

Сервис позволяет:

- Создавать Telegram-сессии через QR-авторизацию
- Поддерживать 2FA (ввод пароля)
- Отправлять сообщения
- Подписываться на входящие сообщения (stream)
- Удалять Telegram-сессии

Каждая сессия изолирована и работает независимо от остальных.

---

## Технологии

- Go 1.26
- gRPC
- gotd/td
- zerolog

---

## Требования

- Go 1.26+
- Telegram APP_ID и APP_HASH

Получить APP_ID и APP_HASH можно на:

https://my.telegram.org → API development tools

---

## Конфигурация

Создайте файл `.env` (пример есть в `.env.example`):

```

TELEGRAM_SERVICE_HOST=localhost

TELEGRAM_SERVICE_PORT=50051


TELEGRAM_APP_ID=YOUR_APP_ID

TELEGRAM_APP_HASH=YOUR_APP_HASH

```

---

## Запуск

1) Без использования docker

Из директории `services/telegram/cmd`:

ВАЖНО: в файле main.go вам нужно будет снять комментарии с участка кода, рядом с которым стоят пояснительные комментарии

```

go run main.go

```

2. С использованием docker
   Из деректории `deployment:`
   1. С Makefile -> make up
   2. Без Makefile -> docker compose up -d

Сервис запустится на:

```

localhost:50051

```

---

## Команды через Makefile

В репозитории есть `Makefile` с готовыми командами для ручного прогона API через `grpcurl` (create_session, submit_password, subscribe_message, delete_session и др.).

Можно запускать так:

```

make create_session

make submit_password

make send_message

make subscribe_message

make delete_session

```

> Названия таргетов смотри в `Makefile` (они соответствуют gRPC методам и тестовым сценариям).

---

## gRPC API

### CreateSession

Создаёт новую Telegram-сессию и запускает поток с QR-кодом.

```

grpcurl -plaintext \

  -d '{}' \

  localhost:50051 \

  pact.telegram.v1.TelegramService/CreateSession

```

---

### SubmitPassword (если включён 2FA)

```

grpcurl -plaintext \

  -d '{

        "sessionId": "PASTE_YOUR_SESSION_ID",

        "password": "PASTE_YOUR_PASSWORD"

      }' \

  localhost:50051 \

  pact.telegram.v1.TelegramService/SubmitPassword

```

---

### SendMessage

```

grpcurl -plaintext \

  -d '{

        "session_id": "PASTE_YOUR_SESSION_ID",

        "peer": "@username",

        "text": "Hello from PactTelegramService"

      }' \

  localhost:50051 \

  pact.telegram.v1.TelegramService/SendMessage

```

---

### SubscribeMessages

Подписка на входящие сообщения (stream).

```

grpcurl -plaintext \

  -d '{

        "session_id": "PASTE_YOUR_SESSION_ID"

      }' \

  localhost:50051 \

  pact.telegram.v1.TelegramService/SubscribeMessages

```

---

### DeleteSession

Удаляет сессию и завершает соединение с Telegram.

```

grpcurl -plaintext \

  -d '{

        "session_id": "PASTE_YOUR_SESSION_ID"

      }' \

  localhost:50051 \

  pact.telegram.v1.TelegramService/DeleteSession

```

---

## Архитектура и ключевые решения

### 1) SessionManager

`SessionManager` хранит активные сессии в памяти и отвечает за:

- создание новой сессии (генерация `session_id`, инициализация каналов, запуск goroutine)
- поиск/маршрутизацию запросов по `session_id`
- удаление сессии (graceful shutdown + очистка)

Идея простая: **одна LiveSession = один независимый Telegram-клиент**.

### 2) LiveSession: “1 Run на сессию”

На каждую сессию создаётся отдельный Telegram-клиент gotd и запускается **ровно один** `cli.Run(...)` в отдельной goroutine.

Зачем:

- gotd клиент удобно держать в одном “runtime loop”
- все операции с Telegram API выполняются последовательно и в одном месте
- меньше гонок и проще управлять shutdown

### 3) Очередь задач через cmdCh (actor-подход)

После авторизации `LiveSession` крутит цикл обработки задач:

- внешние gRPC методы (например `SendMessage`) **не вызывают Telegram API напрямую**
- они упаковывают запрос в “команду” (task/command) и отправляют её в `cmdCh`
- внутри `cli.Run(...)` одна горутина читает `cmdCh` и выполняет `task.run(ctx, api)`

Плюсы:

-**потокобезопасность**: Telegram API вызывается последовательно (один consumer)

-**backpressure**: `cmdCh` буферизирован (например на 64), чтобы кратковременные пики не блокировали отправителя сразу

-**управляемый shutdown**: при закрытии можно прекращать приём новых задач и/или выполнять “финальные” команды (offline/logout) перед cancel

### 4) BroadcastHub для входящих сообщений

Для входящих сообщений используется `UpdateDispatcher` (gotd), а новые сообщения прокидываются в `broadcastHub`.

`SubscribeMessages` добавляет listener в hub и получает свой канал, из которого уже читает gRPC stream.

Плюсы:

- несколько подписчиков на одну сессию
- управление жизненным циклом подписчика через context (отписка по ctx.Done())

---

## Graceful Shutdown

Сессия завершает работу корректно:

- прекращает обработку задач (через close signal/ctx cancel)
- закрывает Telegram-клиент (контекст `cli.Run` завершается)
- закрывает `broadcastHub` и ждёт `Done()` с таймаутом

---
