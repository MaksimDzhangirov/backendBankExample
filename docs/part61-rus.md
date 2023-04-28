# Подтверждение адреса электронной почты в Go: создаём таблицы в БД и отправляем соответствующее письмо

[Оригинал](https://www.youtube.com/watch?v=lEHkwDPHrcc)

Всем привет. Добро пожаловать на мастер-класс по бэкенду! На предыдущей 
лекции мы узнали, как отправлять электронные письма с помощью Gmail. Однако
мы ещё не интегрировали этот функционал в наш асинхронный воркер. Напомню,
что мы пытаемся сделать: отправить приветственное письмо новому 
пользователю и попросить его подтвердить свой адрес электронной почты.

![отправляем приветственное письмо новым пользователям](../images/part61/1.png)

И я покажу как это сделать в этой лекции.

Идея проста: для каждого нового пользователя мы будем генерировать 
случайный секретный код и хранить его в таблице БД. Этот код также будет
отправлен на зарегистрированный адрес электронной почты пользователя 
вместе со ссылкой на страницу на нашем веб-сайте для подтверждения.

![сохраняем случайный секретный код в БД](../images/part61/2.png)

Если пользователь ввёл адрес электронной почты, к которому у него доступ,
то он откроет страницу с правильным секретным кодом, и, таким образом, его 
адрес электронной почты будет подтверждён.

![адрес электронной почты подтверждён](../images/part61/3.png)

Хорошо, давайте начнём!

## Обновляем структуру базы данных

Во-первых, нам нужно обновить структуру базы данных. Поэтому я открою 
файл `db.dbml` внутри папки `doc`. В определение таблицы «users» добавим 
новое логическое поле с названием "is_email_verified". Оно будет указывать 
нам был ли подтвержден адрес электронное почты или нет. Это поле не должно
быть равно `null`, а его значением по умолчанию должно быть `false`.

```shell
email varchar [unique, not null]
is_email_verified bool [not null, default: false]
password_changed_at timestamptz [not null, default: '0001-01-01']
```

Наличие значения по умолчанию очень важно для нового столбца, потому что, 
если у вас уже существуют какие-то записи в базе данных, новый столбец для
них должны быть заполнен, чтобы удовлетворить ограничению на ненулевое 
значение.

Затем я добавлю новую таблицу под названием "verify_emails". В этой 
таблице будет храниться секретный код и вся необходимая информация об 
электронном письме, которое мы отправляем пользователю. Она будет состоять 
из столбца `id` типа `bigserial`, который будет первичным ключом этой таблицы.
Столбец `username` типа `varchar`, будет являться внешним ключом, который 
ссылается на соответствующий столбец в таблице "users". И он также должен 
быть не `null`. Также тут существует столбец для отслеживания адреса 
электронной почты пользователя. Хотя он может показаться избыточным, 
поскольку адрес электронной почты уже хранится в таблице "users", на самом
деле это не так, потому что мы можем разрешить пользователям изменять 
свой адрес электронной почты в будущем. Нам не нужно ограничение на 
уникальность в этой таблице, потому что мы можем отправить несколько 
писем с подтверждением на один и тот же адрес для разных целей. Хорошо, 
далее следует самое важное поле: `secret_code`. Его типом должен быть 
`varchar`, и оно не должен быть `null`. Нам также понадобится логическое 
поле, чтобы знать, использовался ли этот секретный код или нет. По 
соображениям безопасности мы не хотим, чтобы секретный код можно было 
использовать многократно. Это поле не должно быть равно `null` и его 
значение по умолчанию равно `false`. Затем я добавлю в эту таблицу 
поле `created_at`, чтобы отслеживать время создания этой записи в таблице.
И, наконец, поле `expired_at`, чтобы знать, когда срок её действия истек.
Это также сделано из соображений безопасности, потому что мы не хотим, 
чтобы секретный код был валиден в течении длительного временного интервала,
что может увеличить вероятность его утечки. Поэтому я установлю значение по
умолчанию для этого поля равным `now() + interval '15 minutes'`. Это 
гарантирует, что код будет валиден только в течение 15 минут после создания.

```
Table verify_emails {
  id bigserial [pk]
  username varchar [ref: U.username, not null]
  email varchar [not null]
  secret_code varchar [not null]
  is_used bool [not null, default: false]
  created_at timestamptz [not null, default: `now()`]
  expired_at timestamptz [not null, default: `now() + interval '15 minutes'`]
}
```

Отлично, на мой взгляд такой структуры БД должно быть достаточно для 
реализации необходимого нам функционала.

Давайте откроем терминал и запустим команду

```shell
make db_schema
dbml2sql --postgres -o doc/schema.sql doc/db.dbml
  ERROR: You have a syntax error at "db.dbml" line 20 column 26. Expected " " or <>, >, - or < but "U" found.

  A complete log can be found in:
     /Users/quangpham/Projects/techschool/simplebank/dbml-error.log
```

чтобы сгенерировать SQL код для новой структуры. Ой, мы допустили 
синтаксическую ошибку в строке 20, давайте вернемся к коду, чтобы исправить 
её. Да, я понял в чём дело! Здесь, когда я определял ссылку для внешнего
ключа я забыл добавить символ `>`.

```
username varchar [ref: > U.username, not null]
```

Теперь ошибок быть не должно. Давайте вернёмся в окно терминала и запустим
команду

```shell
make db_schema
dbml2sql --postgres -o doc/schema.sql doc/db.dbml
  ✔ Generated SQL dump file (PostgreSQL): schema.sql
```

ещё раз.

На этот раз SQL файл был успешно создан. А в VS Code мы увидим, что в 
файле `schema.sql` появились новые изменения.


```postgresql
CREATE TABLE "users" (
  "username" varchar PRIMARY KEY,
  "hashed_password" varchar NOT NULL,
  "full_name" varchar NOT NULL,
  "email" varchar UNIQUE NOT NULL,
  "is_email_verified" bool NOT NULL DEFAULT false,
  "password_changed_at" timestamptz NOT NULL DEFAULT '0001-01-01',
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "verify_emails" (
  "id" bigserial PRIMARY KEY,
  "username" varchar NOT NULL,
  "email" varchar NOT NULL,
  "secret_code" varchar NOT NULL,
  "is_used" bool NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "expired_at" timestamptz NOT NULL DEFAULT (now() + interval '15 minutes')
);
```

В таблицу "users" был добавлен новый столбец, а также появилась новая 
команда для создания таблицы "verify_emails". В нижней части файла также 
находится команда "alter table", добавляющая ограничение для внешнего ключа 
в эту новую таблицу.

```postgresql
ALTER TABLE "verify_emails" ADD FOREIGN KEY ("username") REFERENCES "users" ("username");
```

Сейчас, когда наш проект уже развернут на продакшене, мы не может просто 
удалить всю БД и воссоздать её с новой структурой. Поэтому следующим нашим
действием будет написание скрипта миграции БД, который обновит структуру,
учитывая внесённые изменения.

В конце файла `README` находится команда, которая поможет нам создать 
новый файл миграции БД.

```shell
migrate create -ext sql -dir db/migration -seq <migration_name>
```

Поскольку эта команда будет часто использоваться при разработке, я добавлю 
её в `Makefile`, чтобы было проще её запускать.

Давайте добавим новую команду с названием "new_migration" и вставим в неё 
команду «migrate create», которую мы только что скопировали.

```makefile
...
migratedown1:
	migrate -path db/migration -database "$(DB_URL)" -verbose down 1

new_migration:
	migrate create -ext sql -dir db/migration -seq <migration_name>
	
...
```

Обратите внимание, что нам нужно заменить здесь `<migration_name>` на 
аргумент командной строки. Таким образом, мы сможем задавать различные
названия для миграций при запуске из терминала. Это можно легко сделать, 
используя символ `$`, за которым следует имя аргумента в круглых скобках.

```makefile
new_migration:
	migrate create -ext sql -dir db/migration -seq $(name)
```

Здесь я использую "name" в качестве аргумента.

Затем давайте добавим команду `new_migration` в список PHONY.

```makefile
.PHONY: postgres createdb dropdb migrateup migratedown migrateup1 migratedown1 new_migration db_docs db_schema sqlc test server mock proto evans redis
```

Сохраните файл и вернитесь в терминал. Теперь мы можем запустить команду 
"make new_migration" и задать для аргумента "name" значение 
"add_verify_emails".

```shell
make new_migration name=add_verify_emails
migrate create -ext sql -dir db/migration -seq add_verify_emails
/Users/quangpham/Projects/techschool/simplebank/db/migration/000004_add_verify_emails.up.sql
/Users/quangpham/Projects/techschool/simplebank/db/migration/000004_add_verify_emails.down.sql
```

Как видите, было создано 2 новых файла миграции (`up` и `down`). Оба они 
имеют префикс «add_verify_emails» в своём названии. Эти новые файлы находятся
в папке "db/migration".

Теперь нам нужно заполнить их SQL кодами. Во-первых, миграцию `up`. Для 
новой таблицы "verify_emails" это довольно просто, потому что нам просто 
нужно скопировать сгенерированный код из файла `schema.sql`. Мы также 
должны скопировать оператор, который добавляет ограничения для внешнего ключа 
в эту новую таблицу. Для нового столбца "is_email_verified" мы также можем
скопировать его код из `schema.sql`,

```postgresql
"is_email_verified" bool NOT NULL DEFAULT false,
```

но мы должны вручную написать первую часть команды, а именно: 
`ALTER TABLE "users" ADD COLUMN `. Затем вставьте в конец скопированную 
часть определения столбца.

```postgresql
ALTER TABLE "users" ADD COLUMN "is_email_verified" bool NOT NULL DEFAULT false;
```

Вот и все!

Файл для миграции `up` готов.

Теперь займёмся миграцией `down`. Для неё нет сгенерированного кода, 
поэтому нам придется написать всё самостоятельно. Но сделать это довольно 
просто. Нам просто нужно отменить те изменения, которые мы внесли при 
миграции `up`. Во-первых, удалить таблицу "verify_emails" 
`DROP TABLE  "verify_emails"`, если она существует. Ключевое слово 
`CASCADE` в конце гарантирует, что если в других таблицах существуют записи, 
которые ссылаются на эту таблицу, все они будут удалены. Затем для 
столбца "is_email_verified" мы можем использовать тот же оператор
`ALTER TABLE "users"`, что и в миграции `up`, но на этот раз заменим 
`ADD COLUMN` на `DROP COLUMN`. После этого оба скрипта с миграциями готовы
к использованию.

```postgresql
DROP TABLE IF EXISTS "verify_emails" CASCADE;

ALTER TABLE "users" DROP COLUMN "is_email_verified";
```

Давайте посмотрим на текущее состояние БД, прежде чем запускать их. Как 
видите, текущая версия миграции — 3.

![](../images/part61/4.png)

Теперь давайте запустим в терминале "make migrateup".

```shell
make migrateup
migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up
2023/02/25 11:36:26 Start buffering 4/u add_verify_emails
2023/02/25 11:36:26 Read and execute 4/u add_verify_emails
2023/02/25 11:36:26 Finished 4/u add_verify_emails (read 9.04674ms, ran 57.574636ms)
2023/02/25 11:36:26 Finished after 72.96994ms
2023/02/25 11:36:26 Closing source and database
```

Миграция успешно выполнена. Поэтому, если мы обновим TablePlus, то мы 
увидим, что версия изменилась на 4.

![](../images/part61/5.png)

Была создана новая таблица "verify_emails".

![](../images/part61/6.png)

А в таблице «users» появился новый столбец: "is_email_verified", значение
которого было установлено в `FALSE` для всех существующих записей. 

![](../images/part61/7.png)

Это именно то, что чего мы хотели. Таким образом, скрипт миграции `up` 
работает хорошо. Теперь давайте попробуем запустить `make migratedown1` в 
терминале, чтобы выполнить только последний скрипт миграции `down`.

```shell
make migratedown1
migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose down 1
2023/02/25 11:36:58 Start buffering 4/d add_verify_emails
2023/02/25 11:36:58 Read and execute 4/d add_verify_emails
2023/02/25 11:36:58 Finished 4/d add_verify_emails (read 9.216659ms, ran 24.391741ms)
2023/02/25 11:36:58 Finished after 40.384528ms
2023/02/25 11:36:58 Closing source and database
```

На этот раз, если мы обновим TablePlus, таблица "verify_emails" исчезнет. 
И текущая версия миграции будет опять равна трём.

![](../images/part61/8.png)

Столбец "is_email_verified" также исчез из таблицы "users".

![](../images/part61/9.png)

Таким образом, скрипт миграции `down` тоже работает хорошо. Теперь мы 
можем запустить

```shell
make migrateup
migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up
2023/02/25 11:37:16 Start buffering 4/u add_verify_emails
2023/02/25 11:37:16 Read and execute 4/u add_verify_emails
2023/02/25 11:37:16 Finished 4/u add_verify_emails (read 8.722009ms, ran 31.07725ms)
2023/02/25 11:37:16 Finished after 46.569695ms
2023/02/25 11:37:16 Closing source and database
```

ещё раз, чтобы получить последнюю версию базы данных, прежде чем 
продолжить реализацию нашего функционала.

Отлично, теперь структура БД приведена к необходимому нам виду.

## Пишем запрос для новой записи в таблице "verify_email"

Вернемся к коду и напишем запрос для создания новой записи в таблице 
"verify_email". В папке `db/query` я создам новый файл с названием 
`verify_email.sql`. Поскольку мы собираемся генерировать код с 
помощью `sqlc`, я напишу для него комментарий, помогающий создать его, 
здесь, в `verify_email.sql`. Название сгенерированной функции должно быть
`CreateVerifyEmail` и она должна возвращать одну запись.

```postgresql
-- name: CreateVerifyEmail :one
```

Под этим комментарием давайте напишем наш запрос:

```postgresql
-- name: CreateVerifyEmail :one
INSERT INTO verify_emails (
                           
)
```

Теперь давайте посмотрим, какие столбцы мы должны задать в этом запросе.
Нет смысла указывать столбец "id", потому что он является первичным ключом 
с автоматическим инкрементом. Мы обязательно должны задать имя пользователя,
адрес электронной почты и секретный код. Для последних трёх столбцов можно
использовать значения по умолчанию, поэтому нам не обязательно их задавать
в нашем `INSERT` запросе. Давайте вернемся к нашему запросу и добавим 
названия трех столбцов: `username`, `email` и `secret_code`, а затем их
значения (`VALUES`), которые должны быть указаны с помощью трёх 
соответствующих входных аргументов. Наконец, мы добавляем ключевое слово 
`RETURNING *` в конце, чтобы вернуть все столбцы только что вставленной 
записи.

```postgresql
-- name: CreateVerifyEmail :one
INSERT INTO verify_emails (
    username,
    email,
    secret_code
) VALUES (
    $1, $2, $3
) RETURNING *;
```

Теперь мы можем выполнить в терминале,

```shell
make sqlc
sqlc generate
```

чтобы сгенерировать Golang код для этого запроса.

Отлично, теперь если мы откроем пакет `sqlc`,

```go
CreateUser(ctx context.Context, arg CreateUserParams) (User, error)
CreateVerifyEmail(ctx context.Context, arg CreateVerifyEmailParams) (VerifyEmail, error)
DeleteAccount(ctx context.Context, id int64) error
```

то увидим, что в интерфейс `Querier` добавлен новый метод: 
`CreateVerifyEmail()`. И также была сгенерирована его реализация в этом 
файле: `verify_emails.sql.go`. Обратите внимание, что только что внесённое
нами изменение нарушит некоторые тесты в пакете `api`, поскольку новый
добавленный метод ещё не был реализован в `MockStore`. Так что нам придется
выполнить

```shell
make mock
mockgen -package mockdb -destination db/mock/store.go github.com/techschool/simplebank/db/sqlc Store
```

в терминале, чтобы повторно сгенерировать `MockStore`.

После этого все ошибки пропадут.

Превосходно!

## Обновляем обработчик задач

Теперь пришло время вернуться к обработчику задач и использовать метод, 
который мы только что добавили, для создания новой "verify_email" записи 
в базе данных. Я вызову `processor.store.CreateVerifyEmail`, передам 
контекст и объект `db.CreateVerifyEmailParams`. Нам нужно будет задать 
значения для трех входных полей: `Username` будет равен `user.Username`,
`Email` — `user.Email`, а для `SecretCode` я воспользуюсь функцией 
`util.RandomString()`, которая сгенерирует случайный код из 32 символов.

```go
func (processor RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
    ...
    
    user, err := processor.store.GetUser(ctx, payload.Username)
    if err != nil {
        // if err == sql.ErrNoRows {
        //	return fmt.Errorf("user doen't exist: %w", asynq.SkipRetry)
        // }
        return fmt.Errorf("failed to get user: %w", err)
    }
    
    processor.store.CreateVerifyEmail(ctx, db.CreateVerifyEmailParams{
        Username:   user.Username,
        Email:      user.Email,
        SecretCode: util.RandomString(32),
    })
    
    ...
}
```

Эту функцию мы написали давным-давно. Строка, которую она сгенерирует, 
будет содержать только строчные латинские буквы.

```go
func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet)

	for i := 0; i < n; i++ {
		c := alphabet[rand.Intn(k)]
		sb.WriteByte(c)
	}

	return sb.String()
}
```

Вы всегда можете написать другую функцию, подходящую для вашей системы, 
но убедитесь, что секретный код достаточно длинный, чтобы избежать атак
«грубой силой» (полным перебором). OK, после своего выполнения эта функция
`processor.store.CreateVerifyEmail()` вернет объект `verifyEmail` и ошибку.
Если ошибка не равна `nil`, мы просто обертываем её и возвращаем с 
сообщением о том, что "failed to create verify email" («не удалось 
создать письмо для подтверждения адреса электронной почты»).

```go
func (processor RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
    ...
    
    user, err := processor.store.GetUser(ctx, payload.Username)
    if err != nil {
        // if err == sql.ErrNoRows {
        //	return fmt.Errorf("user doen't exist: %w", asynq.SkipRetry)
        // }
        return fmt.Errorf("failed to get user: %w", err)
    }

    verifyEmail, err := processor.store.CreateVerifyEmail(ctx, db.CreateVerifyEmailParams{
        Username:   user.Username,
        Email:      user.Email,
        SecretCode: util.RandomString(32),
    })
    if err != nil {
        return fmt.Errorf("failed to create verify email: %w", err)
    }
	
    ...
}
```

В противном случае запись в таблице была успешно создана, и пришло время
отправить пользователю настоящее электронное письмо.

## Отправляем пользователю настоящее электронное письмо

Я воспользуюсь кодом из пакета `mail`, который мы написали на прошлой 
лекции. Итак, в структуру `RedisTaskProcessor` я добавлю новое поле 
`mailer` типа `EmailSender`.

```go
type RedisTaskProcessor struct {
	server *asynq.Server
	store  db.Store
	mailer mail.EmailSender
}
```

Как видите, это интерфейс, который мы написали в предыдущей лекции,
предоставляет приведенный ниже метод

```go
type EmailSender interface {
    SendEmail(
        subject string,
        content string,
        to []string,
        cc []string,
        bcc []string,
        attachFiles []string,
    ) error
}
```

для отправки электронных писем.

Возвращаясь к обработчику задач, мы должны добавить новый аргумент 
`mailer` к этой функции `NewRedisTaskProcessor`, а также сюда при 
инициализации `RedisTaskProcessor`.

```go
func NewRedisTaskProcessor(redisOpt asynq.RedisClientOpt, store db.Store, mailer mail.EmailSender) TaskProcessor {
	...

	return &RedisTaskProcessor{
		server: server,
		store:  store,
		mailer: mailer,
	}
}
```

Хорошо, теперь, когда `mailer` доступен внутри обработчика, мы можем 
вернуться к задаче отправки письма для подтверждения адреса электронной почты
и воспользоваться им. После создания записи в таблице "verify_email" базы
данных я вызову `processor.mailer.SendEmail()`. Мы должны передать в 
функцию тему, содержимое и адрес, на который должно быть отправлено 
электронное письмо. В этом случае мы не используем поля `cc`, `bcc` и не 
прикрепляем файлы, поэтому я использую `nil` для них.

```go
func (processor RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
	...
    if err != nil {
        return fmt.Errorf("failed to create verify email: %w", err)
    }
    
    processor.mailer.SendEmail(subject, content, to, nil, nil, nil)
    
    log.Info().Str("type", task.Type()).Bytes("payload", task.Payload()).
        Str("email", user.Email).Msg("processed task")
    return nil
}
```

Теперь давайте определим тему письма (`subject`), скажем, "Welcome to Simple 
Bank" («Добро пожаловать в Simple Bank»). Затем зададим содержимое 
(`content`), которое будет состоять из нескольких строк. Я хочу сказать 
"Hello" («Здравствуйте»), после чего указать имя пользователя, поэтому давайте 
добавим сюда оператор `fmt.Sprintf()` для форматирования строки и передаем 
`user.FullName` в качестве одного из входных аргументов. Затем я добавлю
тег разрыва строки (`<br>`), чтобы начать следующее предложение с новой
строки. После этого мы говорим "Thank you for registering with us!" 
(«Благодарим за регистрацию на нашем сайте!») и опять переносим строку.
Теперь самая важная часть сообщения. Мы предоставляем пользователю ссылку, 
по которой он может щелкнуть, чтобы подтвердить свой адрес электронной
почты. Здесь я использую тег `<a href>`, чтобы указать URL-адрес 
подтверждения. Поэтому нам нужно объявить эту новую переменную `verifyUrl`
перед созданием содержимого письма. Обычно она должна представлять собой 
URL-адрес на фронтенд страницу, которая отвечает за анализ входного 
аргумента из URL-адреса и вызов соответствующего API на стороне сервера 
для подтверждения. Для этого учебного примера я приравняю его к 
`http://simple-bank.org` с двумя параметрами: первый параметр — это 
идентификатор записи (`id`) в таблице "verify_email", а второй — секретный
код для той же записи. Мы можем получить эти значения из объекта 
`verifyEmail`, который мы создали выше.

```go
func (processor RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
    ...
	if err != nil {
        return fmt.Errorf("failed to create verify email: %w", err)
    }
    
    subject := "Welcome to Simple Bank"
    verifyUrl := fmt.Sprintf("http://simple-bank.org?id=%d&secret_code=%s",
        verifyEmail.ID, verifyEmail.SecretCode)
    content := fmt.Sprintf(`Hello %s,<br>
    Thank you for registering with us!<br>
    Please <a href="%s">click here</a> to verify your email address.<br>
    `, user.FullName, verifyUrl)
    processor.mailer.SendEmail(subject, content, to, nil, nil, nil)
    
    ...
}
```

OK, далее мы должны объявить переменную, где будет храниться адрес,
на который мы хотим отправить электронное письмо. Обратите внимание, что
функция `SendEmail()` может возвращать ошибку. Поэтому нам нужно 
проверить, равна ли эта ошибка `nil` или нет. Если она не `nil`, мы 
просто обернём и вернем её со следующим сообщением: "failed to send 
verify email" («не удалось отправить письмо для подтверждения адреса 
электронной почты»).

```go
func (processor RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
    ...
    content := fmt.Sprintf(`Hello %s,<br>
    Thank you for registering with us!<br>
    Please <a href="%s">click here</a> to verify your email address.<br>
    `, user.FullName, verifyUrl)
    to := []string{user.Email}
    err = processor.mailer.SendEmail(subject, content, to, nil, nil, nil)
    if err != nil {
        return fmt.Errorf("failed to send verify email: %w", err)
    }
    
    ...
}
```

Вот и всё! Мы обновили реализацию задачи по отправке письма для 
подтверждения адреса электронной почты.

Теперь последнее, что нам нужно сделать, это обновить файл `main.go`, потому
что здесь возникла ошибка.

```go
func runTaskProcessor(redisOpt asynq.RedisClientOpt, store db.Store) {
	taskProcessor := worker.NewRedisTaskProcessor(redisOpt, store)
	...
}
```

Поскольку мы добавили новый объект `mailer` в функцию 
`NewRedisTaskProcessor()`, поэтому в начале функции `runTaskProcessor()`
я вызову `mail.NewGmailSender()`. Она требует три входных аргумента:
`name`, `fromEmailAddress` и `fromEmailPassword`, которые мы можем получить
из переменных окружения через объект `config`, который мы обновили в 
предыдущей лекции. Итак, давайте добавим объект `config` в качестве первого
входного аргумента этой функции. И мы можем передать его значение при 
вызове `runTaskProcessor` в функции `main()`.

```go
func runTaskProcessor(config util.Config, redisOpt asynq.RedisClientOpt, store db.Store) {
    mailer := mail.NewGmailSender()
    taskProcessor := worker.NewRedisTaskProcessor(redisOpt, store, mailer)
    ...
}
```

```go
func main() {
    ...
    taskDistributor := worker.NewRedisTaskDistributor(redisOpt)
    go runTaskProcessor(config, redisOpt, store)
    go runGatewayServer(config, store, taskDistributor)
    runGrpcServer(config, store, taskDistributor)
    //runGinServer(config, store)
}
```

OK, теперь с помощью объекта `config` мы можем получить значения имени 
отправителя электронной почты, адреса отправителя электронной почты и 
пароля отправителя электронной почты. Они уже заданы в файле `app.env`.
Таким образом, мы можем использовать их для создания переменной `mailer`
с помощью `NewGmailSender`. И применить `mailer` для инициализации 
`RedisTaskProcessor`.

```go
func runTaskProcessor(config util.Config, redisOpt asynq.RedisClientOpt, store db.Store) {
    mailer := mail.NewGmailSender(config.EmailSenderName, config.EmailSenderAddress, config.EmailSenderPassword)
    taskProcessor := worker.NewRedisTaskProcessor(redisOpt, store, mailer)
    ...
}
```

И на этом по сути всё! Теперь задача отправки письма для подтверждения
адреса электронной почты готова к тестированию.

## Тестируем наш код, используя Postman

Давайте выполним

```shell
make server
go run main.go
11:52AM INF db migrated successfully
11:52AM INF start gRPC server at [::]:9090
11:52AM INF start task processor
11:52AM INF Starting processing
11:52AM INF start HTTP gateway server at [::]:8080
```

в терминале, чтобы запустить сервер.

Затем в Postman я немного изменю тело запроса `create_user`, имя пользователя
будет равно "techschool", полное имя и фамилия — "Tech School" и, 
наконец, адрес электронной почты этого пользователя — 
"techschool.guru@gmail.com".

OK, давайте отправим этот запрос на сервер.

![](../images/part61/10.png)

Он успешно выполнен. Создан новый пользователь. И если мы посмотрим в логи
сервера, то увидим, что в очередь поставлена новая задача по отправке письма 
для подтверждения адреса электронной почты.

```shell
11:53AM INF enqueued task max_retry=10 payload="{\"username\":\"techschool\"}" queue=critical type=task:send_verify_email
11:53AM INF received an HTTP request duration=101.511666 method=POST path=/v1/create_user protocol=http status_code=200 status_text=OK
```

Через несколько секунд задача будет обработана.

```shell
11:53AM INF processed task email=techschool.guru@gmail.com payload="{\"username\":\"techschool\"}" type=task:send_verify_email
```

Это означает, что письмо для подтверждения было отправлено на 
зарегистрированный адрес электронной почты.

Итак, давайте откроем почтовый ящик Gmail Tech School и обновим его.

И вуаля, мы увидим там новое электронное письмо с заголовком: "Welcome to 
Simple Bank" («Добро пожаловать в Simple Bank»).

![](../images/part61/11.png)

Внутри содержимое этого письма точно такое, как мы задали выше:
"Hello Tech School, Thank you for registering with us! Please click
here to verify your email address." («Здравствуйте, Tech School! 
Благодарим за регистрацию на нашем сайте! Пожалуйста, нажмите сюда, 
чтобы подтвердить свой адрес электронной почты»).

![](../images/part61/12.png)

И если мы нажмем на ссылку, она приведёт нас на сайт `simple-bank.org`.

Конечно, этой страницы еще не существует, но в URL мы видим два параметра:
`id` и `secret_code`.

![](../images/part61/13.png)

На самом деле, я думаю, что мы должны немного изменить URL-адрес, чтобы 
он указывал на определенную страницу, а не на домашнюю страницу, например, 
мы можем добавить путь `/verify_email` к URL-адресу, следующим образом.

```go
func (processor RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
    ...
    
    subject := "Welcome to Simple Bank"
    verifyUrl := fmt.Sprintf("http://simple-bank.org/verify_email?id=%d&secret_code=%s",
        verifyEmail.ID, verifyEmail.SecretCode)
    content := fmt.Sprintf(`Hello %s,<br>
    Thank you for registering with us!<br>
    Please <a href="%s">click here</a> to verify your email address.<br>
    `, user.FullName, verifyUrl)
    ...
}
```

И теперь, если мы посмотрим в базу данных, то увидим там новую запись в
таблице `verify_email`, с теми же `id` и `secret_code`, что и в параметрах 
URL запроса.

![](../images/part61/14.png)

Кроме того, в таблице "users" появилась новая запись для пользователя
"tech school", чей адрес электронной почты ещё не подтвержден.

![](../images/part61/15.png)

На следующей лекции мы узнаем, как написать новый API для подтверждения
адреса электронной почты.

А пока вы можете попробовать сделать это самостоятельно, чтобы понять
как всё работает.

И на этом мы закончим эту лекцию. Надеюсь, она была интересной и 
полезной для вас. Большое спасибо за время, потраченное на чтение,
желаю Вам получать удовольствие от обучения и до встречи на следующей
лекции.
