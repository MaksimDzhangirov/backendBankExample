# Реализуем API для подтверждения адреса электронной почты в Go

[Оригинал](https://www.youtube.com/watch?v=50ZN-4UNwnY)

Всем привет, добро пожаловать на мастер-класс по бэкенду! На прошлой лекции
мы узнали, как создать и отправить письмо для подтверждения адреса электронной
почты новому зарегистрированному пользователю. В электронном письме мы 
просим его нажать на эту ссылку

```go
verifyUrl := fmt.Sprintf("http://simple-bank.org/verify_email?id=%d&secret_code=%s",
        verifyEmail.ID, verifyEmail.SecretCode)
```

чтобы подтвердить свой адрес электронной почты. Но мы еще не реализовали
API для обработки этой ссылки. И как это сделать я покажу вам именно на 
этой лекции.

## Пишем API для подтверждения адреса электронной почты

Идея заключается в том, что при получении запроса к API для подтверждение 
адреса электронной почты мы будем искать в базе запись в таблице 
`verify_email`, используя переданные в запросе `id` и `secret_code`.
Если запись существует, не просрочена, не была использована ранее,
мы можем пометить её как использованную и поменять значение в поле 
`is_email_verified` для соответствующего пользователя в таблице `users`
на `true`. Довольно просто, не так ли?

Хорошо, давайте откроем задачу отправки письма для подтверждения адреса 
электронной почты. В предыдущей лекции я использовал этот несуществующий
URL

```go
verifyUrl := fmt.Sprintf("http://simple-bank.org/verify_email?id=%d&secret_code=%s",
        verifyEmail.ID, verifyEmail.SecretCode)
```

для демонстрационных целей. Мы должны отправлять пользователям ссылку на
страницу фронтэнд приложения, которое будет анализировать входные 
параметры запроса, а затем использовать их для вызова API 
подтверждения адреса электронной почты на стороне сервера. И конечно же
на продакшене, мы должны использовать защищенный протокол HTTPS здесь 
`https://simple-bank.org/`. Но для демонстрационных целей сейчас я не 
буду реализовывать страницу настоящего фронтэнд приложения, а заменю её
ссылкой, которая будет указывать непосредственно на API нашего локального
сервера: `http://localhost:8080/v1/verify_email`.

```go
verifyUrl := fmt.Sprintf("http://localhost:8080/v1/verify_email?id=%d&secret_code=%s",
    verifyEmail.ID, verifyEmail.SecretCode)
```

`v1` здесь обозначает версию API, которую мы также используем для всех 
других маршрутов. И чтобы было понятнее, я изменю этот параметр `id` на 
`email_id`.

```go
verifyUrl := fmt.Sprintf("http://localhost:8080/v1/verify_email?email_id=%d&secret_code=%s",
    verifyEmail.ID, verifyEmail.SecretCode)
```

поскольку он обозначает идентификатор записи таблицы `verify_email` в 
нашей базе данных. Итак, теперь давайте начнем создавать API. Поскольку 
мы используем gRPC шлюз, я определю его в `protobuf`. Последовательность 
действий аналогична той, которую мы проводили для других API, поэтому
давайте скопируем этот файл `rpc_create_user.proto` и переименуем его в 
`rpc_verify_email.proto`.

```shell
cd proto
cp rpc_create_user.proto rpc_verify_email.proto
```

В этом файле заменим все строки "CreateUser" на "VerifyEmail". В запросе
у нас будут два поля: `email_id` и `secret_code`. Поле `email_id` должно
быть типа  `int64`, а `secret_code` — `string`.

```protobuf
message VerifyEmailRequest {
  int64 email_id = 1;
  string secret_code = 2;
}
```

В качестве ответа от сервера мы будет отправлять только одно логическое 
значение: `is_verified`. Оно сообщит нам было ли адрес электронной почты 
успешно подтверждён или нет.

```protobuf
message VerifyEmailResponse {
  bool is_verified = 1;
}
```

Хорошо, теперь мы можем избавиться от лишнего оператора импорта вверху.

```protobuf
import "user.proto";
```

В качестве следующего шага, нам необходимо будет добавить новую RPC в файл
`service_simple_bank.proto`. Я импортирую файл `rpc_verify_email.proto` и
в конце файла давайте продублируем `LoginUser` RPC, а затем заменим его
название на `VerifyEmail`, входной аргумент — на `VerifyEmailRequest`, а
результат — на `VerifyEmailResponse`. Теперь HTTP параметр 
`option (google.api.http)` на `GET`, поскольку когда пользователь нажимает 
на ссылку, которую мы отправили в письме, браузер отправляет GET запрос.
Кроме того, путь следует изменить на `/v1/verify_email`. У GET запроса
нет "body", поэтому давайте удалим его. Этот параметр 
`option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation)`
gRPC шлюза предназначен для создания документации, поэтому давайте изменим 
поле `description` на: "Use this API to verify user's email address" 
(«Используйте этот API для подтверждения адреса электронной почты 
пользователя»), а `summary` — на: "Verify email" («Подтверждает адрес 
электронной почты»). Это все изменения, которые нужно внести.

```protobuf
...
import "rpc_login_user.proto";
import "rpc_verify_email.proto";
import "protoc-gen-openapiv2/options/annotations.proto";
...

service SimpleBank {
  ...
  rpc LoginUser(LoginUserRequest) returns (LoginUserResponse) {
    option (google.api.http) = {
      post: "/v1/login_user"
      body: "*"
    };
    option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
      description: "Use this API to login user and get access token & refresh token";
      summary: "Login user";
    };
  }
  rpc VerifyEmail(VerifyEmailRequest) returns (VerifyEmailResponse) {
    option (google.api.http) = {
      get: "/v1/verify_email"
    };
    option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
      description: "Use this API to verify user's email address";
      summary: "Verify email";
    };
  }
}
```

Теперь мы можем запустить

```shell
make proto
rm -f pb/*.go
rm -f doc/swagger/*.swagger.json
protoc --proto_path=proto --go_out=pb --go_opt=paths=source_relative \
        --go-grpc_out=pb --go-grpc_opt=paths=source_relative \
        --grpc-gateway_out=pb --grpc-gateway_opt=paths=source_relative \
        --openapiv2_out=doc/swagger --openapiv2_opt=allow_merge=true,merge_file_name=simple_bank \
        proto/*.proto
statik -src=./doc/swagger -dest=./doc
```

в терминале, чтобы сгенерировать код для этого нового API.

Вуаля, в папке `pb` мы видим новый файл `rpc_verify_email.pb.go`. Однако мы 
могли заметить некоторые выделенные красным файлы в пакете `gapi`. Похоже, 
что-то не так с импортом пакета `pb`. На самом деле, я замечаю, что это 
иногда происходит, потому что мы удаляем все существующие файлы в пакете 
`pb`: `rm -f pb/*.go` — перед тем как сгенерировать новый код, поэтому у 
VS Code возникают проблемы с перезагрузкой пакета после этого. Если такое 
происходит, мы можем просто выйти из VS Code, затем снова открыть его, и 
все должно быть в порядке.

Хорошо, теперь у нас есть сгенерированные заглушки для нового API. Мы 
можем реализовать его обработчик в пакете `gapi`. Я скопирую файл 
`rpc_create_user.go` и переименую его в `rpc_verify_email.go`.

```shell
cd gapi
cp rpc_create_user.go rpc_verify_email.go
```

В этом файле заменим все вхождения "CreateUser" на "VerifyEmail". Теперь
у нас есть обработчик `VerifyEmail`, который принимает `VerifyEmailRequest` 
в качестве входного аргумента и возвращает `VerifyEmailResponse` в 
качестве результата.

```go
func (server *Server) VerifyEmail(ctx context.Context, req *pb.VerifyEmailRequest) (*pb.VerifyEmailResponse, error) {
	...
}
```

Я удалю весь код, который осуществляет создание пользователя, поскольку
нам нужно будет реализовать новую логику для подтверждения адреса 
электронной почты. Хорошо, теперь давайте модифицируем функцию для проверки
запроса. `VerifyEmailRequest` содержит два поля: идентификатор электронной 
почты и секретный код. Поэтому нам нужно будет открыть пакет `validator`, 
чтобы добавить новые функции для их проверки.

Во-первых, функцию для проверки идентификатора электронной почты. Она 
будет принимать входное значение типа `int64` и возвращать ошибку. Если
переданное значение меньше или равно 0, мы вернем ошибку, сообщающую о
том, что входной аргумент должен быть положительным целым числом. В 
противном случае просто возвращаем `nil`.

```go
func ValidateEmailId(value int64) error {
	if value <= 0 {
		return fmt.Errorf("must be a positive integer")
	}
	return nil
}
```

Затем функцию проверки секретного кода примет строковое значение в 
качестве входных данных и также вернет ошибку в качестве результата. В ней
я просто проверю длину строки, поэтому мы просто вызываем функцию 
`ValidateString()`, задавая минимальную длину в 32, а максимальную длину в 
128 символов. Вы можете добавить здесь больше проверок, если хотите, но
я не хочу уделять этому слишком много времени и это не основная тема 
лекции.

```go
func ValidateSecretCode(value string) error {
	return ValidateString(value, 32, 128)
}
```

Теперь я вернусь к обработчику `rpc_verify_email`. Здесь, в 
`validateVerifyEmailRequest()`, давайте вызовем `val.ValidateEmailId()`, 
передав идентификатор электронной почты из входного запроса, и если эта 
функция вернет ошибку, мы добавим `email_id` в список нарушений правил
для полей. Точно так же мы вызываем `val.ValidateSecretCode()` и передаем 
значение секретного кода, предоставленное во входном запросе. Если
возвращаемая ошибка не равна `nil`, мы добавляем `secret_code` в список 
нарушений правил для полей. Наконец, мы можем просто вернуть список нарушений
`violations` вызывающей стороне этой функции.

```go
func validateVerifyEmailRequest(req *pb.VerifyEmailRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	if err := val.ValidateEmailId(req.GetEmailId()); err != nil {
		violations = append(violations, fieldViolation("email_id", err))
	}

	if err := val.ValidatePassword(req.GetSecretCode()); err != nil {
		violations = append(violations, fieldViolation("password", err))
	}

	return violations
}
```

Хорошо, теперь давайте воспользуемся входными значениями `email_id` и 
`secret_code`, чтобы извлечь и обновить информацию в базе данных. Поскольку
мы хотим обновить как таблицу `verify_emails`, так и `users`, нам нужно 
будет сделать это в рамках одной транзакции БД. Поэтому в папке `sqlc` я 
создам копию файла `tx_create_user.go` и переименую его в 
`tx_verify_email.go`.

```go
cd db/sqlc
cp tx_create_user.go tx_verify_email.go
```

Затем в этом файле давайте переименуем все вхождения "CreateUserTx" в 
"VerifyEmailTx". Структура с параметрами транзакции будет содержать всего
два поля: идентификатор электронной почты типа `int64` и секретный код 
типа `string`. Кроме того, структура, хранящая результат транзакции, также
будет состоять из двух полей: для хранения обновленных записей о пользователе
из таблицы `users` и обновленной записи о подтверждаемом адресе электронной
почты из таблицы `verify_emails`.

```go
type VerifyEmailTxParams struct {
	EmailId    int64
	SecretCode string
}

type VerifyEmailTxResult struct {
	User        User
	VerifyEmail VerifyEmail
}
```

OK, внутри транзакции я пока всё удалю, оставив только объявление переменной
для ошибки и вернув её в конце.

```go
func (store *SQLStore) VerifyEmailTx(ctx context.Context, arg VerifyEmailTxParams) (VerifyEmailTxResult, error) {
	var result VerifyEmailTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error
		
		return err
	})

	return result, err
}
```

Затем нам нужно написать новый запрос, который найдёт подтверждаемый адрес
электронной почты по идентификатору и секретному коду и обновит столбец
`is_used` для этой записи на `true`. После этого нам также необходимо 
изменить значение поля `is_email_verified` соответствующей записи в 
таблице `users` на `true`.

Во-первых, давайте откроем файл `verify_email.sql` внутри папки `query`. 
Я добавлю новый запрос под названием `UpdateVerifyEmail`, который вернет 
одну строку обновленной записи подтверждаемого адреса электронной 
почты. SQL-запрос будет таким:

```postgresql
-- name: UpdateVerifyEmail :one
UPDATE verify_emails
SET
    is_used = TRUE
WHERE
    id = @id
    AND secret_code = @secret_code
    AND is_used = FALSE
    AND expired_at > now()
RETURNING *;
```

Эта запись ("@id") ещё один способ именования входных параметров в `sqlc`. 
Вы также можете использовать здесь параметр `$1`, это тоже сработает. Но 
лично я предпочитаю именованный параметр, так как он выглядит понятнее. 
Нам также необходимо проверить, совпадает ли секретный код этого 
подтверждаемого адреса с кодом, предоставленным во входном аргументе, и 
что этот код еще не использовался ранее, и, что наиболее важно, срок его 
действия не истек или, другими словами, метка времени истечения срока действия 
должно быть больше, чем текущая. Наконец, мы используем выражение 
`RETURNING *` в конце запроса, чтобы вернуть обновленную запись 
подтверждаемого адреса электронной почты.

Теперь давайте перейдем к файлу `user.sql`. У нас уже есть `UpdateUser`, 
поэтому все, что нам нужно сделать, это добавить здесь новое выражение, 
чтобы задать для поля `is_email_verified` новое значение, передаваемое с 
помощью входного аргумента. Напоминаем, что мы используем функцию `sqlc`, 
допускающую значение типа `NULL`, так что каждое поле может обновляться 
независимо, не затрагивая другое. Вы можете вернуться на несколько лекций 
назад и просмотреть содержимое лекции 49, чтобы понять, как это работает.

```postgresql
-- name: UpdateUser :one
UPDATE users
SET
    hashed_password = COALESCE(sqlc.narg(hashed_password), hashed_password),
    password_changed_at = COALESCE(sqlc.narg(password_changed_at), password_changed_at),
    full_name = COALESCE(sqlc.narg(full_name), full_name),
    email = COALESCE(sqlc.narg(email), email),
    is_email_verified = COALESCE(sqlc.narg(is_email_verified), is_email_verified)
WHERE
    username = sqlc.arg(username)
RETURNING *;
```

Итак, оба запроса для обновления данных готовы.

Теперь мы можем выполнить 

```shell
make sqlc
sqlc generate
```

в терминале, чтобы сгенерировать новый код для них. Как видите, в файле 
`verify_email.sql.go`

```go
func (q *Queries) UpdateVerifyEmail(ctx context.Context, arg UpdateVerifyEmailParams) (VerifyEmail, error) {
	row := q.db.QueryRowContext(ctx, updateVerifyEmail, arg.ID, arg.SecretCode)
	var i VerifyEmail
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Email,
		&i.SecretCode,
		&i.IsUsed,
		&i.CreatedAt,
		&i.ExpiredAt,
	)
	return i, err
}
```

у нас появилась новая функция для обновления записи подтверждаемого 
адреса электронной почты.

А в файле `user.sql.go` у функции обновляющей запись о пользователе также
появился новый входной аргумент: `is_email_verified`. Прекрасно!

```go
type UpdateUserParams struct {
	HashedPassword    sql.NullString `json:"hashed_password"`
	PasswordChangedAt sql.NullTime   `json:"password_changed_at"`
	FullName          sql.NullString `json:"full_name"`
	Email             sql.NullString `json:"email"`
	IsEmailVerified   sql.NullBool   `json:"is_email_verified"`
	Username          string         `json:"username"`
}
```

Теперь вернёмся к транзакции для подтверждения адреса электронной почты.
В теле функции `execTx` я вызову `q.UpdateVerifyEmail()` и передам 
переменную контекста вместе с объектом `UpdateVerifyEmailParams`. Поле 
`ID` должно быть задано равным значению `EmailId` входного аргумента, и то 
же самое касается и секретного кода. Эта функция вернет обновленную запись 
подтверждаемого адреса электронной почты и ошибку. Поэтому я сохраню 
обновленную запись в `result.VerifyEmail`. Затем проверьте, равна ли 
ошибка `nil` или нет. Если она не равна `nil`, мы просто возвращаем её. В 
противном случае передан правильный секретный код, поэтому пора вызвать 
функцию `q.UpdateUser()`, чтобы обновить значение поля `IsEmailVerified` 
на `true`. Это универсальная функция, которая позволяет нам обновлять 
несколько полей записи пользователя, но в нашем случае нам нужно обновить 
только одно поле. Поэтому нам просто нужно указать требуемое имя 
пользователя и допустимое значение для столбца `IsEmailVerified`. Мы можем 
легко получить имя пользователя из `result.VerifyEmail`, а полю 
`IsEmailVerified` необходимо присвоить объект `sql.NullBool`, у которого оба
поля `Bool` и `Valid` должны быть равны `true`. Хорошо, теперь мы можем
сохранить результат этой функции `UpdateUser` в `result.User` и если она
вернёт не `nil` ошибку, эта ошибка будет передана внешней функции.

```go
func (store *SQLStore) VerifyEmailTx(ctx context.Context, arg VerifyEmailTxParams) (VerifyEmailTxResult, error) {
	var result VerifyEmailTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.VerifyEmail, err = q.UpdateVerifyEmail(ctx, UpdateVerifyEmailParams{
			ID:         arg.EmailId,
			SecretCode: arg.SecretCode,
		})
		if err != nil {
			return err
		}

		result.User, err = q.UpdateUser(ctx, UpdateUserParams{
			Username: result.VerifyEmail.Username,
			IsEmailVerified: sql.NullBool{
				Bool:  true,
				Valid: true,
			},
		})

		return err
	})
	return result, err
}
```

На этом по сути всё! Транзакция для подтверждения адреса электронной почты
готова.

Мы можем вернуться к файлу `rpc_verify_email.go`, чтобы закончить 
реализацию функции обработчика. Но перед этим я забыл об одном важном 
моменте, мы должны добавить сигнатуру функции `VerifyEmailTx()` в интерфейс 
`Store`, потому что на уровне API у нас есть доступ только к этому общему 
интерфейсу, мы ничего не знаем о его конкретной реализации.

```go
type Store interface {
	Querier
	TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error)
	CreateUserTx(ctx context.Context, arg CreateUserTxParams) (CreateUserTxResult, error)
	VerifyEmailTx(ctx context.Context, arg VerifyEmailTxParams) (VerifyEmailTxResult, error)
}
```

Хорошо, теперь мы можем вызвать `server.store.VerifyEmailTx()`, передать 
контекст и объект `db.VerifyEmailTxParams`. Его поле `EmailId` должно быть
равно `req.GetEmailId()`, а `SecretCode` — `req.GetSecretCode()`. Эта 
функция вернет результат транзакции и ошибку, поэтому я сохраняю их в 
соответствующие переменные. Затем проверьте, равна ли ошибка `nil` или нет.
Если она не `nil`, мы просто возвращаем в качестве результата `nil` вместе 
с кодом ошибки gRPC состояния `Internal` и следующим сообщением "failed to
verify email" (не удалось проверить адрес электронной почты). Вы также можете
дополнительно проверить тип ошибки, чтобы вернуть другой код состояния, 
если, например, адрес электронной почты не найден. Здесь я решил упростить 
задачу, всегда возвращая один и тот же код ошибки. Наконец, мы можем 
записать в поле `IsVerified` объекта `VerifyEmailResponse` значение
`txResult.User.IsEmailVerified`. И после этого мы можем сказать, что 
реализация API для подтверждения адреса электронной почты завершена.

```go
func (server *Server) VerifyEmail(ctx context.Context, req *pb.VerifyEmailRequest) (*pb.VerifyEmailResponse, error) {
	violations := validateVerifyEmailRequest(req)
	if violations != nil {
		return nil, invalidArgumentError(violations)
	}

	txResult, err := server.store.VerifyEmailTx(ctx, db.VerifyEmailTxParams{
		EmailId:    req.GetEmailId(),
		SecretCode: req.GetSecretCode(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to verify email")
	}

	rsp := &pb.VerifyEmailResponse{
		IsVerified: txResult.User.IsEmailVerified,
	}
	return rsp, nil
}
```

Возможно вы заметили, что в пакете `api` появились ошибки. Причина в том, 
что имитация БД больше не удовлетворяет интерфейсу `Store`, так как мы 
только что добавили в этот интерфейс новый метод. Итак, чтобы исправить 
это, нам придется запустить

```shell
make mock
mockgen -package mockdb -destination db/mock/store.go github.com/techschool/simplebank/db/sqlc Store
```

в терминале, чтобы повторно сгенерировать `MockStore`. После этого ошибки
должны пропасть и всё должно прийти в норму.

Теперь мы готовы провести несколько тестов.

Но сначала я хочу очистить БД, чтобы было проще отслеживать новые записи.
Один из простых способов сделать это — запустить

```shell
make migratedown
migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose down
2023/03/18 11:23:22 Are you sure you want to apply all down migrations? [y/N]
y
2023/03/18 11:23:24 Applying all down migrations
2023/03/18 11:23:24 Start buffering 4/d add_verify_emails
2023/03/18 11:23:24 Start buffering 3/d add_sessions
2023/03/18 11:23:24 Start buffering 2/d add_users
2023/03/18 11:23:24 Start buffering 1/d init_schema
2023/03/18 11:23:24 Read and execute 4/d add_verify_emails
2023/03/18 11:23:24 Finished 4/d add_verify_emails (read 10.285017ms, ran 37.251805ms)
2023/03/18 11:23:24 Read and execute 3/d add_sessions
2023/03/18 11:23:24 Finished 3/d add_sessions (read 57.570959ms, ran 17.475981ms)
2023/03/18 11:23:24 Read and execute 2/d add_users
2023/03/18 11:23:24 Finished 2/d add_users (read 85.049288ms, ran 19.544725ms)
2023/03/18 11:23:24 Read and execute 1/d init_schema
2023/03/18 11:23:24 Finished 1/d init_schema (read 113.168821ms, ran 35.883769ms)
2023/03/18 11:23:24 Finished after 2.0460477s
2023/03/18 11:23:24 Closing source and database
```

чтобы применить все миграции `down`. Это удалит все таблицы из базы данных.

Затем мы выполним

```shell
make migrateup
migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up
2023/03/18 11:23:30 Start buffering 1/u init_schema
2023/03/18 11:23:30 Start buffering 2/u add_users
2023/03/18 11:23:30 Start buffering 3/u add_sessions
2023/03/18 11:23:30 Start buffering 4/u add_verify_emails
2023/03/18 11:23:30 Read and execute 1/u init_schema
2023/03/18 11:23:30 Finished 1/u init_schema (read 11.11641ms, ran 86.271254ms)
2023/03/18 11:23:30 Read and execute 2/u add_users
2023/03/18 11:23:30 Finished 2/u add_users (read 112.329252ms, ran 30.488839ms)
2023/03/18 11:23:30 Read and execute 3/u add_sessions
2023/03/18 11:23:30 Finished 3/u add_sessions (read 150.083295ms, ran 24.135692ms)
2023/03/18 11:23:30 Read and execute 4/u add_verify_emails
2023/03/18 11:23:30 Finished 4/u add_verify_emails (read 184.733063ms, ran 27.778521ms)
2023/03/18 11:23:30 Finished after 221.804983ms
2023/03/18 11:23:30 Closing source and database
```

чтобы повторно их создать.

Теперь, если мы обновим Table Plus, все существующие данные исчезнут.

![](../images/part62/1.png)

## Тестируем API для подтверждения адреса электронной почты

Я выполню команду

```shell
make server
go run main.go
11:23AM INF db migrated successfully
11:23AM INF start gRPC server at [::]:9090
11:23AM INF start task processor
11:23AM INF Starting processing
11:23AM INF start HTTP gateway server at [::]:8080
```

в терминале, чтобы запустить сервер и открою Postman, чтобы отправить 
следующий, показанный ниже запрос `create_user`. Он создаст нового 
пользователя `techschool` и адресом электронной почты 
`techschool.guru@gmail.com`.

![](../images/part62/2.png)

Вуаля, пользователь успешно создан. В логах сервера

```shell
11:24AM INF enqueued task max_retry=10 payload="{\"username\":\"techschool\"}" queue=critical type=task:send_verify_email
11:24AM INF received an HTTP request duration=98.256762 method=POST path=/v1/create_user protocol=http status_code=200 status_text=OK
```

мы видим, что задача по отправке письма для подтверждения адреса электронной
почты была поставлена в очередь. И через несколько секунд она будет 
обработана.

```shell
11:24AM INF processed task email=techschool.guru@gmail.com payload="{\"username\":\"techschool\"}" type=task:send_verify_email
```

Теперь, если мы посмотрим на входящую сообщения Gmail почты Tech School,
то увидим новое электронное письмо с заголовком "Welcome to Simple Bank" 
(«Добро пожаловать в Simple Bank»).

![](../images/part62/3.png)

Внутри него есть ссылка, которая указывает на наш локальный бекэнд API для
подтверждения адреса электронной почты. Давайте нажмём на неё и посмотрим
что произойдёт.

![](../images/part62/4.png)

Как видите, он отправляет GET запрос на API `verify_email`. И возвращает 
JSON ответ с `is_verified`, равным `true`. Это означает, что адрес 
электронной почты был успешно подтвержден. Опять же, я должен подчеркнуть, 
что данный способ подходит только для демонстрационных целей. В реальном 
приложении мы должны направлять пользователей на соответствующую фронтэнд
страницу, где эта информация будет красиво отображаться с соответствующим 
текстом и изображениями.

Хорошо, прежде чем мы закончим, давайте проверим записи в БД, чтобы 
убедиться, что все корректно обновлено. В таблице «users» у нас есть 
новый пользователь: `techschool`. И его поле `is_email_verified` равно 
`TRUE`, как и следовало ожидать.

![](../images/part62/5.png)

В таблице `verify_emails` также существует одна запись, принадлежащая 
пользователю "techschool". И её поле `is_used` равно `TRUE`, это означает,
что саму запись и хранящийся в записи секретный код больше нельзя будет
повторно использовать в будущем.

![](../images/part62/6.png)

Мы можем проверить это, вернувшись в браузер и обновив эту страницу.

![](../images/part62/7.png)

Он отправит на сервер ещё один `verify_email` запрос, но на этот раз мы 
получим ошибку: "failed to verify email" («не удалось подтвердить адрес 
электронной почты»).

Это именно то поведение, которого мы добивались! На этом давайте также
завершим сегодняшнюю лекцию, посвященную реализации  API для подтверждения 
адреса электронной почты.

Я надеюсь, что она была интересной и полезной для вас.

Большое спасибо за время, потраченное на чтение, желаю Вам получать 
удовольствие от обучения и увидимся на следующей лекции!
