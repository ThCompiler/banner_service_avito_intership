// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {
            "name": "Артём Ветошкин",
            "email": "vet_v2002@mail.ru"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/banner": {
            "get": {
                "security": [
                    {
                        "AdminToken": []
                    }
                ],
                "description": "Возвращает список баннеров на основе фильтра по фиче и/или тегу.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "banner"
                ],
                "summary": "Получение всех баннеров c фильтрацией по фиче и/или тегу",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "Идентификатор тэга группы пользователей",
                        "name": "tag_id",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "Идентификатор фичи",
                        "name": "feature_id",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "Лимит",
                        "name": "limit",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "Оффсет",
                        "name": "offset",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Список баннеров успешно отфильтрован",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/response.Banner"
                            }
                        }
                    },
                    "400": {
                        "description": "Некорректные данные",
                        "schema": {
                            "$ref": "#/definitions/tools.Error"
                        }
                    },
                    "401": {
                        "description": "Пользователь не авторизован"
                    },
                    "403": {
                        "description": "Пользователь не имеет доступа"
                    },
                    "500": {
                        "description": "Внутренняя ошибка сервера",
                        "schema": {
                            "$ref": "#/definitions/tools.Error"
                        }
                    }
                }
            },
            "post": {
                "security": [
                    {
                        "AdminToken": []
                    }
                ],
                "description": "Добавляет баннер включая его содержания, id фичи, список id тэгов и состояние.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "banner"
                ],
                "summary": "Создание нового баннера.",
                "parameters": [
                    {
                        "description": "Информация о добавляемом пользователе",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/request.CreateBanner"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Баннер успешно добавлен в систему",
                        "schema": {
                            "$ref": "#/definitions/response.BannerID"
                        }
                    },
                    "400": {
                        "description": "Некорректные данные",
                        "schema": {
                            "$ref": "#/definitions/tools.Error"
                        }
                    },
                    "401": {
                        "description": "Пользователь не авторизован"
                    },
                    "403": {
                        "description": "Пользователь не имеет доступа"
                    },
                    "409": {
                        "description": "Баннер с указанной парой id фичи и ia тэга уже существует"
                    },
                    "500": {
                        "description": "Внутренняя ошибка сервера",
                        "schema": {
                            "$ref": "#/definitions/tools.Error"
                        }
                    }
                }
            }
        },
        "/banner/{id}": {
            "delete": {
                "security": [
                    {
                        "AdminToken": []
                    }
                ],
                "description": "Удаляет информацию о банере по его id.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "banner"
                ],
                "summary": "Удаление банера.",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "Идентификатор баннера",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "204": {
                        "description": "Баннер успешно удалён"
                    },
                    "400": {
                        "description": "Некорректные данные",
                        "schema": {
                            "$ref": "#/definitions/tools.Error"
                        }
                    },
                    "401": {
                        "description": "Пользователь не авторизован"
                    },
                    "403": {
                        "description": "Пользователь не имеет доступа"
                    },
                    "404": {
                        "description": "Баннер с данным id не найден"
                    },
                    "500": {
                        "description": "Внутренняя ошибка сервера",
                        "schema": {
                            "$ref": "#/definitions/tools.Error"
                        }
                    }
                }
            },
            "patch": {
                "security": [
                    {
                        "AdminToken": []
                    }
                ],
                "description": "Обновляет информацию о баннере по его id.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "banner"
                ],
                "summary": "Обновление баннера.",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "Идентификатор баннера",
                        "name": "id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": "Информация об обновлении",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/request.UpdateBanner"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Баннер успешно обновлён"
                    },
                    "400": {
                        "description": "Некорректные данные",
                        "schema": {
                            "$ref": "#/definitions/tools.Error"
                        }
                    },
                    "401": {
                        "description": "Пользователь не авторизован"
                    },
                    "403": {
                        "description": "Пользователь не имеет доступа"
                    },
                    "404": {
                        "description": "Баннер с данным id не найден"
                    },
                    "409": {
                        "description": "Баннер с указанной парой id фичи и ia тэга уже существует"
                    },
                    "500": {
                        "description": "Внутренняя ошибка сервера",
                        "schema": {
                            "$ref": "#/definitions/tools.Error"
                        }
                    }
                }
            }
        },
        "/filter_banner": {
            "delete": {
                "security": [
                    {
                        "AdminToken": []
                    }
                ],
                "description": "Удаляет баннеры на основе фильтра по фиче или тегу. Обязателен один из query параметров.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "banner"
                ],
                "summary": "Удаление всех баннеров c фильтрацией по фиче или тегу",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "Идентификатор тэга группы пользователей",
                        "name": "tag_id",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "Идентификатор фичи",
                        "name": "feature_id",
                        "in": "query"
                    }
                ],
                "responses": {
                    "204": {
                        "description": "Баннеры успешно удалены"
                    },
                    "400": {
                        "description": "Некорректные данные",
                        "schema": {
                            "$ref": "#/definitions/tools.Error"
                        }
                    },
                    "401": {
                        "description": "Пользователь не авторизован"
                    },
                    "403": {
                        "description": "Пользователь не имеет доступа"
                    },
                    "404": {
                        "description": "Баннер с указанными тэгом и фичёй не найден"
                    },
                    "500": {
                        "description": "Внутренняя ошибка сервера",
                        "schema": {
                            "$ref": "#/definitions/tools.Error"
                        }
                    }
                }
            }
        },
        "/token/admin": {
            "get": {
                "description": "Возвращает токен с правами админа.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "auth"
                ],
                "summary": "Получение токена админа.",
                "responses": {
                    "200": {
                        "description": "Токен успешно создан",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/token/user": {
            "get": {
                "description": "Возвращает токен с правами пользователя.",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "auth"
                ],
                "summary": "Получение токена пользователя.",
                "responses": {
                    "200": {
                        "description": "Токен успешно создан",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/user_banner": {
            "get": {
                "security": [
                    {
                        "UserToken": []
                    }
                ],
                "description": "|",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "banner"
                ],
                "summary": "Получение баннера для пользователя.",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "Идентификатор тэга группы пользователей",
                        "name": "tag_id",
                        "in": "query",
                        "required": true
                    },
                    {
                        "type": "integer",
                        "description": "Идентификатор фичи",
                        "name": "feature_id",
                        "in": "query",
                        "required": true
                    },
                    {
                        "type": "integer",
                        "description": "Версия баннера",
                        "name": "version",
                        "in": "query"
                    },
                    {
                        "type": "boolean",
                        "description": "Получать актуальную информацию",
                        "name": "use_last_revision",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "JSON-отображение баннера",
                        "schema": {
                            "type": "object"
                        }
                    },
                    "400": {
                        "description": "Некорректные данные",
                        "schema": {
                            "$ref": "#/definitions/tools.Error"
                        }
                    },
                    "401": {
                        "description": "Пользователь не авторизован"
                    },
                    "403": {
                        "description": "Пользователь не имеет доступа"
                    },
                    "404": {
                        "description": "Баннер с указанными тэгом и фичёй не найден"
                    },
                    "500": {
                        "description": "Внутренняя ошибка сервера",
                        "schema": {
                            "$ref": "#/definitions/tools.Error"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "request.CreateBanner": {
            "type": "object",
            "properties": {
                "content": {
                    "description": "Содержимое баннера",
                    "type": "object"
                },
                "feature_id": {
                    "description": "Идентификатор фичи",
                    "type": "integer",
                    "format": "uint64"
                },
                "is_active": {
                    "description": "Флаг активности баннера",
                    "type": "boolean"
                },
                "tag_ids": {
                    "description": "Идентификаторы тегов",
                    "type": "array",
                    "items": {
                        "type": "integer"
                    }
                }
            }
        },
        "request.UpdateBanner": {
            "type": "object",
            "properties": {
                "content": {
                    "description": "Содержимое баннера",
                    "type": "object"
                },
                "feature_id": {
                    "description": "Идентификатор фичи",
                    "type": "integer",
                    "format": "uint64"
                },
                "is_active": {
                    "description": "Флаг активности баннера",
                    "type": "boolean"
                },
                "tag_ids": {
                    "description": "Идентификаторы тегов",
                    "type": "array",
                    "items": {
                        "type": "integer"
                    }
                }
            }
        },
        "response.Banner": {
            "type": "object",
            "properties": {
                "banner_id": {
                    "description": "Идентификатор баннера",
                    "type": "integer",
                    "format": "uint64"
                },
                "created_at": {
                    "description": "Дата создания баннера",
                    "type": "string",
                    "format": "date-time"
                },
                "feature_id": {
                    "description": "Идентификатор фичи",
                    "type": "integer"
                },
                "is_active": {
                    "description": "Флаг активности баннера",
                    "type": "boolean",
                    "format": "uint64"
                },
                "tag_ids": {
                    "description": "Идентификаторы тэгов",
                    "type": "array",
                    "items": {
                        "type": "integer"
                    }
                },
                "updated_at": {
                    "description": "Дата обновления баннера",
                    "type": "string",
                    "format": "date-time"
                },
                "versions": {
                    "description": "Последние три версии баннера",
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/response.Content"
                    }
                }
            }
        },
        "response.BannerID": {
            "type": "object",
            "properties": {
                "banner_id": {
                    "description": "Идентификатор созданного баннера",
                    "type": "integer",
                    "format": "uint64"
                }
            }
        },
        "response.Content": {
            "type": "object",
            "properties": {
                "content": {
                    "description": "Содержимое баннера",
                    "type": "object"
                },
                "created_at": {
                    "description": "Дата создания версии",
                    "type": "string",
                    "format": "date-time"
                },
                "version": {
                    "description": "Версия содержимого баннера",
                    "type": "integer",
                    "format": "uint32"
                }
            }
        },
        "tools.Error": {
            "type": "object",
            "properties": {
                "error": {
                    "type": "string"
                }
            }
        }
    },
    "securityDefinitions": {
        "AdminToken": {
            "description": "Токен доступа админа",
            "type": "apiKey",
            "name": "token",
            "in": "header"
        },
        "UserToken": {
            "description": "Токен доступа пользователя",
            "type": "apiKey",
            "name": "token",
            "in": "header"
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8080",
	BasePath:         "/api/v1",
	Schemes:          []string{"http"},
	Title:            "Сервис баннеров",
	Description:      "Rest API Для управления для сервиса баннеров",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
