# Как управлять сессиями пользователя с помощью refresh токена - Golang

[Оригинал](https://www.youtube.com/watch?v=rT20ylRLm5U)

Привет, всем!

Рад вас снова видеть на мастер-классе по бэкенду. Возможно вы удивились, 
потому что на [предыдущей лекции](part36-rus.md) я сказал, что она будет 
последней. 

Я планировал начать новый курс с более сложными темами по бэкенду. Однако, 
немного подумав, я решил, что целесообразнее продолжать публиковать их в 
этом курсе, поскольку основная цель курса — помочь вам стать специалистом 
в области бэкенд-разработки.

Без рассмотрения сложных для изучения тем это курс не был бы мастер-классом,
не так ли?

Хорошо, начнем с первой темы: как управлять сессиями пользователя?

## Как управлять сессиями пользователя

Если вы ещё помните, на лекциях 19 - 22 мы узнали, как применять PASETO 
или JWT для аутентификации с использованием токенов. 

Однако я должен подчеркнуть, что вы не должны использовать их в течение 
длительного периода времени.

![](../images/part37/1.png)

Из-за особенностей, связанных с отсутствием возможности сохранения своего 
состояния, эти токены доступа не хранятся в базе данных, поэтому их 
невозможно отозвать в случае утечки.

Следовательно, их срок действия должен быть очень коротким, например, 10 
или 15 минут.

Но если мы будем использовать только токены доступа, то по истечении срока 
их действия пользователям нужно будет снова залогиниться, вводя свои 
учётные данные (имя пользователя и пароль).

![](../images/part37/2.png)

С точки зрения удобства пользователям не понравиться, если их придётся заново
логиниться каждые 10 или 15 минут, верно?

Так что здесь вступает в игру другой тип токена, называемый refresh токеном.
Основная идея заключается в том, что мы будем использовать его для 
управления сессией с отслеживанием состояния на сервере, а клиент может 
использовать refresh токен с длительным сроком действия, чтобы запросить 
новый токен доступа по истечении его срока действия.

![](../images/part37/3.png)

Refresh токеном может быть даже простая строка из случайных символов или
мы также можем использовать PASETO, если хотим. Но он должен хранится в 
таблице сессий базы данных с одним вспомогательным полем логического типа, 
позволяющим заблокировать токен в случае его утечки.

Благодаря возможности отозвать refresh токен его срок действия может быть
намного дольше, чем у токена доступа, например, несколько дней или даже 
недель.

Итак, давайте займёмся написанием кода.

Во-первых, давайте запустим API сервер нашего простого банковского приложения.
Затем попробуйте отправить запрос на вход в систему в Postman.

![](../images/part37/4.png)

Как видите, прямо сейчас сервер возвращает только PASETO токен доступа.
И если мы посмотрим на файл `app.env` в Visual Studio Code,

```
DB_DRIVER=postgres
DB_SOURCE=postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable
SERVER_ADDRESS=0.0.0.0:8080
TOKEN_SYMMETRIC_KEY=12345678901234567890123456789012
ACCESS_TOKEN_DURATION=15m
```

то увидим, что срок действия токена доступа очень короткий, всего 15 минут.
Мы хотим добиться, чтобы сервер возвращал refresh токен с большим сроком 
действия при отправке запроса к API на вход в систему. Поэтому я добавлю 
сюда новую переменную `REFRESH_TOKEN_DURATION` и задам для неё значение 
равное 24 часам.

```
REFRESH_TOKEN_DURATION=24h
```

Затем в файле `config.go` мы должны обновить структуру `Config`, чтобы 
загрузить эту новую переменную.

Новое поле будет называться `RefreshTokenDuration` с типом `time.Duration` и
мы должны добавить для него дескриптор `mapstructure` с тем же именем, что 
и у переменной окружения.

```go
type Config struct {
	DBDriver             string        `mapstructure:"DB_DRIVER"`
	DBSource             string        `mapstructure:"DB_SOURCE"`
	ServerAddress        string        `mapstructure:"SERVER_ADDRESS"`
	TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
}
```

Готово!

Следующим шагом будет создание новой таблицы `sessions` в базе данных.

Итак, давайте воспользуемся этой командой в файле README, чтобы создать 
новую миграцию БД.

```shell
migrate create -ext sql -dir db/migration -seq <migration_name>
```

Я скопирую и вставлю её в терминал. Затем изменю название миграции на
`add_sessions`.

```shell
migrate create -ext sql -dir db/migration -seq add_sessions
```

![](../images/part37/5.png)

Таким образом, будет создано 2 файла миграции, `up` и `down`.

Команда для файла `up` будет похожа на ту, которая использовалась для
таблицы `users`, поэтому давайте просто скопируем её оттуда и вставим в
новый файл миграции.

Я изменю название таблицы на `sessions`. Первый столбец в этой таблице 
будет идентификатором сессии, который может совпадать с идентификатором
refresh токена, определяемый в полезной нагрузке токена. Так что его тип 
должен быть `uuid`, и он будет первичным ключом этой таблицы. Мы также будем 
хранить имя пользователя в этой таблице, но, конечно, оно больше не 
будет первичным ключом. В третьем столбце будет храниться refresh токен
этой сессии, если мы хотим отслеживать тип клиента, и откуда пользователь 
подключается к серверу, мы также можем хранить агента пользователя и IP-адрес
клиента в этой таблице. Затем мы добавим один важный логический столбец,
блокирующий сессию в случае утечки refresh токена. Наконец, давайте 
добавим столбец `expires_at`, указывающий время, когда истечет срок 
действия refresh токена. Кроме того, добавим ограничение внешнего ключа
для столбца `username`, который ссылается на тот же столбец таблицы
`users`. Файл миграции `up` готов.

```postgresql
CREATE TABLE "sessions" (
     "id" uuid PRIMARY KEY,
     "username" varchar NOT NULL,
     "refresh_token" varchar NOT NULL,
     "user_agent" varchar NOT NULL,
     "client_ip" varchar NOT NULL,
     "is_blocked" boolean NOT NULL DEFAULT false,
     "expires_at" timestamptz NOT NULL,
     "created_at" timestamptz NOT NULL DEFAULT (now())
);

ALTER TABLE "sessions" ADD FOREIGN KEY ("username") REFERENCES "users" ("username");
```

Для файла миграции `down`, по аналогии с файлом миграции `add_users.down`, 
всё, что нам нужно сделать, это удалить таблицу `sessions`.

```postgresql
DROP TABLE IF EXISTS "sessions";
```

Хорошо, теперь давайте откроем терминал и запустим `make migrateup`, 
чтобы применить новую миграцию.

```shell
make migrateup
```

Прекрасно, она успешно отработала! Давайте просмотрим обновленную структуру 
БД в Table Plus. Здесь на рисунке мы видим новую таблицу `sessions`.

![](../images/part37/6.png)

У неё есть столбец `username`, который ссылается на таблицу `users`. 
Столбцы  `refresh_token` и `is_blocked` определены именно так, как мы их 
описали в файле миграции. Превосходно!

На следующем шаге мы добавим несколько новых SQL-запросов для создания и 
извлечения сессии.

Я создам новый файл `session.sql` в папке `query`. Затем скопируем сюда 
содержимое файла `user.sql`. Первый запрос будет создавать новую сессию, 
поэтому пишем `INSERT INTO sessions`, затем указываем столбцы `id`, 
`username`, но на самом деле быстрее будет просто скопировать остальные 
столбцы из файла миграции `up`. А затем удалить лишнее, чтобы команда имела 
правильный синтаксис.

В Visual Studio Code мы можем легко создать несколько курсоров, нажимая 
клавишу Option (или Alt) и щелкая в разных местах редактора кода. Это 
позволяет нам редактировать несколько строк одновременно, что намного 
быстрее, чем делать это по одной.

Итак, мы только что добавили все необходимые столбцы: `refresh_token`, 
`user_agent`, `client_ip`, `is_blocked` и `expires_at`. Всего у нас будет 7 
столбцов в этой команде `INSERT`. Поэтому мы должны добавить еще три 
параметра в список `VALUES`. И запрос `INSERT` будет готов!

```postgresql
-- name: CreateSession: one
INSERT INTO sessions (
    id,
    username,
    refresh_token,
    user_agent,
    client_ip,
    is_blocked,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;
```

Теперь давайте перейдем к следующему запросу: `GetSession`. В нём мы будем
искать конкретную сессию по её идентификатору. Поэтому я изменю код в этом
условии

```postgresql
-- name: GetUser :one
SELECT * FROM users
WHERE username = $1 LIMIT 1;
```

на `WHERE id` равен первому параметру.

```postgresql
--name: GetSession :one
SELECT * FROM sessions
WHERE id = $1 LIMIT 1;
```

Вот и всё! Теперь выполним

```shell
make sqlc
```

в терминале, чтобы сгенерировать Golang код для двух новых запросов, 
которые мы только что написали.

Она прошла успешно. Итак, если мы откроем файл `session.sql.go` внутри 
папки `sqlc`, мы увидим, что код был сгенерирован. `sqlc` использует 
`google/uuid` для столбца идентификатора сессии. И в файл добавлена
одна функция для создания нового сессии и еще одна — для получения сессии 
по её идентификатору. Здорово!

Теперь, когда в интерфейс `Store` были добавлены ещё две функции, обратите 
внимание, что некоторые из наших unit тестов API выдают некоторые ошибки.
Это связано с тем, что эти две новые функции ещё не реализованы в имитации
`Store`.

Чтобы исправить их, мы должны выполнить

```shell
make mock
```

в терминале, чтобы повторно сгенерировать имитацию `Store`.

Только после этого ошибки пропадут.

Мы можем запустить

```shell
make test
```

чтобы убедиться, что все работает так, как ожидалось.

![](../images/part37/7.png)

И это действительно так! Все тесты успешно пройдены. Превосходно!

Хорошо, теперь перейдём к важной части, мы изменим API входа в систему, 
чтобы создать и вернуть refresh токен вместе с токеном доступа.

Итак, давайте откроем файл `api/user.go`. В функции `loginUser` после 
создания токена доступа здесь,

```go
	accessToken, err := server.tokenMaker.CreateToken(
		user.Username,
		server.config.AccessTokenDuration,
	)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, errorResponse(err))
        return
    }
```

мы вызовем `server.tokenMaker.CreateToken` еще раз, чтобы создать refresh 
токен. Мы по-прежнему передаем тот же `user.Username`, но срок действия 
refresh токена обновления должен быть другой. Его значение берется из 
переменной `config.RefreshTokenDuration`. Эта функция вернет строку 
`refreshToken` и ошибку. Если ошибка не равна `nil`, мы просто возвращаем 
внутреннюю ошибку сервера, как и для токена доступа выше.

```go
	refreshToken, err := server.tokenMaker.CreateToken(
		user.Username,
		server.config.RefreshTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
	}
```

Если ошибки не возникнет, мы вызовем `server.store.CreateSession`, чтобы 
вставить новую сессию в базу данных. Мы передаем контекст и объект 
`db.CreateSessionParam`. Я скопирую все необходимые поля этой структуры

```go
type CreateSessionParams struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	RefreshToken string    `json:"refresh_token"`
	UserAgent    string    `json:"user_agent"`
	ClientIp     string    `json:"client_ip"`
	IsBlocked    bool      `json:"is_blocked"`
	ExpiresAt    time.Time `json:"expires_at"`
}
```

и вставлю их внутрь этого объекта `CreateSessionParams`.

```go
	server.store.CreateSession(ctx, db.CreateSessionParams{
		ID           uuid.UUID `json:"id"`
		Username     string    `json:"username"`
		RefreshToken string    `json:"refresh_token"`
		UserAgent    string    `json:"user_agent"`
		ClientIp     string    `json:"client_ip"`
		IsBlocked    bool      `json:"is_blocked"`
		ExpiresAt    time.Time `json:"expires_at"`
	})
```

В этом объекте у нас должен быть идентификатор сессии и, как я уже 
говорил, мы будем использовать идентификатор refresh токена для этого 
поля. Но проблема в том, что функция `createToken` не возвращает полезную 
нагрузку токена. Он просто возвращает токен в виде зашифрованной строки, 
поэтому мы не знаем, чему равен идентификатор токена. Поэтому я добавлю 
полезную нагрузку токена в список возвращаемых значений этой функции.

```go
type Maker interface {
    CreateToken(username string, duration time.Duration) (string, *Payload, error)
	...
}
```

Теперь, когда мы изменили интерфейс `TokenMaker`, мы также должны 
изменить `PasetoMaker` и `JwtMaker`, потому что они реализуют 
интерфейс `TokenMaker`. Во-первых, для `PasetoMaker` я добавлю `Payload` 
в список возвращаемых значений этой функции.

```go
func (maker *PasetoMaker) CreateToken(username string, duration time.Duration) (string, *Payload, error) {
	...
}
```

Затем изменим этот оператор `return`, добавив в него `payload`.

```go
    return "", payload, err
```

Теперь для этого вызова функции `Encrypt` мы должны сохранить её 
результат в двух переменных: `token` и `err`. Затем мы возвращаем все три
значения токен, полезную нагрузку и ошибку в конце функции.

```go
func (maker *PasetoMaker) CreateToken(username string, duration time.Duration) (string, *Payload, error) {
	payload, err := NewPayload(username, duration)
	if err != nil {
		return "", payload, err
	}

	token, err := maker.paseto.Encrypt(maker.symmetricKey, payload, nil)
	return token, payload, err
}
```

Итак, реализация `PasetoMaker` исправлена.

Теперь я собираюсь сделать то же самое для `JwtMaker`. Обычно в реальном 
проекте мы просто выбираем одну из реализаций, но не обе. И я настоятельно 
рекомендую использовать Paseto вместо JWT. Если вы не знаете, почему, 
пожалуйста, вернитесь и прочитайте ещё раз лекцию 19 курса.

```go
func (maker *JWTMaker) CreateToken(username string, duration time.Duration) (string, *Payload, error) {
	payload, err := NewPayload(username, duration)
	if err != nil {
		return "", payload, err
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	token, err := jwtToken.SignedString([]byte(maker.secretKey))
	return token, payload, err
}
```

Хорошо, теперь `JwtMaker` тоже исправлен.

## Исправляем unit тесты

Но некоторые unit тесты всё ещё выдают ошибки, так что давайте исправим и 
их. В функции `TestPasetoMaker` мы должны добавить здесь переменную 
полезной нагрузки. 

```go
	token, payload, err := maker.CreateToken(util.RandomOwner(), -time.Minute)
```

и удалить это двоеточие, потому что `payload` здесь больше не новая 
переменная.

```go
	payload, err = maker.VerifyToken(token)
```

Затем я также потребую, чтобы она не была пустой.

```go
    require.NotEmpty(t, payload)
```

Вы можете добавить сюда больше проверок, чтобы сделать тест более надежным, 
если хотите. Давайте сделаем то же самое для `TestExpiredPasetoMaker`.

Затем таким же образом я исправлю остальные unit тесты для JWT.

```go
func TestJWTMaker(t *testing.T) {
	maker, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)

	username := util.RandomOwner()
	duration := time.Minute

	issuedAt := time.Now()
	expiredAt := issuedAt.Add(duration)

	token, payload, err := maker.CreateToken(username, duration)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotEmpty(t, payload)

	payload, err = maker.VerifyToken(token)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	require.NotZero(t, payload.ID)
	require.Equal(t, username, payload.Username)
	require.WithinDuration(t, issuedAt, payload.IssuedAt, time.Second)
	require.WithinDuration(t, expiredAt, payload.ExpiredAt, time.Second)
}

func TestExpiredJWTToken(t *testing.T) {
	maker, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)

	token, payload, err := maker.CreateToken(util.RandomOwner(), -time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotEmpty(t, payload)

	payload, err = maker.VerifyToken(token)
	require.Error(t, err)
	require.EqualError(t, err, ErrExpiredToken.Error())
	require.Nil(t, payload)
}
```

Всё готово!

Давайте повторно запустим все тесты из пакета `token`. Все они успешно 
пройдены! Превосходно!

Я закрою все эти файлы.

Но погодите, в файле `api/middleware_test.go` все ещё есть ошибки. Давайте
посмотрим что там не так!

Итак, здесь, в функции `addAuthorization`, мы также должны добавить 
объект `payload` и потребовать, чтобы он не был пустым.

```go
	token, payload, err := tokenMaker.CreateToken(username, duration)
    require.NoError(t, err)
    require.NotEmpty(t, payload)
```

Отлично, ошибки пропали.

Теперь вернемся к функции-обработчику `loginUser`!

Поскольку мы изменили интерфейс `TokenMaker`, в ней должно было возникнуть
много ошибок. Но почему-то они не появляются, может быть, из-за этого 
незавершенного фрагмента кода.

```go
	server.store.CreateSession(ctx, db.CreateSessionParams{
		ID           
		Username     string    `json:"username"`
		RefreshToken string    `json:"refresh_token"`
		UserAgent    string    `json:"user_agent"`
		ClientIp     string    `json:"client_ip"`
		IsBlocked    bool      `json:"is_blocked"`
		ExpiresAt    time.Time `json:"expires_at"`
	})
```

Итак, сначала я закомментирую все эти поля объекта `CreateSessionParams`.

Здорово, теперь ошибки появились. Давайте исправим их!

Здесь при создании токена доступа нам нужно добавить ещё одну переменную для 
хранения полезной нагрузки токена доступа.

```go
accessToken, accessPayload, err := server.tokenMaker.CreateToken(
    user.Username,
    server.config.AccessTokenDuration,
)
```

Затем давайте перейдём в самое начало файла.

Я добавлю ещё несколько полей в структуру `loginUserResponse`. Во-первых, 
поле, чтобы знать, когда истечет срок действия токена доступа. Оно будет
полезно для клиента, чтобы настроить периодическое обновление токена 
доступа. Я перемещу это поле вверх к `AccessToken`. Затем мы добавим еще два 
аналогичных поля для refresh токена и срока его действия.

```go
type loginUserResponse struct {
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  userResponse `json:"user"`
}
```

Хорошо, теперь вернемся к обработчику `loginUser`.

Давайте добавим переменную `refreshPayload` к этому вызову `CreateToken`.

```go
	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		server.config.RefreshTokenDuration,
	)
```

Хорошо, теперь я раскомментирую эти параметры.

```go
	server.store.CreateSession(ctx, db.CreateSessionParams{
		//ID, 
		//Username, string    `json:"username"`
		//RefreshToken string    `json:"refresh_token"`
		//UserAgent    string    `json:"user_agent"`
		//ClientIp     string    `json:"client_ip"`
		//IsBlocked    bool      `json:"is_blocked"`
		//ExpiresAt    time.Time `json:"expires_at"`
	})
```

Мы передадим в `ID` сессии значение `refreshPayload.ID`, `Username` 
возьмём из `user.Username`, `RefreshToken` - это просто значение, 
возвращаемое вызовом функции `CreateToken`, расположенной выше. Значения
для `UserAgent` и `ClientIP` временно будут равны пустой строке. Мы вернёмся
к ним немного позже и заполним их правильными значениями. Далее, значение
для поля `IsBlocked`, конечно, должно быть `false`. И, наконец, для
`ExpiresAt` возьмём его из `refreshPayload.ExpiredAt`. Этот вызов метода
`CreateSession` вернет объект `session` и ошибку.

```go
    session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
        ID: refreshPayload.ID,
        Username: user.Username,
        RefreshToken: refreshToken,
        UserAgent: "", // TODO: fill it
        ClientIp: "", // TODO: fill it
        IsBlocked: false,
        ExpiresAt: refreshPayload.ExpiredAt,
    })
```

Если ошибка не `nil`, то мы просто возвращаем внутреннюю ошибку сервера, 
как и в других случаях. В противном случае мы готовы отправить ответ 
клиенту.

```go
    if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
	}
    rsp := loginUserResponse{
        AccessToken: accessToken,
        User:        newUserResponse(user),
    }
    ctx.JSON(http.StatusOK, rsp)
```

Но я хочу добавить ещё одно поле в структуру ответа, которым будет 
идентификатор сессии (или по совместительству идентификатор refresh токена).

```go
type loginUserResponse struct {
	SessionID             uuid.UUID    `json:"session_id"`
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  userResponse `json:"user"`
}
```

Теперь для этого объекта `loginUserResponse` значение `SessionID` возьмём из
`session.ID`, `AccessTokenExpiresAt` - из `accessPayload.ExpiredAt`, 
`RefreshToken` - из `refreshToken`, и, наконец, ` RefreshTokenExpiresAt` - из
`refreshPayload.ExpiredAt`.

```go
	rsp := loginUserResponse{
		SessionID: session.ID,
		AccessToken: accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiredAt,
		RefreshToken: refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User:        newUserResponse(user),
	}
```

Вот и всё!

API для входа пользователя в систему с поддержкой refresh токена готов.

Давайте откроем файл `user_test.go` и попробуем запустить `TestLoginUserAPI`,
чтобы посмотреть, не возникло ли ошибок.

![](../images/part37/8.png)

Итак, ошибка в одном unit тесте. И причина в следующем: "there are no
expected calls of the method `CreateSession`" («неожиданный вызовов метода 
`CreateSession`»). Логично, так как мы ещё не добавили в заглушку этот 
новый метод.

Исправить это очень просто. В функцию `buildStubs` для случая OK я добавлю 
ещё один оператор `store.EXPECT()`, но на этот раз мы ожидаем, что 
функция `CreateSession` будет вызываться с любым контекстом и любыми 
параметрами. И вызов должен происходить равно один раз. Вот как показано
во фрагменте ниже!

```go
        {
			name: "OK",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(user, nil)
				store.EXPECT().
					CreateSession(gomock.Any(), gomock.Any()).
					Times(1)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
```

На этом всё! Конечно вы можете сделать тест более надёжным, заменив 
`gomock.Any()` более конкретным объектом. Вы можете прочитать лекцию 18,
если не знаете, как это сделать.

Хорошо, давайте перезапустим unit тесты.

![](../images/part37/9.png)

Теперь все тесты пройдены. Превосходно!

Давайте выполним

```shell
make server
```

в терминале, чтобы запустить сервер.

Затем я снова отправлю запрос к API входа в систему, используя Postman.

![](../images/part37/10.png)

На этот раз у нас гораздо больше полей в ответе от сервера.

В первую очередь `session_id`, `access_token` и срок его действия, 
затем refresh токен и его срок действия. Вы видите, что его срок
действия намного дольше, один день, по сравнению с 15 минутами токена 
доступа.

Теперь давайте посмотрим содержимое базы данных. В таблице `sessions` мы
видим одну запись. Подробную информацию о ней можно просмотреть в этом окне 
справа.

![](../images/part37/11.png)

Как видите, у него тот же идентификатор, что и у сессии, который мы 
получили в Postman. А с помощью поля `is_blocked` мы можем легко 
заблокировать эту сессию в случае утечки его refresh токена. Здесь мы также 
видим время его создания и истечения срока действия. Но столбцы 
`user_agent` и IP-адрес клиента все еще пусты. Теперь давайте напишем код, 
чтобы заполнить их правильными значениями.

На самом деле это довольно просто, потому что мы используем фреймворк 
Gin. Для `UserAgent` мы можем просто вызвать `ctx.Request.UserAgent()`.
Эта информация уже доступна внутри объекта `context` Gin. Аналогично, для 
IP-адреса клиента мы можем просто вызвать `ctx.ClientIP()`. Вот и всё! 
Очень легко, не так ли?

Вы также можете получить другие метаданные из контекста, например 
`Content-Type`, если хотите. Хорошо, теперь давайте перезапустим сервер и 
протестируем его работу!

![](../images/part37/12.png)

Я собираюсь повторно отправить этот запрос на вход в систему. Он выполнен
успешно! Итак, давайте просмотрим содержимое таблицы `sessions`.

![](../images/part37/13.png)

На этот раз была добавлена новая запись и посмотрите на поля `user_agent` 
и `client_ip`! Там появились данные с правильными значениями. Довольно круто, 
не так ли?

Итак, теперь, когда API для входа пользователя в систему с refresh токеном 
и записью сессии хорошо работает, единственное, чего все ещё не 
хватает, — это API для обновления токена доступа по истечении срока его 
действия.

Поскольку этот API очень похож на API входа пользователя, я скопирую 
содержимое этого обработчика `loginUser` и создам новый файл с именем 
`token.go` внутри пакета `api`. Затем давайте вставим содержимое, которое 
мы только что скопировали. Теперь мы должны изменить все названия в 
запросе, ответе и функции-обработчике с `loginUser` на `renewAccessToken`.

Как видите, в Visual Studio Code я использую комбинацию клавиш `Command + 
Shift + L`, чтобы отредактировать это значение в нескольких местах 
одновременно.

Хорошо, теперь для `renewAccessTokenRequest` единственное поле, которое 
нам понадобится, — это `refreshToken`.

```go
type renewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
```

Поэтому я собираюсь изменить это поле `Username` на `refreshToken` и
удалить поле `Password`.

По аналогии для `renewAccessTokenResponse`, мы просто вернём два поля:
`AccessToken` и срок его действия.

Хорошо, затем в функции `renewAccessToken` мы сначала преобразуем входной
JSON в объект запроса. Затем здесь нам нужно проверить, действителен ли
refresh токен. Поэтому я вызову `server.tokenMaker.VerifyToken` и 
передам `req.RefreshToken`. Результатом вызова этой функции является 
`refreshPayload` и ошибка.

```go
func (server *Server) renewAccessToken(ctx *gin.Context) {
	var req renewAccessTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	refreshPayload, err := server.tokenMaker.VerifyToken(req.RefreshToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}
	...
}
```

Если ошибка не равна `nil`, это означает, что refresh токен недействителен 
или срок его действия истек. В этом случае мы просто возвращаем клиенту
код состояния `Unauthorized`. В противном случае мы найдем сессию в базе 
данных, вызвав `server.store.GetSession`, и передадим идентификатор
refresh токена в качестве идентификатора сессии.

```go
func (server *Server) renewAccessToken(ctx *gin.Context) {
	...
	session, err := server.store.GetSession(ctx, refreshPayload.ID)
    if err != nil {
        if err == sql.ErrNoRows {
            ctx.JSON(http.StatusNotFound, errorResponse(err))
            return
        }
        ctx.JSON(http.StatusInternalServerError, errorResponse(err))
        return
    }
	...
}
```

Он вернет объект `session` или ошибку. Если ошибка не равна `nil`, мы
просто обрабатываем ее так же, как когда пользователь не найден в API 
входа пользователя в систему. Возвращаем 404 `Not found`, если сессии не
существует или внутреннюю ошибку сервера в других случаях.

Нам не нужно сравнивать пароль в этом `renewAccessToken` API, но вместо 
этого мы должны проверить, заблокирована ли сессия или нет.

```go
func (server *Server) renewAccessToken(ctx *gin.Context) {
	...
	if session.IsBlocked {
        err := fmt.Errorf("blocked session")
        ctx.JSON(http.StatusUnauthorized, errorResponse(err))
        return
    }
	...
}
```

Если она заблокирована, мы создаём новый объект ошибки с сообщением
"blocked session" («заблокированная сессия») и вернём его клиенту вместе 
с кодом состояния `401 Unauthorized`. Мы также должны проверить, совпадает 
ли `session.Username` с именем, хранящимся в refresh токене.

```go
func (server *Server) renewAccessToken(ctx *gin.Context) {
	...
	if session.Username != refreshPayload.Username {
        err := fmt.Errorf("incorrect session user")
        ctx.JSON(http.StatusUnauthorized, errorResponse(err))
        return
    }
	...
}
```

Если не совпадают, то мы также возвращаем клиенту сообщение об ошибке
"incorrect session user" («неправильный пользователь сессии»). Давайте 
также проверим, совпадает ли `session.RefreshToken` с `req.RefershToken` 
или нет.

```go
func (server *Server) renewAccessToken(ctx *gin.Context) {
	...
	if session.RefreshToken != req.RefreshToken {
        err := fmt.Errorf("mismatched session token")
        ctx.JSON(http.StatusUnauthorized, errorResponse(err))
        return
    }
	...
}
```

Обычно они должны быть одинаковыми. Но на всякий случай, если это не так, 
мы возвращаем сообщение об ошибке "mismatched session token" 
(«несовпадающий токен сессии»).

Наконец, хотя время истечения срока действия токена проверялось в 
функции `VerifyToken`, мы можем проверить его здесь ещё раз. В 
некоторых редких случаях мы можем захотеть принудительно изменить его срок
действия, чтобы он истёк раньше ранее назначенного времени.

```go
func (server *Server) renewAccessToken(ctx *gin.Context) {
	...
	if time.Now().After(session.ExpiresAt) {
        err := fmt.Errorf("expired session")
        ctx.JSON(http.StatusUnauthorized, errorResponse(err))
        return
    }
	...
}
```

Поэтому, если текущий момент времени находится в будущем относительно 
`session.ExpiresAt`, мы возвращаем ошибку, говорящую о том, что сеанс истёк.
И я думаю, что этого должно быть достаточно для проверки. Если срок действия
токена не истёк, то мы можем создать новый токен доступа.

```go
func (server *Server) renewAccessToken(ctx *gin.Context) {
	...
	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
        refreshPayload.Username,
        server.config.AccessTokenDuration,
    )
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, errorResponse(err))
        return
    }
	...
}
```

На этот раз мы можем получить имя пользователя из объекта `RefreshPayload`. 
Вот и все! Часть, касающаяся обработки ошибок, остаётся прежней. И мы можем
удалить оставшийся код, который создает новый refresh токен обновления и 
сессию.

Наконец, мы удалим все лишние поля в объекте ответа от сервера, оставим два
поля: `accessToken` и `accessTokenExpiresAt`.

```go
func (server *Server) renewAccessToken(ctx *gin.Context) {
    ...
	rsp := renewAccessTokenResponse{
        AccessToken:          accessToken,
        AccessTokenExpiresAt: accessPayload.ExpiredAt,
    }
    ctx.JSON(http.StatusOK, rsp)
}
```

Функция-обработчик `renewAccessToken` готова. Далее мы должны 
зарегистрировать этот обработчик как новый маршрут в файле `server.go`.

Я добавлю сюда новый `router.POST`, у которого путь будет равен 
`/token/renew_access`, а обработчик – `server.renewAccessToken`.

```go
func (server *Server) setupRouter() {
	router := gin.Default()

	router.POST("/users", server.createUser)
	router.POST("/users/login", server.loginUser)
	router.POST("/token/renew_access", server.renewAccessToken)
	...
}
```

И мы закончили!

Давайте откроем терминал и перезапустим сервер!

```shell
make server
```

## Посылаем запросы, используя Postman

Затем откройте Postman, я создам новый запрос. Измените метод на POST, 
URL-адрес запроса должен быть `localhost:8080/tokens/renew_access`.
Затем во вкладке `Body` давайте выберем `Raw` и JSON формат. Единственное 
поле, которое нам понадобится, это `refresh_token`. Я скопирую его значение 
из ответа API входа в систему. Итак, теперь давайте отправим запрос!

![](../images/part37/14.png)

Ура! Он успешно выполнен!

Мы получили новый токен доступа, который будет действовать следующие 15 минут!
Превосходно!

Хорошо, теперь я сохраню этот новый запрос в моей коллекции `Simple Bank` 
приложения Postman.

![](../images/part37/15.png)

И перед тем как мы закончим, я протестирую один граничный случай, когда 
мы используем токен доступа вместо refresh токена в API.

Посмотрим, сможем ли мы получить новый токен доступа или нет.

Итак, я отправлю запрос!

![](../images/part37/16.png)

О, срок действия токена доступа уже истек. Поэтому давайте снова 
войдём в систему. Затем скопируйте новый токен доступа, вставьте его в API
обновления токена доступа и отправьте запрос еще раз!

![](../images/part37/17.png)

На этот раз мы получили ошибку: "no rows in result set" («в результате 
выполнения запрос не было получено каких-либо строк»), потому что в базе 
данных нет записи о сессии с идентификатором токена доступа, который мы 
использовали.

Хорошо, а если попробовать использовать неправильный refresh токен, например,
"abc"? Мы получим ошибку "token is invalid" («неправильный токен»). 

![](../images/part37/18.png)

И в этом заключается очень приятный бонус при использовании PASETO в 
качестве refresh токена, потому что в подобных случаях серверу вообще не 
нужно обращаться к базе данных, чтобы узнать, что refresh токен неправильный.

Это определенно уменьшит нагрузку на базу данных!

И на этом мы закончим сегодняшнюю лекцию. Я надеюсь, что она была 
интересной и изученный материал каким-то образом пригодится вам.

Большое спасибо за время, потраченное на чтения и до встречи на следующей
лекции!