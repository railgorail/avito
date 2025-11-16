### [назад](../README.md#структура-сервиса---tree)
.
├── cmd 
│   └── migrator                          мигратор
├── docs                                  ТЗ, openapi, tree
├── internal                            
│   ├── config                            применение конфигураций
│   ├── entity                            сущности для сервисного слоя
│   ├── lib                               библиотеки/тулзы
│   │   ├── logger      
│   │   └── sl
│   ├── repo                              репозиторий слой
│   ├── server                            сервер, запуск
│   ├── service                           сервисный слой
│   ├── storage                           инициализация БД и применение миграций
│   └── transport                         транспортный слой(пока только http)
│       └── http                          
│           ├── dto                       обьекты передачи данных
│           ├── handlers                  обработка endpoint'ов
│           ├── middleware                
│           └── router
├── migrations                            миграции БД
└── tests
    └── e2e                               end-to-end тесты
