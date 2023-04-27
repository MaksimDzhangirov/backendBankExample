# Unit test gRPC API with mock DB & Redis

[Original video](https://www.youtube.com/watch?v=QFxZlKb7W2k)

Hello everyone, welcome to the backend master class! So far, we've written
a lot of codes for our gRPC web services, but we haven't written any unit
tests for them yet. So in this lecture, I'm gonna show you how to do that.

# Unit test gRPC CreateUser API

The API we're gonna test today is the CreateUser RPC. It's a bit more
complicated than other APIs, because it involves a DB transaction that
creates a new user, and if the user is successfully created, the 
`AfterCreate` callback function is called to distribute an async task that
sends a verification email to the user.

As you already know, the task is normally stored in Redis. So in order
to unit test this RPC, we'll have to deal with 2 mock entities, one is 
the mock store for the database, and the other one is a mock task 
distributor for Redis.

First, let's create a new file called `rpc_create_user_test.go` inside
the `gapi` package. The tests we're gonna write will be pretty similar
to the ones we've written in the `api` package for the Gin HTTP services.
So let's open the `user_test.go` file, and copy the function 
`TestCreateUserAPI` from it, then let's paste it to the new file we've
just created.

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

AS you can see, this test features a custom `gomock.Matcher`, that we've
written in lecture 18 of the course, in order to correctly compare the input
arguments of the `CreateUser` function, since the hashed password field
is a bit tricky to handle. You can rewatch that lecture to understand what
we're doing here before continue.

Alright, now let's update the code of this test to make it work with our 
gRPC server.

First, we have to create a random user with this function. It is written
in the `api` package, so let's head over there to copy it and paste it
here, right before the `TestCreateUserAPI` function.

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

Next, we're gonna define some test cases. This `name` field will store
the name of the case. Then, the `body` field will have to be changed,
because we're not using Gin here, but gRPC instead. If we look at the
CreateUser RPC handler, we will see that the input data is sent via this
`CreateUserRequest` object. So that's exactly the data type that we're 
gonna store in this case.

```go
testCases := []struct {
    name          string
    req           *pb.CreateUserRequest
}{
    ...
}
```

OK, now we have to update the content of this request. It is super easy to
write codes with gRPC, since everything is strongly typed, so we will
have the autocomplete feature from Visual Studio Code. We have to change
this asterisk to ampersand, because we want to obtain the address of this
request object.

```go
req: &pb.CreateUserRequest{
    Username: user.Username,
    Password: password,
    FullName: user.FullName,
    Email:    user.Email,
}
```

OK, next let's move to the `buildStubs` function. This is where we tell
`gomock`, which function we expect to be called, with which parameters, 
and return which output. When we implemented the CreateUser HTTP API with
Gin, we didn't have the async task, that sends emails, so that's why
we expect the `CreateUser` method of the `store` to be called directly.

```go
store.EXPECT().
    CreateUser(gomock.Any(), EqCreateUserParams(arg, password)).
    Times(1).
    Return(user, nil)
```

However, in out new implementation of the CreateUser RPC, we're not calling
that method directly, but we're calling the CreateUser transaction instead.

```go
txResult, err := server.store.CreateUserTx(ctx, arg)
```

The `CreateUser` method only gets called inside that transaction, together
with the callback function. So here, in `buildStubs` function, I'm 
gonna change this argument to `db.CreateUserTxParams` and set the 
`CreateUserParams` as an inner field of it.

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

Note that this struct also has another field for the `AfterCreate` 
callback, but there's no way to compare 2 functions in Golang, so let's
ignore this callback for now. We will deal with it later.

OK, here we should expect the CreateUser transaction of the mock
`store` to be called exactly 1 time with the created input argument.

```go
store.EXPECT().
    CreateUserTx(gomock.Any(), EqCreateUserParams(arg, password)).
    Times(1).
    Return(user, nil)
```

However, since we're using this custom `gomock` matcher to compare
the input, we will have to update its data type a little bit.

First, I'm gonna change its name to `eqCreateUserTxParamsMatcher`. Then
this `arg db.CreateUserParams` argument's type should be 
`db.CreateUserTxParams`. By the way, to make it easier to understand,
I'm gonna change the name of this `func (e eqCreateUserTxParamsMatcher)`
variable to "expected", since it contains the expected value of the 
arguments. And this `x` variable in `Matches(x interface{})` will contain
the actual value, that the gRPC handler will use when it calls the 
CreateUser transaction. So, when we convert it to `db.CreateUserTxParams`,
I'm gonna assign the result to a variable named `actualArg`. Next, we will
use `util.CheckPassword` function to check that the expected password 
matches the actual hashed password. And finally, we assign the actual hashed
password to the expected argument.

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

This is a trick, because we want to use the `DeepEqual` function to 
compare them. I already explained the reason we have to do this in 
lecture 18. Every time we hash the password, a random salt us added, so 
the output hash value will be different, which will help us prevent rainbow
table attacks.

OK, now we have to change the type of this `func 
EqCreateUserTxParams(arg db.CreateUserParams, password string)` argument
to `db.CreateUserTxParams`. Then go back to the unit test, that we're
writing.

We will have to update the `checkResponse` function, because for gRPC 
services, we won't use an HTTP recorder to store the response, but we will
get a `CreateUserResponse` object or an error directly from the RPC 
handler. So I'm gonna put them as the input arguments of this 
`checkResponse` function.

```go
checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
    ...
}
```

I also add a `testing.T` object as the first input argument, since we're
gonna need it when comparing the real output with the expected value.
Alright, let's copy this function signature, and paste it into the 
`testCases` struct definition.

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

OK, now let's fix the content of the `checkResponse` function. Since this
is a successful case, we expect no errors to be returned. And the response
should be not `nil`. Next, we can get the created user object from the
response. And require the created user's username to equal the input 
`user.Username`. We expect the same thing for the other fields, such as 
the full name and email address.

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

Oh, I've just noticed that we should change this

```go
store.EXPECT().
    CreateUserTx(gomock.Any(), EqCreateUserParams(arg, password)).
    Times(1).
    Return(user, nil)
```

`EqCreateUserParams` function's name to `EqCreateUserTxParams`, because the
input argument is now, in fact, a `CreateUserTxParams`. And one more 
thing, we're expecting the `CreateUserTx` function to be called once, and
according to the code, it will return a `CreateUserTxResult` object. So
here, we cannot just return a `user` object like this, but we will have
to change it into a `db.CreateUserTxResult`, and `User` will be just one
field of this object. And that's how the test data for the happy case 
should look like.

```go
store.EXPECT().
    CreateUserTx(gomock.Any(), EqCreateUserTxParams(arg, password)).
    Times(1).
    Return(db.CreateUserTxResult{User: user}, nil)
```

We will have to update the unsuccessful case as well, but let's save it
for later, and just focus on the happy case first. So for now, to keep
things simple, I'm just gonna delete all of them.

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

The next thing we need to do is update the body of the unit test. As you 
can see here,

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

we created a `gomock` controller, and use it to create a mock store for
the database. We will then use the mock store to create a test server. This
`newTestServer` function is missing in the `gapi` package. We can find its
content from the `main_test.go` file in the `api` package.

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

There's also a `TestMain()` function in this file, which we used to set 
the Gin test mode, but we don't need it for our gRPC service, so let's copy
only the `newTestServer` function.

I'm gonna create a `main_test.go` file inside the `gapi` package, and 
paste in the content of the function we've just copied.

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

We'll have to update this a bit though, since the `NewServer` function
of the `gapi` package requires 1 more input argument for the task 
distributor. And if we open its definition, we'll see that it's an interface
with this only method.

```go
type TaskDistributor interface {
	DistributeTaskSendVerifyEmail(
		ctx context.Context,
		payload *PayloadSendVerifyEmail,
		opts ...asynq.Option,
	) error
}
```

This is good for writing unit tests, because we don't want to connect to
a real Redis server. So just like what we did for the `Store` interface,
we can also generate a mock for the `TaskDistributor` interface, and use
it in the test.

To do that, I'm gonna open the `Makefile`, and scroll down to the "mock"
command.

```makefile
mock:
    mockgen -package mockdb -destination db/mock/store.go github.com/MaksimDzhangirov/backendBankExample/db/sqlc Store
```

Let's duplicate this command that generates a mock `Store`. Then change
the package name to `mockwk` (which means "mock worker"). The output 
destination will be inside the `worker` folder. A `mock` folder will be
created, and inside it, there will be a `distributor.go` file to contain
the generated codes of the mock task distributor. We'll have to change
the path of this input package `backendBankExample/db/sqlc` to 
`backendBankExample/worker` like this.

```makefile
mock:
    mockgen -package mockdb -destination db/mock/store.go github.com/MaksimDzhangirov/backendBankExample/db/sqlc Store
    mockgen -package mockwk -destination worker/mock/distributor.go github.com/MaksimDzhangirov/backendBankExample/worker TaskDistributor
```

And of course, the name of the input interface should be `TaskDistributor`.

OK, now we can run

```shell
make mock
```

in the terminal to generate the mock objects. It's successful.

So in Visual Studio Code, we will see a new `mock` folder inside the
`worker` folder, and inside this folder there's a `distributor.go` file
with the `MockTaskDistributor` struct, which we can use in the unit tests.

Now let's get back to the `main_test.go` file. I'm gonna add a task 
distributor argument to the `newTestServer` function. And we can pass it
into this `NewServer(config, store)` function to create a new server.

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

Alright, with all these changes in place, we can continue writing our unit
tests.

As the `newTestServer` function now requires a task distributor, we'll
create a mock for it, just like how we created a mock store. So I'm 
gonna define a `taskDistributor` variable, and assign it to 
`mockwk.NewMockTaskDistributor`.

```go
taskDistributor := mockwk.NewMockTaskDistributor()
```

Since this is the first time we use this package, Visual Studio Code 
doesn't know about it yet, so we'll have to import this package manually. 
Let's  duplicate this `import mockdb` line, then change its name to 
`mockwk`, and the path of the package should be 
`backendBankExample/worker/mock`.

```go
import (
    ...
    mockwk "github.com/MaksimDzhangirov/backendBankExample/worker/mock"
    ...
)
```

OK, now if we go back to the test, we will see that Visual Studio Code
has recognized the package. It also requires a mock controller object as
input, so let's pass in the same controller we used for the mock store.
Then, we can now use the mock task distributor to create the test server.

```go
store := mockdb.NewMockStore(ctrl)
tc.buildStubs(store)

taskDistributor := mockwk.NewMockTaskDistributor(ctrl)

server := newTestServer(t, store, taskDistributor)
recorder := httptest.NewRecorder()
```

Next, I'm gonna get rid of all these codes that set up and send HTTP
requests.

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

For gRPC server unit testing, we can just call the RPC handler function
directly like this: `server.CreateUser`. Then pass in a background 
context and the input request object of the test case. It will return a
response object and an error, so we can send them to the `checkResponse()`
function of the test case together with the `testing.T` object for final
verification of the result.

```go
server := newTestServer(t, store, taskDistributor)
res, err := server.CreateUser(context.Background(), tc.req)
tc.checkResponse(t, res, err)
```

Note that this `t` is different from the global `t`, because it has been
shadowed by the input argument of this `t.Run` function. So basically,
it's the `t` object of the sub-test, created by the `Run()` function. This
way, the check response call for each case will be independent, and not
interfere with each other, when we add more cases in the future.

Alright, the happy case is ready. Let's run it to see what happens!

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

It failed with this error: "missing a call to mockStore.CreateUserTx. The
expected call doesn't match the argument at index 1", which is the 
`CreateUserTxParams`. Ok, so we know this expectation

```go
store.EXPECT().
    CreateUserTx(gomock.Any(), EqCreateUserTxParams(arg, password)).
    Times(1).
    Return(db.CreateUserTxResult{User: user}, nil)
```

of function call is not satisfied. But what can we do about it? How can
we figure out what's wrong with our implementation?

Well, one way to debug this kind of error is, to print out some logs.
We must find out up to which part the code has been executed. So I'm gonna
add several `fmt.Println()` messages, one before validating the request,
one before hashing the password and one right before running the CreateUser
transaction.

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

OK, now let's rerun the test.

```shell
=== RUN   TestCreateUserAPI
=== RUN   TestCreateUserAPI/OK
validate request
request: username:"fthteb" full_name:"vklrvh" email:"lkswpj@email.com" password:"ouhhdy"
create user tx {{fthteb $2a$10$vPpxT8MAnxflwHnw7mw6auUSwqRBdGnLbahTljqPMMnbnWol6.Xpe vklrvh lkswpj@email.com} 0x15ea480}
...
```

This time, we can see the 3 lines of logs that we've just added. So it
has indeed run up to the CreateUser transaction, but it couldn't go
through, because the argument doesn't match with the one we expect.

So let's take a look at the custom `gomock` matcher. I'm gonna add some
more debug logs here.

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

First, before converting the params, second, before checking the password,
third, before comparing the arguments with `DeepEqual`. And I want to add
1 more log at the end, so let's change this 
`return reflect.DeepEqual(expected.arg, actualArg)` into an if clause. If
the arguments are not equal, we return `false`. Else, we print a message
saying "param matches!", and return `true`.

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

Alright, let's rerun the test one more time.

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

This time, we can see all the logs up until the `DeepEqual` statement. 
But there's no log saying "param matches!", so the issue must come from the
`DeepEqual` comparison of the arguments. But we've already taken care of
the tricky hashed password, then what makes this deep equal call fail?

Well, I just remember another tricky field in the `CreateUserTxParams`,
which is the `AfterCreate` callback function. As I said before, there's
no way to compare 2 functions in Go, so this is exactly what makes
the `DeepEqual` statement fail.

To avoid this, we should compare only the `CreateUserParams` fields instead
of the whole argument objects.

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

Just like that, and I think the error should be gone when we rerun the test.

```shell
=== RUN   TestCreateUserAPI
=== RUN   TestCreateUserAPI/OK
...
--- PASS: TestCreateUserAPI (0.19s)
    --- PASS: TestCreateUserAPI/OK (0.12s)
PASS
ok      github.com/MaksimDzhangirov/backendBankExample/gapi     0.514s
```

Indeed, this time the test passed. So, I'm gonna delete all the debug logs
that we've added before.

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

There's a warning here, since this `if` can be replaced with 1 single 
return statement, but I'm gonna keep it like this for now, because we're
gonna do something with the callback later if the arguments match.

OK, let's remove the 3 debug logs in the `rpc_create_user.go` file as 
well.

Now, the test has passed, but it only checks that the CreateUser 
transaction is executed with the expected arguments. One thing it hasn't
checked is, the `AfterCreate` callback function. In this case, we would
like to be sure that the `DistributeTaskSendVerifyEmail` function is called
every time a new user is created. That way, an email will be delivered
to user in the future. So, how can we test this?

Well, we can use the `MockTaskDistributor` for this purpose, just like
what we did for the mock store. I'm gonna add a new `taskDistributor`
argument to the `buildStubs()` function and update its function signature
in the `testCases` struct.

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

Then, in the body of the test, let's move the `taskDistributor` declaration
up here.

```go
store := mockdb.NewMockStore(ctrl)
taskDistributor := mockwk.NewMockTaskDistributor(ctrl)
tc.buildStubs(store)
```

And pass it into the `tc.buildStubs()` call, together with the mock store.

```go
store := mockdb.NewMockStore(ctrl)
taskDistributor := mockwk.NewMockTaskDistributor(ctrl)

tc.buildStubs(store, taskDistributor)
server := newTestServer(t, store, taskDistributor)
```

Just like that!

Then at the end of the `buildStubs` function, I'm gonna call 
`taskDistributor.EXPECT()`. This time, we want the 
`DistributeTaskSendVerifyEmail()` function to be called. It has several
input arguments, such as the context, the task payload, and some `asynq`
options. So, I'm gonna use `gomock.Any()` for the context, then a 
`taskPayload`, which we will declare a bit later, finally, another 
`gomock.Any()` matcher for the `asynq` options.

```go
taskDistributor.EXPECT().
    DistributeTaskSendVerifyEmail(gomock.Any(), taskPayload, gomock.Any())
```

This `Any()` matcher can match variadic arguments, so it will replace the 
3 options we send in this `opt` array.

```go
opts := []asynq.Option{
    asynq.MaxRetry(10),
    asynq.ProcessIn(10 * time.Second),
    asynq.Queue(worker.QueueCritical),
}
```

If you want, you can also use `gomock.Eq()` matcher to match each of them.
For me, I'm gonna keep it simple with `gomock.Any()`, and focus on the
more important param, which is the `taskPayload`. We can copy it from the
callback function here.

```go
AfterCreate: func(user db.User) error {
    taskPayload := &worker.PayloadSendVerifyEmail{
        Username: user.Username,
    }
    ...
},
```

Basically, it should be of type `PayloadSendVerifyEmail`, with the username
field set to the input `user.Username`.

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

So, now, we can expect this distribute function to be called exactly one
time and it will return an output result of type error. This is a happy case,
so we're going to return a `nil` error here.

```go
taskDistributor.EXPECT().
    DistributeTaskSendVerifyEmail(gomock.Any(), taskPayload, gomock.Any()).
    Times(1).
    Return(nil)
```

And that's it! Let's try to run the test to see what happens!

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

It failed! And the error is missing calls to the 
`DistributeTaskSendVerifyEmail()` function. So what happened? Why is this
function not getting called? This is a bit tricky, so you can pause the 
video and think about it for a while.

Did you figure it out? Well, remember that besides the `TaskDistributor`,
we're also mocking the CreateUser transaction. And if we look at its
implementation, we'll see that the function to distribute task send verify
email is only called inside the `AfterCreate` callback function. That 
means, it will only be called if the real DB transaction is executed, and
a real user is successfully created inside the database. However, since 
we're mocking this CreateUser transaction, the actual code that talks to 
the DB is not executed. Therefore, the `AfterCreate` function doesn't get
a chance to run either.

So what can we do? Can we somehow still trigger the `AfterCreate` callback
in the mock transaction? And it should not be a fake trigger, but should
come from the actual `AfterCreate` callback function.

Well, luckily we have this custom `gomock` matcher, that compares the 
actual arguments of the transaction with the expected one. And we know 
that, if it is a match, then it's kind of the same as if the transaction
is executed. So, we can safely assume that the `AfterCreate` function
can be executed here.

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

And since we have the actual implementation of the callback function inside
this `actualArg` object, we can simply call `actualArg.AfterCreate()`,
and pass in a `db.User` object as input. Although, it's possible to 
construct the `User` from the actual input arguments, it will be faster
if we just store the expected user object inside this 
`eqCreateUserTxParamsMatcher` matcher struct, and just use it here when
executing this callback function.

```go
type eqCreateUserTxParamsMatcher struct {
	arg      db.CreateUserTxParams
	password string
	user     db.User
}
```

The callback function will return an error, so we will return `true` 
if that error is `nil` at the end.

```go
func (expected eqCreateUserTxParamsMatcher) Matches(x interface{}) bool {
	...

	// call the AfterCreate function here
	err = actualArg.AfterCreate(expected.user)

	return err == nil
}
```

Alright, now we have to update this `EqCreateUserTxParams` constructor
function to accept one more parameter for the expected user object and
use it to create the custom matcher.

```go
func EqCreateUserTxParams(arg db.CreateUserTxParams, password string, user db.User) gomock.Matcher {
	return eqCreateUserTxParamsMatcher{arg, password, user}
}
```

Then, in the `buildStubs()` function, we'll add the target user to this
constructor.

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

And that should be it!

Let's rerun the test to see if it's working or not.

```shell
=== RUN   TestCreateUserAPI
=== RUN   TestCreateUserAPI/OK
```

Wow, it seems to take a really long time! It feels like the test gets stuck
somewhere. If that's the case, it will be timed out after 30 seconds. And
indeed, we've got a panic, because the test cannot finish in time.

```shell
panic: test timed out after 30s
```

The reason for this can be difficult to spot if you don't fully understand
the way `gomock` works. In fact, the problem comes from the way we use the 
same controller for both the mock store and the mock task distributor.

```go
ctrl := gomock.NewController(t)
defer ctrl.Finish()

store := mockdb.NewMockStore(ctrl)
taskDistributor := mockwk.NewMockTaskDistributor(ctrl)
```

There's a locking mechanism in the controller. Every time it checks for
a matching function call. So when the `CreteUserTx` function is being
checked for matching arguments, the mock controller will be locked.

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

That's why when we call the `AfterCreate()` callback function, it can no
longer acquire the lock to record the call to the mock task distributor.
To fix this, we can simply use two different controllers, one for the 
mock store, and the other one for the mock task distributor.

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

Just like that, and we're done. This time, I expect the test to pass.

Let's run it one more time!

```shell
=== RUN   TestCreateUserAPI
=== RUN   TestCreateUserAPI/OK
--- PASS: TestCreateUserAPI (0.19s)
    --- PASS: TestCreateUserAPI/OK (0.12s)
PASS
ok      github.com/MaksimDzhangirov/backendBankExample/gapi     0.521s
```

And it really does! Excellent!

Now, to make sure the test can  detect a wrong implementation, let's 
pretend that we forget to create a task to send emails. I'm gonna comment
out all the codes inside the `AfterCreate` callback function. And just 
return `nil` here.

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

This time, if the test is string enough, it should fail. Let's run it
to verify!

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

Indeed, the test has failed due to a missing call to the 
`DistributeTaskSendVerifyEmail` function.

So that's how we implement a unit test for the gRPC API, which involves
multiple mock entities interacting with each other. However, the test 
we've just written is only for the successful case.

Can you try adding some more failure cases to this on your own? It's 
pretty easy, isn't it?

Let's implement the "internal server error" case. I'm gonna duplicate
this "OK" case,

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

and change the name to "InternalError".

The request object will be the same, but in the `buildStubs()` function,
we don't need to create a real input argument of the CreateUser 
transaction. We can just use `gomock.Any()` matcher here, and fake the 
return of this call with an empty result, and a not-`nil` error, such as
`sql.ErrConnDone`. And since the transaction fail in this case, we expect
the `DistributeTaskSendVerifyEmail` to be called zero times, no matter
what the input arguments are.

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

Then, in the `checkResponse()` function, we should expect to see a 
not-`nil` error. There's no need to check the response struct, but we should
check the error code to make sure it's really Internal Error. To do this,
we simply call `status.FromError()`, and pass in the `err` object. This
"status" is a sub-package of the gRPC framework. And this function will 
return a `status.Status` object, together with a bool value to tell us
if the conversion is successful or not. I'm gonna store them in the `st`
and `ok` variables. Then require the value of `ok` to be `true`, and the 
value of `st.Code()` to be equal to `codes.Internal`.

```go
checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
    require.Error(t, err)
    st, ok := status.FromError(err)
    require.True(t, ok)
    require.Equal(t, codes.Internal, st.Code())
},
```

And that's basically it!

Let's run the tests!

```shell
=== RUN   TestCreateUserAPI/OK
=== RUN   TestCreateUserAPI/InternalError
--- PASS: TestCreateUserAPI (0.24s)
    --- PASS: TestCreateUserAPI/OK (0.12s)
    --- PASS: TestCreateUserAPI/InternalError (0.05s)
PASS
ok      github.com/MaksimDzhangirov/backendBankExample/gapi     0.565s
```

They all passed! both the "OK" case and the "InternalError" case.

Awesome!

There are still several more cases we can test, such as: when the username
already exists, or when the provided input arguments are invalid. But I 
leave that as an exercise for you to do on your own.

You can always check my codes on GitHub of you want to see how I implement
them.

And that brings us to the end today's lecture about writing unit tests for
gRPC services. I hope it was interesting and useful for you. Thanks a lot
for watching, Happy learning, and see you in the next lecture!