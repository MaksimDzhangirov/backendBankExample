# Как написать и запустить миграцию базы данных в Golang

[Оригинал](https://dev.to/techschoolguru/how-to-write-run-database-migration-in-golang-5h6g)

При работе с базой данных миграция схемы является одной из важных задач, которую 
нам часто приходится выполнять на протяжении всего жизненного цикла приложения, 
чтобы адаптироваться к новым бизнес-требованиям.

В этой лекции мы узнаем, как написать и запустить миграцию схемы базы данных в 
Golang, используя библиотеку [golang-migrate](https://github.com/golang-migrate/migrate).

* Ссылка на [плейлист с видео лекциями на Youtube](https://bit.ly/backendmaster)
* И на [Github репозиторий](https://github.com/techschool/simplebank)

## Устанавливаем golang-migrate

Golang-migrate работает со многими различными движками баз данных, такими как 
Postgres, Mysql, Mongo, Cockroach и т. д.

Давайте откроем [эту CLI документацию](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate), чтобы узнать, как его установить. У меня 
Mac, поэтому я буду использовать Homebrew.

```shell
brew install golang-migrate
```

Golang-migrate позволяет нам использовать несколько команд:

* Первая — `create`, которую мы можем использовать для создания новых файлов 
  миграции.
* Вторая — это `goto`, которая осуществит миграцию на определенную версию.
* Команды `up` или `down` осуществляют все или N миграций вперёд или назад.

Существуют ещё несколько команд, но чаще всего мы будем работать с `create`, `up` 
и `down`.

После того как golang-migrate успешно установлен, мы можем запустить команду
`migrate -help`, чтобы прочитать его справочное руководство.

![](../images/part3/zlc7u7javz1fcvja5a4j.jpeg)

## Создаём новую миграцию

Итак, теперь я создам новую папку для нашего проекта Simple Bank, а внутри 
неё - новую папку `db/migration` для хранения всех наших файлов миграции.

```shell
cd ~/Projects/techschool
mkdir simple_bank
cd simple_bank
mkdir -p db/migration
```

Затем давайте создадим первый файл миграции, чтобы инициализировать схему базы 
данных нашего приложения Simple Bank.

Начните ввод в терминале с `migrate create`. Затем укажите расширение файла `sql`
и каталог для его хранения — `db/migration`.

```shell
migrate create -ext sql -dir db/migration -seq init_schema
```

Мы используем флаг `-seq` для создания последовательного номера версии файла 
миграции. И, наконец, укажите название миграции, которое в данном случае равно 
`init_schema`.

![](../images/part3/k6q2ptxpxdwtk4oyewof.jpeg)

Как видно на рисунке, для нас было сгенерировано 2 файла миграции. Оба они имеют 
версию 1 в префиксе имени файла, но их суффиксы разные: у одного файла `up`, а 
другой — `down`. Почему?

## Миграции up/down

Причина в том, что это лучшая практика при написании миграции базы данных. Скрипт
`up` запускается для применения изменений к схеме БД, а скрипт `down` запускается, 
если мы хотим отменить изменения, сделанные скриптом `up`.

![](../images/part3/7a1ail02bmla1uf51j4s.png)

Поэтому когда мы запускаем команду `migrate up`, файлы скрипта `up` внутри папки 
`db/migration` будут запускаться последовательно по возрастанию их версии префикса.

![](../images/part3/si2z26vqmo62miregkk6.png)

Наоборот, когда мы запускаем команду `migrate down`, файлы скрипта `down` внутри 
папки db/migration будут запускаться последовательно в порядке, обратном их 
версии префикса.

![](../images/part3/9v5vsgabledu35oc3rg7.png)

Хорошо, теперь давайте откроем файл `simple_bank.sql`, который мы создали в 
предыдущих лекциях. Я скопирую все содержимое этого файла и вставлю его в файл 
`init_schema.up.sql`.

```postgresql
CREATE TABLE "accounts" (
  "id" bigserial PRIMARY KEY,
  "owner" varchar NOT NULL,
  "balance" bigint NOT NULL,
  "currency" varchar NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "entries" (
  "id" bigserial PRIMARY KEY,
  "account_id" bigint NOT NULL,
  "amount" bigint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "transfers" (
  "id" bigserial PRIMARY KEY,
  "from_account_id" bigint NOT NULL,
  "to_account_id" bigint NOT NULL,
  "amount" bigint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

ALTER TABLE "entries" ADD FOREIGN KEY ("account_id") REFERENCES "accounts" ("id");

ALTER TABLE "transfers" ADD FOREIGN KEY ("from_account_id") REFERENCES "accounts" ("id");

ALTER TABLE "transfers" ADD FOREIGN KEY ("to_account_id") REFERENCES "accounts" ("id");

CREATE INDEX ON "accounts" ("owner");

CREATE INDEX ON "entries" ("account_id");

CREATE INDEX ON "transfers" ("from_account_id");

CREATE INDEX ON "transfers" ("to_account_id");

CREATE INDEX ON "transfers" ("from_account_id", "to_account_id");

COMMENT ON COLUMN "entries"."amount" IS 'can be negative or positive';

COMMENT ON COLUMN "transfers"."amount" IS 'must be positive';
```

Для файла `init_schema.down.sql` мы должны отменить изменения, внесенные 
сценарием `up`. В нашем случае скрипт `up` создает 3 таблицы: `accounts`, `transfers` 
и `entries`. Таким образом, скрипт `down` должен удалить их все. Для этого мы 
воспользуемся командой `DROP TABLE`.

```postgresql
DROP TABLE IF EXISTS entries;
DROP TABLE IF EXISTS transfers;
DROP TABLE IF EXISTS accounts;
```

Здесь мы сначала удаляем таблицы `entries` и `transfers`, а потом `accounts`, из-за
наличия в них внешних ключей, которые ссылаются на записи в таблице `accounts`.

Итак, наши скрипты для миграций готовы. Попробуем их запустить.

## Проверяем статус postgres контейнера

Но перед этим мы должны проверить, работает ли еще наш контейнер `postgres`:

```shell
docker ps
```

Кстати, я покажу вам еще несколько docker команд для работы с контейнерами. Если 
мы хотим остановить запущенный контейнер, мы используем `docker stop` с именем 
или идентификатором контейнера.

```shell
docker stop postgres12
```

После этого, если мы запустим `docker ps`, мы больше не увидим контейнер postgres
в списке, потому что он не запущен. Чтобы вывести список всех контейнеров, 
независимо от их текущего состояния, выполните команду:

```shell
docker ps -a
```

Теперь мы видим наш контейнер `postgres` со статусом `exited`.

![](../images/part3/x9us9i1722tgfv1lhk3b.jpeg)

Чтобы снова запустить его, нам просто нужно выполнить команду: `docker start` и 
передать ей имя или идентификатор контейнера.

```shell
docker start postgres12
```

После этого контейнер `postgres12` снова запущен и готов к работе.

## Доступ к командной оболочке контейнера postgres

Мы можем получить доступ к его командной оболочке с помощью команды `docker exec`.
Поскольку мы используем образ postgres alpine, у нас нет оболочки `/bin/bash`, 
как в Ubuntu, поэтому вместо нее мы используем оболочку `/bin/sh`:

```shell
docker exec -it postgres12 /bin/sh
```

Внутри оболочки у нас есть доступ ко всем стандартным командам Linux.

![shell](../images/part3/uampgev7odq2t67fw9me.jpeg)

И поскольку это контейнер postgres, он также предоставляет нам некоторые CLI 
команды для взаимодействия с сервером postgres непосредственно из оболочки.

## Создаём/удаляем базу данных внутри контейнера postgres

Мы должны создать базу данных, прежде чем мы сможем запустить первую миграцию.

Итак, давайте выполним команду `createdb` внутри оболочки контейнера postgres, 
чтобы создать новую базу данных для нашего приложения Simple Bank:

```shell
createdb --username=root --owner=root simple_bank
```

* Мы используем параметр `--username`, чтобы указать, что мы подключаемся как 
  пользователь root.
* И параметр `--owner`, чтобы указать, что база данных, которую мы собираемся 
  создать, также будет принадлежать пользователю `root`.
* Последний аргумент — это название базы данных: `simple_bank`.

Отлично, база данных создана. Мы можем получить доступ к ней через консоль с 
помощью команды `psql`.

![database_in_console](../images/part3/j32zemncvf89zsgmqscs.jpeg)

Мы также можем удалить базу данных с помощью команды `dropdb` и передав 
название базы данных.

```shell
dropdb simple_bank
```

Мы используем команду `exit`, чтобы выйти из командной оболочки контейнера.

## Создаём/удаляем базу данных вне контейнера postgres

Извне контейнера мы также можем запустить `createdb` напрямую с помощью команды 
`docker exec`.

```shell
docker exec -it postgres12 createdb --username=root --owner=root simple_bank
```

И получить доступ к консоли базы данных, минуя командную оболочку контейнера.

```shell
docker exec -it postgres12 psql -U root simple_bank
```

## Создаём Makefile

Итак, теперь я создам Makefile в нашем проекте, затем добавлю команду `createdb` 
для создания базы данных Simple Bank и команду `dropdb` для ее удаления.

При работе в команде эти команды будут полезны вашим коллегам, чтобы им было проще 
настроить проект на своей локальной машине для разработки.

Давайте также добавим в Makefile команду, которую мы использовали для запуска 
контейнера postgres в предыдущей лекции.

```makefile
postgres:
    docker run --name postgres12 -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:12-alpine

createdb:
    docker exec -it postgres12 createdb --username=root --owner=root simple_bank

dropdb:
    docker exec -it postgres12 dropdb simple_bank

.PHONY: postgres createdb dropdb
```

Хорошо, теперь давайте остановим текущий контейнер postgres.

```shell
docker stop postgres12
```

Контейнер остановлен. Я полностью удалю его с помощью команды `docker rm`.

```shell
docker rm postgres12
```

Теперь когда мы выполним команду

```shell
make postgres
```

Стартует новый контейнер postgres. И мы можем запустить

```shell
make createdb
```

чтобы создать базу данных `simple_bank`.

## Просматриваем базу данных с помощью TablePlus

ОК, база данных создана. Давайте подключимся к нему с помощью [TablePlus](https://tableplus.com/).

Соединение, которое мы создали в предыдущей лекции, подключит нас к базе данных 
root. Мы можем щелкнуть на значок базы данных, чтобы открыть нашу новую базу 
данных `simple_bank` (смотри рисунок).

![](../images/part3/ts68d13wwjxggrt83rjc.png)

Хорошо, теперь вы видите здесь 2 базы данных: `root` и `simple_bank`. На данный 
момент база данных `simple_bank` пуста. Поэтому вернемся в терминал и запустим 
первую миграцию.

## Запускаем миграцию

Начнём написание команды со слова `migrate`. Затем воспользуемся параметром
`-path`, чтобы указать папку, содержащую наши файлы миграции, то есть `db/migration`.

```shell
migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank" -verbose up
```

Параметр `-database` используется для указания URL-адреса сервера базы данных.

* Мы используем postgres, поэтому название драйвера — `postgresql`.
* Имя пользователя - `root`.
* Пароль - `secret`.
* Адрес `localhost`, порт `5432`.
* И имя базы данных — `simple_bank`.

Мы используем параметр `-verbose`, чтобы команда `migrate` выводила на экран все
служебные сообщения.

И, наконец, мы используем аргумент `up`, чтобы указать команде `migrate`, что нужно 
запустить команду `migrate up`.

![](../images/part3/pvuoxe9s61pndkpdg67s.jpeg)

Ой, мы получили ошибку: SSL is not enabled on the server (на сервере не включен SSL). 
Она возникла потому, что наш контейнер postgres не включает SSL по умолчанию.

Поэтому мы должны добавить параметр sslmode=disable к URL-адресу базы данных. 
Теперь запустите такую команду:

```shell
migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up
```

И миграция успешно прошла!

Если мы обновим базу данных `simple_bank` в TablePlus, то теперь увидим 4 таблицы:
`accounts`, `entries`, `transfers` и `schema_migrations`.

![](../images/part3/o9ws42vgqlpsxj9vf0lu.png)

В таблице `schema_migrations` хранится последняя примененная версия миграции, в 
нашем случае это версия 1, поскольку мы запустили только один файл миграции.

Столбец `dirty` сообщает нам, была ли последняя миграция удачной или нет. Если
миграция не прошла, мы должны вручную исправить проблемы, чтобы восстановить корректное 
состояние базы данных, прежде чем пытаться запускать любые другие миграции.

## Добавляем migrate up/down в Makefile

Итак, теперь я добавлю команды `migrate up` и `migrate down` в Makefile:

```makefile
postgres:
    docker run --name postgres12 -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:12-alpine

createdb:
    docker exec -it postgres12 createdb --username=root --owner=root simple_bank

dropdb:
    docker exec -it postgres12 dropdb simple_bank

migrateup:
    migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up

migratedown:
    migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose down

.PHONY: postgres createdb dropdb migrateup migratedown
```

Теперь давайте попробуем выполнить их в терминале! Сначала я запущу:

```shell
make migratedown
```

Вернитесь в TablePlus и обновите страницу.

![](../images/part3/ki2n6fl83bjim2ppdi46.png)

Все таблицы исчезли, кроме `schema_migrations`.

Хорошо, теперь запустим:

```shell
make migrateup
```

Затем обновите TablePlus.

![](../images/part3/6hansa2xyfp9is4wjgcp.png)

Все таблицы опять на месте. Прекрасно!

На этом завершим сегодняшнюю лекцию о миграции баз данных. Спасибо за время, 
потраченное на чтение, и до встречи на следующей лекции! 