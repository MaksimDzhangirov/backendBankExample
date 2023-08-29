# Unit-тестирование gRPC API с помощью фиктивной БД и Redis

[Оригинал](https://www.youtube.com/watch?v=QFxZlKb7W2k)

Всем привет, добро пожаловать на мастер-класс по бэкенду! На данный момент 
мы написали много кода для наших gRPC веб-сервисов, но до сих пор не 
написали для них unit тестов. Итак, на этой лекции я покажу вам, как это 
сделать.

# Unit тест для gRPC CreateUser API

API, для которого мы будем писать тесты сегодня, — это CreateUser RPC. Сделать
это будет немного сложнее, чем для других API, поскольку он содержит 
транзакцию БД, которая создает нового пользователя, и, если пользователь 
успешно создан, вызывается функция обратного вызова `AfterCreate`, чтобы 
распределить асинхронную задачу отправки письма для подтверждения адреса 
электронной почты.

Как вы уже знаете, задача обычно хранится в Redis. Таким образом, для 
unit тестирования этого RPC мы будем работать с двумя фиктивными сущностями,
одной из которых является фиктивная БД, а другой — фиктивный распределитель
задач для Redis.

Во-первых, давайте создадим новый файл с именем `rpc_create_user_test.go` 
внутри пакета `gapi`. Тесты, которые мы собираемся написать, будут очень 
похожи на те, которые мы написали в пакете `api` для Gin HTTP-сервисов.
Поэтому, давайте откроем файл `user_test.go` и скопируем из него 
функцию `TestCreateUserAPI`, а затем вставим её в новый файл, который 
только что создали.

```go
package gapi

import (
    "bytes"
    "database/sql"
    "encoding/json"
    "fmt"
    "net/http"
    "net/http/httptest"
    "reflect"
    "testing"
    
    mockdb "github.com/MaksimDzhangirov/backendBankExample/db/mock"
    db "github.com/MaksimDzhangirov/backendBankExample/db/sqlc"
    "github.com/MaksimDzhangirov/backendBankExample/util"
    "github.com/gin-gonic/gin"
    "github.com/golang/mock/gomock"
    "github.com/lib/pq"
    "github.com/stretchr/testify/require"
)

type eqCreateUserParamsMatcher struct {
    arg      db.CreateUserParams
    password string
}

func (e eqCreateUserParamsMatcher) Matches(x interface{}) bool {
    arg, ok := x.(db.CreateUserParams)
    if !ok {
        return false
    }
    
    err := util.CheckPassword(e.password, arg.HashedPassword)
    if err != nil {
        return false
    }
    
    e.arg.HashedPassword = arg.HashedPassword
    return reflect.DeepEqual(e.arg, arg)
}

func (e eqCreateUserParamsMatcher) String() string {
    return fmt.Sprintf("matches arg %v and password %v", e.arg, e.password)
}

func EqCreateUserParams(arg db.CreateUserParams, password string) gomock.Matcher {
    return eqCreateUserParamsMatcher{arg, password}
}

func TestCreateUserAPI(t *testing.T) {
    user, password := randomUser(t)
    
    testCases := []struct {
        name          string
        body          gin.H
        buildStubs    func(store *mockdb.MockStore)
        checkResponse func(recoder *httptest.ResponseRecorder)
    }{
        {
            name: "OK",
            body: gin.H{
                "username":  user.Username,
                "password":  password,
                "full_name": user.FullName,
                "email":     user.Email,
            },
            buildStubs: func(store *mockdb.MockStore) {
                arg := db.CreateUserParams{
                    Username: user.Username,
                    FullName: user.FullName,
                    Email:    user.Email,
                }
                store.EXPECT().
                    CreateUser(gomock.Any(), EqCreateUserParams(arg, password)).
                    Times(1).
                    Return(user, nil)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusOK, recorder.Code)
                requireBodyMatchUser(t, recorder.Body, user)
            },
        },
        {
            name: "InternalError",
            body: gin.H{
                "username":  user.Username,
                "password":  password,
                "full_name": user.FullName,
                "email":     user.Email,
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().
                    CreateUser(gomock.Any(), gomock.Any()).
                    Times(1).
                    Return(db.User{}, sql.ErrConnDone)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusInternalServerError, recorder.Code)
            },
        },
        {
            name: "DuplicateUsername",
            body: gin.H{
                "username":  user.Username,
                "password":  password,
                "full_name": user.FullName,
                "email":     user.Email,
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().
                    CreateUser(gomock.Any(), gomock.Any()).
                    Times(1).
                    Return(db.User{}, &pq.Error{Code: "23505"})
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusForbidden, recorder.Code)
            },
        },
        {
            name: "InvalidUsername",
            body: gin.H{
                "username":  "invalid-user#1",
                "password":  password,
                "full_name": user.FullName,
                "email":     user.Email,
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().
                    CreateUser(gomock.Any(), gomock.Any()).
                    Times(0)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusBadRequest, recorder.Code)
            },
        },
        {
            name: "InvalidEmail",
            body: gin.H{
                "username":  user.Username,
                "password":  password,
                "full_name": user.FullName,
                "email":     "invalid-email",
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().
                    CreateUser(gomock.Any(), gomock.Any()).
                    Times(0)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusBadRequest, recorder.Code)
            },
        },
        {
            name: "TooShortPassword",
            body: gin.H{
                "username":  user.Username,
                "password":  "123",
                "full_name": user.FullName,
                "email":     user.Email,
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().
                    CreateUser(gomock.Any(), gomock.Any()).
                    Times(0)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusBadRequest, recorder.Code)
            },
        },
    }
    
    for i := range testCases {
        tc := testCases[i]
    
        t.Run(tc.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()
    
            store := mockdb.NewMockStore(ctrl)
            tc.buildStubs(store)
    
            server := newTestServer(t, store)
            recorder := httptest.NewRecorder()
    
            // Marshal body data to JSON
            data, err := json.Marshal(tc.body)
            require.NoError(t, err)
    
            url := "/users"
            request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
            require.NoError(t, err)
    
            server.router.ServeHTTP(recorder, request)
            tc.checkResponse(recorder)
        })
    }
}
```

Как видите, в этом тесте используется пользовательский `gomock.Matcher`, 
который мы написали в лекции 18 курса, для корректного сравнения входных 
аргументов функции `CreateUser`, а именно поля хэшированного пароля. Вы 
можете пересмотреть эту лекцию, чтобы понять, что мы здесь делаем, прежде 
чем продолжить.

Хорошо, теперь давайте обновим код этого теста, чтобы он работал с нашим 
gRPC сервером.

Во-первых, мы должны создать случайного пользователя с помощью этой функции 
`randomUser()`. Она была создана в пакете `api`, поэтому давайте перейдем 
туда, скопируем её и вставить сюда, непосредственно перед функцией 
`TestCreateUserAPI`.

```go
func randomUser(t *testing.T) (user db.User, password string) {
    password = util.RandomString(6)
    hashedPassword, err := util.HashPassword(password)
    require.NoError(t, err)
    
    user = db.User{
        Username:       util.RandomOwner(),
        HashedPassword: hashedPassword,
        FullName:       util.RandomOwner(),
        Email:          util.RandomEmail(),
    }
    return
}

func TestCreateUserAPI(t *testing.T) {
    ...
}
```

Далее мы определим несколько тестовых случаев. В этом поле `name` будет 
храниться название случая. Затем нужно будет изменить поле `body`, потому 
что здесь мы используем не Gin, а gRPC. Если мы посмотрим на обработчик 
RPC CreateUser, то увидим, что входные данные отправляются через объект 
`CreateUserRequest`. Так что именно этот тип данных мы будем использовать в
данном случае.

```go
testCases := []struct {
    name          string
    req           *pb.CreateUserRequest
}{
    ...
}
```

OK, теперь нам нужно обновить содержимое этого запроса. С помощью gRPC 
очень легко писать код, поскольку все строго типизировано и будет срабатывать 
функция автозаполнения из Visual Studio Code. Мы должны изменить эту 
звездочку `*pb.CreateUserRequest` на амперсанд, потому что мы хотим 
получить адрес этого объекта запроса.

```go
req: &pb.CreateUserRequest{
    Username: user.Username,
    Password: password,
    FullName: user.FullName,
    Email:    user.Email,
}
```

OK, теперь давайте перейдем к функции `buildStubs`. Здесь мы сообщаем 
`gomock`, вызов какой функции ожидается, с какими параметрами и какие 
выходные данные возвращаем. Когда мы реализовали HTTP API CreateUser с 
помощью Gin, у нас не было асинхронной задачи, которая отправляет письма, 
поэтому мы ожидали, что метод `CreateUser` объекта `store` будет вызываться 
непосредственно.

```go
store.EXPECT().
    CreateUser(gomock.Any(), EqCreateUserParams(arg, password)).
    Times(1).
    Return(user, nil)
```

Однако в нашей новой реализации CreateUser RPC мы не вызываем этот метод 
напрямую, а вместо этого используем транзакцию CreateUser.

```go
txResult, err := server.store.CreateUserTx(ctx, arg)
```

Метод `CreateUser` вызывается только внутри этой транзакции вместе с 
функцией обратного вызова. Итак, здесь, в функции `buildStubs`, я изменю 
этот аргумент `arg := db.CreateUserParams` на `db.CreateUserTxParams` и 
задам `CreateUserParams` в качестве его вложенного поля.

```go
buildStubs: func(store *mockdb.MockStore) {
    arg := db.CreateUserTxParams{
        CreateUserParams: db.CreateUserParams{
            Username: user.Username,
            FullName: user.FullName,
            Email:    user.Email,
        },
    }
    ...
}
```

Обратите внимание, что в этой структуре также существует другое поле для 
обратного вызова `AfterCreate`, но в Golang нет возможности сравнить две 
функции, поэтому давайте пока проигнорируем этот обратный вызов. Мы 
займемся этим позже.

OK, здесь мы ожидаем, что транзакция CreateUser фиктивного `store` будет 
вызвана строго один раз с только что созданным входным аргументом.

```go
store.EXPECT().
    CreateUserTx(gomock.Any(), EqCreateUserParams(arg, password)).
    Times(1).
    Return(user, nil)
```

Однако, поскольку мы используем этот нестандартный, пользовательский 
сопоставитель `gomock` для сравнения входных данных, нам придется немного 
обновить его тип данных.

Во-первых, я изменю его название на `eqCreateUserTxParamsMatcher`. Затем
тип аргумента `arg db.CreateUserParams` должен быть равен 
`db.CreateUserTxParams`. Кстати, чтобы было легче понять, я изменю имя 
этой переменной `func (e, eqCreateUserTxParamsMatcher)` на "expected", 
так как она содержит ожидаемое значение аргументов. А эта переменная `x` 
в `Matches(x interface{})` будет содержать фактическое значение, которое 
обработчик gRPC будет использовать при вызове транзакции CreateUser. Итак, 
когда мы преобразуем его в `db.CreateUserTxParams`, я присвою результат 
переменной под названием `actualArg`. После этого мы воспользуемся 
функцией `util.CheckPassword`, чтобы проверить, соответствует ли ожидаемый 
пароль фактическому хешированному паролю. И, наконец, мы присваиваем значение 
фактического хешированного пароля ожидаемому аргументу.

```go
func (expected eqCreateUserTxParamsMatcher) Matches(x interface{}) bool {
    actualArg, ok := x.(db.CreateUserTxParams)
    if !ok {
        return false
    }
    
    err := util.CheckPassword(expected.password, actualArg.HashedPassword)
    if err != nil {
        return false
    }
    
    expected.arg.HashedPassword = actualArg.HashedPassword
    return reflect.DeepEqual(expected.arg, actualArg)
}
```

Мы воспользовались этим приёмом, потому что хотим использовать функцию 
`DeepEqual` для их сравнения. Я уже объяснял причину, из-за которой мы 
должны это делать, в лекции 18. Каждый раз, когда мы хешируем пароль, мы 
добавляем случайную соль, поэтому выходное значение хеш-функции будет 
другим, что поможет нам предотвратить атаки с помощью радужной таблице.

OK, теперь мы должны изменить тип аргумента в этой функции `func EqCreateUserTxParams(arg db.CreateUserParams, password string)` на `db.CreateUserTxParams`. Затем 
вернёмся к unit тесту, который мы пишем.

Нам придется модифицировать функцию `checkResponse`, поскольку для gRPC 
сервисов мы не будем использовать HTTP-рекордер для сохранения ответа,
а получим объект `CreateUserResponse` или ошибку непосредственно из RPC 
обработчика. Поэтому я задам их в качестве входных аргументов этой 
функции `checkResponse`.

```go
checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
    ...
}
```

Я также добавляю объект `testing.T` в качестве первого входного аргумента, 
так как он нам понадобится при сравнении реального результата с ожидаемым 
значением. Хорошо, давайте скопируем эту сигнатуру функции и вставим её в 
определение структуры `testCases`.

```go
testCases := []struct {
    name          string
    req           *pb.CreateUserRequest
    buildStubs    func(store *mockdb.MockStore)
    checkResponse func(t *testing.T, res *pb.CreateUserResponse, err error)
}{
    ...
}
```

Хорошо, теперь давайте исправим содержимое функции `checkResponse`. 
Поскольку это успешный случай, мы ожидаем, что ошибок не возникнет. И 
ответ должен быть не `nil`. Далее мы можем получить созданный объект 
пользователя из ответа. И потребовать, чтобы имя созданного пользователя 
совпадало с входным `user.Username`. Мы ожидаем того же для других полей, 
таких как полное имя и адрес электронной почты.

```go
checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
    require.NoError(t, err)
    require.NotNil(t, res)
    createdUser := res.GetUser()
    require.Equal(t, user.Username, createdUser.Username)
    require.Equal(t, user.FullName, createdUser.FullName)
    require.Equal(t, user.Email, createdUser.Email)
}
```

О, я только что заметил, что мы должны изменить это

```go
store.EXPECT().
    CreateUserTx(gomock.Any(), EqCreateUserParams(arg, password)).
    Times(1).
    Return(user, nil)
```

название функции `EqCreateUserParams` на `EqCreateUserTxParams`, поскольку
входным аргументом теперь фактически является `CreateUserTxParams`. А также
мы ожидаем, что функция `CreateUserTx` будет вызвана один раз, и, согласно 
коду, она вернет объект `CreateUserTxResult`. Здесь мы не можем просто 
вернуть объект `user`, поэтому нам придётся изменить его на 
`db.CreateUserTxResult`, а `User` будет всего лишь одним полем этого объекта.
Вот как должны выглядеть элементы структуры `testCases` для успешного случая.

```go
store.EXPECT().
    CreateUserTx(gomock.Any(), EqCreateUserTxParams(arg, password)).
    Times(1).
    Return(db.CreateUserTxResult{User: user}, nil)
```

Нам придётся модифицировать код и для случаев, когда происходит ошибка при 
выполнении RPC, но давайте займёмся этим поздне, а пока сосредоточимся на 
успешном случае. Так что пока, чтобы не усложнять, я просто удалю остальные
тестовые случаи.

```go
testCases := []struct {
    name          string
    req           *pb.CreateUserRequest
    buildStubs    func(store *mockdb.MockStore)
    checkResponse func(t *testing.T, res *pb.CreateUserResponse, err error)
}{
    {
        name: "OK",
        req: &pb.CreateUserRequest{
            Username: user.Username,
            Password: password,
            FullName: user.FullName,
            Email:    user.Email,
        },
        buildStubs: func(store *mockdb.MockStore) {
            arg := db.CreateUserTxParams{
                CreateUserParams: db.CreateUserParams{
                    Username: user.Username,
                    FullName: user.FullName,
                    Email:    user.Email,
                },
            }
            store.EXPECT().
                CreateUserTx(gomock.Any(), EqCreateUserTxParams(arg, password)).
                Times(1).
                Return(db.CreateUserTxResult{User: user}, nil)
        },
        checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
            require.NoError(t, err)
            require.NotNil(t, res)
            createdUser := res.GetUser()
            require.Equal(t, user.Username, createdUser.Username)
            require.Equal(t, user.FullName, createdUser.FullName)
            require.Equal(t, user.Email, createdUser.Email)
        },
    },
}
```

Следующее, что нам нужно сделать, это обновить тело unit теста. Как вы 
видите здесь,

```go
t.Run(tc.name, func(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    store := mockdb.NewMockStore(ctrl)
    tc.buildStubs(store)

    server := newTestServer(t, store)
    recorder := httptest.NewRecorder()

    // Marshal body data to JSON
    data, err := json.Marshal(tc.body)
    require.NoError(t, err)

    url := "/users"
    request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
    require.NoError(t, err)

    server.router.ServeHTTP(recorder, request)
    tc.checkResponse(recorder)
})
```

мы создали `gomock` контроллер и использовали его для создания фиктивного 
`store` для базы данных. Затем мы воспользовались фиктивным `store` для 
создания тестового сервера. Эта функция — `newTestServer` отсутствует в 
пакете `gapi`. Мы можем найти его содержимое в файле `main_test.go`
пакета `api`.

```go
func TestMain(m *testing.M) {
    gin.SetMode(gin.TestMode)
    os.Exit(m.Run())
}

func newTestServer(t *testing.T, store db.Store) *Server {
    config := util.Config{
        TokenSymmetricKey:   util.RandomString(32),
        AccessTokenDuration: time.Minute,
    }
    
    server, err := NewServer(config, store)
    require.NoError(t, err)
    
    return server
}
```

В этом файле также есть функция `TestMain()`, которую мы использовали для 
перевода Gin в режим тестирования, но она нам не нужна для нашего gRPC 
сервиса, поэтому давайте скопируем только функцию `newTestServer`.

Я создам файл `main_test.go` внутри пакета `gapi` и вставлю в него 
содержимое функции, которую мы только что скопировали.

```go
package gapi

import (
	"testing"
	"time"

	db "github.com/MaksimDzhangirov/backendBankExample/db/sqlc"
	"github.com/MaksimDzhangirov/backendBankExample/util"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, store db.Store) *Server {
	config := util.Config{
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
	}

	server, err := NewServer(config, store)
	require.NoError(t, err)

	return server
}
```

Однако нам придется немного обновить её, поскольку функция `NewServer` 
пакета `gapi` требует ещё один входной аргумент для распределителя задач 
(`TaskDistributor`). И если мы откроем его определение, то увидим, что 
это интерфейс со следующим единственным методом.

```go
type TaskDistributor interface {
	DistributeTaskSendVerifyEmail(
		ctx context.Context,
		payload *PayloadSendVerifyEmail,
		opts ...asynq.Option,
	) error
}
```

Это хорошо для написания unit тестов, потому что мы не хотим 
подключаться к реальному серверу Redis. Таким образом, точно так же, как 
мы сделали для интерфейса `Store`, мы также можем создать имитацию для
интерфейса `TaskDistributor` и использовать его в тесте.

Для этого я открою `Makefile` и прокручу вниз до команды «mock».

```makefile
mock:
    mockgen -package mockdb -destination db/mock/store.go github.com/MaksimDzhangirov/backendBankExample/db/sqlc Store
```

Давайте скопируем эту команду, которая создаст фиктивный `Store`. Затем 
измените название пакета на `mockwk` (что означает "mock worker"). В качестве
папки, куда будет генерироваться результат, выберем `worker`. Будет создана 
папка «mock», а внутри неё будет файл `distributor.go`, содержащий 
сгенерированный код фиктивного распределителя задач. Нам нужно изменить путь
пакета, для которого будет создаваться имитация, с 
`backendBankExample/db/sqlc` на `backendBankExample/worker` следующим
образом.

```makefile
mock:
    mockgen -package mockdb -destination db/mock/store.go github.com/MaksimDzhangirov/backendBankExample/db/sqlc Store
    mockgen -package mockwk -destination worker/mock/distributor.go github.com/MaksimDzhangirov/backendBankExample/worker TaskDistributor
```

И, конечно, название интерфейса, который нужно сымитировать, должно 
быть равно `TaskDistributor`.

OK, теперь мы можем выполнить

```shell
make mock
```

в терминале, чтобы сгенерировать фиктивные объекты. Генерация прошла 
успешно.

Итак, в Visual Studio Code мы увидим новую папку `mock` внутри
папка `worker`, а внутри этой папки находится файл `distributor.go`
со структурой `MockTaskDistributor`, которую мы можем использовать в 
unit тестах.

Теперь вернемся к файлу main_test.go. Я добавлю аргумент для распределителя 
задач в функцию `newTestServer`. И мы можем передать его в эту функцию 
`NewServer(config, store)`, чтобы создать новый сервер.

```go
func newTestServer(t *testing.T, store db.Store, taskDistributor worker.TaskDistributor) *Server {
	config := util.Config{
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
	}

	server, err := NewServer(config, store, taskDistributor)
	require.NoError(t, err)

	return server
}
```

Хорошо, внеся все эти изменения, мы можем продолжить писать наши unit тесты.

Поскольку для функции `newTestServer` теперь требуется распространитель 
задач, мы создадим для него имитацию, точно так же, как мы создали имитацию 
`store`. Итак, я определю переменную `taskDistributor` и присвою ей 
`mockwk.NewMockTaskDistributor`.

```go
taskDistributor := mockwk.NewMockTaskDistributor()
```

Поскольку мы впервые используем этот пакет, Visual Studio Code ещё не знает 
о нём, поэтому нам придется импортировать его вручную. Давайте продублируем 
эту строку `import mockdb`, затем изменим название на `mockwk`, а путь к 
пакету должен быть равен `backendBankExample/worker/mock`.

```go
import (
    ...
    mockwk "github.com/MaksimDzhangirov/backendBankExample/worker/mock"
    ...
)
```

OK, теперь, если мы вернемся к тесту, то увидим, что Visual Studio Code 
распознал пакет. В качестве входных данных также требуется передать 
фиктивный объект контроллера, поэтому давайте передадим тот же контроллер, 
который мы использовали для фиктивного `store`. Затем мы можем использовать 
фиктивный распространитель задач для создания тестового сервера.

```go
store := mockdb.NewMockStore(ctrl)
tc.buildStubs(store)

taskDistributor := mockwk.NewMockTaskDistributor(ctrl)

server := newTestServer(t, store, taskDistributor)
recorder := httptest.NewRecorder()
```

Далее я избавлюсь от кода, который настраивает и отправляет HTTP-запросы.

```go
recorder := httptest.NewRecorder()

// Marshal body data to JSON
data, err := json.Marshal(tc.body)
require.NoError(t, err)

url := "/users"
request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
require.NoError(t, err)

server.router.ServeHTTP(recorder, request)
```

Для unit тестирования gRPC сервера мы можем просто напрямую вызвать 
RPC функцию-обработчик следующим образом: `server.CreateUser`. Затем 
передайте фоновый контекст и объект входного запроса для тестового случая. 
Он вернет объект ответа и ошибку, поэтому мы можем отправить их в 
функцию `checkResponse()` тестового случая вместе с объектом `testing.T` 
для окончательной проверки результата.

```go
server := newTestServer(t, store, taskDistributor)
res, err := server.CreateUser(context.Background(), tc.req)
tc.checkResponse(t, res, err)
```

Обратите внимание, что эта переменная `t` отличается от глобальной `t`, 
потому что она была перекрыта входным аргументом этой функции `t.Run`. По 
сути, это объект `t` подтеста, созданный функцией `Run()`. Таким образом, 
вызов `checkResponse` для каждого случая будет независимым и не будет 
мешать друг другу, когда мы добавим больше случаев в будущем.

Хорошо, тест для успешного случая готов. Давайте запустим его, чтобы 
посмотреть, что произойдет!

```shell
=== RUN   TestCreateUserAPI
=== RUN   TestCreateUserAPI/OK
...
--- FAIL: TestCreateUserAPI (0.18s)
    --- FAIL: TestCreateUserAPI/OK (0.11s)
FAIL
FAIL    github.com/MaksimDzhangirov/backendBankExample/gapi     0.545s
FAIL
```

Он завершился со следующей ошибкой: "missing a call to mockStore.CreateUserTx. 
The expected call doesn't match the argument at index 1" («отсутствует 
вызов mockStore.CreateUserTx. Несоответствие аргументов с ожидаемым значением 
для индекса 1»),которым является `CreateUserTxParams`. Итак, из ошибки мы 
видим, что это вот этого ожидаемого 

```go
store.EXPECT().
    CreateUserTx(gomock.Any(), EqCreateUserTxParams(arg, password)).
    Times(1).
    Return(db.CreateUserTxResult{User: user}, nil)
```

вызова функции не произошло. Но что мы можем с этим сделать? Как мы можем 
выяснить, что не так с нашей реализацией?

Что ж, один из способов отладки такого рода ошибок — добавить в определенные 
части теста логи. Мы должны выяснить, до какого места был выполнен код.
Итак, я собираюсь добавить несколько сообщений `fmt.Println()`, одно перед 
проверкой запроса, одно перед хэшированием пароля и одно непосредственно 
перед запуском транзакции CreateUser.

```go
func (server *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	fmt.Println("validate request")
	violations := validateCreateUserRequest(req)
	if violations != nil {
		return nil, invalidArgumentError(violations)
	}

	fmt.Println("request:", req)
	hashedPassword, err := util.HashPassword(req.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %s", err)
	}

	arg := db.CreateUserTxParams{
		CreateUserParams: db.CreateUserParams{
			Username:       req.GetUsername(),
			HashedPassword: hashedPassword,
			FullName:       req.GetFullName(),
			Email:          req.GetEmail(),
		},
		AfterCreate: func(user db.User) error {
			taskPayload := &worker.PayloadSendVerifyEmail{
				Username: user.Username,
			}
			opts := []asynq.Option{
				asynq.MaxRetry(10),
				asynq.ProcessIn(10 * time.Second),
				asynq.Queue(worker.QueueCritical),
			}
			return server.taskDistributor.DistributeTaskSendVerifyEmail(ctx, taskPayload, opts...)
		},
	}

	fmt.Println("create user tx", arg)
	txResult, err := server.store.CreateUserTx(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				return nil, status.Errorf(codes.AlreadyExists, "username already exists: %s", err)
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create user: %s", err)
	}

	rsp := &pb.CreateUserResponse{
		User: convertUser(txResult.User),
	}
	return rsp, nil
}
```

Хорошо, теперь давайте заново запустим тест.

```shell
=== RUN   TestCreateUserAPI
=== RUN   TestCreateUserAPI/OK
validate request
request: username:"fthteb" full_name:"vklrvh" email:"lkswpj@email.com" password:"ouhhdy"
create user tx {{fthteb $2a$10$vPpxT8MAnxflwHnw7mw6auUSwqRBdGnLbahTljqPMMnbnWol6.Xpe vklrvh lkswpj@email.com} 0x15ea480}
...
```

На этот раз мы видим 3 строки логов, которые мы только что добавили. Так 
что тест действительно дошел до транзакции CreateUser, но не смог её 
выполнить, потому что аргумент не совпадает с тем, который мы ожидаем.

Поэтому давайте взглянем на пользовательский сопоставитель `gomock`. Я 
добавлю сюда ещё несколько логов для отладки.

```go
func (expected eqCreateUserTxParamsMatcher) Matches(x interface{}) bool {
	fmt.Println(">> check param matches")
	actualArg, ok := x.(db.CreateUserTxParams)
	if !ok {
		return false
	}

	fmt.Println(">> check password", actualArg)
	err := util.CheckPassword(expected.password, actualArg.HashedPassword)
	if err != nil {
		return false
	}

	fmt.Println(">> deep equal")
	expected.arg.HashedPassword = actualArg.HashedPassword
	return reflect.DeepEqual(expected.arg, actualArg)
}
```

Во-первых, перед преобразованием параметров, во-вторых, перед проверкой 
пароля, в-третьих, перед сравнением аргументов с помощью `DeepEqual`. И я 
хочу добавить ещё один лог в конце, поэтому давайте заменим эту команду 
`return Reflect.DeepEqual(expected.arg, factArg)` на оператор `if`. Если 
аргументы не равны, мы возвращаем `false`. В противном случае мы выводим 
сообщение "param matches!" («параметры совпадают!»), и возвращаем `true`.

```go
func (expected eqCreateUserTxParamsMatcher) Matches(x interface{}) bool {
	...

	fmt.Println(">> deep equal")
	expected.arg.HashedPassword = actualArg.HashedPassword
	if !reflect.DeepEqual(expected.arg, actualArg) {
		return false
	}

	fmt.Println(">> param matches!")
	return true
}
```

Хорошо, давайте запустим тест ещё раз.

```shell
=== RUN   TestCreateUserAPI
=== RUN   TestCreateUserAPI/OK
validate request
request: username:"ohhojh"  full_name:"hstuwy"  email:"pphhyh@email.com"  password:"onzfew"
create user tx {{ohhojh $2a$10$r9mAzKfxXw1lCI19zb.dt07RvIkdeTZKnc8AdMdNqhZJMpb/yhLBC hstuwy pphhyh@email.com} 0x15ea480}
>> check param matches
>> check password {{ohhojh $2a$10$r9mAzKfxXw1lCI19zb.dt07RvIkdeTZKnc8AdMdNqhZJMpb/yhLBC hstuwy pphhyh@email.com} 0x15ea480}
>> deep equal
...
```

На этот раз мы видим все логи вплоть до функции `DeepEqual`. Но нет лога
"param matches!", поэтому проблема должна быть из сравнения аргументов в 
`DeepEqual`. Но мы уже учли проблему, связанную с хешированием пароля, из-за
чего же этот вызов `DeepEqual` завершается с ошибкой?

Что ж, я только что вспомнил об ещё одном проблематичном поле в 
`CreateUserTxParams`, которым является функция обратного вызова 
`AfterCreate`. Как я уже сказал, в Go невозможно сравнить две функции,
именно это приводит к тому, что функция `DeepEqual` завершается с ошибкой.

Чтобы избежать этого, мы должны сравнивать только поля `CreateUserParams`, а
не весь объект-аргумент.

```go
func (expected eqCreateUserTxParamsMatcher) Matches(x interface{}) bool {
	...

	fmt.Println(">> deep equal")
	expected.arg.HashedPassword = actualArg.HashedPassword
	if !reflect.DeepEqual(expected.arg.CreateUserParams, actualArg.CreateUserParams) {
		return false
	}

	fmt.Println(">> param matches!")
	return true
}
```

Вот так, и мне кажется, что ошибка должна пропасть, когда мы перезапустим 
тест.

```shell
=== RUN   TestCreateUserAPI
=== RUN   TestCreateUserAPI/OK
...
--- PASS: TestCreateUserAPI (0.19s)
    --- PASS: TestCreateUserAPI/OK (0.12s)
PASS
ok      github.com/MaksimDzhangirov/backendBankExample/gapi     0.514s
```

Действительно, на этот раз тест был успешно пройден. Поэтому я удалю все 
отладочные логи, которые мы добавили ранее.

```go
func (expected eqCreateUserTxParamsMatcher) Matches(x interface{}) bool {
	actualArg, ok := x.(db.CreateUserTxParams)
	if !ok {
		return false
	}

	err := util.CheckPassword(expected.password, actualArg.HashedPassword)
	if err != nil {
		return false
	}

	expected.arg.HashedPassword = actualArg.HashedPassword
	if !reflect.DeepEqual(expected.arg.CreateUserParams, actualArg.CreateUserParams) {
		return false
	}

	return true
}
```

Здесь среда разработки выдаёт предупреждение, так как это `if` можно 
заменить одним оператором `return`, но я пока оставлю всё как есть, потому
что позже мы кое-что сделаем с обратным вызовом, если аргументы совпадают.

Хорошо, давайте также удалим 3 отладочных лога в файле `rpc_create_user.go`.

Теперь тест пройден, но он только проверяет, выполняется ли транзакция 
CreateUser с ожидаемыми аргументами. Но он не проверяет выполнение функции 
обратного вызова `AfterCreate`. В этом случае мы хотели бы быть уверены, что 
функция `DistributeTaskSendVerifyEmail` вызывается каждый раз, когда 
создается новый пользователь. Таким образом, электронное письмо будет 
доставлено пользователю в будущем. Итак, как мы можем проверить это?

Для этой цели мы можем использовать `MockTaskDistributor`, точно так же, 
как мы делали это для фиктивного `store`. Я добавлю новый аргумент 
`taskDistributor` в функцию `buildStubs()` и обновлю её сигнатуру в 
структуре `testCases`.

```go
testCases := []struct {
    name          string
    req           *pb.CreateUserRequest
    buildStubs    func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor)
    checkResponse func(t *testing.T, res *pb.CreateUserResponse, err error)
}{
    {
        name: "OK",
        req: &pb.CreateUserRequest{
            Username: user.Username,
            Password: password,
            FullName: user.FullName,
            Email:    user.Email,
        },
        buildStubs: func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor) {
            arg := db.CreateUserTxParams{
                CreateUserParams: db.CreateUserParams{
                    Username: user.Username,
                    FullName: user.FullName,
                    Email:    user.Email,
                },
            }
            store.EXPECT().
                CreateUserTx(gomock.Any(), EqCreateUserTxParams(arg, password)).
                Times(1).
                Return(db.CreateUserTxResult{User: user}, nil)
        },
        checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
            ...
        },
    },		
}
```

Затем, в теле теста, давайте переместим вот сюда объявление `task Distributor`.

```go
store := mockdb.NewMockStore(ctrl)
taskDistributor := mockwk.NewMockTaskDistributor(ctrl)
tc.buildStubs(store)
```

И передадим его в вызов `tc.buildStubs()` вместе с фиктивным `store`.

```go
store := mockdb.NewMockStore(ctrl)
taskDistributor := mockwk.NewMockTaskDistributor(ctrl)

tc.buildStubs(store, taskDistributor)
server := newTestServer(t, store, taskDistributor)
```

Вот так!

Затем в конце функции `buildStubs` я вызову `taskDistributor.EXPECT()`.
На этот раз мы хотим, чтобы вызывалась функция 
`DistributeTaskSendVerifyEmail()`. Она имеет несколько входных 
аргументов, таких как контекст, полезная нагрузка задачи и некоторые 
параметры `asynq`. Поэтому, я буду использовать `gomock.Any()` для 
контекста, затем `taskPayload`, которую мы объявим чуть позже, наконец, 
ещё один сопоставитель `gomock.Any()` для параметров `asynq`.

```go
taskDistributor.EXPECT().
    DistributeTaskSendVerifyEmail(gomock.Any(), taskPayload, gomock.Any())
```

Этот сопоставитель `Any()` может сравнивать объекты с переменным 
количеством аргументов, поэтому им можно заменить три параметра для 
настройки `asynq`, которые мы отправляем в этом массиве `opt`.

```go
opts := []asynq.Option{
    asynq.MaxRetry(10),
    asynq.ProcessIn(10 * time.Second),
    asynq.Queue(worker.QueueCritical),
}
```

По желанию, вы также можете использовать сопоставитель `gomock.Eq()` для
сравнения каждого из них. Я же не буду усложнять, воспользуюсь `gomock.Any()`
и сосредоточусь на более важном параметре, а именно на `taskPayload`. Мы 
можем скопировать его из функции обратного вызова вот здесь.

```go
AfterCreate: func(user db.User) error {
    taskPayload := &worker.PayloadSendVerifyEmail{
        Username: user.Username,
    }
    ...
},
```

Главное, чтобы он имел тип `PayloadSendVerifyEmail`, а поле `Username` было
задано равным полю `Username` аргумента `user`, поступающего на вход функции.

```go
buildStubs: func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor) {
    arg := db.CreateUserTxParams{
        CreateUserParams: db.CreateUserParams{
            Username: user.Username,
            FullName: user.FullName,
            Email:    user.Email,
        },
    }
    store.EXPECT().
        CreateUserTx(gomock.Any(), EqCreateUserTxParams(arg, password)).
        Times(1).
        Return(db.CreateUserTxResult{User: user}, nil)

    taskPayload := &worker.PayloadSendVerifyEmail{
        Username: user.Username,
    }
    taskDistributor.EXPECT().
        DistributeTaskSendVerifyEmail(gomock.Any(), taskPayload, gomock.Any())
},
```

Итак, теперь мы ожидаем, что эта функция распределения задач будет вызвана 
ровно один раз и вернет в качестве результата переменную типа `error`. Это
успешный случай, поэтому здесь мы вернем ошибку равную `nil`.

```go
taskDistributor.EXPECT().
    DistributeTaskSendVerifyEmail(gomock.Any(), taskPayload, gomock.Any()).
    Times(1).
    Return(nil)
```

Вот и всё что нужно было сделать! Давайте попробуем запустить тест, чтобы 
увидеть, что произойдёт!

```shell
=== RUN   TestCreateUserAPI
=== RUN   TestCreateUserAPI/OK
...
--- FAIL: TestCreateUserAPI (0.18s)
    --- FAIL: TestCreateUserAPI/OK (0.11s)
FAIL
FAIL    github.com/MaksimDzhangirov/backendBankExample/gapi     0.531s
FAIL
```

Он завершился с ошибкой! И ошибка связана с отсутствием вызовов функции 
`DistributeTaskSendVerifyEmail()`. В чём же дело? Почему эта функция не 
вызывается? Всё не так просто, поэтому прежде чем продолжить сделайте паузу
и немного подумайте.

Поняли в чём причина? Вспомните, что помимо `TaskDistributor`, мы также 
имитируем транзакцию CreateUser. И если мы посмотрим на её реализацию, то
увидим, что функция распределения задач по отправке писем для подтверждения
адреса электронной почты вызывается только внутри функции обратного 
вызова `AfterCreate`. Это означает, что она будет вызываться только в том 
случае, если будет выполнена реальная транзакция БД и в базе данных будет 
успешно создан реальный пользователь. Однако, поскольку мы имитируем эту 
транзакцию CreateUser, фактический код, который взаимодействует с БД, 
не выполняется. Следовательно, функция `AfterCreate` также не запускается.

Что же нам делать? Можем ли мы каким-то образом вызвать функцию обратного 
вызова `AfterCreate` в фиктивной транзакции? И это должна быть не имитация
вызова, а фактическое выполнение функции обратного вызова `AfterCreate`.

Что ж, к счастью, у нас есть специальный сопоставитель `gomock`, который 
сравнивает фактические аргументы транзакции с ожидаемыми. И мы знаем, что 
если они совпали, то можно считать, что транзакция была успешно выполнена.
Поэтому здесь мы можем безбоязненно осуществить вызов функции `AfterCreate`.

```go
func (expected eqCreateUserTxParamsMatcher) Matches(x interface{}) bool {
	actualArg, ok := x.(db.CreateUserTxParams)
	if !ok {
		return false
	}

	err := util.CheckPassword(expected.password, actualArg.HashedPassword)
	if err != nil {
		return false
	}

	expected.arg.HashedPassword = actualArg.HashedPassword
	if !reflect.DeepEqual(expected.arg.CreateUserParams, actualArg.CreateUserParams) {
		return false
	}

    // call the AfterCreate function here

	return true
}
```

И поскольку у нас есть фактическая реализация функции обратного вызова 
внутри этого объекта `actualArg`, мы можем просто вызвать 
`actualArg.AfterCreate()` и передать объект `db.User` в качестве входных 
данных. Хотя возможно создать `User` из фактических входных аргументов,
быстрее будет, если мы просто сохраним ожидаемый объект пользователя 
внутри этой структуры сопоставления `eqCreateUserTxParamsMatcher` и 
просто используем его здесь при выполнении этой функции обратного вызова.

```go
type eqCreateUserTxParamsMatcher struct {
	arg      db.CreateUserTxParams
	password string
	user     db.User
}
```

Функция обратного вызова возвращает ошибку, поэтому мы вернем в конце 
`true`, если эта ошибка равна `nil`.

```go
func (expected eqCreateUserTxParamsMatcher) Matches(x interface{}) bool {
	...

	// call the AfterCreate function here
	err = actualArg.AfterCreate(expected.user)

	return err == nil
}
```

Хорошо, теперь мы должны обновить эту функцию-конструктор 
`EqCreateUserTxParams`, чтобы она могла принимать ещё один параметр - 
ожидаемый объект пользователя - и использовала его для создания определённого
нами специального сопоставителя.

```go
func EqCreateUserTxParams(arg db.CreateUserTxParams, password string, user db.User) gomock.Matcher {
	return eqCreateUserTxParamsMatcher{arg, password, user}
}
```

Затем в функции `buildStubs()` мы добавим нужного пользователя в этот
конструктор.

```go
buildStubs: func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor) {
    arg := db.CreateUserTxParams{
        CreateUserParams: db.CreateUserParams{
            Username: user.Username,
            FullName: user.FullName,
            Email:    user.Email,
        },
    }
    store.EXPECT().
        CreateUserTx(gomock.Any(), EqCreateUserTxParams(arg, password, user)).
        Times(1).
        Return(db.CreateUserTxResult{User: user}, nil)

    ...
},
```

И этого должно быть достаточно!

Давайте перезапустим тест, чтобы увидеть, работает он или нет.

```shell
=== RUN   TestCreateUserAPI
=== RUN   TestCreateUserAPI/OK
```

Вау, похоже, что его выполнение займёт слишком много времени! Такое 
ощущение, что тест где-то завис. Если это действительно так, то время ожидания 
истечёт через 30 секунд. И действительно, возникла паника, потому что тест
не смог завершиться вовремя.

```shell
panic: test timed out after 30s
```

Причину такого поведения может быть трудно обнаружить, если вы не до конца 
понимаете, как работает `gomock`. На самом деле проблема возникает из-за 
того, что мы используем один и тот же контроллер как для фиктивного 
хранилища (`store`), так и для фиктивного распределителя задач.

```go
ctrl := gomock.NewController(t)
defer ctrl.Finish()

store := mockdb.NewMockStore(ctrl)
taskDistributor := mockwk.NewMockTaskDistributor(ctrl)
```

В контроллере существует блокирующий механизм. Каждый раз происходит 
блокировка при вызове функции, за которые он отвечает. Поэтому, когда функция 
`CreteUserTx` проверяется на соответствие аргументам, контроллер, отвечающий 
за имитацию, будет заблокирован.

```go
store.EXPECT().
    CreateUserTx(gomock.Any(), EqCreateUserTxParams(arg, password, user)).
    Times(1).
    Return(db.CreateUserTxResult{User: user}, nil)

taskPayload := &worker.PayloadSendVerifyEmail{
    Username: user.Username,
}
taskDistributor.EXPECT().
    DistributeTaskSendVerifyEmail(gomock.Any(), taskPayload, gomock.Any()).
    Times(1).
    Return(nil)
```

```go
// DistributeTaskSendVerifyEmail indicates an expected call of DistributeTaskSendVerifyEmail.
func (mr *MockTaskDistributorMockRecorder) DistributeTaskSendVerifyEmail(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DistributeTaskSendVerifyEmail", reflect.TypeOf((*MockTaskDistributor)(nil).DistributeTaskSendVerifyEmail), varargs...)
}
```

```go
// RecordCallWithMethodType is called by a mock. It should not be called by user code.
func (ctrl *Controller) RecordCallWithMethodType(receiver interface{}, method string, methodType reflect.Type, args ...interface{}) *Call {
	ctrl.T.Helper()

	call := newCall(ctrl.T, receiver, method, methodType, args...)

	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()
	ctrl.expectedCalls.Add(call)

	return call
}
```

Вот почему, когда мы вызываем функцию обратного вызова `AfterCreate()`,
она больше не может заблокировать контроллер при вызове фиктивного 
распределителя задач. Чтобы исправить это, мы можем просто использовать 
два разных контроллера: один для фиктивного хранилища (`store`), а 
другой — для фиктивного распределителя задач.

```go
t.Run(tc.name, func(t *testing.T) {
    storeCtrl := gomock.NewController(t)
    defer storeCtrl.Finish()
    store := mockdb.NewMockStore(storeCtrl)

    taskCtrl := gomock.NewController(t)
    defer taskCtrl.Finish()
    taskDistributor := mockwk.NewMockTaskDistributor(taskCtrl)

    tc.buildStubs(store, taskDistributor)
    server := newTestServer(t, store, taskDistributor)
    res, err := server.CreateUser(context.Background(), tc.req)
    tc.checkResponse(t, res, err)
})
```

Вот так вот и этого будет достаточно. На этот раз я предполагаю, что тест
будет успешно пройден.

Давайте запустим его ещё раз!

```shell
=== RUN   TestCreateUserAPI
=== RUN   TestCreateUserAPI/OK
--- PASS: TestCreateUserAPI (0.19s)
    --- PASS: TestCreateUserAPI/OK (0.12s)
PASS
ok      github.com/MaksimDzhangirov/backendBankExample/gapi     0.521s
```

И это действительно так! Отлично!

Теперь, чтобы убедиться, что тест может обнаружить неправильную 
реализацию, давайте представим, что мы забыли создать задачу для отправки 
электронных писем. Я закомментирую весь код внутри функции обратного 
вызова `AfterCreate`. И просто верну здесь `nil`.

```go
AfterCreate: func(user db.User) error {
    // taskPayload := &worker.PayloadSendVerifyEmail{
    // 	Username: user.Username,
    // }
    // opts := []asynq.Option{
    // 	asynq.MaxRetry(10),
    // 	asynq.ProcessIn(10 * time.Second),
    // 	asynq.Queue(worker.QueueCritical),
    // }

    // return server.taskDistributor.DistributeTaskSendVerifyEmail(ctx, taskPayload, opts...)
    return nil
},
```

На этот раз, если тест достаточно строгий, он должен завершиться с ошибкой.
Запустим его, чтобы проверить так ли это!

```shell
=== RUN   TestCreateUserAPI
=== RUN   TestCreateUserAPI/OK
...
--- FAIL: TestCreateUserAPI (0.18s)
    --- FAIL: TestCreateUserAPI/OK (0.11s)
FAIL
FAIL    github.com/MaksimDzhangirov/backendBankExample/gapi     0.414s
FAIL
```

так и произошло, тест не был пройден из-за отсутствия вызова функции 
`DistributeTaskSendVerifyEmail`.

Вот так мы реализовали unit тест для gRPC API, который включает в себя 
несколько фиктивных сущностей, взаимодействующих друг с другом. Однако 
тест, который мы только что написали, предназначен только для успешного 
случая.

Попробуйте сами добавить сюда ещё несколько тестовых случаев, когда по 
какой-то причине возникает ошибка в работе gRPC API? Сделать это довольно 
просто, не так ли?

Давайте реализуем случай "internal server error" («внутренней ошибки 
сервера»). Я продублирую этот "OK" (успешный) случай,

```go
{
    name: "OK",
    req: &pb.CreateUserRequest{
        Username: user.Username,
        Password: password,
        FullName: user.FullName,
        Email:    user.Email,
    },
    buildStubs: func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor) {
        arg := db.CreateUserTxParams{
            CreateUserParams: db.CreateUserParams{
                Username: user.Username,
                FullName: user.FullName,
                Email:    user.Email,
            },
        }
        store.EXPECT().
            CreateUserTx(gomock.Any(), EqCreateUserTxParams(arg, password, user)).
            Times(1).
            Return(db.CreateUserTxResult{User: user}, nil)

        taskPayload := &worker.PayloadSendVerifyEmail{
            Username: user.Username,
        }
        taskDistributor.EXPECT().
            DistributeTaskSendVerifyEmail(gomock.Any(), taskPayload, gomock.Any()).
            Times(1).
            Return(nil)
    },
    checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
        require.NoError(t, err)
        require.NotNil(t, res)
        createdUser := res.GetUser()
        require.Equal(t, user.Username, createdUser.Username)
        require.Equal(t, user.FullName, createdUser.FullName)
        require.Equal(t, user.Email, createdUser.Email)
    },
},
```

и изменю название на "InternalError".

Объект запроса будет таким же, но в функции `buildStubs()` нам не нужно 
создавать реальный аргумент, который нужно передать в транзакцию CreateUser.
Мы можем просто использовать здесь сопоставление `gomock.Any()` и имитировать
результат этого вызова пустым `db.CreateUserTxResult` и ошибкой, отличной 
от `nil`, например, `sql.ErrConnDone`. А поскольку в этом случае транзакция
не была выполнена, мы ожидаем, что `DistributeTaskSendVerifyEmail` будет 
вызвана ноль раз (`Times(0)`), независимо от того, какие аргументы были 
переданы в качестве входных.

```go
buildStubs: func(store *mockdb.MockStore, taskDistributor *mockwk.MockTaskDistributor) {
    store.EXPECT().
        CreateUserTx(gomock.Any(), gomock.Any()).
        Times(1).
        Return(db.CreateUserTxResult{}, sql.ErrConnDone)

    taskDistributor.EXPECT().
        DistributeTaskSendVerifyEmail(gomock.Any(), gomock.Any(), gomock.Any()).
        Times(0)
},
```

Затем в функции `checkResponse()` мы ожидаем, что ошибка будет не равна 
`nil`. Нет необходимости проверять структуру ответа, но мы должны проверить 
код ошибки, чтобы убедиться, что это действительно внутренняя ошибка 
сервера. Для этого мы просто вызываем `status.FromError()` и передаем объект 
`err`. "status" является подпакетом gRPC фреймворка. И эта функция 
вернет объект `status.Status` вместе с логическим значением, сообщающим нам,
было ли преобразование успешным или нет. Я сохраню их в переменных `st` 
и `ok`. Затем потребуем, чтобы значение `ok` было равно `true`, а 
`st.Code()` — `codes.Internal`.

```go
checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
    require.Error(t, err)
    st, ok := status.FromError(err)
    require.True(t, ok)
    require.Equal(t, codes.Internal, st.Code())
},
```

И на этом по сути всё!

Давайте запустим тесты!

```shell
=== RUN   TestCreateUserAPI/OK
=== RUN   TestCreateUserAPI/InternalError
--- PASS: TestCreateUserAPI (0.24s)
    --- PASS: TestCreateUserAPI/OK (0.12s)
    --- PASS: TestCreateUserAPI/InternalError (0.05s)
PASS
ok      github.com/MaksimDzhangirov/backendBankExample/gapi     0.565s
```

Оба успешно пройдены! И для случая "OK", и для "InternalError".

Превосходно!

Есть ещё несколько случаев, которые мы можем протестировать, например: 
когда имя пользователя уже существует или когда предоставленные входные 
аргументы не соответствуют принятым ограничениям. Но я оставляю это в 
качестве упражнения, которое вы можете выполнить самостоятельно.

Вы всегда можете просмотреть мой код на GitHub, если хотите увидеть, как 
я их реализовал.

И на этом закончим сегодняшнюю лекцию о том как написать unit тесты для 
gRPC сервисов. Надеюсь, она была интересной и полезной для вас. Большое 
спасибо за время, потраченное на чтение, желаю вам получать удовольствие 
от обучения и до встречи на следующей лекции!