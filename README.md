# Запуск проекта через `docker`

1.
```bash 
git clone https://github.com/railgorail/avito.git
```
2.
```bash
make start
```
3.
```bash
http://localhost:8080/health
```


## Выполненные дополнительные задания

- Эндпоинт статистики
- E2E-тестирование
- Описание конфигурации линтера

---
## Дополнительные возможности
e2e тест в отдельном окружении 
```bash
make e2e
```
запуск линтеров
``` bash
make lint
```
## Структура сервиса -> [tree](docs/tree.md)


#

_Inspired by [clean architecture](https://github.com/evrone/go-clean-template) and [tuzov](https://github.com/GolangLessons/url-shortener)_

