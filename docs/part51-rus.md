# Добавляем авторизацию для защиты gRPC API

[Оригинал](https://www.youtube.com/watch?v=_jqNs3d99ps)

Всем привет, рад вас снова видеть на мастер-классе по бэкенду! На 
предыдущей лекции мы реализовали `UpdateUser` RPC, однако, он ещё не
защищён, поэтому любой может вызвать API для изменения данных любого 
другого пользователя. Итак, сегодня давайте узнаем, как добавить слой
авторизации для защиты этого API и гарантировать, что только владелец 
может изменять информацию, связанную с его учётной записью.

## Добавляем слой авторизации

Хорошо, давайте начнём с создания нового файла, назвав его 
`authorization.go` внутри пакета `gapi`. В этом файле я добавлю новый
метод в структуру `Server`. Давайте назовем его `authorizeUser()`.
Эта функция примет объект контекста в качестве входных данных и 
вернет объект `token.Payload` и ошибку в качестве результата.

```go
package gapi

import (
	"context"
	"github.com/MaksimDzhangirov/backendBankExample/token"
)

func (server *Server) authorizeUser(ctx context.Context) (*token.Payload, error) {
	
}
```

Если вы ещё помните `token.Payload` — это объект, который мы использовали 
для создания токена доступа.

```go
type Payload struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiredAt time.Time `json:"expired_at"`
}
```

Он содержит некую информацию о пользователе, такую как имя пользователя и 
время истечения срока действия токена. Итак, в этой функции мы проверим 
токен доступа, чтобы убедиться, что он действителен. И если это так, мы 
вернем полезную нагрузку токена RPC обработчику, чтобы сообщить ему, 
какой пользователь вызывает API. Обычно токен доступа отправляется 
клиентом внутри метаданных, поэтому сначала мы должны вызвать 
`metadata.FromIncomingContext()`, чтобы получить данные, хранящиеся в 
контексте. Эта функция вернет объект метаданных и логическое значение `ok`.
Если `ok` равно `false`, то значит метаданные не предоставлены. В этом 
случае мы просто возвращаем полезную нагрузку в виде `nil` и 
ошибку: "missing metadata" («отсутствуют метаданные»).

```go
func (server *Server) authorizeUser(ctx context.Context) (*token.Payload, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing metadata")
	}
}
```

То, что мы делаем сейчас, на самом деле очень похоже на то, что мы делали 
в лекции 44, где мы извлекали агента пользователя и IP-адрес клиента из 
метаданных. Таким образом, мы можем использовать функцию `md.Get()`, чтобы 
получить определенное значение заголовка, хранящееся внутри метаданных.
Как правило, токен доступа следует отправлять, используя заголовок
`Authorization`. Поэтому я определю для него константу в начале этого 
файла.

```go
const (
	authorizationHeader = "authorization"
)
```

Затем здесь давайте вызовем `md.Get()`, чтобы получить значение заголовка
`Authorization`. Обратите внимание, что эта функция возвращает массив строк, 
так что давайте сохраним его внутри переменной `values`.

```go
func (server *Server) authorizeUser(ctx context.Context) (*token.Payload, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing metadata")
	}

    values := md.Get(authorizationHeader)
}
```

И мы должны проверить, пуст этот массив или нет. Если длина `values` равна
0, то мы возвращаем полезную нагрузку в виде `nil` и сообщение об ошибке
"missing authorization header" («отсутствует заголовок авторизации»). В 
противном случае заголовок авторизации будет первым элементом массива.

```go
func (server *Server) authorizeUser(ctx context.Context) (*token.Payload, error) {
	...

	values := md.Get(authorizationHeader)
	if len(values) == 0 {
		return nil, fmt.Errorf("missing authorization header")
	}

	authHeader := values[0]
}
```

Как мы узнали из лекции 22, значение заголовка авторизации должно быть 
строкой с префиксом `Bearer`, за которым следует пробел и токен доступа.
Здесь `Bearer` — это тип авторизации, а токен доступа — это данные 
авторизации. Мы используем этот формат, потому что сервер может поддерживать
несколько способов авторизации.

Итак, чтобы получить токен, мы должны разбить этот заголовок авторизации,
используя в качестве разделителя пробел. В стандартном пакете `strings` для 
этой цели уже существует функция `Fields`. Мы ожидаем, что в возвращаемом 
функцией массиве будет два элемента, первый будет содержать строку `Bearer`,
а второй — токен доступа. Поэтому, если его длина меньше двух, мы возвращаем
ошибку: "invalid authorization header format" («неверный формат заголовка 
авторизации»). В противном случае первое поле будет типом авторизации.
Я переведу его в нижний регистр, чтобы было легче сравнивать.

```go
func (server *Server) authorizeUser(ctx context.Context) (*token.Payload, error) {
	...

	authHeader := values[0]
	fields := strings.Fields(authHeader)
	if len(fields) < 2 {
		return nil, fmt.Errorf("invalid authorization header format")
	}
	
	authType := strings.ToLower(fields[0])
	
}
```

Допустим, наш сервер поддерживает пока только тип токена `Bearer`. 
Таким образом, я определю константу для него в начале файла

```go
const (
	authorizationHeader = "authorization"
	authorizationBearer = "bearer"
)
```

затем здесь, если тип авторизации не `bearer`, мы просто возвращаем 
ошибку, сообщающую о "unsupported authorization type" («неподдерживаемом 
типе авторизации»).

```go
func (server *Server) authorizeUser(ctx context.Context) (*token.Payload, error) {
	...

	authType := strings.ToLower(fields[0])
	if authType != authorizationBearer {
		return nil, fmt.Errorf("unsupported authorization type: %s", authType)
	}
}
```

В противном случае токен доступа должен быть во втором элементе массива.
Мы проверим это, вызвав `server.tokenMaker.VerifyToken()`. Эта функция 
вернет полезную нагрузку токена и ошибку. Если ошибка не `nil`, мы вернем 
полезную нагрузку в виде `nil` и ошибку с сообщением "invalid access token"
(«недопустимый токен доступа»). Наконец, если все прошло хорошо, мы просто 
возвращаем полезную нагрузку и ошибку `nil`.

```go
func (server *Server) authorizeUser(ctx context.Context) (*token.Payload, error) {
	...

	accessToken := fields[1]
	payload, err := server.tokenMaker.VerifyToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("invalid access token: %s", err)
	}

    return payload, nil
}
```

Напомним, что метод `VerifyToken` является частью интерфейса `TokenMaker`, 
и мы узнали, как реализовать его с помощью JWT и PASETO в лекциях 19 и 20 
курса. Это довольно просто! Мы просто расшифровываем строку токена с 
помощью симметричного ключа, чтобы получить полезную нагрузку. Если 
расшифровать не удается, предоставленный токен недействителен. В противном 
случае вызывается `payload.Valid()`, чтобы проверить, истек ли срок действия 
токена.

```go
func (maker *PasetoMaker) VerifyToken(token string) (*Payload, error) {
	payload := &Payload{}

	err := maker.paseto.Decrypt(token, maker.symmetricKey, payload, nil)
	if err != nil {
		return nil, ErrInvalidToken
	}

	err = payload.Valid()
	if err != nil {
		return nil, err
	}

	return payload, nil
}
```

Хорошо, теперь когда метод `authorizedUser()` готов, мы можем вернуться к
RPC обработчику `UpdateUser`, чтобы использовать его. В верхней части 
функции давайте вызовем `server.authorizedUser()`, передав контекст, и
сохраним результат в переменные `authPayload` и `error`. Если ошибка не 
`nil`, то мы должны вернуть клиенту ошибку с кодом состояния 
`Unauthenticated`.

```go
func (server *Server) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
        return nil, unauthenticatedError(err)
	}
	
	...
}
```

Точно так же, как мы сделали для ошибки неверного аргумента, я определю
отдельную функцию для ошибки `Unauthenticated`, поскольку она будет повторно 
использоваться во многих местах. Эта функция примет ошибку в качестве 
входного аргумента, а также вернет ошибку в качестве результата. Но 
возвращаемая ошибка будет преобразована, добавлением в неё gRPC код состояния
`Unauthenticated`, а также сообщения "unauthorized" («не авторизовано»).

```go
func unauthenticatedError(err error) error {
	return status.Errorf(codes.Unauthenticated, "unauthorized: %s", err)
}
```

Хорошо, используя эту функцию, теперь здесь мы можем просто вернуть полезную 
нагрузку в виде `nil` и ошибку `Unauthenticated`.

```go
func (server *Server) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	authPayload, err := server.authorizeUser(ctx)
	if err != nil {
        return nil, unauthenticatedError(err)
	}
	
	...
}
```

Обратите внимание, что полезная нагрузка аутентификации ещё не используется.
На данный момент, если пользователь не предоставит токен доступа или если 
предоставленный токен недействителен или срок его действия истек, остальной
код не будет выполняться и клиент получит код состояния `Unauthenticated`.

Однако это не мешает некому пользователю использовать свой собственный 
токен доступа для обновления информации других пользователей. Вот когда в 
игру вступает полезная нагрузка аутентификации! Итак, после проверки 
параметров запроса мы проверим, совпадает ли `authPayload.Username` с 
именем пользователя, указанным в запросе, или нет. Если они отличаются, мы 
вернем полезную нагрузку в виде `nil`, ошибку с кодом состояния 
`PermissionDenied` и сообщение о том, что "cannot update other user's 
info" («невозможно обновить информацию другого пользователя»).

```go
func (server *Server) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	...
	
	if authPayload.Username != req.GetUsername() {
        return nil, status.Errorf(codes.PermissionDenied, "cannot update other user's info")
    }
	
	...
}
```

Хорошо, теперь API `UpdateUser` полностью защищен. Вы также можете 
реализовать эту логику авторизации с помощью gRPC перехватчика, но если вы 
это сделаете, она не будет работать для HTTP-шлюза, и вам также придется 
реализовать отдельный HTTP middleware.

Однако в нашем случае, поскольку мы реализуем эту логику внутри метода
RPC обработчика, она будет работать из коробки как для gRPC, так и для 
сервера HTTP-шлюза.

Давайте попробуем запустить сервер и протестировать его!

```shell
make server
go run main.go
2022/09/14 16:48:53 db migrated successfully
2022/09/14 16:48:53 start gRPC server at [::]:9090
2022/09/14 16:48:53 start HTTP gateway server at [::]:8080
```

## Тестируем HTTP сервер

Итак, и gRPC, и сервер HTTP-шлюза готовы обслуживать запросы на разных 
портах. В первую очередь я собираюсь протестировать HTTP-сервер. Вот 
запрос `UpdateUser`, который мы добавили в Postman на прошлой лекции.

![](../images/part51/1.png)

Я изменю поле полного имени и фамилии на "New Alice" и отправлю этот 
запрос без заголовка авторизации.

![](../images/part51/2.png)

Как видите, мы получили в качестве ответа код состояния 401 `Unauthorized`
с сообщением: "unauthorized, missing authorization header" («пользователь 
не авторизован, отсутствует заголовок авторизации»). Поэтому мы должны 
добавить Bearer токен доступа в заголовок этого запроса.

Откроем вкладку `Authorization` и выберем `Bearer Token` в качестве 
типа авторизации.

![](../images/part51/3.png)

Затем мы можем ввести токен доступа в это поле ввода `Token`. Давайте 
временно зададим его равным "abc".

![](../images/part51/4.png)

Затем на вкладке `Headers` мы увидим новый заголовок `Authorization` со 
значением "Bearer abc".

![](../images/part51/5.png)

Итак, теперь вы знаете зачем мы определили константу для заголовка 
авторизации следующим образом

```go
const (
	authorizationHeader = "authorization"
)
```

в коде.

Обратите внимание, что не имеет значения, используете ли вы строчные или 
прописные буквы для названия заголовка, потому что «под капотом» gRPC 
преобразует его в нижний регистр.

```go
func (md MD) Get(k string) []string {
	k = strings.ToLower(k)
	return md[k]
}
```

Хорошо, теперь давайте попробуем отправить этот запрос с неправильным 
токеном.

![](../images/part51/6.png)

Вуаля! На этот раз мы по-прежнему получили в качестве ответа код
состояния 401 `Unauthorized`, но с другим сообщением: "invalid access token"
(«неправильный токен доступа»).

Чтобы получить действующий токен доступа, мы должны вызвать API для входа
пользователя в систему. Затем скопируйте значение токена доступа из тела 
ответа

![](../images/part51/7.png)

и вставьте его в поле для ввода `Token` заголовка `Authorization`.

![](../images/part51/8.png)

Теперь, если мы повторно отправим запрос, он успешно выполнится.

![](../images/part51/9.png)

Полное имя и фамилия было изменено на "New Alice", как мы и ожидали.

Так что HTTP API работает отлично!

Однако довольно неудобно тестировать API, если нам приходится копировать 
токен доступа каждый раз, когда мы повторно входим в систему или обновляем
его. Вместо этого лучше воспользоваться `Collection variables` в Postman.

Итак, в `LoginUser` API, давайте откроем вкладку `Tests`.

![](../images/part51/10.png)

Здесь мы можем написать несколько скриптов для проверки ответа от API.
Тут показано несколько примеров тестовых скриптов, я выберу этот тест для 
проверки равен ли код состояния 200.

![](../images/part51/11.png)

Во вкладку `Test` добавится этот небольшой фрагмент кода

```js
pm.test("Status code is 200", function () {
    pm.response.to.have.status(200)
})
```

который проверит, что запрос выполнен успешно. Теперь давайте попробуем 
дописать к нему ещё небольшой фрагмент кода, чтобы получить токен доступа 
из ответа и добавить его в переменные коллекции.

Во-первых, мы вызовем `JSON.parse()`, чтобы проанализировать тело ответа.
Затем мы используем функцию `pm.collectionValues.set()`, чтобы задать 
значение токена доступа равным `jsonData.access_token`. Обратите внимание, 
что это имя поля должно совпадать с тем, которое мы получили в теле ответа.

```js
var jsonData = JSON.parse(responseBody);
pm.collectionVariables.set("access_token", jsonData.access_token)
```

Хорошо, давайте повторно отправим запрос. Он успешно выполнен.

![](../images/part51/12.png)

И мы можем просмотреть результат теста в этой вкладке.

![](../images/part51/13.png)

Он успешно пройден. Итак, теперь, если мы щёлкнем на название коллекции и
откроем её список `Variables`, то увидим новую переменную с названием
"access_token", и её текущее значение, которое мы получили в ответе при входе
пользователя в систему.

![](../images/part51/14.png)

Теперь мы можем использовать эту переменную в API `UpdateUser`. Просто 
замените это вручную заданное значение на переменную `access_token`.
Мы можем обратиться к нему с помощью пары двойных фигурных скобок следующим
образом.

![](../images/part51/15.png)

![](../images/part51/16.png)

Хорошо, теперь давайте попробуем изменить `email` на "new_alice@gmail.com" 
и повторно отправить запрос.

![](../images/part51/17.png)

Он успешно выполнен! Превосходно.

Таким образом, сервер HTTP-шлюза работает правильно.

## Тестируем gRPC API 

А что насчёт gRPC?

Мы также можем протестировать gRPC API с помощью Postman, хотя на момент 
записи этого видео этот функционал всё ещё на стадии бета-тестирования.

Чтобы использовать его, нам нужно просто нажать кнопку `New` в левом меню
рядом с `My Workspace` и выбрать `gRPC Request`.

To use it, we just have to click on this `New` button on
the left-hand side menu, next to `My Workspace` and select
`gRPC Request`.

![](../images/part51/18.png)

![](../images/part51/19.png)

В первую очередь нам нужно ввести URL-адрес gRPC сервера. Как вы видели в
логах, наш локальный gRPC сервер работает на порту `9090`. Поэтому я введу 
здесь `localhost:9090`.

![](../images/part51/20.png)

После этого все доступные RPC на сервере отобразятся в поле для ввода 
метода.

![](../images/part51/21.png)

Давайте сначала попробуем вызвать `LoginUser` RPC. В поле для ввода 
сообщения мы можем ввести обычный JSON объект, поэтому давайте зададим
имя пользователя равным "alice", а пароль "secret". Затем нажмите на эту
кнопку "Invoke", чтобы отправить запрос.

![](../images/part51/22.png)

Вуаля, запрос успешно выполнен. И мы можем увидеть все возвращенные данные 
в ответе. Он очень похож на тело HTTP-запроса, за исключением некоторых 
полей с метками времени.

![](../images/part51/23.png)

![](../images/part51/24.png)

Хорошо, теперь давайте скопируем токен доступа. И давайте сохраним этот 
запрос, используя следующее название: "Login User RPC". Мы не можем 
сохранить его в коллекции `Simple Bank`, поскольку эта коллекция 
предназначена только для HTTP-запросов. Поэтому я создам новую коллекцию
под названием `Simple Bank gRPC`.

![](../images/part51/25.png)

![](../images/part51/26.png)

Итак, была создана новая коллекция, содержащая "Logic User RPC".

Чтобы создать новый запрос, мы можем нажать на эту кнопку с троеточием и 
выбрать "Add Request", а затем `gRPC request`.

![](../images/part51/27.png)

Я переименую его в "Update User RPC", затем изменю URL-адрес сервера на
`localhost:9090` и выберу метод `UpdateUser`.

![](../images/part51/28.png)

Далее в сообщении давайте попробуем изменить имя пользователя на 
«alice», а также полное имя и фамилию — на «Alice».

Если мы вызовем метод сейчас, мы получим сообщение об ошибке
"missing authorization header" («отсутствует заголовок авторизации») и
код состояния `16 Unauthenticated`.

![](../images/part51/29.png)

Чтобы такого не происходило, давайте откроем вкладку `Metadata` и добавим 
новый ключ: `Authorization` со значением `Bearer abc`. Давайте протестируем
реакцию системы на этот некорректный токен и посмотрим что произойдёт.

![](../images/part51/30.png)

Мы по-прежнему получили ответ с кодом состояния `16 Unauthenticated`, но 
сообщение изменилось на "token is invalid" («некорректный токен»). Так что,
вроде бы всё работает хорошо. Теперь я собираюсь вставить действующий 
токен доступа, который мы получили из "Login User RPC" ранее и вызвать 
метод ещё раз.

![](../images/part51/31.png)

На этот раз запрос успешно выполнен и полное имя и фамилия было обновлено.
Как мы и хотели. Мы также можем добавить адрес электронной почты в сообщение, 
чтобы изменить его на «alice@gmail.com» и повторно отправить запрос.

![](../images/part51/32.png)

Вуаля теперь и адрес электронной почты также был обновлен до нового значения. 

И на этом закончим эту лекцию. Сегодня мы успешно добавили слой
авторизации на gRPC сервер для защиты нашего `UpdateUser` API.

Я надеюсь, что лекция была интересной и приобретенные знания будут вам
полезны. Большое спасибо за время, потраченное на чтение! Желаю Вам получать
удовольствие от обучения и до встречи на следующей лекции!