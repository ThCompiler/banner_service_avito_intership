[//]: # ([![codecov]&#40;https://codecov.io/gh/ThCompiler/bannersrv_test/graph/badge.svg?token=0XHCNFY6DJ&#41;]&#40;https://codecov.io/gh/ThCompiler/bannersrv_test&#41;)

# Тестовое задание на стажировку "Backend" в Avito

## Оглавление

- [Возникшие вопросы](md/Questions.md)
- [Полное описание задания](md/Task.md)
- [Нагрузочное тестирование](md/Test.md)
- [О мониторинге](md/Grafana.md)

## Сервис баннеров

В Авито есть большое количество неоднородного контента, для которого необходимо иметь единую систему управления. В 
частности, необходимо показывать разный контент пользователям в зависимости от их принадлежности к какой-либо группе. 
Данный контент мы будем предоставлять с помощью баннеров.

## Описание задачи
Необходимо реализовать сервис, который позволяет показывать пользователям баннеры, в зависимости от требуемой фичи и 
тега пользователя, а также управлять баннерами и связанными с ними тегами и фичами.

## Общее описание решения

- Сервис реализован на языке `Golang` версии `1.22` с использованием чистой архитектуры, разделяющей систему
  на уровни `delivery`, `usecase`, `repository` `.
- В качестве web фреймворка используется [gin](https://github.com/gin-gonic/gin).
- Логирование операций в файл в папку `/app-log`, настраиваемое в файле конфигураций.
- Реализованы `Middlewares` для: отслеживания паники, логирования, проверки авторизации и прав доступа,
  а также кэширования.
- Валидация реализована с помощью [vjson](https://github.com/miladibra10/vjson).
- Сервис поднимается в `Docker` контейнерах: база данных, хранилище кэша и основное приложение.
  Дополнительно поднимается prometeus для сбора метрик, grafana для визуализации метрик и nginx для удобства работы с
  prometeus и grafana.
- Контейнеры конфигурируются в  `docker-compose`. Для сборки сервиса используется multi-stage сборка в `Docker`.
- В качестве СУБД используется `PostgreSQL`. В качестве библиотеки для работы с запросами к `PostgreSQL` используется
  [pgxpool](https://github.com/jackc/pgx), а в качестве драйвера [pgx](https://github.com/jackc/pgx), позволяющие быстро обрабатывать запросы.
- В качестве хранилища кэша используется `Redis`. В качестве библиотеки для работы с `Redis` используется
  [go-redis](https://github.com/redis/go-redis).
- API задокументировано с использованием Swagger по адресу `http:://localhost:8080/api/v1/swagger/`.
- Все методы имеют префикс `/api/v1`.
- Взаимодействие с проектом организовано посредством `Makefile`.
- Подключен `Github Actions` для проверки стиля, тестирования и сборки приложения.


## Пункты задания

1. [x] Используйте этот [API](https://github.com/avito-tech/backend-trainee-assignment-2024/blob/main/api.yaml).
   * Расширен API и сохранён в генерируемый `swagger.yaml` в папке `docs`, 
   * Также в папке `docs` находится файл `banner.postman_collection.json` который можно открыть в Postman..
2. [x] Тегов и фичей небольшое количество (до 1000), RPS — 1k, SLI времени ответа — 50 мс, SLI успешности ответа — 99.99%
   * Для отслеживания SLI поднята grafana, и собираются метрики успешности ответов и времени ответов в prometheus. 
      *Дополнительно можно настроить алертменеджер для оперативного реагирования на состояние сервиса*.
   * Также для улучшения производительности изменены параметры подключения к Postgresql
   (их можно настроить в конфигурационном файле сервиса).
   * Результаты тестирования приведены в разделе [нагрузочное тестирование](md/Test.md).
3. [x] Для авторизации доступов должны использоваться 2 вида токенов: пользовательский и админский.
   Получение баннера может происходить с помощью пользовательского или админского токена, а все остальные 
   действия могут выполняться только с помощью админского токена. 
   * Дополнительно для удобства тестирования реализована эмуляция сервиса токенов
4. [x] Реализуйте интеграционный или E2E-тест на сценарий получения баннера.
   * Реализован интеграционный тест включающий: поднятие окружения в контейнерах `Docker` и запуск теста с использованием
   библиотеки `apitest`.
   * Тест находится в пакете `/internal/app` в файле `api_test.go`.
   * Детально о запуске интеграционных тестов написано ниже.
   * Результаты тестирования можно найти по [ссылке](https://thcompiler.github.io/banner_service_avito_intership)
5. [x] Если при получении баннера передан флаг use_last_revision, необходимо отдавать самую актуальную информацию.
   В ином случае допускается передача информации, которая была актуальна 5 минут назад.
   * Реализовано кэширование запросов на метод /user_banner. Для хранения кэша используется `Redis`. При передачи
   флага use_last_revision, запрос не проверяется на наличие в кэше и передаётся на обработку дальше
6. [x] Баннеры могут быть временно выключены. Если баннер выключен, то обычные пользователи не должны его получать, 
   при этом админы должны иметь к нему доступ.

## Пункты дополнительного задания

1. [x] Адаптировать систему для значительного увеличения количества тегов и фичей, при котором допускается 
   увеличение времени исполнения по редко запрашиваемым тегам и фичам. 
   * Было проведено нагрузочное тестирование с увеличенным числом тегов и фичей и были проанализированы запросы
   с помощью EXPLAIN ANALYSE, после чего были добавлены индексы, повещающие производительность запросов, 
   в скрипт инициализации базы, а также переработаны запросы.
2. [x] Провести нагрузочное тестирование полученного решения и приложить результаты тестирования к решению.
   * Детальная информация о проведённом тестировании в разделе [Нагрузочное тестирование](md/Test.md)
3. [x] Иногда получается так, что необходимо вернуться к одной из трех предыдущих версий баннера в связи с 
   найденной ошибкой в логике, тексте и т.д.  Измените API таким образом, чтобы можно было просмотреть существующие 
   версии баннера и выбрать подходящую версию.
   * Добавлена таблица контролирующая версии. Сохраняется только три последние версии. 
   * Дополнительно для каждой версии сохраняется дата и время её создания. 
   * Для работы с версиями в метод `/user_banner` добавлено поле `version`, при передачи
   которого будет возвращена указанная версия баннера. Если этот параметр не указан, то возвращается последняя версия.
4. [x] Добавить метод удаления баннеров по фиче или тегу, время ответа которого не должно превышать 100 мс, 
   независимо от количества баннеров.  В связи с небольшим временем ответа метода, рекомендуется ознакомиться 
   с механизмом выполнения отложенных действий.
   * Добавлен delete метод `/filter_banner`, который в параметрах запроса принимает id фичи или/и тега, и удаляет 
   найденный по критериям баннер. 
   * Для отложенных задач используется библиотека [gocron](https://github.com/go-co-op/gocron).
   * Для обеспечения времени ответа на удаление баннер помечается удалённым, что закрывает к нему доступ из других методов. 
   * Для удаления помеченных баннеров запущен отдельный сервис, поднимаемый в docker-compose, 
   который с помощью [gocron](https://github.com/go-co-op/gocron) запускает задачу на удаление раз в 5 часов.
5. [x] Реализовать интеграционное или E2E-тестирование для остальных сценариев
   * Реализованно тестирование всех методов сервиса баннеров.
6. [x] Описать конфигурацию линтера
   * В Github Actions добавлены проверки go vet и staticcheck, а также запуск golangci-lint с конфигурацией
   в файле .golangci.yml

## Инструкция по запуску:

### Исполняемый файл сервиса баннеров

Описание аргументов командной строки при работы с исполняемым файлом сервиса баннеров.

***Использование:***
```bash
server [-c=<file> | --config=<file>] [-h | --help]
````

***Опции:***
```bash
   -c --config=<file> - путь к файлу с конфигурациями (по умолчанию путь до локальной конфигурации (./configs/localhsot-config.yaml)).
   -h --help - выводит список допустимых опций и их описание.
```

### Исполняемый файл сервиса очисти удалённых баннеров

Описание аргументов командной строки при работы с исполняемым файлом сервиса баннеров.

***Использование:***
```bash
service [[-c=<file> | --config=<file>] [-p=<number> | --period=<number>]] [-h | --help]
````

***Опции:***
```bash
   -c --config=<file> - путь к файлу с конфигурациями (по умолчанию путь до локальной конфигурации (./configs/localhsot-config.yaml)).
   -p --period=<number> - период выполнения задачи по удалению баннеров в милисекундах.
   -h --help - выводит список допустимых опций и их описание.
```


### Конфигурационный файл

Все конфигурационные файлы находятся в папке `config`. Папка `env` содержит файл с переменными среды для запуска окружения для 
интеграционного теста (api_test.env) и файлы -- для запуска боевого окружения в docker.

Папка `prometheus` содержит настройки сбора метрик для инстанса `prometheus`.

Папка `services` содержит конфигурацию `nginx` и `postgreSQL`.

Файлы `docker-config.yaml` и `localhost-config.yaml` являются файлами конфигурации сервиса для запуска в боевом окружении
в Docker и для локального запуска вне Docker контейнера.

Конфигурационный файл имеет следующие поля:
```yaml
port: 8080 # Порт на котором запускается сервер
mode: release # Режим запуска системы
postgres: # Настройки подключения к PostgreSQL
   url: "host=banner-bd port=5432 user=intern password=fyr8as4da6 dbname=banner_db sslmode=disable" # Строка подключения к базе PostgreSQL
   max_connections: 10 # Максимальное число активных соединений к PostgreSQL
   min_connections: 5 # Минимальное число активных соединений к PostgreSQL
   ttl_idle_connections: 100 # Время, на протяжении которого сохраняется бездействующее соединение сверх их ограничения
redis:  # Настройки подключения к Redis
   url: "redis://chaches/0" # Строка подключения к хранилищу Redis
logger: # Настройки логгера
   app_name: "banner" # Имя приложения, будет выводиться в лог
   level: 'info'  # Минимальный уровень вывода информации в лог
   directory: './app-log/' # Папка куда сохранять логи
   use_std_and_file: false # Если установлено в true, то лог будет выводиться как в файл так и в stdErr
   allow_show_low_level: false # Если установлено в true и use_std_and_file тоже true, то в stdErr будет выводиться лог всех уровней
```

Существует четыре режима работы:
* `release` -- Запуск в режиме релиза (влияет на запуск gin в режиме Release).
* `debug` -- Запуск в режиме отладки (влияет на запуск gin в режиме Debug).
* `debug+prof` -- Запуск как в режиме `debug`, но с подключением профилирования.
* `release+prof` -- Запуск как в режиме `release`, но с подключением профилирования.

**Все поля обязательны.**

### Сборка контейнера с сервером

Теперь необходимо собрать докер образ с сервисом баннеров и сервисом очистки удалённых баннеров:

```bash
make build-docker-all
```

Отдельно собрать докер образ с сервисом баннеров можно с помощью команды:

```bash
make build-docker-banner
```

Отдельно собрать докер образ с сервисом очистки удалённых баннеров можно с помощью команды:

```bash
make build-docker-cron
```

### Образы

После запуска команды на сборку докер образов появятся два образа `banner` и `cron` содержащие сервис баннеров и сервис 
очистки удалённых баннеров, соответственно.

Образы `banner` и `cron` поддерживает env переменную `CONFIG_PATH`, которая позволяет установить путь до 
конфигурационного файла в аргумент `--config` запускаемого сервиса.

Образ `cron` дополнительно поддерживает env переменную `TASK_PERIOD`, которая позволяет установить период выполнения
задачи по удалению баннера в аргумент  `--period` запускаемого сервиса.

### Запуск всей системы

Для запуска необходимо выполнить следующую команду:

```bash
make run
```

Если необходимо запустить docker-compose не в режиме демона, то можно выполнить следующую команду:

```bash
make run-verbose
```

Система запущена. Сервер доступен на http://localhost:8080/.

Api можно посмотреть и запускать на http://localhost:8080/api/v1/swagger/index.html.

### Остановка

Для остановки с сохранением контейнеров необходимо выполнить следующую команду:

```bash
make stop
```

Для полной остановки необходимо выполнить следующую команду:

```bash
make down
```

## Инструкция по интеграционных тестов:

### Конфигурационные файлы:

В папке `config` находится файл `api-test-config.yaml` содержащий конфигурацию для запуска сервиса очистки удалённых баннеров
в тестовом окружение. 

В папке `api_test.env` настраивает конфигурацию базы данных PostgreSQL в тестовом окружение, а также
строки подключения тестов к тестовому окружению:


- `POSTGRES_PASSWORD=fyr8as4da6` -- Пароль для базы данных PostgreSQL в тестовом окружении
- `POSTGRES_USER=intern`  -- Пользователь для базы данных PostgreSQL в тестовом окружении
- `POSTGRES_DB=banner_db` -- Название базы данных PostgreSQL в тестовом окружении
- `PG_STRING=host=localhost port=5432 user=intern password=fyr8as4da6 dbname=banner_db sslmode=disable` -- 
   Строка подключения тестов к тестовому окружению PostgreSQL
- `REDIS_STRING=redis://localhost:6379/0`  -- Строка подключения тестов к тестовому окружению Redis

### Запуск тестов

Для запуска тестов сначала необходимо запустить тестовое окружение:
```bash
make run-environment
```

Если на системе не собран образ сервиса очистки удалённых баннеров, можно запустить окружение с его сборкой:
```bash
make run-environment-with-build
```

После запуска окружения запускаются тесты командой:
```bash
make run-api-test
```

После тестирования обязательно нужно остановить окружение:
```bash
make down-environment
```

### Дополнительно

Дополнительно можно запустить просмотр отчёта, который будет сгенерирован в папке `internal/app/allure-results` с
помощью утилиты [allure](https://allurereport.org/docs/gettingstarted-installation/).

```bash
allure serve ./internal/app/allure-results
```
