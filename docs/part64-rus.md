# Как протестировать gRPC API, требующий аутентификации

[Оригинал](https://www.youtube.com/watch?v=MI7ucbAlZPM)

Всем привет, добро пожаловать на мастер-класс по бэкенду! На прошлой лекции
мы написали несколько unit тестов для CreateUser RPC. Однако этот API не 
требует никакой аутентификации, любой может вызвать его без входа в 
систему и получения токена доступа. Поэтому сегодня я хочу показать вам,
как написать unit тесты для UpdateUser RPC, который содержит этот
метод `authorizeUser()`

```go
authPayload, err := server.authorizeUser(ctx)
```

требующий предоставления действующего токена доступа. Определение этого 
API дано в файле `rpc_update_user.proto`.

```proto
message UpdateUserRequest {
  string username = 1;
  optional string full_name = 2;
  optional string email = 3;
  optional string password = 4;
}
```

Его запрос содержит `username` и некоторые необязательные поля, такие как
`full_name`, `email` и `password`, которые можно обновлять. Итак, давайте 
начнём писать тесты!

## Пишем unit тест для gRPC UpdateUser API

Я создам новый файл с названием `rpc_update_user_test.go` внутри пакета 
`gapi`. Затем давайте скопируем модульные тесты, которые мы написали на 
прошлой лекции, вставим их в новый файл и изменим название функции 
на `TestUpdateUserAPI`. Хорошо, теперь давайте изменим тип запроса на 
`pb.UpdateUserRequest`. Для этого API у нас нет асинхронной задачи, поэтому
давайте удалим фиктивный распределитель задач из метода `buildStubs`.
Затем давайте изменим тип ответа на `pb.UpdateUserResponse`.

```go
func TestUpdateUserAPI(t *testing.T) {
	user, password := randomUser(t)

	testCases := []struct {
		name          string
		req           *pb.UpdateUserRequest
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, res *pb.UpdateUserResponse, err error)
	}{
        ...
    }
    ...
}
```

Теперь давайте обновим параметры случая `OK`. Во-первых, тип запроса 
должен быть `UpdateUserRequest`. А поскольку поля `FullName`, `Email` и 
`Password` являются необязательными, их типы являются указателями на 
строку. Я определю новую переменную для нового имени пользователя и 
воспользуюсь функцией `util.RandomOwner()`, чтобы сгенерировать для неё 
случайное значение. Аналогично, новый адрес электронной почты генерируется с 
помощью `util.RandomEmail()`.

```go
newName := util.RandomOwner()
newEmail := util.RandomEmail()
```


Чтобы не усложнять тест я пока не буду изменять пароль. Поэтому давайте 
удалим это поле из запроса. Затем задайте для поля `FullName` значение
`newName`. Как видите, VSCode автоматически добавляет в начало переменной 
символ амперсанда, так как нам нужен указатель на это поле. Точно так же 
для поля `Email`, мы зададим значение `newEmail`.

```go
name: "OK",
req: &pb.UpdateUserRequest{
    Username: user.Username,
    FullName: &newName,
    Email:    &newEmail,
},
```

Хорошо, теперь давайте удалим фиктивный распределитель задач из
функции `buildStubs`. В этой функции нам придется имитировать запрос 
`UpdateUser`, поэтому давайте изменим здесь `db.CreateUserTxParams` на 
`db.UpdateUserParams`. Единственное обязательное поле в этой структуре —
`Username`. Все остальные поля являются необязательными, как видно из их 
типов, допускающих значение NULL. В этом тестовом случае мы хотим обновить 
только `FullName` и `Email`, поэтому давайте установим для поля `Username` 
значение `user.Username`, для `FullName` — `sql.NullString` со 
значением `newName` для поля `String` и `true` — для `Valid`. По 
аналогии поступаем с полем `Email`, приравниваем его к структуре 
`sql.NullString`, у которой поле `String` имеет значение `newEmail`, а 
`Valid` — `true`.

```go
arg := db.UpdateUserParams{
    Username: user.Username,
    FullName: sql.NullString{
        String: newName,
        Valid:  true,
    },
    Email: sql.NullString{
        String: newEmail,
        Valid:  true,
    },
}
```

Теперь нам следует `EXPECT` (`ОЖИДАТЬ`) вызов функции `UpdateUser` с только 
что объявленными нами аргументами.

```go
store.EXPECT().
    UpdateUser(gomock.Any(), gomock.Eq(arg)).
```

Здесь я использую сопоставитель `gomock.Eq` для сравнения входного аргумента.
Для успешного тестового случая этот метод должен вызываться ровно один 
раз и возвращать пользователя с обновлёнными значениями полей вместе с 
объектом ошибки. Поэтому мы определим здесь переменную `updatedUser` и
вернем её вместе с ошибкой равной `nil`. Пользователь с обновлёнными 
значениями полей будет объектом `db.User`, где все поля останутся такими 
же, как и у старого пользователя, за исключением `FullName` и `Email`, 
значения которых должны быть равны `newName` и `newEmail`.

```go
updatedUser := db.User{
    Username:          user.Username,
    HashedPassword:    user.HashedPassword,
    FullName:          newName,
    Email:             newEmail,
    PasswordChangedAt: user.PasswordChangedAt,
    CreatedAt:         user.CreatedAt,
    IsEmailVerified:   user.IsEmailVerified,
}
store.EXPECT().
    UpdateUser(gomock.Any(), gomock.Eq(arg)).
    Times(1).
    Return(updatedUser, nil)
```

Хорошо, теперь мы можем удалить команду `EXPECT` для фиктивного 
распределителя задач. И перейти к функции `checkResponse`.

```go
buildStubs: func(store *mockdb.MockStore) {
    arg := db.UpdateUserParams{
        Username: user.Username,
        FullName: sql.NullString{
            String: newName,
            Valid:  true,
        },
        Email: sql.NullString{
            String: newEmail,
            Valid:  true,
        },
    }
    updatedUser := db.User{
        Username:          user.Username,
        HashedPassword:    user.HashedPassword,
        FullName:          newName,
        Email:             newEmail,
        PasswordChangedAt: user.PasswordChangedAt,
        CreatedAt:         user.CreatedAt,
        IsEmailVerified:   user.IsEmailVerified,
    }
    store.EXPECT().
        UpdateUser(gomock.Any(), gomock.Eq(arg)).
        Times(1).
        Return(updatedUser, nil)
},
```

В ходе этого теста она получит `UpdateUserResponse` и объект ошибки в качестве 
результата работы API. Поскольку это успешный случай, мы ожидаем отсутствия 
ошибок и результат не должен быть равен `nil`. Мы извлечём пользователя с 
обновлёнными значениями полей из ответа и проверим, что имя пользователя 
осталось прежним, а полное имя и фамилия были изменены на `newName` и 
адрес электронной почты — на `newEmail`. Вот и всё!

```go
checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
    require.NoError(t, err)
    require.NotNil(t, res)
    updatedUser := res.GetUser()
    require.Equal(t, user.Username, updatedUser.Username)
    require.Equal(t, newName, updatedUser.FullName)
    require.Equal(t, newEmail, updatedUser.Email)
},
```

Мы внесли необходимые изменения для случая `OK`. Пока что я удалю все
остальные тестовые случаи, чтобы мы могли сосредоточиться на нём и в 
первую очередь сделать так, чтобы он был успешно пройден.

Хорошо, давайте модифицируем содержимое функции `Run`!

Мы можем избавиться от контроллера и распределителя задач. Удалите 
распределитель задач при создании заглушек и просто используйте `nil` в 
качестве распределителя задач при создании нового тестового сервера. Здесь 
нам следует вызывать `server.UpdateUser` вместо `CreateUser`. И это всё что 
нам нужно сделать!

```go
t.Run(tc.name, func(t *testing.T) {
    storeCtrl := gomock.NewController(t)
    defer storeCtrl.Finish()
    store := mockdb.NewMockStore(storeCtrl)

    tc.buildStubs(store)
    server := newTestServer(t, store, nil)

    res, err := server.UpdateUser(context.Background(), tc.req)
    tc.checkResponse(t, res, err)
})
```

Здесь компилятор предупреждает нас, что эта переменная `password` 
Oh, the compiler is still complaining about this `password` variable

```go
user, password := randomUser(t)
```

не используется, поэтому я заменю её пустым идентификатором.

```go
user, _ := randomUser(t)
```

И теперь у нас всё готово к работе! Давайте запустим тест и посмотрим, 
что произойдет!

```shell
go test -timeout 30s -run ^TestUpdateUserAPI$ github.com/techschool/simplebank/gapi -v count=1
=== RUN   TestUpdateUserAPI
=== RUN   TestUpdateUserAPI/OK
    /Users/quangpham/Projects/techschool/simplebank/gapi/rpc_update_user_test.go:62: 
            Error Trace:    rpc_update_user_test.go:62
                                                    rpc_update_user_test.go:84
            Error:          Received unexpected error:
                            rpc error: code = Unauthenticated desc = unauthorized: missing metadata
```

Он не был пройден! И возникла следующая ошибка `Unauthenticated: 
missing metadata` («Пользователь неаутентифицирован: отсутствуют метаданные»).
Такая реакция вполне ожидаема, поскольку RPC UpdateUser требует, чтобы 
действующий токен доступа был отправлен через метаданные контекста 
запроса, но мы еще не добавили ни одного токена доступа в контекст. Здесь, 
в функции `authorizeUser`,

```go
func (server *Server) authorizeUser(ctx context.Context) (*token.Payload, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing metadata")
	}

	values := md.Get(authorizationHeader)
	if len(values) == 0 {
		return nil, fmt.Errorf("missing authorization header")
	}

	authHeader := values[0]
	fields := strings.Fields(authHeader)
	if len(fields) < 2 {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	authType := strings.ToLower(fields[0])
	if authType != authorizationBearer {
		return nil, fmt.Errorf("unsupported authorization type: %s", authType)
	}

	accessToken := fields[1]
	payload, err := server.tokenMaker.VerifyToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("invalid access token: %s", err)
	}

	return payload, nil
}

```

мы пытаемся получить метаданные из входящего контекста, затем заголовок 
авторизации из метаданных, и значение этого заголовка должно содержать 
токен доступа `Bearer`. Затем мы используем `tokenMaker` чтобы убедиться, 
что токен действующий. Итак, в unit тесте нам нужно будет добавить
действующий токен доступа к метаданным контекста перед отправкой запроса.

Для этого я добавлю метод `buildContext` в структуру c тестовыми случаями. 
В качестве входных аргументов он примет объект `test.T` и интерфейс 
`token.Maker` и вернет объект контекста, содержащий метаданные с 
токеном доступа.

```go
testCases := []struct {
    name          string
    req           *pb.UpdateUserRequest
    buildStubs    func(store *mockdb.MockStore)
    buildContext  func(t *testing.T, tokenMaker token.Maker) context.Context
    checkResponse func(t *testing.T, res *pb.UpdateUserResponse, err error)
}{
    ...
}
```

Мы сделали этот метод в структуру с тестовыми случаями, потому что для 
разных случаев у нас могут быть разные метаданные, например, с 
не действительным токеном доступа или с истекшим сроком действия или мы
даже можем возвращать `context.Background()`, если не хотим отправлять 
токен.

```go
buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
    return context.Background()
},
```

В функции `Run` теста мы будем вызывать `tc.buildContext()` и передавать 
объект `t` и `server.tokenMaker`, чтобы получить контекст с добавленными 
метаданными. Затем будем использовать этот контекст в качестве первого 
аргумента при вызове `UpdateUser`.

```go
ctx := tc.buildContext(t, server.tokenMaker)
res, err := server.UpdateUser(ctx, tc.req)
```

Хорошо, теперь для успешного тестового случая нам нужно будет добавить 
действующий токен доступа к метаданным контекста. Итак, сначала я присвою 
этот `context.Background()` переменной `ctx`. Затем мы можем использовать 
функцию `NewIncomingContext` пакета `metadata` из `grpc`, чтобы добавить
объект метаданных (`md`) в фоновый контекст.

```go
ctx := context.Background()
return metadata.NewIncomingContext(ctx, md)
```

Мы можем создать объект метаданных с помощью структуры `metadata.MD`.
К этому объекту мы добавим заголовок авторизации с токеном доступа. 
Обратите внимание, что объект `md` — это просто карта со строковым ключом и
значениями типа срез строк, поэтому здесь нам нужно объявить срез строк 
только с одним элементом: `BearerToken`.

```go
md := metadata.MD{
    authorizationHeader: []string{
        bearerToken,
    },
}
```

На следующем шаге я создам `bearerToken` с помощью функции `fmt.Sprint()`.
Он будет состоять из двух подстрок, разделенных пробелом. Первая — это 
тип авторизации, в данном случае "bearer". И вторая — это токен доступа, 
который мы скоро создадим. Здесь мы будем использовать `tokenMaker`, чтобы 
создать действующий токен доступа.

```go
tokenMaker.CreateToken()
bearerToken := fmt.Sprintf("%s %s", authorizationBearer, accessToken)
```

Я передаю этому методу `CreateToken` имя пользователя и корректное значение
для продолжительности периода времени в 1 минуту. Этот метод вернет строку 
c токеном доступа, полезную нагрузку токена и ошибку. Мы не используем 
полезную нагрузку токена, поэтому я воспользуюсь пустым идентификатором. 
Затем проверим, что возвращаемая ошибка равна `nil` с помощью функции 
`require.NoError()`.

```go
accessToken, _, err := tokenMaker.CreateToken(user.Username, time.Minute)
require.NoError(t, err)
```

Мы можем немножко сократить код, поместив фоновый контекст в начале функции 
(`ctx := context.Background()`) непосредственно в этот вызов 
`metadata.NewIncomingContext(ctx, md)`. И на этом по сути всё. Мы 
закончили с написанием функции для добавления токена доступа к метаданным 
контекста.

```go
buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
    ctx := context.Background()
    accessToken, _, err := tokenMaker.CreateToken(user.Username, time.Minute)
    require.NoError(t, err)
    bearerToken := fmt.Sprintf("%s %s", authorizationBearer, accessToken)
    md := metadata.MD{
        authorizationHeader: []string{
            bearerToken,
        },
    }
    return metadata.NewIncomingContext(ctx, md)
},
```

Давайте повторно запустим тест и посмотрим, что произойдет!

```shell
go test -timeout 30s -run ^TestUpdateUserAPI$ github.com/techschool/simplebank/gapi -v count=1
=== RUN   TestUpdateUserAPI
=== RUN   TestUpdateUserAPI/OK
--- PASS: TestUpdateUserAPI (0.07s)
    --- PASS: TestUpdateUserAPI/OK (0.00s)
PASS
ok      github.com/techschool/simplebank/gapi     0.410s
```

На этот раз тест успешно пройден! Отлично!

Далее попробуем добавить еще несколько тестовых случаев для проверки 
других ситуаций, например, когда пользователь не найден. Я скопирую
`OK` случай и изменю его название на `UserNotFound`.

```go
name: "UserNotFound",
req: &pb.UpdateUserRequest{
    Username: user.Username,
    FullName: &newName,
    Email:    &newEmail,
},
```

Поскольку это тестовый случай, когда возникает ошибка, в функции `buildStubs` 
нам не нужны реальные параметры в качестве аргументов функции `UpdateUser` 
или пользователь с обновлёнными значениями полей, так что давайте от них 
избавимся.

```go
buildStubs: func(store *mockdb.MockStore) {
    store.EXPECT().
        UpdateUser(gomock.Any(), gomock.Any()).
        Times(1).
        Return(db.User{}, sql.ErrNoRows)
},
```

Здесь мы можем просто использовать `gomock.Any()`, что соответствует 
произвольному значению параметров и возвращаем пустой объект пользователя
с ошибкой: `sql.ErrNoRows`. Драйвер БД вернет эту ошибку, если 
пользователь не найден в базе данных.

Теперь, поскольку функция `buildContext` точно такая же, как и в случае `OK`, 
давайте проведем рефакторинг кода, чтобы у нас не было дублирующихся частей.
Этот метод на самом деле можно вынести в глобальную функцию, которую 
можно будет повторно использовать и в других RPC unit тестах. Поэтому я 
помещу его в файл `main_test.go` и назову `newContextWithBearerToken()`.
Нам придется добавить сюда параметр `username`, поскольку объект `user` 
недоступен в этой функции. Затем передадим `username` в вызов 
`tokenMaker.CreateToken()`. Кстати, я думаю, нам также следует изменить 
это жестко закодированное значение `time.Minute` на параметр длительности, 
чтобы мы могли использовать отрицательное значение для получения 
токена доступа с истекшим сроком действия при тестировании этого случая.
Хорошо, теперь вернемся к unit тестам!

```go
func newContextWithBearerToken(t *testing.T, tokenMaker token.Maker, username string, duration time.Duration) context.Context {
	ctx := context.Background()
	accessToken, _, err := tokenMaker.CreateToken(username, duration)
	require.NoError(t, err)
	bearerToken := fmt.Sprintf("%s %s", authorizationBearer, accessToken)
	md := metadata.MD{
		authorizationHeader: []string{
			bearerToken,
		},
	}
	return metadata.NewIncomingContext(ctx, md)
}
```

Я удалю все тело функции `buildContext` и заменю её только одной строкой, где 
возвращается результат выполнения `newContextWithBearerToken()`. 
Передадим `t`, `tokenMaker`, `user.Username` и `time.Minute` в 
качестве входных аргументов.

```go
buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
    return newContextWithBearerToken(t, tokenMaker, user.Username, time.Minute)
},
```

Этот же фрагмент кода можно использовать в `buildContext` для случая 
`UserNotFound`. Теперь код стал выглядеть гораздо лаконичнее, не так ли?

Далее, в функции `checkResponse` вместо того, чтобы требовать отсутствия 
ошибок, мы будем требовать возврата ненулевой ошибки. Нам также следует 
проверить код состояния, поэтому давайте скопируем его из тестового случая, 
когда возникает ошибка, который мы написали при тестировании `CreateUser` на 
предыдущей лекции.

```go
st, ok := status.FromError(err)
require.True(t, ok)
require.Equal(t, codes.NotFound, st.Code())
```

По сути, мы преобразуем возвращаемую ошибку в объект состояния и требуем, 
чтобы поле `Code` этого объекта было `NotFound`, поскольку это значение
должно быть возвращено сервером в этом случае.

Хорошо, всё готово. Теперь пора повторно запустить повторить тест!

```shell
go test -timeout 30s -run ^TestUpdateUserAPI$ github.com/techschool/simplebank/gapi -v count=1
=== RUN   TestUpdateUserAPI
=== RUN   TestUpdateUserAPI/OK
=== RUN   TestUpdateUserAPI/UserNotFound
--- PASS: TestUpdateUserAPI (0.07s)
    --- PASS: TestUpdateUserAPI/OK (0.00s)
    --- PASS: TestUpdateUserAPI/UserNotFound (0.00s)
PASS
ok      github.com/techschool/simplebank/gapi     0.385s
```

Он успешно пройден. Превосходно! Остальные тестовые случаи могут быть 
написаны подобным образом, поэтому давайте продублируем этот случай
"UserNotFound" и протестируем случай, когда у токена истёк срок действия.

```go
name: "ExpiredToken",
req: &pb.UpdateUserRequest{
    Username: user.Username,
    FullName: &newName,
    Email:    &newEmail,
},
```

В этом случае мы ожидаем, что функция `UpdateUser` не будет вызываться 
(или вызовется 0 раз).

```go
buildStubs: func(store *mockdb.MockStore) {
    store.EXPECT().
        UpdateUser(gomock.Any(), gomock.Any()).
        Times(0)
},
```

Поскольку станет понятно, что запрос не нужно выполнять уже при вызове
`authorizeUser()`.

И, как я уже говорил, мы можем создать контекст с токеном, у которого истёк 
срок действия. Для этого достаточно передать отрицательное значение для 
параметра `duration`.

```go
buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
    return newContextWithBearerToken(t, tokenMaker, user.Username, -time.Minute)
},
```

Наконец, в функции `checkResponse` мы ожидаем, что код состояния будет
равен `Unauthenticated`. Вот и всё что нужно было изменить!

```go
checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
    require.Error(t, err)
    st, ok := status.FromError(err)
    require.True(t, ok)
    require.Equal(t, codes.Unauthenticated, st.Code())
},
```

Давайте повторно запустим тесты!

```shell
go test -timeout 30s -run ^TestUpdateUserAPI$ github.com/techschool/simplebank/gapi -v count=1
=== RUN   TestUpdateUserAPI
=== RUN   TestUpdateUserAPI/OK
=== RUN   TestUpdateUserAPI/UserNotFound
=== RUN   TestUpdateUserAPI/ExpiredToken
--- PASS: TestUpdateUserAPI (0.07s)
    --- PASS: TestUpdateUserAPI/OK (0.00s)
    --- PASS: TestUpdateUserAPI/UserNotFound (0.00s)
    --- PASS: TestUpdateUserAPI/ExpiredToken (0.00s)
PASS
ok      github.com/techschool/simplebank/gapi     0.380s
```

Все они успешно пройдены!

Попробуете добавить другие случаи самостоятельно? Например, тест `No
Authorization`, когда пользователь не предоставляет данных для его 
аутентификации (или токен доступа).

```go
name: "NoAuthorization",
req: &pb.UpdateUserRequest{
    Username: user.Username,
    FullName: &newName,
    Email:    &newEmail,
},
```

Довольно просто, не так ли? Все, что нам нужно сделать, это изменить 
в операторе `return` функции `buildContext` значение на 
`context.Background()`. Затем повторно запустим тесты.

```shell
go test -timeout 30s -run ^TestUpdateUserAPI$ github.com/techschool/simplebank/gapi -v count=1
=== RUN   TestUpdateUserAPI
=== RUN   TestUpdateUserAPI/OK
=== RUN   TestUpdateUserAPI/UserNotFound
=== RUN   TestUpdateUserAPI/ExpiredToken
=== RUN   TestUpdateUserAPI/NoAuthorization
--- PASS: TestUpdateUserAPI (0.07s)
    --- PASS: TestUpdateUserAPI/OK (0.00s)
    --- PASS: TestUpdateUserAPI/UserNotFound (0.00s)
    --- PASS: TestUpdateUserAPI/ExpiredToken (0.00s)
    --- PASS: TestUpdateUserAPI/NoAuthorization (0.00s)
PASS
ok      github.com/techschool/simplebank/gapi     0.383s
```

И вуаля, все они опять успешно пройдены!

Прежде чем мы закончим, я покажу вам ещё один, последний на сегодня 
тестовый случай, когда указан неправильный адрес электронной почты. В 
этом случае мы не можем просто использовать символ амперсанда для получения
адреса строковой константы вот так `&invalid-email`. Поэтому нам придётся 
использовать переменную с названием `invalidEmail`, и объявить её в верхней 
части функции следующим образом.

```go
newName := util.RandomOwner()
newEmail := util.RandomEmail()
invalidEmail := "invalid-email"
```

```go
name: "InvalidEmail",
req: &pb.UpdateUserRequest{
    Username: user.Username,
    FullName: &newName,
    Email:    &invalidEmail,
},
```

Теперь функция UpdateUser должна вызываться 0 раз, а контекст должен 
содержать действующий токен доступа.

```go
buildStubs: func(store *mockdb.MockStore) {
    store.EXPECT().
        UpdateUser(gomock.Any(), gomock.Any()).
        Times(0)
},
buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
    return newContextWithBearerToken(t, tokenMaker, user.Username, time.Minute)
},
```

Но на этот раз мы ожидаем, что сервер вернет код состояния `InvalidArgument`.

```go
checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
    require.Error(t, err)
    st, ok := status.FromError(err)
    require.True(t, ok)
    require.Equal(t, codes.InvalidArgument, st.Code())
},
```

Вот и всё! Мы внесли все необходимые изменения! Давайте повторно запустим 
тесты ещё один, последний раз!

```shell
go test -timeout 30s -run ^TestUpdateUserAPI$ github.com/techschool/simplebank/gapi -v count=1
=== RUN   TestUpdateUserAPI
=== RUN   TestUpdateUserAPI/OK
=== RUN   TestUpdateUserAPI/UserNotFound
=== RUN   TestUpdateUserAPI/ExpiredToken
=== RUN   TestUpdateUserAPI/NoAuthorization
=== RUN   TestUpdateUserAPI/InvalidEmail
--- PASS: TestUpdateUserAPI (0.07s)
    --- PASS: TestUpdateUserAPI/OK (0.00s)
    --- PASS: TestUpdateUserAPI/UserNotFound (0.00s)
    --- PASS: TestUpdateUserAPI/ExpiredToken (0.00s)
    --- PASS: TestUpdateUserAPI/NoAuthorization (0.00s)
    --- PASS: TestUpdateUserAPI/InvalidEmail (0.00s)
PASS
ok      github.com/techschool/simplebank/gapi     0.407s
```

Все тесты были успешно пройдены.

Итак, теперь вы знаете, как написать unit тесты для gRPC API, 
требующий аутентификации. На основе этой реализации вы можете добавить 
другие тестовые случаи для этого API обновляющего значения полей 
пользователя или самостоятельно написать тесты для других API в проекте!

И на этом мы закончим сегодняшнюю лекцию.

Я надеюсь, что она была интересной и полезной для вас. Большое
спасибо за время, потраченное на чтение! Желаю вам получать удовольствие
от обучения и до встречи на следующей лекции!
