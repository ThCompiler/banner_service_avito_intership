[//]: # ([![codecov]&#40;https://codecov.io/gh/ThCompiler/bannersrv_test/graph/badge.svg?token=0XHCNFY6DJ&#41;]&#40;https://codecov.io/gh/ThCompiler/bannersrv_test&#41;)

# Тестовое задание на стажировку "Backend" в Avito

## Сервис баннеров
В Авито есть большое количество неоднородного контента, для которого необходимо иметь единую систему управления. В частности, необходимо показывать разный контент пользователям в зависимости от их принадлежности к какой-либо группе. Данный контент мы будем предоставлять с помощью баннеров.

## Описание задачи
Необходимо реализовать сервис, который позволяет показывать пользователям баннеры, в зависимости от требуемой фичи и тега пользователя, а также управлять баннерами и связанными с ними тегами и фичами.

Полное описание задания доступно в файле [Task.md](./Task.md).

## Вопросы:

### Первый вопрос:

**Вопрос**: Согласно условию:`Баннеры могут быть временно выключены. Если баннер выключен, то обычные пользователи не должны его получать, при этом админы должны иметь к нему доступ.`
Как разграничивается получение баннера админом и пользователем?

**Ответ**: Так понимаю, что get запрос на `/user_banner` согласно API будет соответствовать получению баннера пользователем, а 
get запрос на `/banner` будет соответствовать запросу админа. Следовательно для обработки запроса на `/user_banner` будет необходимо 
учитывать состояние баннера, а для обработки запроса на `/banner` выводить админу баннер в любом состоянии.

### Второй вопрос:

**Вопрос**: Согласно условию:`Если при получении баннера передан флаг use_last_revision, необходимо отдавать самую актуальную информацию. В ином случае допускается передача информации, которая была актуальна 5 минут назад.`
Каким образом обеспечивать передачу информации, которая была актуальна 5 минут назад?

**Ответ**: Буду использовать кэширование запросов на получение пользователем баннера, с временем жизни кэша 5 минут. 
В качестве хранилища буду использовать Redis вместо хранения внутри сервиса, например с помощью map.
Такое решение в случае масштабирования сервиса использования Redis позволит нескольким экземплярам сервиса поддерживать 
общую информацию о кэшированных запросов, т.е. избавит сервис от дублирования одних и тех же кэшированных запросов в 
каждом экземпляре сервиса.

### Третий вопрос:

**Вопрос**: Согласно условию:`Для авторизации доступов должны использоваться 2 вида токенов: пользовательский и админский. Получение баннера может происходить с помощью пользовательского или админского токена, а все остальные действия могут выполняться только с помощью админского токена. `
Кто выдаёт токены для работы?

**Ответ**: По идее токены должен выдавать ответственный за авторизацию пользователей сервис. Сервис который мне необходим реализовать
имеет другую зону ответственности и не предполагает в себе авторизацию и выдачу авторизованному пользователю токена. Поэтому
для проверки работоспособности сервиса получения баннеров имеет смысл сымитировать работу сервиса авторизации, путём создания двух
методов API, которые выдают без проверки пользовательский и админский токен. А также добавления функциональности проверки принадлежности токена.
Т.к. мы эмулируем сервис авторизации, для упрощения хранения будем создавать токен с помощью uuid и добавления приписки `user` или `admin` для пользовательского и админского токена,
соответственно. Такой подход позволит не хранить токены и быстро проверить их принадлежность. 
Сервис баннеров доступ к этой функциональности будет иметь через интерфейс, что позволит в случае необходимости, заменить реализацию интерфейса
для работы с полноценным сервисом авторизации.

### Четвёртый вопрос:

**Вопрос**: Согласно условию:`Необходимо реализовать сервис, который позволяет показывать пользователям баннеры, в зависимости от требуемой фичи и тега пользователя, а также управлять баннерами и связанными с ними тегами и фичами.`
Как учитывать пользователей при управлении тэгами и из чего состоят тэги и фичи?

**Ответ**: Т.к. создаваем сервис является частью уже существующей системы, логично предположить, что в системе уже существуют группы пользователей и фичи.
Следовательно для реализации сервиса в отдельности имеет смысл рассматривать тэги и фичи как уникальные идентификаторы и хранить их. А при интеграции
сервиса с существующей системой добавить ограничения внешнего ключа на эти идентификаторы, таким образом связав баннеры с существующими сущностями фич и тэгов.

### Пятый вопрос:

**Вопрос**: Согласно API для get метода `/banner` offset и limit могут быть не представлены. В случае если они не заданы каким образом выбирать баннеры?

**Ответ**: Следует ожидать, что баннеров может быть много, а, следовательно, вычитывание всего списка баннеров может оказать слишком долгим процессом. 
Поэтому имеет смысл установить в качестве значений по умолчанию для параметра limit, например, 100 записей. А для параметра 
offset -- 0.

### Шестой вопрос:

**Вопрос**: Согласно условию `Фича и тег однозначно определяют баннер`. Что делать, если при создании или обновлении баннера будет передан уже используемая пара тэг + фича.

**Ответ**: Для решения такого конфликта имеет смысл добавить в API для методов создания и обновления баннеров ответ с кодом 409, который будет означать
что баннер с переданной парой фича+тэг уже существует.

### Седьмой вопрос:

**Вопрос**: Согласно API для get метода `/banner` в строке запроса передаются параметры метода. Что делать если вместо числа
в численные параметры передали другой тип данных?

**Ответ**: Для решения проблем с неверным типом в запросе имеет смысл добавить в API для этого метода ответ с кодом 400, который будет означать
что в полях параметров запроса есть ошибка.

## Инструкция по запуску:

Задание выполнено на `Golang`. Данные хранятся в СУБД `PostgreSQL`.

В качестве дополнения был добавлен тип задач "Random", который с вероятностью 0,5 засчитывает пользователю задачу.
Также расширена сущность Задачи и в историю добавлено время выполнения задачи. Полную API можно посмотреть в swagger.yaml в папке docs. 
Или при запуске сервера на соответствующей странице.

### Запуск

#### Конфигурационный файл

В качестве примера конфигурационный файл находится в корне репозитория с название 'config.yaml'.
Его формат выглядит следующим образом:
```yaml
port: 8080 # Порт на котором запускается сервер
postgres:
  url: "host=banner-bd port=5432 user=intern password=fyr8as4da6 dbname=banner_db sslmode=disable" # Строка подключения к базе Postgres
logger:  # Настройки логгера
  app_name: "bannersrv"        # Имя приложения, будет выводиться в лог
  level: 'debug'              # Минимальный уровень вывода информации в лог
  directory: './app-log/'     # Папка куда сохранять логи
  use_std_and_file: true      # Если установлено в true, то лог будет выводиться как в файл так и в stdErr
  allow_show_low_level: true  # Если установлено в true и use_std_and_file тоже true, то в stdErr будет выводиться лог всех уровней
```

#### Сборка контейнера с сервером

Перед запуском необходимо собрать Docker образ:

```cmd
sudo make build-docker
```

#### Запуск всей системы

Для запуска всей системы можно выполнить команду с выводом информации в консоль:

```cmd
sudo make run-verbose
```

Или команду которая запускает docker compose в режиме daemon:

```cmd
sudo make run
```

Система запущена. Сервер доступен на http://localhost:8080/.

Api можно посмотреть и запускать на http://localhost:8080/api/v1/swagger.

