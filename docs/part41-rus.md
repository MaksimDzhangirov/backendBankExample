# Как запустить gRPC сервер на Golang и вызвать его API

[Оригинал](https://www.youtube.com/watch?v=BkfBJIS0_ro)

Привет, ребята, рад вас снова видеть на мастер-классе по бэкенду. На 
[предыдущей лекции](part40-rus.md) мы научились определять gRPC API с 
помощью `protobuf` и генерировать на его основе Golang код. А сегодня 
давайте узнаем, как использовать сгенерированный код для запуска gRPC 
сервера, и затем подключиться к нему с помощью интерактивного клиентского 
инструмента под названием `Evans`. Итак, давайте начнём!

## Реализуем сервисы, используя gRPC фреймворк

Если вы еще помните, ранее мы реализовали наши веб-сервисы на базе
HTTP JSON API, используя Gin фреймворка. И этот код хранятся внутри 
пакета `api`, как показано на рисунке.

![](../images/part41/1.png)

Теперь мы хотим реализовать тот же набор сервисов, но с использованием gRPC 
фреймворка. Поэтому я собираюсь создать новый отдельный пакет для этого. 

Назовём его `gapi`. И внутри этого пакета я создам новый файл: `server.go`.
Этот файл будет содержать структуру `Server`, похожую на структуру Gin 
`Server`, которую мы реализовали ранее. Единственная разница в том, что мы 
будем обслуживать gRPC запросы вместо HTTP. Итак, я скопирую этот код 
функции `NewServer` из этого файла `api/server.go` и вставлю его в наш 
новый файл `server.go`.

```go
// Server обслуживает gRPC запросы нашего банковского сервиса.
type Server struct {
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
	router     *gin.Engine
}

// NewServer создаёт новый HTTP сервер и настраивает маршрутизацию.
func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("currency", validCurrency)
	}

	server.setupRouter()
	return server, nil
}

func (server *Server) setupRouter() {
	router := gin.Default()

	router.POST("/users", server.createUser)
	router.POST("/users/login", server.loginUser)
	router.POST("/token/renew_access", server.renewAccessToken)

	router.POST("/accounts", server.createAccount)
	router.GET("/accounts/:id", server.getAccount)
	router.GET("/accounts", server.listAccounts)

	router.POST("/transfers", server.createTransfer)

	server.router = router
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}
```

Этот сервер будет обслуживать gRPC запросы для нашего банковского сервиса.
Нам по-прежнему нужны все эти поля для хранения объектов `config`, `store`
для БД и `tokenMaker`. Они будут использоваться позже, когда мы реализуем 
RPC. Но теперь нам больше не нужен механизм валидации, потому что он не 
используется gRPC фреймворком, в отличие от Gin. И настройку маршрутизации
тоже можно убрать, потому что, в отличие от HTTP, в gRPC нет маршрутов.

```go
// Server обслуживает gRPC запросы нашего банковского сервиса.
type Server struct {
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
}

// NewServer создаёт новый HTTP сервер и настраивает маршрутизацию.
func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
	}

	return server, nil
}
```

Клиент обратится к серверу, просто выполнив RPC, так, как будто он вызывает 
локальную функцию.

Итак, теперь у нас есть функция для создания нового сервера. Однако она 
пока не является gRPC сервером. Для того чтобы она им стала, нам нужно 
использовать код, который `protoc` сгенерировал для нас в 
[предыдущей лекции](part40-rus.md). Давайте взглянем на файл 
`service_simple_bank_grpc.pb.go`. Там вы увидите интерфейс 
`SimpleBankServer`. Наш сервер может стать gRPC сервером только тогда, 
когда он реализует этот интерфейс. И ещё обратите внимание здесь на
наличие метода: `mustEmbedUnimplementedSimpleBankServer()`. Для чего
он нужен?

На самом деле, в последних версиях gRPC, кроме интерфейса сервера,
`protoc` также генерирует эту структуру
`UnimplementedSimpleBankServer`, где уже созданы все RPC функции, но
все они возвращают ошибку `codes.Unimplemented`. И мы должны добавить 
`UnimplementedSimpleBankServer` в нашу структуру `Server`, следующим 
образом.

```go
// Server обслуживает gRPC запросы нашего банковского сервиса.
type Server struct {
	pb.UnimplementedSimpleBankServer
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
}
```

Это необходимо, чтобы включить прямую совместимость, т. е. сервер
уже сможет принимать RPC вызовы `CreateUser` и `LoginUser` до того, как 
они будут фактически реализованы. Затем мы можем постепенно добавить их 
реализации позднее. Я думаю, что это полезная особенность, облегчающая 
команде работу над несколькими RPC параллельно, не блокируя и не 
конфликтуя друг с другом.

Хорошо, теперь, чтобы показать вам, что наши RPC уже могут принимать 
вызовы от клиента, давайте попробуем запустить gRPC сервер и вызвать 
его API.

В файле `main.go` мы сейчас запускаем HTTP Gin сервер. Я оставлю его
для тех, кто будет изучать курс в будущем в качестве примера, поэтому
создам отдельную функцию для запуска Gin сервера. Затем переместим этот 
фрагмент кода в новую функцию.

```go
func runGinServer(config util.Config, store db.Store) {
	server, err := api.NewServer(config, store)
	if err != nil {
		log.Fatal("cannot create server:", err)
	}

	err = server.Start(config.ServerAddress)
	if err != nil {
		log.Fatal("cannot start server:", err)
	}
}
```

Эта функция должна будет принимать 2 параметра: объект типа `util.Config`
и `db.Store`.

Хорошо, теперь я объявлю другую функцию для запуска gRPC сервера с той 
же сигнатурой. Затем в функции `main` мы можем вызвать
`runGrpcServer` и передать `config` и `store` для БД.

```go
func main() {
	...
	runGrpcServer(config, store)
}

func runGrpcServer(config util.Config, store db.Store) {

}
```

Вы можете легко изменить её на `runGinServer`, если хотите вместо 
этого запустить HTTP сервис. Хорошо, теперь давайте реализуем функцию 
`runGrpcServer()`.

Сначала мы должны вызвать `grpc.NewServer`, чтобы создать новый объект
gRPC сервера. Затем мы вызываем `pb.RegisterSimpleBankServer` с
этим объектом gRPC сервера в качестве первого параметра. Второй параметр, 
который нам нужно передать, — это наша собственная реализация простого 
банковского сервера.

```go
func runGrpcServer(config util.Config, store db.Store) {
	grpcServer := grpc.NewServer()
	
	pb.RegisterSimpleBankServer(grpcServer, server)
}
```

Таким образом, мы должны создать её здесь. Это будет похоже на то как
мы создавали новый сервер для Gin, поэтому я скопирую эти строки.

```go
grpcServer := grpc.NewServer()
server, err := api.NewServer(config, store)
if err != nil {
    log.Fatal("cannot create server:", err)
}
pb.RegisterSimpleBankServer(grpcServer, server)
```

Но мы должны изменить имя пакета с `api` на `gapi`, потому что 
именно там мы определяем объект для gRPC сервера нашего простого
банковского приложения. Я собираюсь немного изменить код, чтобы его 
было легче читать.

Давайте переместим переменную `grpcServer` сюда, непосредственно 
перед вызовом `RegisterSimpleBankServer`.

```go
func runGrpcServer(config util.Config, store db.Store) {
	server, err := gapi.NewServer(config, store)
	if err != nil {
		log.Fatal("cannot create server:", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterSimpleBankServer(grpcServer, server)
}
```

Следующий шаг необязателен, но я настоятельно рекомендую вам его 
сделать, то есть зарегистрировать gRPC reflection для нашего сервера.
Эта команда выглядит очень просто, но на самом деле оказывает 
значительное влияние, поскольку позволяет gRPC клиенту легко 
узнать, какие RPC доступны на сервере и как их вызывать. Представьте
будто она автоматически генерирует некую документацию для
сервера.

```go
func runGrpcServer(config util.Config, store db.Store) {
    ...
    
	grpcServer := grpc.NewServer()
	pb.RegisterSimpleBankServer(grpcServer, server)
	reflection.Register(grpcServer)
}
```

Теперь самый важный шаг: запустить сервер для прослушивания gRPC запросов 
на определенном порту. В нашем файле `app.env` на данный момент у нас есть 
только адрес сервера для HTTP-запросов. Итак, сначала я изменю название 
этой переменной на `HTTP_SERVER_ADDRESS`. Затем нам нужно будет добавить 
сюда еще одну переменную для адреса gRPC сервера. HTTP-запросы 
обслуживаются через порт `8080`, поэтому, допустим, мы будем 
обслуживать gRPC запросы через порт `9090`.

```
GRPC_SERVER_ADDRESS=0.0.0.0:9090
```

Вы можете сами выбрать номер порта. Вам не обязательно использовать те же 
значения, что и у меня. Так как мы обновили переменные окружения, мы 
должны обновить нашу структуру `config`, чтобы отразить это изменение.

Во-первых, тут `mapstructure:"SERVER_ADDRESS"` изменим название на
`HTTP_SERVER_ADDRESS`. Затем я добавлю еще одно поле для адреса gRPC 
сервера.

```go
type Config struct {
	...
	HTTPServerAddress    string        `mapstructure:"HTTP_SERVER_ADDRESS"`
	GRPCServerAddress    string        `mapstructure:"GRPC_SERVER_ADDRESS"`
	...
}
```

Хорошо, теперь вернёмся к файлу `main.go`. В функции `runGinServer()`
мы должны изменить адрес на `config.HTTPServerAddress`. Затем в 
`runGrpcServer()` я создам новый слушатель с `net.Listen()`, передам 
`tcp` в качестве протокола и `config.GRPCServerAddress`. Этот вызов может 
вернуть ошибку. Итак, если ошибка не `nil`, мы запишем сообщение в лог, 
что "cannot create listener" («невозможно создать слушатель») и 
завершаем работу.

```go
func runGrpcServer(config util.Config, store db.Store) {
	...

	listener, err := net.Listen("tcp", config.GRPCServerAddress)
    if err != nil {
        log.Fatal("cannot create listener")
    }
}
```

В противном случае мы пишем в лог сообщение о том, что мы собираемся
запустить сервер gRPC по этому адресу. Теперь все, что нам нужно 
сделать, чтобы запустить сервер, это вызвать `grpcServer.Serve()` и 
передать `listener` в качестве входных данных. Если этот вызов возвращает
не `nil` ошибку, то мы просто пишем в лог сообщение, что "cannot
start gRPC server" («невозможно запустить gRPC сервер».) и завершаем 
работу.

```go
func runGrpcServer(config util.Config, store db.Store) {
    ...
	log.Printf("start gRPC server at %s", listener.Addr().String())
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatal("cannot start gRPC server")
	}
}
```

И в принципе этого должно быть достаточно! Теперь наш сервер можно 
запустить.

Давайте откроем терминал и выполним

```shell
make server
2022/04/10 11:47:38 start gRPC server at [::]:9090
```

Ошибок не возникло, значит gRPC сервер успешно запущен на порту 9090.

## Устанавливаем и используем Evans для тестирования

Теперь попробуем вызвать его API. Для тестирования я буду использовать 
инструмент под названием Evans.

Evans — очень крутой gRPC клиент, который позволяет вам создавать и 
отправлять gRPC запросы в интерактивной консоли. На их 
[странице Github](https://github.com/ktr0731/evans) вы можете легко 
найти бинарные файлы для Mac, Linux и Windows. Поскольку я на Mac, я 
предпочту установить его с помощью Homebrew. Итак, сначала давайте 
запустим эту

```shell
brew tab ktr0731/evans
```

команду.

А затем

```shell
brew install evans
```

![](../images/part41/2.png)

Хорошо, как видно на рисунке, теперь Evans успешно установлен. И поскольку 
на нашем сервере уже включено gRPC reflection, мы можем запустить эту 
команду

```shell
evans -r repl
evans: failed to run REPL mode: failed to instantiate a new spec: failed to 
instantiate the spec: failed to list packages by gRPC reflection: failed to 
list services from reflecton enabled gRPC server: rpc error: code = Unavailable
desc = connection error: desc = "transport: Error while dialing dial tcp 
127.0.0.1:50051: connect: connection refused"
```

для подключения к серверу.

Ой, возникла ошибка. Она связана с тем, что Evans пытается подключиться к
gRPC порту по умолчанию: 50051, тогда как на самом деле наш сервер вместо 
этого прослушивает порт 9090. Поэтому нам нужно добавить еще несколько 
параметров в команду. Мы можем добавить хост: в данном случае localhost, 
затем порт, который должен быть равен 9090.

```shell
evans --host localhost --port 9090 -r repl

  ______
 |  ____|
 | |__    __   __   __ _   _ __    ___
 |  __|   \ \ / /  / _. | | '_ \  / __|
 | |____   \ V /  | (_| | | | | | \__ \
 |______|   \_/    \__,_| |_| |_| |___/

 more expressive universal gRPC client


pb.SimpleBank@localhost:9090> 
```

И вуаля, теперь мы внутри консоли Evans и можем взаимодействовать с 
сервером. Мы можем запустить

```shell
show service
+------------+------------+-------------------+--------------------+
|  SERVICE   |    RPC     |   REQUEST TYPE    |   RESPONSE TYPE    |
+------------+------------+-------------------+--------------------+
| SimpleBank | CreateUser | CreateUserRequest | CreateUserResponse |
| SimpleBank | LoginUser  | LoginUserRequest  | LoginUserResponse  |
+------------+------------+-------------------+--------------------+
```

чтобы просмотреть все сервисы и RPC доступные на сервере.

Как видите, мы еще не реализовали никаких API, но уже доступы два
RPC: `CreateUser` и `LoginUser`. Всё потому, что мы встроили структуру 
`UnimplementedSimpleBankServer` в нашу структуру `Server` ранее. Теперь 
попробуем вызвать `CreateUser` RPC.

```shell
call CreateUser
username (TYPE_STRING) => 
```

Как видите, Evans теперь запрашивает некоторые входные данные, которые 
требуются RPC. Итак, давайте введем имя пользователя, его имя и фамилию, 
адрес электронной почты и пароль. Как только мы ввели последний 
параметр, запрос будет отправлен на сервер.

```shell
call CreateUser
username (TYPE_STRING) => quang
full_name (TYPE_STRING) => Quang
email (TYPE_STRING) => quang@gmail.com
password (TYPE_STRING) => secret
command call: rpc error: code = Unimplemented desc = method CreateUser not implemented
```

И мы немедленно получаем ответ. Но, конечно, ошибку, потому что метод 
`CreateUser` еще не реализован на сервере.

Однако этого достаточно, чтобы показать, что gRPC сервер уже работает и 
может принимать gRPC запросы от клиента. Мы можем выполнить команду

```shell
exit
Good Bye :)
```

чтобы выйти из консоли Evans.

Теперь я добавлю команду `evans` в Makefile, чтобы мы могли легко 
запустить его для тестирования наших gRPC API-интерфейсов позже.

```makefile
evans:
	evans --host localhost --port 9090 -r repl
```

С помощью этой команды для входа в консоль Evans нам достаточно запустить

```shell
make evans
```

И на этом закончим сегодняшнюю лекцию. Мы успешно запустили gRPC сервер 
и вызвали его gRPC API с помощью клиента Evans. Однако пока что
всё RPC для создания и входа пользователя в систему не реализованы.
Они по-прежнему возвращают код ошибки по умолчанию.

Поэтому на следующей лекции я покажу вам как реализовать их.

А пока желаю вам получать удовольствие от обучения и до встречи на 
следующей лекции!