[//]: # ([![codecov]&#40;https://codecov.io/gh/ThCompiler/bannersrv_test/graph/badge.svg?token=0XHCNFY6DJ&#41;]&#40;https://codecov.io/gh/ThCompiler/bannersrv_test&#41;)

# Тестовое задание на стажировку "Backend" в Avito

## Оглавление

- [Возникшие вопросы](md/Questions.md)
- [Полное описание задания](md/Task.md)
- [Нагрузочное тестирование](md/Test.md)

## Сервис баннеров

В Авито есть большое количество неоднородного контента, для которого необходимо иметь единую систему управления. В 
частности, необходимо показывать разный контент пользователям в зависимости от их принадлежности к какой-либо группе. 
Данный контент мы будем предоставлять с помощью баннеров.

## Описание задачи
Необходимо реализовать сервис, который позволяет показывать пользователям баннеры, в зависимости от требуемой фичи и 
тега пользователя, а также управлять баннерами и связанными с ними тегами и фичами.

## Общее описание решения

- Сервис реализован на языке `Golang` с использованием чистой архитектуры, разделяющей систему
  на уровни `delivery`, `usecase`, `repository` `.
- Логирование операций в файл в папку `/app-log`, настраиваемое в `config.yaml`.
- Реализованы `Middlewares` для: отслеживания паники, логирования, проверки авторизации и прав доступа,
  а также кэширования.
- Валидация реализована с помощью [vjson](https://github.com/miladibra10/vjson).
- Сервис поднимается в `Docker` контейнерах: база данных, хранилище кэша и основное приложение.
  Дополнительно поднимается prometeus для сбора метрик, grafana для визуализации метрик и nginx для удобства работы с
  prometeus и grafana.
- Контейнеры конфигурируются в  `docker-compose`.
- В качестве СУБД используется `PostgreSQL`. В качестве библиотеки для работы с запросами к `PostgreSQL` используется
  [sqlx](https://jmoiron.github.io/sqlx/), а в качестве драйвера [pgx](https://github.com/jackc/pgx).
- В качестве хранилища кэша используется `Redis`. В качестве библиотеки для работы с `Redis` используется
  [go-redis](https://github.com/redis/go-redis).
- API задокументировано с использованием Swagger по адресу `http:://localhost:8080/api/v1/swagger/`.
- Все методы имеют префикс `/api/v1`.
- Взаимодействие с проектом организовано посредством `Makefile`.
- Подключен `Github Actions` для проверки стиля, тестирования и сборки приложения.


## Пункты задания

1. [x] Используйте этот [API](https://github.com/avito-tech/backend-trainee-assignment-2024/blob/main/api.yaml).
   * Расширен API и сохранён в генерируемый swagger.yaml в папке docs.
2. [x] Тегов и фичей небольшое количество (до 1000), RPS — 1k, SLI времени ответа — 50 мс, SLI успешности ответа — 99.99%
   * Для отслеживания SLI поднята grafana, и собираются метрики успешности ответов и времени ответов в prometheus. 
      *Дополнительно можно настроить алертменеджер для оперативного реагирование на состояние сервиса*.
   * Также для улучшения производительности изменены параметры подключения к Postgresql
   (их можно настроить в конфигурационном файле сервиса).
3. [x] Для авторизации доступов должны использоваться 2 вида токенов: пользовательский и админский.  
   Получение баннера может происходить с помощью пользовательского или админского токена, а все остальные 
   действия могут выполняться только с помощью админского токена. 
   * Дополнительно для удобства тестирования реализована эмуляция сервиса токенов
4. [x] Реализуйте интеграционный или E2E-тест на сценарий получения баннера.
   * Реализован интеграционный тест включающий: поднятие окружения в контейнерах `Docker` и запуск теста с использованием
   библиотеки `apitest`.
5. [x] Если при получении баннера передан флаг use_last_revision, необходимо отдавать самую актуальную информацию.  
   В ином случае допускается передача информации, которая была актуальна 5 минут назад.
   * Реализовано кэширование запросов на метод /user_banner. Для хранения кэша используется `Redis`. При передачи
   флага use_last_revision, запрос не проверяется на наличие в кэше и передаётся на обработку дальше
6. [x] Баннеры могут быть временно выключены. Если баннер выключен, то обычные пользователи не должны его получать, 
   при этом админы должны иметь к нему доступ.

## Пункты дополнительного задания

1. [x] Адаптировать систему для значительного увеличения количества тегов и фичей, при котором допускается 
   увеличение времени исполнения по редко запрашиваемым тегам и фичам. 
   * После нагрузочного тестирования были исследованы запросы через Explain analyze, по итогу анализа были добавлены 
   индексы, повещающие производительность запросов, в крипт инициализации базы. 
   * Добиться 50 мс для запросов на get метод /banner при указании, только фичи или тэга не получилось, но получилось 
   сократить до 200 мс.
2. [ ] Провести нагрузочное тестирование полученного решения и приложить результаты тестирования к решению
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
   найденный по критерием баннер. 
   * Для отложенных задач используется библиотека [gocron](https://github.com/go-co-op/gocron).
   * Для обеспечения времени ответа на удаление баннер помечается удалённым, что закрывает к нему доступ из других методов. 
   * Для удаления помеченных баннеров создаётся задача на удаление баннера в [gocron](https://github.com/go-co-op/gocron).
   * На весь репозиторий баннеров запускается одна задача, очищаю все удалённые баннеры раз в 5 часов.
   Также при вызове метода `/filter_banner` запускается одноразовая задача, которые должна выполнится немедленно. 
   * Решено было добавить глобальную задачу раз в 5 часов, на случай если при выполнении незамедлительной задачи 
   произойдёт ошибка, система всё равно бы очистила удалённые банеры. 
5. [ ] Реализовать интеграционное или E2E-тестирование для остальных сценариев
6. [ ] Описать конфигурацию линтера

## Инструкция по запуску:

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
postgres: # Настройки подключения к PostgreSQL
   url: "host=banner-bd port=5432 user=intern password=fyr8as4da6 dbname=banner_db sslmode=disable" # Строка подключения к базе PostgreSQL
   max_connections: 10 # Максимальное число активных соединения к PostgreSQL
   max_idle_connections: 5 # Максимальное число бездействующих соединения к PostgreSQL
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

**Все поля обязательны.**

### Сборка контейнера с сервером

Теперь необходимо собрать докер образ с сервисом:

```cmd
make build-docker
```

### Запуск всей системы

Для запуска необходимо выполнить следующую команду:

```cmd
make run
```

Если необходимо запустить docker-compose не в режиме демона, то можно выполнить следующую команду:

```cmd
make run-verbose
```

Система запущена. Сервер доступен на http://localhost:8080/.

Api можно посмотреть и запускать на http://localhost:8080/api/v1/swagger.

### Остановка

Для остановки с сохранением контейнеров необходимо выполнить следующую команду:

```cmd
make stop
```

Для полной остановки необходимо выполнить следующую команду:

```cmd
make down
```

