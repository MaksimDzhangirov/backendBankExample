# Switch DB driver from lib/pq to pgx

[Original video](https://www.youtube.com/watch?v=m9gYy5U0edQ)

Hello guys, welcome to the backend master class! As promised in the 
previous lecture, today we will learn how to switch the DB driver from
`lib/pq` to `pgx`. We've been using `lib/pq` in our project since the 
beginning of the course, and it has proved to be a good DB driver. However,
this package has been put into [maintenance mode](https://github.com/lib/pq#status), 
which means, it doesn't accept new features, but still accepts bug fixes and 
non-breaking changes from the community. The maintainers usually don't 
resolve the reported issues, so you can only get help from other community 
members. They also recommend switching to `pgx`, if you want to have new
features or a more reliable solution. Indeed, `pgx` is also a very good [DB
driver](https://github.com/jackc/pgx), and it is still under active 
development. We can install it by copying this "github.com/jackc/pgx/v5" 
import URL, paste it to the `main.go` file with a blank identifier. This will
keep the package in the import list, even when we're not directly using it
in the code yet.

```go
import (
    ...
    "github.com/hibiken/asynq"
    _ "github.com/jackc/pgx/v5"
    _ "github.com/lib/pq"
    "github.com/rakyll/statik/fs"
    ...
)
```

Then we save the file, and run

```shell
go mod tidy
```

in the terminal to download the package.

Alright, now if we open the `go.mod` file, we will see that `pgx` version 5
has been added to the list of dependencies.

```
github.com/hibiken/asynq v0.23.0
github.com/jackc/pgx/v5 v5.3.1
github.com/jordan-wright/email v4.0.1-0.20210109023952-943e75fe5223+incompatible
```

As we're using `sqlc` to generate codes to talk to the database, we'll have
to update the settings to tell it to use `pgx` DB driver. In the `sqlc`
documentation, under the `gen go` 
[section](https://docs.sqlc.dev/en/latest/reference/config.html#go), there's 
an option for the `sql_package`, which can be either `pgx/v4`, `pgx/v5`, or
`database/sql`. So far we've been using the default one, which is 
`database/sql`. Now we'll have to change this to `pgx/v5`, so I'm gonna
copy this `sql_package` option, paste it to our `sqlc.yaml` file, and set
its value to "pgx/v5".

```yaml
version: "2"
sql:
- schema: "db/migration"
  queries: "db/query"
  engine: "postgresql"
  gen:
    go: 
      package: "db"
      out: "db/sqlc"
      sql_package: "pgx/v5"
      emit_json_tags: true
      emit_interface: true
      emit_empty_slices: true
```

Before generating the codes, make sure your `sqlc` binary version is at least
1.18

```shell
sqlc version
v1.18.0
```

Then we can run 

```shell
make sqlc
sqlc generate
```

to regenerate Golang codes.

Of course, there will be some braking changes, but I'm gonna show you how
to fix them, one by one.

Let's start with the `account_test.go` file. Here,

```go
func TestGetAccount(t *testing.T) {
    ...
	require.WithinDuration(t, account1.CreatedAt, account2.CreatedAt, time.Second)
}
```

there are some issues with the `created_at` field, which is caused by 
the incompatible data types of the time.

`Pgx` defines its own time type called `Timestamptz`, while the 
`WithinDuration` function expects a `Time` type from the standard `time`
package. If we look into the generated `Account` struct

```go
type Account struct {
	ID        int64              `json:"id"`
	Owner     string             `json:"owner"`
	Balance   int64              `json:"balance"`
	Currency  string             `json:"currency"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
}
```

we'll see that the type of the `CreatedAt` field has been changed from 
`time.Time` to `pgtype.Timestamptz`. Although it's possible to access the
original `time.Time` value, since it's an inner field of the `Timestamptz`
type, I prefer to use the original `time.Time` type, since it's more 
convenient to use when the field is not nullable. The `Timestamptz` field
is only useful when the field can be null. We'll see how to use that type
later. For now, I'm gonna tell `sqlc` to use `time.Time` type instead.

To do that, we can use the `overrides` option with the `go_type` mapping 
key, as written here in the [Type overriding section](https://docs.sqlc.dev/en/latest/reference/config.html#type-overriding) of the documentation page. You can also config the
generated go struct tag (`go_struct_tag`), or tell `sqlc` to only use this
type, when the column is nullable (`nullable`).

In our case, I want `sqlc` to always use `time.Time`, so let's paste the
`overrides` option to our `sqlc.yaml` file.

```yaml
version: "2"
sql:
- schema: "db/migration"
  queries: "db/query"
  engine: "postgresql"
  gen:
    go: 
      package: "db"
      out: "db/sqlc"
      sql_package: "pgx/v5"
      emit_json_tags: true
      emit_interface: true
      emit_empty_slices: true
      overrides:
        - db_type: "uuid"
          go_type: "github.com/gofrs/uuid.UUID"
```

The `db_type` should be set to `timestamptz`, since it is the type we're
using in our DB schema design. And for the `go_type`, I'm gonna set it
to `time.Time`.

Just like that,

```yaml
version: "2"
sql:
- schema: "db/migration"
  queries: "db/query"
  engine: "postgresql"
  gen:
    go: 
      package: "db"
      out: "db/sqlc"
      sql_package: "pgx/v5"
      emit_json_tags: true
      emit_interface: true
      emit_empty_slices: true
      overrides:
        - db_type: "timestamptz"
          go_type: "time.Time"
```

and we can rerun

```shell
make sqlc
sqlc generate
```

in the terminal.

Now if we look at the model, all `CreatedAt` fields' types have been changed
back to `time.Time` as we wanted.

```go
type Account struct {
	ID        int64     `json:"id"`
	Owner     string    `json:"owner"`
	Balance   int64     `json:"balance"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
}
```

And for this reason, a lot of errors have been gone automatically. We still
have some more issues to fix, though. But first, I also notice that the 
UUID type has also been changed to pgx's own UUID type.

```go
type Session struct {
	ID           pgtype.UUID `json:"id"`
	Username     string      `json:"username"`
	RefreshToken string      `json:"refresh_token"`
	UserAgent    string      `json:"user_agent"`
	ClientIp     string      `json:"client_ip"`
	IsBlocked    bool        `json:"is_blocked"`
	ExpiresAt    time.Time   `json:"expires_at"`
	CreatedAt    time.Time   `json:"created_at"`
}
```

I don't like this very much, since that type is somehow a bit annoying to 
use compared to the Google UUID package that we've been using in the course.
So I'm gonna tell `sqlc` to use Google UUID type instead.

Just like before, we can override the UUID type with the type we want. In
this case, it's gonna be "github.com/google/uuid.UUID".

```yaml
version: "2"
sql:
- schema: "db/migration"
  queries: "db/query"
  engine: "postgresql"
  gen:
    go: 
      package: "db"
      out: "db/sqlc"
      sql_package: "pgx/v5"
      emit_json_tags: true
      emit_interface: true
      emit_empty_slices: true
      overrides:
        - db_type: "timestamptz"
          go_type: "time.Time"
        - db_type: "uuid"
          go_type: "github.com/google/uuid.UUID"
```

Then, let's run 

```shell
make sqlc
sqlc generate
```

again.

This time, if we look at one of the generated code files, we'll see that
UUID type's package has been changed to the one from Google.

```go
import (
	"context"
	"time"

	"github.com/google/uuid"
)

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

Alright, now let's continue fixing incompatible issues in other files. The
most important one is the `store.go` file, where we define the store and
implement the execute transaction function. But it's a bit more complicated,
so let's check other files first.

In the `tx_verify_email.go` file, we see an error with the `sql.NullBool` 
type.

```go
result.User, err = q.UpdateUser(ctx, UpdateUserParams{
    Username: result.VerifyEmail.Username,
    IsEmailVerified: sql.NullBool{
      Bool:  true,
      Valid: true,
    },
  })
```

That's because `pgx` uses its own nullable boolean type, so we can fix 
this by changing the type of this field to `pgtype.Bool`. Make sure you
import the correct package, which is `pgx/v5/pgtype`.

```go
import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

result.User, err = q.UpdateUser(ctx, UpdateUserParams{
    Username: result.VerifyEmail.Username,
    IsEmailVerified: pgtype.Bool{
      Bool:  true,
      Valid: true,
    },
  })
```

There's another similar issue in the `user_test.go` file, where we're 
passing in a `sql.NullString` object, while the generated code wants a
`pgtype.Text` type.

```go
updatedUser, err := testQueries.UpdateUser(context.Background(), UpdateUserParams{
  Username: oldUser.Username,
  FullName: sql.NullString{
    String: newFullName,
    Valid:  true,
  },
})
```

So, I'm gonna change all of the occurrences of this `sql.NullString` type
to `pgtype.Text`.

```go
updatedUser, err := testQueries.UpdateUser(context.Background(), UpdateUserParams{
  Username: oldUser.Username,
  FullName: pgtype.Text{
    String: newFullName,
    Valid:  true,
  },
})
```

They share the same inner field structure, so that should be enough to solve
the problem.

Alright, now it's time to fix the errors in the `main_test.go` file. To 
switch to `pgx`, we'll have to replace the `sql.DB` with pgx's connection
pool.

```go
func NewStore(db *sql.DB) Store {
	return &SQLStore{
		db:      db,
		Queries: New(db),
	}
}
```

It is defined in the `pgxpool` subpackage of `pgx/v5`, so make sure your
import path is correct. Now, if you look at sqlc's documentation, they also
have a section to talk about how to [use Go with `pgx`](https://docs.sqlc.dev/en/latest/guides/using-go-and-pgx.html). As you can see, `pgx/v5` is supported starting from
version 1.18, so make sure you have the latest `sqlc` binary version. 
There's one thing you must be careful here though. In their example code,
they use `pgx.Connect()` to create a connection to Postgres, but in fact,
it's not a good example to be used in production, because it just creates
1 single connection, while in reality, we'll always need a pool of 
connections in order to handle multiple requests in parallel. The `pgxpool`
package will help us manage the connection pools. There's an error with
this import, so let's try running


```shell
go mod tidy
```

in terminal to download the package.

Somehow the error doesn't disappear. But I'll get back to this later,
once we've fixed all other issues.

Now I'm gonna replace the `db` field in the `SQLStore` struct with a 
connection pool object. And initialize its value with the injected one
we receive in the `NewStore` function. We should also use that connection
pool to create a new `Queries` object.

```go
type SQLStore struct {
	connPool *pgxpool.Pool
	*Queries
}

func NewStore(connPool *pgxpool.Pool) Store {
	return &SQLStore{
		connPool: connPool,
		Queries:  New(connPool),
	}
}
```

OK, next I'm gonna show you how to fix the execute transaction function.

```go
tx, err := store.db.BeginTx(ctx, nil)
```

But first, I think it would be cleaner to move this piece of code into a 
separate file. I'm gonna call that file `exec_tx.go`. And it should be in
the same `db` package as other files.

OK, now, to start a transaction, we call the `BeginTx` method of the 
connection pool. It comes with some transaction options. But since we
don't need any custom options, we can simply use the `Begin()` method 
instead.

```go
func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.connPool.Begin(ctx)
  ...
}
```

Behind the scene, it also calls the `BeginTx()` method, but with
an empty transaction option. The options are there in case you need to set

```go
func (p *Pool) Begin(ctx context.Context) (pgx.Tx, error) {
	return p.BeginTx(ctx, pgx.TxOptions{})
}

type TxOptions struct {
	IsoLevel       TxIsoLevel
	AccessMode     TxAccessMode
	DeferrableMode TxDeferrableMode
	BeginQuery string
}
```

an isolation level, access mode, or deferrable mode for your transaction.

OK, let's go back to our code. There's still something we need to update 
here. The rollback function of `pgx` requires a context, so we have to pass
in the input context parameter. Same thing for the `tx.Commit()` function.
And that's basically it.

```go
func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.connPool.Begin(ctx)
	if err != nil {
		return err
	}

	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit(ctx)
}
```

We've fixed all issues in the `execTx()` function. Next, let's move to the
`main_test.go` file. I'm gonna remove both the `testQueries` and `testDB`
object, and replace them with only 1 single `testStore` variable.

```go
var testStore Store
```

Then inside the `TestMain()` function, let's create a new connection pool
by calling `pgxpool.New()`. It doesn't need a DB driver name here, since
it knows for sure that we're connecting to Postgres when we use `pgx`
package. But we'll have to pass in a context as the first argument of the
function. So I'm gonna use a background context here. Then, instead of
creating a `testQueries`, I'm gonna create a new `testStore` with the 
connection pool above. Now, we can use this `testStore` in all of the tests.

```go
func TestMain(m *testing.M) {
	config, err := util.LoadConfig("../..")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	connPool, err := pgxpool.New(context.Background(), config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	testStore = NewStore(connPool)

	os.Exit(m.Run())
}
```

For example, in the `account_test.go` file, I'm gonna replace all occurrences
of `testQueries` with `testStore`. In fact, we can do this for all other
files as well. Just right click the "sqlc" folder, and choose "Find in 
folder", click on this arrow to expand the field. Then, in the replace box,
let's put "testStore" and in the search box, "testQueries".

![](../images/part66/1.png)

As you can see, `testQueries` are being used in a lot of places, so let's
press this "Replace All" button to replace all of its occurrences with 
"testStore".

![](../images/part66/2.png)

Once it's done, all errors in those test files are gone.

But we still have some issues in the `store_test.go` file. In this file,
we're also creating a new store, but we don't have to do that anymore,
since we already have the `testStore`. So here I can just replace all
occurrences of "store" with "testStore". Then let's remove the store 
creation from the `TestTransferTxDeadlock` function. And that should be it!
No more errors in the `store_test.go` file.

However, we still have an issue with the import `pgxpool` statement in the
`main_test.go` file. Let me try to run 

```shell
go mod tidy
go: downloading github.com/jackc/puddle v1.1.3
```

again in the terminal to see if that can fix the issue or not. OK, so now
it seems to work.

We don't have any errors in the `sqlc` package. I'm gonna run the whole
package tests to see what happens.

```shell
Running tool: /usr/local/go/bin/go test -timeout 30s -coverprofile=/var/folders/wd/yz00fp3j0dv8lcc986n2whgc0000gn/T/vscode-goG2otyF/go-code-cover github.com/techschool/simplebank/db/sqlc
-v -count=1

=== RUN   TestCreateAccount
--- PASS: TestCreateAccount (0.09s)
=== RUN   TestGetAccount
--- PASS: TestGetAccount (0.07s)
=== RUN   TestUpdateAccount
--- PASS: TestUpdateAccount (0.07s)
=== RUN   TestDeleteAccount
    /Users/quangpham/Projects/techschool/simplebank/db/sqlc/account_test.go:79:
        	Error Trace:	/Users/quangpham/Projects/techschool/simplebank/db/sqlc/account_test.go:79
        	Error:      	Error message not equal:
        	            	expected: "sql: no rows in result set"
        	            	actual  : "no rows in result set"
        	Test:       	TestDeleteAccount
--- FAIL: TestDeleteAccount (0.07s)
=== RUN   TestListAccounts
--- PASS: TestListAccounts (0.60s)
...
```

Alright, so it seems all tests are able to run, but we've got a failing
case for the `TestDeleteAccount`. The expected error message is a bit
different so let's take a look at the content of the test case to see 
what's going on.

```go
func TestDeleteAccount(t *testing.T) {
	account1 := createRandomAccount(t)
	err := testStore.DeleteAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	account2, err := testStore.GetAccount(context.Background(), account1.ID)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, account2)
}
```

Here, after deleting the account, we're trying to get it from the DB, and
we expect to see an error because the deleted account should not be found.
But the problem is, we're expecting to see the `ErrNoRows` from 
`database/sql` package, while in fact, `pgx` doesn't return this type of
error in its implementation. So, to make our code less dependent on the 
implementation details of the library, I'm gonna create a new file called
`error.go` inside the `db` package. Then let's define our own 
record-not-found error variable, and assign its value to `pgx.ErrNoRows`.

```go
package db

import "github.com/jackc/pgx/v5"

var ErrRecordNotFound = pgx.ErrNoRows
```

Now every time we want to check for record not found, we will refer to this
error variable. This way, if we ever have to change the DB driver again, we
only need to update its value in 1 single place.

OK, so let's go back to the `account_test.go` file, and change this 
`sql.ErrNoRows` to `ErrRecordNotFound`.

```go
func TestDeleteAccount(t *testing.T) {
	account1 := createRandomAccount(t)
	err := testStore.DeleteAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	account2, err := testStore.GetAccount(context.Background(), account1.ID)
	require.Error(t, err)
	require.EqualError(t, err, ErrRecordNotFound.Error())
	require.Empty(t, account2)
}
```

Now, if we run the whole package tests again, everything will pass.

```shell
Running tool: /usr/local/go/bin/go test -timeout 30s -coverprofile=/var/folders/wd/yz00fp3j0dv8lcc986n2whgc0000gn/T/vscode-goG2otyF/go-code-cover github.com/techschool/simplebank/db/sqlc
-v -count=1


PASS
coverage: 65.0% of statements
ok  	github.com/techschool/simplebank/db/sqlc	3.085s	coverage: 65.0% of statements
```

But it's not the end! WE're still using `sql.ErrNoRows` in other places.
Let's search for it in the whole project.

![](../images/part66/3.png)

As you can see, there are many occurrences in multiple files. Now, I'm
gonna replace all of them with the `ErrRecordNotFound` variable, that we've
defined in the `db` package.

![](../images/part66/4.png)

Alright, let's look at some files to see what has been changed.

In `account_test.go` file, we're returning the error as part of the 
building stub step.

```go
buildStubs: func(store *mockdb.MockStore) {
  store.EXPECT().
    GetAccount(gomock.Any(), gomock.Eq(account.ID)).
    Times(1).
    Return(db.Account{}, db.ErrRecordNotFound)
},
```

And in the `account.go` file, we're checking the error type in order to 
return the correct status code to the client.

```go
if err != nil {
  if err == db.ErrRecordNotFound {
    ctx.JSON(http.StatusNotFound, errorResponse(err))
    return
  }

  ctx.JSON(http.StatusInternalServerError, errorResponse(err))
  return
}
```

It's fine to compare the error with an equal operator like this, but what
could be better is using `errors.Is()` function, because it can match the
error even when it is wrapped. 

```go
if err != nil {
  if errors.Is(err, db.ErrRecordNotFound) {
    ctx.JSON(http.StatusNotFound, errorResponse(err))
    return
  }

  ctx.JSON(http.StatusInternalServerError, errorResponse(err))
  return
}
```

So let's try to replace all occurrences of the equal error comparison with
`errors.Is()`.

![](../images/part66/5.png)

There are 6 places like that, all have been updated.

Now I think we're good to go!

Let's try to run all tests in the `api` package.

```shell
Running tool: /usr/local/go/bin/go test -timeout 30s -coverprofile=/var/folders/wd/yz00fp3j0dv8lcc986n2whgc0000gn/T/vscode-goG2otyF/go-code-cover github.com/techschool/simplebank/api -v -count=1

...

PASS
coverage: 75.5% of statements
ok  	github.com/techschool/simplebank/api	1.455s	coverage: 75.5% of statements
```

OK, they all passed. Perfect!

Alright, let's check other packages to see if there are still errors 
somewhere.

I see some red files in the `gapi` package, so let's take a look at them!

In the `rpc_update_user.go` file, we still have some nullable string 
fields, whose type should be `pgtype.Text` instead of `sql.NullString`.
So let's search for them, and replace all with `pgtype.Text`.

![](../images/part66/6.png)

OK, it's done.

However, there's a red line under the import `pgtype` package.

```go
import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgtype"
	...
)
```

Sometimes VSCode tries to import the package for us, but it imports the 
wrong one. In this case, it should not import `jackc/pgtype`, but should 
be the subpackage of `pgx/v5` instead.

```go
import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
  ...
)
```

There's still one more error with the `PasswordChangedAt` field when we 
try to update it.

```go
arg.PasswordChangedAt = sql.NullTime{
  Time:  time.Now(),
  Valid: true,
}
```

Here, since the field is optional, `sqlc` cannot use the `time.Time` type,
so it uses the `pgtype.Timestamptz` type instead. Therefore, we have to 
change this to the correct type, that the generated struct wanted.

```go
arg.PasswordChangedAt = pgtype.Timestamptz{
  Time:  time.Now(),
  Valid: true,
}
```

OK, it seems we still have some issues in the test file.

```go
import (
	"context"
	"database/sql"
	"testing"
	"time"
	
	"github.com/jackc/pgtype"
  ...
)
```

It's wrong import issue again, so let's update it to `pgx/v5/pgtype`.

```go
import (
	"context"
	"database/sql"
	"testing"
	"time"
	
	"github.com/jackc/pgx/v5/pgtype"
  ...
)
```

Now we can run the package test to make sure it's working well.

```shell
Running tool: /usr/local/go/bin/go test -timeout 30s -coverprofile=/var/folders/wd/yz00fp3j0dv8lcc986n2whgc0000gn/T/vscode-goG2otyF/go-code-cover github.com/techschool/simplebank/gapi
-v -count=1

=== RUN   TestCreateUserAPI
=== RUN   TestCreateUserAPI/OK
...
PASS
coverage: 43.2% of statements
ok      github.com/techschool/simplebank/gapi     0.761s  coverage: 43.2% of statements
```

Indeed, all tests passed. Excellent!

Next step, let's try to run the server.

```shell
make server
go run main.go
# command-line-arguments
./main.go:50:23 cannot use conn (variable of type *sql.DB) as type *pgpool.Pool in argument to db.NewStore
make: *** [server] Error 2
```

Oops, we've got an error: cannot use the connection variable of type sql.DB.
That's because when creating a new store, it requires a `pgxpool.Pool` 
object.

So, let's open the `main.go` file and fix this!

First, I'm gonna remove both of these blank imports.

```go
_ "github.com/jackc/pgx/v5"
_ "github.com/lib/pq"
```

Then, in the `main` function, I'm gonna create a connection pool by calling
`pgxpool.New()`, and passing in a background context, together with the DB
source URL.

```go
connPool, err := pgxpool.New(context.Background(), config.DBSource)
if err != nil {
	log.Fatal().Err(err).Msg("cannot connect to db")
}
```

We should check the import to make sure it's importing the right package.

```go
"github.com/jackc/pgx/v5/pgxpool"
```

OK, now we can use the connection pool to create a new store.

```go
store := db.NewStore(connPool)
```

And since we're no longer using the DB_DRIVER environment variable, I'm
gonna remove it from the `Config` struct, and from the app.env file as well.

```go
type Config struct {
	Environment          string        `mapstructure:"ENVIRONMENT"`
	DBSource             string        `mapstructure:"DB_SOURCE"`
	MigrationURL         string        `mapstructure:"MIGRATION_URL"`
	RedisAddress         string        `mapstructure:"REDIS_ADDRESS"`
	HTTPServerAddress    string        `mapstructure:"HTTP_SERVER_ADDRESS"`
	GRPCServerAddress    string        `mapstructure:"GRPC_SERVER_ADDRESS"`
	TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	EmailSenderName      string        `mapstructure:"EMAIL_SENDER_NAME"`
	EmailSenderAddress   string        `mapstructure:"EMAIL_SENDER_ADDRESS"`
	EmailSenderPassword  string        `mapstructure:"EMAIL_SENDER_PASSWORD"`
}
```

```
ENVIRONMENT=development
DB_SOURCE=postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable
MIGRATION_URL=file://db/migration
HTTP_SERVER_ADDRESS=0.0.0.0:8080
GRPC_SERVER_ADDRESS=0.0.0.0:9090
TOKEN_SYMMETRIC_KEY=12345678901234567890123456789012
ACCESS_TOKEN_DURATION=15m
REFRESH_TOKEN_DURATION=24h
REDIS_ADDRESS=0.0.0.0:6379
EMAIL_SENDER_NAME=Simple Bank
EMAIL_SENDER_ADDRESS=simplebanktest@gmail.com
EMAIL_SENDER_PASSWORD=jekfcygyenvzekke
```

Now let's try running "make server" in the terminal to see if it's working
or not.

```shell
make server
go run main.go
11:54PM INF db migrated successfully
11:54PM INF start gRPC server at [::]:9090
11:54PM INF start task processor
11:54PM INF Starting processing
11:54PM INF start HTTP gateway server at [::]:8080
```

The server has been started successfully. In Postman, I'm gonna try sending
a request to create a new user.

![](../images/part66/7.png)

We've got a duplicate violation error because this username has been taken.
So the API is working, but there's something not right about it, because
the status code it returned in this case is 500 Internal Server Error, while
it should be something like "409 Conflict" instead.

Looking at the code of the create user handler function, you'll see that 
we're still checking the error using lib/pq's `Error` interface.

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

But the error returned this time actually comes from `pgx`. So that's why
it's not recognizable, and thus a 500 status code is returned to the 
client. If we search for import `lib/pq`, we'll see that it is still being
used in many other places.

![](../images/part66/8.png)

And it will require a significant amount of code modification to fix all of
them. Since this video is already long, I'll show you how to do that in 
the next lecture. In the meantime, you can try to fix this issue on your
own, consider it as an exercise, and you can compare your codes with mine
in the next video.

Thanks a lot for watching this video. I hope it was interesting and useful
for you. Happy learning and see you in the next lecture!