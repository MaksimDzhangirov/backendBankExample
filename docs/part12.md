# Загружаем настройки из файла и переменных окружения в Golang с помощью Viper

[Оригинал](https://dev.to/techschoolguru/implement-restful-http-api-in-go-using-gin-4ap1)

При разработке и развертывании бекенд веб-приложения нам обычно нам обычно 
приходится использовать разные настройки для разных окружений, таких как 
разработка, тестирование, стейдж и продакжен.

Сегодня мы узнаем, как использовать [Viper](https://github.com/spf13/viper) для 
загрузки настроек из файла или переменных окружения.

Ниже:
* Ссылка на [плейлист с видео лекциями на Youtube](https://bit.ly/backendmaster)
* И на [Github репозиторий](https://github.com/techschool/simplebank)

## Почему мы используем файлы и переменные окружения

![](../images/part12/2bh5o1lqfxkzapnzpl4s.png)

Чтение значений из файла позволяет нам легко указать настройки по умолчанию 
для локальной разработки и тестирования.

В то же время чтение значений из переменных окружения поможет нам 
переопределить настройки по умолчанию при развертывании нашего приложения в 
стейдж или продакшен окружении с использованием docker контейнеров.

## Почему Viper

![](../images/part12/lfgy1v72y2agoib2j4ge.png)

[Viper](https://github.com/spf13/viper) — очень популярный пакет Golang для 
этого.

* Он может находить, загружать и распаковывать значения из файла настроек.
* Он поддерживает различные типов файлов, таких как JSON, TOML, YAML, ENV или 
  INI.
* Он также может считывать значения из переменных окружения или флагов командной 
  строки.
* Это позволяет нам устанавливать или переопределять значения по умолчанию.
* Более того, если вы предпочитаете хранить свои настройки в удаленной системе, 
  такой как [Etcd](https://etcd.io/) или [Consul](https://www.consul.io/), то 
  вы можете использовать Viper для чтения данных из них напрямую.
* Его можно использовать как для незашифрованных, так и для зашифрованных 
  значений.
* Еще одна интересная особенность Viper заключается в том, что он может 
  отслеживать изменения в файле настроек и уведомлять об этом приложение.
* Мы также можем использовать Viper для сохранения любых изменений 
  настроек, которые мы внесли в файл.

Много полезных функций, не так ли?

## Что мы изменим

Сейчас в нашем простом банковском приложении мы жестко закодировали 
некоторые константы для `dbDriver`, `dbSource` в файле `main_test.go`, а также 
ещё одну константу для `serverAddress` в файле `main.go`.

```go
const (
    dbDriver      = "postgres"
    dbSource      = "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable"
    serverAddress = "0.0.0.0:8080"
)
```

Итак, на этой лекции мы узнаем, как использовать Viper для чтения этих 
настроек из файлов и переменных окружения.

## Устанавливаем Viper

Хорошо, давайте начнем с установки Viper! Откройте браузер и осуществите
поиск по ключевым словам `golang viper`.

Затем откройте его [Github страницу](https://github.com/spf13/viper). Прокрутите немного вниз, скопируйте эту 
команду `go get` и запустите ее в терминале, чтобы установить пакет:

```shell
❯ go get github.com/spf13/viper
```

После этого в файле `go.mod` [нашего проекта](https://github.com/techschool/simplebank) 
мы видим, что Viper был добавлен в качестве зависимости.

## Создаём файл с настройками

Сейчас я собираюсь создать новый файл `app.env` для хранения наших значений 
настроек для разработки. Затем скопируем эти переменные из файла 
`main.go` и вставим в этот файл. Поскольку мы используем расширение файла 
env, мы должны изменить способ объявления этих переменных. Он будет похож на 
тот, который используется для переменные окружения:

* Каждая переменная должна быть объявлена на отдельной строке.
* Имя переменной написано в верхнем регистре, а отдельные слова в ней 
  разделены символом подчеркивания.
* Значением переменной определяется после имени и символа равенства.

```env
DB_DRIVER=postgres
DB_SOURCE=postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable
SERVER_ADDRESS=0.0.0.0:8080
```

Вот и все! Этот файл `app.env` теперь содержит настройки по умолчанию для 
нашей локальной среды разработки. Затем мы будем использовать Viper для 
загрузки этого файла с настройками.

## Загружаем файл с настройками

Давайте создадим новый файл `config.go` внутри пакета `util`.

Затем объявите в этом файле структуру `Config`. Эта структура `Config` 
будет содержать все переменные настроек приложения, которые мы считываем 
из файла или переменных окружения. На данный момент мы используем только 3 
переменные:

* Во-первых, `DBDriver`,
* Во-вторых, `DBSource`,
* И, наконец, `ServerAddress`.

Все три переменные имеют тип string.

```go
type Config struct {
    DBDriver      string `mapstructure:"DB_DRIVER"`
    DBSource      string `mapstructure:"DB_SOURCE"`
    ServerAddress string `mapstructure:"SERVER_ADDRESS"`
}
```

Чтобы получить значение переменных и сохранить их в этой структуре, нам 
нужно использовать функционал преобразования Viper. Viper использует пакет 
`mapstructure` под капотом для преобразования значений, поэтому мы 
используем дескрипторы `mapstructure`, чтобы указать имя каждой переменной в
файле с настройками.

В нашем случае имя каждой переменной должно совпадать с тем, которое было 
объявлено в файле `app.env`. Например, имя дескриптора поля `DBDriver` должно 
быть `DB_DRIVER`, имя дескриптора поля `DBSource` должно быть `DB_SOURCE`, и, по
аналогии для `ServerAddress`, должно быть `SERVER_ADDRESS`.

Хорошо, теперь я собираюсь определить новую функцию `LoadConfig()`, которая 
принимает путь в качестве входных данных и возвращает объект с настройками 
или ошибку. Эта функция будет считывать настройки из файла по указанному пути, 
если он существует, или переопределять их значения с помощью переменных 
окружения, если они переданы.

```go
func LoadConfig(path string) (config Config, err error) {
    viper.AddConfigPath(path)
    viper.SetConfigName("app")
    viper.SetConfigType("env")

    ...
}
```

Во-первых, мы вызываем `viper.AddConfigPath()`, чтобы сообщить Viper 
расположение файла с настройками. В нашем случае путь к файлу определяется
входным аргументом `path`.

Затем мы вызываем `viper.SetConfigName()`, чтобы указать Viper искать файл 
с определенным именем. Мы используем файл `app.env`, поэтому его имя — `app`.

Мы также сообщаем Viper тип файла с настройками, в данном случае `env`, 
вызывая `viper.SetConfigFile()` и передавая `env`. Вы также можете 
использовать здесь JSON, XML или любой другой формат, просто убедитесь, что 
ваш файл с настройками имеет правильный формат и расширение.

Теперь, помимо чтения настроек из файла, мы также хотим, чтобы Viper 
считывал значения из переменных окружения. Поэтому мы вызываем 
`viper.AutomaticEnv()`, чтобы указать Viper автоматически переопределять 
значения, которые он прочитал из файла с настройками, значениями 
соответствующих переменных окружения, если они существуют.

```go
// LoadConfig считывает настройки из файла или переменных окружения.
func LoadConfig(path string) (config Config, err error) {
    viper.AddConfigPath(path)
    viper.SetConfigName("app")
    viper.SetConfigType("env")

    viper.AutomaticEnv()

    err = viper.ReadInConfig()
    if err != nil {
        return
    }

    err = viper.Unmarshal(&config)
    return
}
```

После этого мы вызываем `viper.ReadInConfig()`, чтобы начать считывание 
значений настроек. Если ошибка не равна `nil`, то мы просто возвращаем её.

В противном случае мы вызываем `viper.Unmarshal()` для преобразования 
значений в переданный объект. И, наконец, мы просто возвращаем объект 
с настройками и любую ошибку, если она произойдет.

Итак, в принципе функцию для загрузки настроек уже можно применять. 
Теперь мы можем воспользоваться ею в файле `main.go`.

## Используем LoadConfig в функции main

Давайте удалим все предыдущие жестко заданные в коде значения. Затем в 
функции `main()` вызовем `util.LoadConfig()` и передадим в неё `"."`, что 
означает текущую папку, так как наш файл с настройками `app.env` находится в 
том же месте, что и этот файл `main.go`.

Если возникла ошибка, то мы просто пишем сообщение в лог о том, что не удаётся
загрузить настройки и завершаем работу приложения. В противном случае значения
из файла с настройками записываются в `config.DBDriver`, `config.DBSource` и 
`config.ServerAdress`.

```go
func main() {
    config, err := util.LoadConfig(".")
    if err != nil {
        log.Fatal("cannot load config:", err)
    }

    conn, err := sql.Open(config.DBDriver, config.DBSource)
    if err != nil {
        log.Fatal("cannot connect to db:", err)
    }

    store := db.NewStore(conn)
    server := api.NewServer(store)

    err = server.Start(config.ServerAddress)
    if err != nil {
        log.Fatal("cannot start server:", err)
    }
}
```

Готово! Попробуем запустить этот сервер.

```shell
❯ make server
go run main.go
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] POST   /accounts                 --> github.com/techschool/simplebank/api.(*Server).createAccount-fm (3 handlers)
[GIN-debug] GET    /accounts/:id             --> github.com/techschool/simplebank/api.(*Server).getAccount-fm (3 handlers)
[GIN-debug] GET    /accounts                 --> github.com/techschool/simplebank/api.(*Server).listAccount-fm (3 handlers)
[GIN-debug] Listening and serving HTTP on 0.0.0.0:8080
```

Он заработал! Сервер прослушивает порт 8080 localhost, как мы и указали в 
файле `app.env`.

Давайте откроем Postman и отправим запрос к API. Я вызову API для получения 
списка счетов.

![](../images/part12/litml0g2ppnh6s52v6dh.jpeg)

К сожалению, возникла ошибка `500 Internal Server Error`. Она связана с 
тем, что наш веб-сервер не может подключиться к базе данных Postgres через 
порт `5432`. Давайте откроем терминал и проверим, запущен ли Postgres:

```
❯ docker ps
```

![](../images/part12/mxselwizllznarp2qmd9.jpeg)

Контейнеры не запущены. Если мы выполним команду `docker ps -a`, то увидим, 
что работа контейнера Postgres была завершена. Итак, мы должны запустить его, 
выполнив:

```shell
❯ docker start postgres12
```

Хорошо, теперь база данных запущена и работает. Вернемся к Postman и снова 
отправим запрос.

![](../images/part12/jalhvyywwcpodx5p0s7e.jpeg)

На этот раз он выполнен успешно. Мы получили список счетов. Итак, наш код для 
загрузки настроек из файла прекрасно работает.

Давайте попробуем переопределить эти настройки с помощью переменных окружения. 
Я задам значение для переменной `SERVER_ADDRESS` равное `localhost:8081` 
перед вызовом `make server`.

```shell
❯ SERVER_ADDRESS=0.0.0.0:8081 make server
go run main.go
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] POST   /accounts                 --> github.com/techschool/simplebank/api.(*Server).createAccount-fm (3 handlers)
[GIN-debug] GET    /accounts/:id             --> github.com/techschool/simplebank/api.(*Server).getAccount-fm (3 handlers)
[GIN-debug] GET    /accounts                 --> github.com/techschool/simplebank/api.(*Server).listAccount-fm (3 handlers)
[GIN-debug] Listening and serving HTTP on 0.0.0.0:8081
```

Здесь мы видим, что сервер теперь прослушивает порт 8081 вместо 8080, как раньше. 
Теперь, если мы попытаемся отправить тот же запрос API в Postman на порт 8080, 
то получим ошибку отказа в соединении:

![](../images/part12/68002w5oekob1ns7qdkh.jpeg)

Только когда мы изменим этот порт на 8081, запрос будет успешным:

![](../images/part12/bejjeuf5mnvh7a7vesz4.jpeg)

Таким образом, мы можем сделать вывод, что Viper успешно переопределил 
значения, считанные из файла с настройками, на переменные окружения. Это очень 
удобно, когда в будущем мы захотим развернуть приложение в различных окружениях, 
таких как стейдж или продакшен.

## Используем LoadConfig в тестах

Теперь, прежде чем мы закончим, давайте обновим файл `main_test.go`, чтобы 
воспользоваться новой функцией `LoadConfig()`.

Во-первых, удалите все жестко закодированные константы. Затем в функции 
`TestMain()` мы вызываем `util.LoadConfig()`.

Но на этот раз файл `main_test.go` находится внутри папки `db/sqlc`, а файл 
с настройками `app.env` — в корне репозитория, поэтому мы должны указать такой 
относительный путь: `«../..»`. Эти две точки `..` означают переход в 
родительскую папку.

```go
func TestMain(m *testing.M) {
    config, err := util.LoadConfig("../..")
    if err != nil {
        log.Fatal("cannot load config:", err)
    }

    testDB, err = sql.Open(config.DBDriver, config.DBSource)
    if err != nil {
        log.Fatal("cannot connect to db:", err)
    }

    testQueries = New(testDB)

    os.Exit(m.Run())
}
```

Если ошибка не равна `nil`, то просто пишем её в лог и завершаем работу 
приложения. В противном случае мы должны изменить `DBDriver` и `DBSource` на 
`config.DBDriver` и `config.DBSource`.

Вот и все! Все необходимые изменения внесены. Давайте запустим тест 
всего пакета.

![](../images/part12/pite0yu09jpnuj4qjx6i.jpeg)

Они успешно пройдены! На этом завершим лекцию о чтении настроек из файла
и переменных окружения.

Я настоятельно рекомендую вам ознакомиться с документацией и попробовать 
некоторые другие интересные функции Viper, такие как [отслеживание значений 
в реальном времени](https://github.com/spf13/viper#watching-and-re-reading-config-files) 
или [считывание из удаленной системы](https://github.com/spf13/viper#remote-keyvalue-store-support).

Желаю вам получать удовольствие от написания кода и увидимся на следующей 
лекции!