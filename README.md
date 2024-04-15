# Telegram-бот "Мой фонд"
## Сборка и запуск
```sh
git clone https://github.com/HennOgyrchik/diplom.git
cd diplom
docker build -t my_fund:0.1 .
docker compose up
```
## Docker compose environments

Переменные окружения, используемые в `compose.yaml`

postgres
- POSTGRES_PASSWORD - пароль суперпользователя для PostgreSQL

ftp_server
- FTP_USER - имя пользователя (по умолчанию `foo`)
- FTP_PASSWORD - пароль (по умолчанию `bar`)

my_fund

- PSQL_HOST - адрес хоста БД (по умолчанию `localhost`)
- PSQL_PORT - порт БД (по умолчанию `5432`)
- PSQL_NAME - имя БД (по умолчанию `postgres`)
- PSQL_USER - имя пользователя БД (по умолчанию `postgres`)
- PSQL_PASSWORD - пароль БД (по умолчанию `postgres`)
- PSQL_SSLMODE - режим подключения (по умолчанию `disable`)
- PSQL_CONN_TIMEOUT - таймаут подключения (по умолчанию `5`)
- FTP_HOST - адрес хоста ftp_сервера (по умолчанию `localhost`)
- FTP_PORT - порт ftp-сервера (по умолчанию `21`)
- FTP_USER - имя пользователя
- FTP_PASSWORD - пароль
- TOKEN - токен Telegram-бота (выдается при регистрации бота в Telegram)
