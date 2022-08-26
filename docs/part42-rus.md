# Реализуем gRPC API для создания и входа пользователей в систему на Go

[Оригинал](https://www.youtube.com/watch?v=7xiWqyZW9lE)

Привет, ребята, рад вас снова видеть на мастер-классе по бэкенду! На
[последней лекции](part41-rus.md) мы узнали, как запустить gRPC сервер, 
используя код, сгенерированный `protoc`. Мы также попытались вызвать RPC
на сервере с помощью клиента Evans. Однако все RPC для создания и 
входа пользователя в систему по-прежнему возвращают ошибки, потому что 
в данный момент они ещё не реализованы.

```shell
call CreateUser
username (TYPE_STRING) => quang
full_name (TYPE_STRING) => Quang
email (TYPE_STRING) => quang@gmail.com
password (TYPE_STRING) => secret
command call: rpc error: code = Unimplemented desc = method CreateUser not implemented
```

Поэтому сегодня давайте научимся их реализовывать!

## Реализуем API для создания пользователей

Если мы перейдём к реализации `UnimplementedSimpleBankServer`, то
увидим сгенерированный код gRPC сервиса, где также находится
интерфейс SimpleBankServer, который нам нужно реализовать. И здесь для
`UnimplementedSimpleBankServer` уже существует базовая реализация двух
методов, которым должны быть у интерфейса: `CreateUser` и `LoginUser`.
Сейчас нам нужно реализовать их в нашей собственной структуре `Server`.

Итак, сначала я скопирую этот метод `CreateUser`.

```go
func (UnimplementedSimpleBankServer) CreateUser(context.Context, *CreateUserRequest) (*CreateUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateUser not implemented")
}
```

Затем давайте создадим новый файл: `rpc_create_user.go` внутри 
пакета `gapi` и вставим в него скопированный код. Чтобы добавить 
этот метод к нашему серверу, мы должны изменить приёмник на
наш объект `Server`. Структура `Server` уже определена в файле 
`server.go` на [предыдущей лекции](part41-rus.md).

Хорошо, теперь давайте добавим имена к параметрам контекста и
запроса. `CreateUserRequest` и `CreateUserResponse` находятся в 
пакете `pb`, который сгенерировал для нас `protoc`. Как только мы 
сохраним файл, все необходимые пакеты будут автоматически импортированы, 
и в файле больше не будет ошибок.

```go
func (server *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateUser not implemented")
}
```

Теперь нам нужно написать реализацию этого метода. Она будет очень 
похожа на ту, которую мы создали ранее для Gin HTTP API сервера. Итак,
давайте откроем файл `user.go` внутри пакета `api`.

Там мы увидим метод-обработчик `createUser`

```go
func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	arg := db.CreateUserParams{
		Username:       req.Username,
		HashedPassword: hashedPassword,
		FullName:       req.FullName,
		Email:          req.Email,
	}

	user, err := server.store.CreateUser(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				ctx.JSON(http.StatusForbidden, errorResponse(err))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := newUserResponse(user)
	ctx.JSON(http.StatusOK, rsp)
}
```

Как видите, для Gin, мы должны сами связать входные параметры с 
объектом запроса. Но для gRPC нам не нужно этого делать, потому что 
об этом уже позаботился фреймворк.

Поэтому давайте проигнорируем код для преобразования входных данных
и просто скопируем основную часть этой функции, где мы взаимодействуем 
с БД для создания нового пользователя. Вставьте его в наш RPC метод.

```go
func (server *Server) CreateUser(context.Context, *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	arg := db.CreateUserParams{
		Username:       req.Username,
		HashedPassword: hashedPassword,
		FullName:       req.FullName,
		Email:          req.Email,
	}

	user, err := server.store.CreateUser(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				ctx.JSON(http.StatusForbidden, errorResponse(err))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	
	return nil, status.Errorf(codes.Unimplemented, "method CreateUser not implemented")
}
```

Хорошо, теперь давайте посмотрим, как мы можем изменить код, чтобы 
он работал с gRPC. Во-первых, как видите, пароль, введённый пользователем,
мы можем легко получить из `CreateUserRequest`, поскольку gRPC уже связал 
все входные данные с этом объектом за нас. Тут мы можем получить 
значение непосредственно из поля `Password` структуры,

```go
hashedPassword, err := util.HashPassword(req.Password)
```

или, что лучше, используя геттер-функцию: `GetPassword()`, поскольку
она обеспечит дополнительную проверку, на тот случай, если объект 
запроса равен `nil`.

```go
func (x *CreateUserRequest) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}
```

Хорошо, вернемся к нашему коду. Здесь мы хешируем пароль, потому что не 
хотим хранить его текстовое значение в базе данных. Если произойдет 
ошибка, мы должны будем вернуть клиенту внутреннюю ошибку.

```go
hashedPassword, err := util.HashPassword(req.GetPassword())
if err != nil {
    ctx.JSON(http.StatusInternalServerError, errorResponse(err))
    return
}
```

Как мы можем сделать это в gRPC?

Что ж, если вы посмотрите на оператор return, сгенерированный для нас
`protoc`, то не увидите ничего сложного.

```go
return nil, status.Errorf(codes.Unimplemented, "method CreateUser not implemented")
```

Все, что нам нужно сделать, это создать ошибку с помощью 
функции `status.Errorf()`. `status` на самом деле является 
подпакетом gRPC. Итак, эта функция `Errorf` позволяет нам 
передавать два параметра. Первый параметр — это код состояния, 
который в нашем случае должен быть `codes.Internal`. Опять же,
`codes` — это ещё один подпакет gRPC, в котором определены все 
коды состояния gRPC. А второй параметр — это сообщение об ошибке, 
скажем: "failed to hash password" («не удалось захешировать пароль»).

```go
hashedPassword, err := util.HashPassword(req.GetPassword())
if err != nil {
    return nil, status.Errorf(codes.Internal, "failed to hash password")
}
```

Мы также можем встроить исходную ошибку в это сообщение, что была
понятна причина, из-за которой произошла ошибка.

```go
hashedPassword, err := util.HashPassword(req.GetPassword())
if err != nil {
    return nil, status.Errorf(codes.Internal, "failed to hash password: %s", err)
}
```

После этого шага мы вызовем функцию в хранилище БД для создания нового 
пользователя. Вот входной аргумент, который нам нужно передать в 
функцию. Его можно не менять.

```go
arg := db.CreateUserParams{
    Username:       req.Username,
    HashedPassword: hashedPassword,
    FullName:       req.FullName,
    Email:          req.Email,
}
```

Мы можем обращаться к полям непосредственно из объекта `request`.
Или, что правильнее, вызвав геттер-функции для получения значений.
Таким образом, мы передаём `request.GetUsername()`, хешированный 
пароль – это значение, которое мы только что вычислили. Затем 
`request.GetFullName()` и, наконец, `request.GetEmail()`.
Теперь мы вызываем `server.store.CreateUser()` и передаем входной 
аргумент.

```go
arg := db.CreateUserParams{
    Username:       req.GetUsername(),
    HashedPassword: hashedPassword,
    FullName:       req.GetFullName(),
    Email:          req.GetEmail(),
}

user, err := server.store.CreateUser(ctx, arg)
```

Если ошибка не равна `nil`, то нужно проверить два случая. Если ошибка
это нарушение уникальности столбца, то это означает, что пользователь 
с таким именем уже существует. Поэтому здесь нам придется вернуть 
другой код состояния ошибки. Я скопирую оператор `return` выше,
затем изменю код состояния на `AlreadyExists`, и сообщение об
ошибке на "username already exists" («такое имя пользователя 
уже существует»). В других случаях мы точно не знаем, что 
вызвало ошибку в базе данных, поэтому просто возвращаем внутреннюю 
ошибку с сообщением "failed to create user" («не удалось создать 
пользователя»).

```go
if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				return nil, status.Errorf(codes.AlreadyExists, "username already exists: %s", err)
			}
		}
        return nil, status.Errorf(codes.Internal, "failed to create user: %s", err)
	}
```

Наконец, если ошибок не возникает, именно здесь мы должны создать
объект ответа и вернуть его gRPC клиенту. Итак, давайте создадим 
здесь объект `CreateUserResponse` и вернём этот объект с ошибкой
`nil` в конце функции.

```go
rsp := &pb.CreateUserResponse{}
return rsp, nil
```

Итак, что нам нужно передать в ответе? Все необходимые поля уже 
определены в сгенерированной структуре. В нашем случае в структуре
существует только одно поле `User`. И тип этого поля — `pb.User`, который 
является еще одной структурой, созданной `protoc` из определения 
`protobuf` в лекции 40.

```go
rsp := &pb.CreateUserResponse{
    User: &pb.User{

    },
}
```

Мы не можем просто вернуть здесь объект `db.User`, поскольку у него 
другой тип. И на самом деле мы не должны смешивать слой БД со 
структурой API слоя. Их лучше отделить друг от друга, потому что 
иногда мы не хотим возвращать клиенту каждое поле в БД. Например, 
хешированный пароль — это конфиденциальная информация, которую мы 
хотим скрыть. Теперь нам нужна функция для преобразования 
`db.User` в `pb.User`.

Я создам новый файл с названием `converter.go` внутри пакета `gapi`.
И в этом файле давайте добавим функцию: `convertUser()`, которая будет 
принимать объект `db.User` в качестве входных данных и возвращать 
объект `pb.User` в качестве результата.

```go
func convertUser(user db.User) pb.User {
	
}
```

Эту функцию мы можем просто вызвать в `CreateUser` RPC, чтобы 
преобразовать внутренний объект `db.User` в требуемый тип для 
gRPC ответа от сервера.

```go
rsp := &pb.CreateUserResponse{
    User: convertUser(user),
}
```

Теперь давайте реализуем функцию `convertUser()`. На самом деле 
возвращаемым типом должен быть указателем на `pb.User`, потому 
что именно он нужен сгенерированной структуре ответа от сервера.
Преобразование реализовать в принципе довольно просто. Нам 
нужно преобразовать каждое поле структуры. Во-первых,
`Username` должно быть `user.Username`, `Fullname` — `user.FullName`,
`Email` — `user.Email`, `PasswordChangedAt` — `user.PasswordChangedAt`,
но эти поля немного отличаются, потому что тип метки времени в
`protobuf` не такой же, как тип `time` в Golang. Итак, здесь мы 
должны использовать функцию `timestamppb.New()` для преобразования 
его значения. Аналогично для поля `user.CreatedAt`. И на этом 
по сути всё!

```go
func convertUser(user db.User) *pb.User {
	return &pb.User{
		Username: user.Username,
		FullName: user.FullName,
		Email: user.Email,
		PasswordChangedAt: timestamppb.New(user.PasswordChangedAt),
		CreatedAt: timestamppb.New(user.CreatedAt),
	}
}
```

Функция `convertUser()` готова.

Наш RPC `CreateUser` теперь также готов обработать запрос.

Давайте выполним команду

```shell
make server
```

в терминале, чтобы запустить gRPC сервер.

Затем в другой вкладке, давайте выполним

```shell
make evans
```

чтобы открыть консоль клиента Evans.

Затем давайте вызовем RPC `CreateUser`. Введите имя пользователя, имя
и фамилию, адрес электронной почты и пароль.

```shell
call CreateUser
username (TYPE_STRING) => quang1
full_name (TYPE_STRING) => Quang
email (TYPE_STRING) => quang@gmail.com
password (TYPE_STRING) => secret
{
  "user": {
    "username": "quang1",
    "fullName": "Quang",
    "email": "quang@gmail.com",
    "passwordChangedAt": "0001-01-01T00:00:00Z",
    "createdAt": "2022-04-10T10:10:37.688043Z"
  }
}
```

И вуаля, новый пользователь создан, мы получили ответ от сервера 
с информацией о созданном пользователе.

Здорово, не так ли?

Вот как мы реализуем `CreateUser` API в gRPC.

## Реализуем API для входа пользователя в систему

А что насчёт API для входа пользователя в систему?

Ну его реализация должны быть очень похожа.

Сейчас самое время прекратить чтение лекции и попробовать 
реализовать его самостоятельно.

Чуть ниже мы покажем как сделать это. Итак, удалось ли вам успешно его 
реализовать?

Первое, что нам нужно сделать, это перейти к файлу, где 
находится структура `UnimplementedSimpleBankServer`.

В этом файле `service_simple_bank_grpc.pb.go`, давайте скопируем
метод `LoginUser()`.

```go
func (UnimplementedSimpleBankServer) LoginUser(context.Context, *LoginUserRequest) (*LoginUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method LoginUser not implemented")
}
```

Нам нужно будет добавить этот метод к нашей реализации Simple Bank
сервера.

Итак, далее я собираюсь создать новый файл с названием
`rpc_login_user.go` внутри папки `gapi`. Чтобы код оставался 
чистым и хорошо структурированным, я рекомендую хранить каждый RPC 
в отдельном файле.

Хорошо, теперь давайте вставим код, который мы только что 
скопировали.

Измените приёмник этой функции на наш объект `server`. Задайте
имена переменных для контекста и запроса. И добавьте пакет `pb`
к `LoginUserRequest` и `LoginUserResponse`.

Затем давайте откроем HTTP-обработчик API входа в систему, который 
мы ранее реализовали для Gin сервера. Игнорируем привязку запроса и 
скопируем оставшийся код обработки вплоть до той части, где создается 
ответ от сервера. Затем вставьте этот код в новую RPC функцию входа 
пользователя в систему.

Итак, первым шагом при обработке запроса на вход пользователя 
в систему является извлечение пользователя из базы данных. 

Поэтому здесь мы вызываем `server.store.GetUser()`, чтобы найти 
пользователя с таким `username`.

```go
func (server *Server) LoginUser(ctx context.Context, req *pb.LoginUserRequest) (*pb.LoginUserResponse, error) {
    user, err := server.store.GetUser(ctx, req.GetUsername())
    ...
}
```

Если пользователь не найден, мы вернем ответ `nil` вместе с новой 
ошибкой, созданной функцией `status.Errorf()`. В этом случае в 
ошибке будет храниться код состояния `NotFound` и сообщение о том, 
что "user not found" («пользователь не найден»). Я удалю этот JSON
ответ Gin обработчика. В случае возникновения непредвиденной ошибки 
мы возвращаем другую ошибку с кодом состояния `Internal` и сообщением
"failed to find user" («не удалось найти пользователя»).

```go
if err != nil {
    if err == sql.ErrNoRows {
        return nil, status.Errorf(codes.NotFound, "user not found")
    }
    return nil, status.Errorf(codes.NotFound, "failed to find user")
}
```

Если пользователь существует и был успешно извлечен из БД, то далее 
мы должны проверить, совпадает ли введенный пароль с паролем в нашей 
БД или нет. Обратите внимание, что мы сохраняем только хешированное 
значение пароля, поэтому здесь мы вызываем функцию 
`util.CheckPassword()`, чтобы сравнить открытый текстовый пароль с 
сохраненным хешированным значением. Если ошибка не равна `nil`, 
это означает, что предоставленный пароль неверен.

В этом случае мы возвращаем клиенту ответ `nil` и ошибку с кодом 
состояния `NotFound`.

```go
err = util.CheckPassword(req.Password, user.HashedPassword)
if err != nil {
    return nil, status.Errorf(codes.NotFound, "incorrect password")
}
```

Хорошо, теперь, если пароль совпадает, мы вызываем функцию 
`tokenMaker.CreateToken()` для создания нового токена доступа.
Если этот вызов выдаёт не `nil` ошибку, то мы возвращаем
внутреннюю ошибку клиенту с сообщением о том, что "failed to create 
access token" («не удалось создать токен доступа»).

```go
accessToken, accessPayload, err := server.tokenMaker.CreateToken(
    user.Username,
    server.config.AccessTokenDuration,
)
if err != nil {
    return nil, status.Errorf(codes.Internal, "failed to create access token")
}
```

По аналогии мы делаем то же самое, если вызов для создания refresh токена 
завершается неудачно.

```go
refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(
    user.Username,
    server.config.RefreshTokenDuration,
)
if err != nil {
    return nil, status.Errorf(codes.Internal, "failed to create refresh token")
}
```

Вы можете вернуться и перечитать лекцию 37, если не знаете зачем нужен
refresh токен.

Итак, теперь последний шаг — создать новую запись для сессии в БД.

```go
session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
    ID:           refreshPayload.ID,
    Username:     user.Username,
    RefreshToken: refreshToken,
    UserAgent:    ctx.Request.UserAgent(),
    ClientIp:     ctx.ClientIP(),
    IsBlocked:    false,
    ExpiresAt:    refreshPayload.ExpiredAt,
})
```

Здесь видно, что большинство параметров уже доступны, за исключением
агента пользователя и IP-адреса клиента. Мы узнаем, как получить их 
из метаданных gRPC контекста, в другой лекции. А пока давайте зададим
в качестве значений для этих полей пустые строки. 

```go
session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
    ID:           refreshPayload.ID,
    Username:     user.Username,
    RefreshToken: refreshToken,
    UserAgent:    "",
    ClientIp:     "",
    IsBlocked:    false,
    ExpiresAt:    refreshPayload.ExpiredAt,
})
```

Опять же, если вызов завершается с ошибкой, мы внутреннюю ошибку с
сообщением "failed to create session" («не удалось создать сессию»).

```go
session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
    ID:           refreshPayload.ID,
    Username:     user.Username,
    RefreshToken: refreshToken,
    UserAgent:    "",
    ClientIp:     "",
    IsBlocked:    false,
    ExpiresAt:    refreshPayload.ExpiredAt,
})
if err != nil {
    return nil, status.Errorf(codes.Internal, "failed to create session")
}
```

Если всё прошло без ошибок, мы создаём объект `pb.LoginUserResponse`
и вернем его вместе с ошибкой `nil`. Теперь первым полем этого объекта 
ответа от сервера является `User`, поэтому мы просто используем 
функцию `convertUser()` для преобразования `db.User` в `pb.User`. 
Идентификатор сессии равен `session.ID.String()`, `AccessToken` —
`accessToken` и по аналогии присваиваем значение для refresh токена.
Для `AccessTokenExpiresAt` используется `timestampb.New()` и мы можем
получить его значение из `accessPayload`. Точно так же значение 
`RefreshTokenExpiresAt` извлекаем из `refreshPayload`. Вот и всё!

```go
rsp := &pb.LoginUserResponse{
    User: convertUser(user),
    SessionId: session.ID.String(),
    AccessToken: accessToken,
    RefreshToken: refreshToken,
    AccessTokenExpiresAt: timestamppb.New(accessPayload.ExpiredAt),
    RefreshTokenExpiresAt: timestamppb.New(refreshPayload.ExpiredAt),
}
return rsp, nil
```

Мы реализовали `LoginUser` RPC. Давайте откроем терминал и 
перезапустим gRPC сервер. Повторно подключитесь к нему с помощью
клиента Evans. На этот раз мы будем вызывать `LoginUser`
RPC. Введите имя пользователя и пароль, которые мы создали ранее.

```shell
call LoginUser
username (TYPE_STRING) => quang1
password (TYPE_STRING) => secret
{
  "user": {
    "username": "quang1",
    "fullName": "Quang",
    "email": "guang@gmail.com",
    "passwordChangedAt": "0001-01-01T00:00:00Z",
    "createdAt": "2022-04-10T10:10:37.688043Z"
  },
  "sessionId": "e71d4c75-827e-4a4c-84f3-4c40b3554439",
  "accessToken": "v2.local.uy2QOalUcpdYWXdXQrYMJCSiohbf6FX_sa6eaa6_BtWGRAfX9NYbAGSzV-0AcVOd8YxOS-Fi3jxZrIjPoLIvu3bj9hDqaL2apx9vv4noGmDxfZUTmut6YRPrQMiR2I2FRVP283ZQL1ocmDjAkI2QhXnN0IZaiWFlypmGE_xq37zThDcIjCU6WX_4lrekMNrFkRKOdky2EVyPwCsRV1J_MDeq00ePEjaA-gwx7kkwy3rmfO3ir3R2eR36HNGYKaUUQH77ZuxtTkj7YZZjclk.bnVsbA",
  "refreshToken": "v2.local.ztop-Ppht-YlqJ9myb_vPTCzIWDD2LiR3n7bVIEsKPoxJ_71379knOo7LsGAxg6cQskbn1fCzrBHAbXE-hg0-YAHfNr1Y69lRc12pKVgq8nh-odcvIpwDha03vdtkA4_DY_eeqkUvBRL928No_HnH32OfnyrBOX_yo6OZmPnOmw4HjyFSFh1U-6sBZBmgELdcBwaxo6Pci6sdYrCPOU4Qs0vh5aiJ7IfqmraGN-Yx5qVnuGyi0n8kHInL444HUSxKWUZKqf4T-kkK4B6E10.bnVsbA",
  "accessTokenExpiresAt": "2022-04-10T10:39:32.035910Z",
  "refreshTokenExpiresAt": "2022-04-11T10:24:32.036171Z"
}
```

И вуаля, мы получили успешный ответ со всей информацией о пользователе, 
а также токеном доступа и refresh токеном.

Потрясающе!

Теперь попробуем войти под несуществующим пользователем.

```shell
call LoginUser
username (TYPE_STRING) => quang2
password (TYPE_STRING) => secret
command call: rpc error: code = NotFound desc = user not found
```

На этот раз мы получили ошибку с кодом состояния `NotFound`
и сообщением: "user not found" («пользователь не найден»).

А если попытаться войти с неверным паролем?

```shell
call LoginUser
username (TYPE_STRING) => quang1
password (TYPE_STRING) => wrong
command call: rpc error: code = NotFound desc = incorrect password
```

В этом случае мы всё равно получили ошибку `NotFound`, но с 
сообщением "incorrect password" («неверный пароль»).

Таким образом, всё работает так как ожидалось. И на этом мы закончим
сегодняшнюю лекцию.

Мы успешно реализовали 2 унарных gRPC API для создания
и входа пользователя в систему.

Я надеюсь, что лекция была интересной и её материал пригодится вам.

На следующей лекции мы узнаем как использовать gRPC шлюз, чтобы
можно было перенаправлять gRPC и HTTP запросы на соответствующие
API.

А пока желаю вам получать удовольствие от обучения и до встречи на
следующей лекции!