# Implement role-based access control (RBAC) in Go

[Original video](https://www.youtube.com/watch?v=Py7dRhtuJ3E)

Hello guys, welcome to the backend master class! In this video, we're 
gonna learn about **RBAC**, or **R**ole-**B**ased **A**ccess **C**ontrol. It
is simply a way tp control who can access your API based on their role.
For example, in our simple bank project, we have an RPC to update user's
information. And for now, only the authenticated user can update their 
own information. In a real banking system, we might want to allow an 
employee of the bank, to be able to update the information on behalf of
the user. So there will be multiple types of user in the system, with 
different role.

Let's create a `role.go` file in the `util` package to keep track of all
those roles. For demo purposes, I'm gonna define only 2 roles: a depositor
role for normal users, who deposit their money in the bank. And a banker 
role for an employee of the bank, who's in charge of customer service.

```go
package util

const (
    DepositorRole = "depositor"
    BankerRole    = "banker"
)
```

Let's say, a depositor can only update their own information, while a 
banker can update the information of any users. OK, so next we'll have to
modify the DB schema to manage user role.

## Modify the DB schema

In the `db.dbml` file, let's add a new `role` column of type `varchar`. 
It should be `not null`, with a default value `depositor`. This default
value is important to make sure that all existing users in the DB, as well
as new users, will be a `depositor` if we don't explicitly set a different
value for this column.

```
Table users as U {
  username varchar [pk]
  role varchar [not null, default: 'depositor']
  hashed_password varchar [not null]
  full_name varchar [not null]
  email varchar [unique, not null]
  is_email_verified bool [not null, default: false]
  password_changed_at timestamptz [not null, default: '0001-01-01']
  created_at timestamptz [not null, default: `now()`]
}
```

Now let's run

```shell
make db_schema
dbml2sql --postgres -o doc/schema.sql doc/db.dbml
  ✔ Generated SQL dump file (PostgreSQL): schema.sql
```

in the terminal to generate new SQL schema from the modified DBML file.

And voilà, in the `schema.sql` file,

```sql
CREATE TABLE "users" (
  "username" varchar PRIMARY KEY,
  "role" varchar NOT NULL DEFAULT 'depositor',
  "hashed_password" varchar NOT NULL,
  "full_name" varchar NOT NULL,
  "email" varchar UNIQUE NOT NULL,
  "is_email_verified" bool NOT NULL DEFAULT false,
  "password_changed_at" timestamptz NOT NULL DEFAULT '0001-01-01',
  "created_at" timestamptz NOT NULL DEFAULT (now())
);
```

the `role` column has been added to the `CREATE TABLE "users"` query. Let's
copy this `"role" varchar NOT NULL DEFAULT 'depositor'` line, as we're
gonna use it for the DB migration. In the terminal, I'm gonna run 

```sql
make new_migration name=add_role_to_users
    migrate create -ext sql -dir db/migration -seq add_role_to_users
/Users/quangpham/Projects/techschool/simplebank/db/migration/000005_add_role_to_users.up.sql
/Users/quangpham/Projects/techschool/simplebank/db/migration/000005_add_role_to_users.down.sql
```

with this name: `add_role_to_users`.

Then, in the `db/migration` folder, let's open the migration up file. The
command we'll use add a new column is:

```sql
ALTER TABLE "users" ADD COLUMN
```

then paste in the definition of the "role" column that we've copied before.

```sql
ALTER TABLE "users" ADD COLUMN "role" varchar NOT NULL DEFAULT 'depositor';
```

Now, for the migration down file, we'll use the same `ALTER TABLE` query,
but with a `DROP COLUMN` clause.

```sql
ALTER TABLE "users" DROP COLUMN "role";
```

And that's it!

Let's run 

```shell
make migrateup
migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up
2023/10/30 14:02:39 Start buffering 5/u add_role_to_users
2023/10/30 14:02:39 Read and execute 5/u add_role_to_users
2023/10/30 14:02:39 Finished 5/u add_role_to_users (read 11.215458ms, ran 5.746125ms)
2023/10/30 14:02:39 Finished after 19.971625ms
2023/10/30 14:02:39 Closing source and database
```

in the terminal to execute the migration up script.

After it's done, we can open TablePlus to check the "users" table.

![](../images/part70/1.png)

And here it is, the `role` column has been added.

Now, to make sure the migration down script is also working, I'm gonna 
run

```shell
make migratedown1
migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose down 1
2023/10/30 14:03:04 Start buffering 5/d add_role_to_users
2023/10/30 14:03:04 Read and execute 5/d add_role_to_users
2023/10/30 14:03:04 Finished 5/d add_role_to_users (read 10.254917ms, ran 7.342625ms)
2023/10/30 14:03:04 Finished after 20.684625ms
2023/10/30 14:03:04 Closing source and database
```

in the terminal and check TablePlus again.

This time, the `role` column is gone, as expected.

![](../images/part70/2.png)

It's all good now, so we can run

```shell
make migrateup
migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up
2023/10/30 14:03:14 Start buffering 5/u add_role_to_users
2023/10/30 14:03:14 Read and execute 5/u add_role_to_users
2023/10/30 14:03:14 Finished 5/u add_role_to_users (read 11.718625ms, ran 5.179959ms)
2023/10/30 14:03:14 Finished after 19.319083ms
2023/10/30 14:03:14 Closing source and database
```

to add back the `role` column.

Now, although the `role` column is added to the DB, it's not added to the
`User` struct model yet. So we'll have to run

```shell
make sqlc
sqlc generate
```

to regenerate Golang code from SQL queries.

```go
type User struct {
    Username          string    `json:"username"`
    HashedPassword    string    `json:"hashed_password"`
    FullName          string    `json:"full_name"`
    Email             string    `json:"email"`
    PasswordChangedAt time.Time `json:"password_changed_at"`
    CreatedAt         time.Time `json:"created_at"`
    IsEmailVerified   bool      `json:"is_email_verified"`
    Role              string    `json:"role"`
}
```

Only then, the new `Role` field will show up in the `User` struct.

Alright, next step, we will have to add this "role" to the access
token. So I'm gonna modify the CreateToken method of the token `Maker`
interface with a new `role` string parameter.

```go
// Maker is an interface for managing tokens
type Maker interface {
    // CreateToken creates a new token for a specific username and duration
    CreateToken(username string, role string, duration time.Duration) (string, *Payload, error)
    
    // VerifyToken checks if the token is valid or not
    VerifyToken(token string) (*Payload, error)
}
```

Then, we'll need to modify the `PasetoMaker` struct, which is implementation
the token `Maker` interface. The `role` string param should be added to 
this `CreateToken` function.

```go
func (maker *PasetoMaker) CreateToken(username string, role string, duration time.Duration) (string, *Payload, error) {
    ...
}
```

And more importantly, in the function that creates a new token payload.

```go
func NewPayload(username string, role string, duration time.Duration) (*Payload, error) {
    ...
}
```

We must add a new `Role` field to the Payload struct, and save the input
`role` value to this field.

```go
type Payload struct {
    ID        uuid.UUID `json:"id"`
    Username  string    `json:"username"`
    Role      string    `json:"role"`
    IssuedAt  time.Time `json:"issued_at"`
    ExpiredAt time.Time `json:"expired_at"`
}
```

So that later, when parsing the token, we will know the role of the 
logged-in user.

```go
func NewPayload(username string, role string, duration time.Duration) (*Payload, error) {
    ...
    
    payload := &Payload{
        ID:        tokenID,
        Username:  username,
        Role:      role,
        IssuedAt:  time.Now(),
        ExpiredAt: time.Now().Add(duration),
    }
    return payload, nil
}
```

After modifying this, we can go back to the `PasetoMaker`, and send the 
`role` param to this `NewPayload` function call.

```go
func (maker *PasetoMaker) CreateToken(username string, role string, duration time.Duration) (string, *Payload, error) {
    payload, err := NewPayload(username, role, duration)
    ...
}
```

There will be some broken tests that we must fix as well.

As the `CreateToken` method now requires a role, I'm gonna define a `role`
variable here,

```go
func TestPasetoMaker(t *testing.T) {
    ...
    
    username := util.RandomOwner()
    role :=
    duration := time.Minute
    
    ...
}
```

with the value of `util.DepositorRole` and pass it into this function.

```go
func TestPasetoMaker(t *testing.T) {
    ...
    
    token, payload, err := maker.CreateToken(username, role, duration)
    ...
}
```

Then, add one more check here 

```go
func TestPasetoMaker(t *testing.T) {
    ...
    
    require.NotZero(t, payload.ID)
    require.Equal(t, username, payload.Username)
    require.Equal(t, role, payload.Role)
    require.WithinDuration(t, issuedAt, payload.IssuedAt, time.Second)
    require.WithinDuration(t, expiredAt, payload.ExpiredAt, time.Second)
}
```

to make sure that the role is correctly stored in the token payload. For
the expired token case, I'll just set the depositor role directly here,

```go
func TestExpiredPasetoToken(t *testing.T) {
    ...
    
    token, payload, err := maker.CreateToken(util.RandomOwner(), util.DepositorRole, -time.Minute)
    ...
}
```

since we don't need to verify the payload of the expired token.

OK, so the `PasetoMaker` is now ready.

If you are using JWT, you will have to make the same modification to the
`JWTMaker` struct.

First, add the `role` string to the `CreateToken` method, and pass it to
the `NewPayload` function call.

```go
func (maker *JWTMaker) CreateToken(username string, role string, duration time.Duration) (string, *Payload, error) {
    payload, err := NewPayload(username, role, duration)
    ...
}
```

Then fix the test, by introducing a new `role` variable, passing it to the
`CreateToken` method, check the payload's role to make sure it matches the 
input value, fix the expired token case just like we did before, and do
the same for the Algorithm None test case.

```go
func TestJWTMaker(t *testing.T) {
    ...
    
    username := util.RandomOwner()
    role := util.DepositorRole
    duration := time.Minute
    
    ...
    
    token, payload, err := maker.CreateToken(username, role, duration)
    ...
    
    require.NotZero(t, payload.ID)
    require.Equal(t, username, payload.Username)
    require.Equal(t, role, payload.Role)
    require.WithinDuration(t, issuedAt, payload.IssuedAt, time.Second)
    require.WithinDuration(t, expiredAt, payload.ExpiredAt, time.Second)
}

func TestExpiredJWTToken(t *testing.T) {
    maker, err := NewJWTMaker(util.RandomString(32))
    require.NoError(t, err)
    
    token, payload, err := maker.CreateToken(util.RandomOwner(), util.DepositorRole, -time.Minute)
    ...
}

func TestInvalidJWTTokenAlgNone(t *testing.T) {
    payload, err := NewPayload(util.RandomOwner(), util.DepositorRole, time.Minute)
    require.NoError(t, err)
	
    ...
}
```

That's basically it.

Once everything is fixed, we should run the whole package tests to make 
sure the token makers are working well.

```shell
Running tool: /usr/local/go/bin/go test -timeout 30s -coverprofile=/var/folders/9c/5cd58chd43753c504gryg82m0000gn/T/vscode-goxIPGhh/go-code-cover github.com/techschool/simplebank/token
-v -count=1
=== RUN   TestJWTMaker
--- PASS: TestJWTMaker (0.00s)
=== RUN   TestExpiredJWTToken
--- PASS: TestExpiredJWTToken (0.00s)
=== RUN   TestInvalidJWTTokenAlgNone
--- PASS: TestInvalidJWTTokenAlgNone (0.00s)
=== RUN   TestPasetoMaker
--- PASS: TestPasetoMaker (0.00s)
=== RUN   TestExpiredPasetoToken
--- PASS: TestExpiredPasetoToken (0.00s)
PASS
coverage: 85.7% of statements
ok      github.com/techschool/simplebank/token    0.288s  coverage: 85.7% of statements
```

All passed! Excellent!

However, there are still some errors in the `gapi` and `api` packages 
that we need to fix. First, in the `main_test.go` file of the `gapi` 
package, the `newContextWithBearerToken()` function is used to create
a context with gRPC metadata, that contains the access token for unit
testing. And now, since the `CreateToken` method requires a `role` string,
we have to add that role to the input arguments of this function, and use 
it to call the `CreateToken` method.

```go
func newContextWithBearerToken(t *testing.T, tokenMaker token.Maker, username string, role string, duration time.Duration) context.Context {
    ctx := context.Background()
    accessToken, _, err := tokenMaker.CreateToken(username, role, duration)
    ...
}
```

Alright, now comes one important thing, we must update the login user RPC
so that it store the role of the user in the payload of the access token
and refresh token. It's pretty simple, because the `Role` field has been 
added to the `User` struct model before. So all we have to do is, passing
its value to the function calls that creates the access token and the
refresh token.

```go
func (server *Server) LoginUser(ctx context.Context, req *pb.LoginUserRequest) (*pb.LoginUserResponse, error) {
    ...
    
    accessToken, accessPayload, err := server.tokenMaker.CreateToken(
        user.Username,
        user.Role,
        server.config.AccessTokenDuration,
    )
    ...
    
    refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(
        user.Username,
        user.Role,
        server.config.RefreshTokenDuration,
    )
    ...
}
```

Finally, we must fix the errors in the `rpc_update_user_test.go` file, by
adding `user.Role` to calls of the `newContextWithBearerToken` function.

```go
buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
    return newContextWithBearerToken(t, tokenMaker, user.Username, user.Role, time.Minute)
}
```

There are several places like that. Once you have updated everything, 
let's run the whole package tests.

```shell
...
--- PASS: TestUpdateUserAPI (0.06s)
    --- PASS: TestUpdateUserAPI/OK (0.00s)
    --- PASS: TestUpdateUserAPI/UserNotFound (0.00s)
    --- PASS: TestUpdateUserAPI/InvalidEmail (0.00s)
    --- PASS: TestUpdateUserAPI/ExpiredToken (0.00s)
    --- PASS: TestUpdateUserAPI/NoAuthorization (0.00s)
PASS
coverage: 42.9% of statements
ok      github.com/techschool/simplebank/gapi    0.652s  coverage: 42.9% of statements
```

All tests passed. So we're good to go! Note that there are still some
errors in the `api` package, where we implement HTTP server using Gin
framework instead of gRPC.

But the fix should be pretty similar to what we've done for the `gapi`
package, So I'll leave that as an exercise for you to do on your own.

That way, we can keep this video focused on the implementation of the 
role-based access control. Don't forget that, you can always check out
my code on GitHub whenever you want.

OK, so now, we've added role to the token payload in the login API. The
next step would be: using that role to control access to other APIs.

We will do that by modifying the `authorizeUser()` function, since this
function will be called by all APIs that requires user authentication.
I'm gonna add a new argument called `accessibleRoles` of type string 
array.

```go
func (server *Server) authorizeUser(ctx context.Context, accessibleRoles []string) (*token.Payload, error) {
    ...
}
```

This argument will contain the list of roles that can access the target
API. Then, at the bottom of this function, when the token payload has been
parsed and verified, we're gonna check if the `payload.Role` has permission
to call the target API or not. If it doesn't, we will return an error 
saying "permission denied".

```go
func (server *Server) authorizeUser(ctx context.Context, accessibleRoles []string) (*token.Payload, error) {
    ...
    
    if !hasPermission(payload.Role, accessibleRoles) {
        return nil, fmt.Errorf("permission denied")
    }
    
    return payload, nil
}
```

Now let's implement the `hasPermission()` function. It will take the user's
role and a list of accessible roles as input arguments, and return a `bool`
value, indicating whether the user's role belong to the list of accessible
ones or not.

```go
func hasPermission(userRole string, accessibleRoles []string) bool {
    ...	
}
```

To check that, we simply iterate through each value of the accessible roles.
If one of the value matches user's role, we return `true`. If at the end
of the loop we didn't find any matching role, then we return `false` in 
this case.

```go
func hasPermission(userRole string, accessibleRoles []string) bool {
    for _, role := range accessibleRoles {
        if userRole == role {
            return true
        }
    }
    return false
}
```

Alright, now the `authorizeUser()` function is ready, we can open the 
UpdateUser RPC to use it. As I said at the beginning, we will allow both
the banker role and depositor role to access this API, so here we're 
passing them as the second argument of this function.

```go
func (server *Server) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
    authPayload, err := server.authorizeUser(ctx, []string{util.BankerRole, util.DepositorRole})
    ...
}
```

But that's not enough, because here

```go
if authPayload.Username != req.GetUsername() {
    return nil, status.Errorf(codes.PermissionDenied, "cannot update other user's info")
}
```

we also check that the username in the auth payload must match the username
of the account we want to update. So if a banker calls this API, they still
can't update information of a user with a different username. That's why
we have to change this condition, so that we only force matching username
when the authenticated role is not banker.

```go
if authPayload.Role != util.BankerRole && authPayload.Username != req.GetUsername() {
    return nil, status.Errorf(codes.PermissionDenied, "cannot update other user's info")
}
```

Alright, now it's time to run some tests to see how it's working. First, 
I'm gonna open the `rpc_update_user_test.go` file, and run the package
tests.

```shell
Running tool: /usr/local/go/bin/go test -timeout 30s -coverprofile=/var/folders/9c/5cd58chd43753c504gryg82m0000gn/T/vscode-goxIPGhh/go-code-cover github.com/techschool/simplebank/gapi -v -count=1
...
=== RUN   TestUpdateUserAPI/ExpiredToken    
=== RUN   TestUpdateUserAPI/NoAuthorization
--- FAIL: TestUpdateUserAPI (0.06s)
    --- FAIL: TestUpdateUserAPI/OK (0.00s)
    --- FAIL: TestUpdateUserAPI/UserNotFound (0.00s)
    --- FAIL: TestUpdateUserAPI/InvalidEmail (0.00s)
    --- PASS: TestUpdateUserAPI/ExpiredToken (0.00s)
    --- PASS: TestUpdateUserAPI/NoAuthorization (0.00s)
FAIL
coverage: 32.2% of statements
FAIL    github.com/techschool/simplebank/gapi   0.540s
```
 
OK, so there are several failing cases. The reason for this is that, when
mocking a random user object here,

```go
user, _ := randomUser(t)
```

we didn't specify the role of the user, which means, the role was empty
in the access token. And thus, we got permission denied when calling the
API.

To fix this, I'm gonna set the role to `util.DepositorRole` here.

```go
func randomUser(t *testing.T) (user db.User, password string) {
    ...
    
    user = db.User{
        Username:       util.RandomOwner(),
        Role: util.DepositorRole,
        HashedPassword: hashedPassword,
        FullName:       util.RandomOwner(),
        Email:          util.RandomEmail(),
    }
    return
}
```

And rerun the package tests.

```shell
Running tool: /usr/local/go/bin/go test -timeout 30s -coverprofile=/var/folders/9c/5cd58chd43753c504gryg82m0000gn/T/vscode-goxIPGhh/go-code-cover github.com/techschool/simplebank/gapi -v -count=1
...
=== RUN   TestUpdateUserAPI/ExpiredToken    
=== RUN   TestUpdateUserAPI/NoAuthorization
--- PASS: TestUpdateUserAPI (0.06s)
    --- PASS: TestUpdateUserAPI/OK (0.00s)
    --- PASS: TestUpdateUserAPI/UserNotFound (0.00s)
    --- PASS: TestUpdateUserAPI/InvalidEmail (0.00s)
    --- PASS: TestUpdateUserAPI/ExpiredToken (0.00s)
    --- PASS: TestUpdateUserAPI/NoAuthorization (0.00s)
PASS
coverage: 43.7% of statements
ok    github.com/techschool/simplebank/gapi   0.799s   coverage: 43.7% of statements
```

This time, all tests passed as usual.

You can also write more unit tests to cover the case with `Banker` role.

## Perform some manual tests using Postman

For now, I'm gonna test this case manually using Postman. So let's open
the terminal and start the server.

```shell
make server
go run main.go
2:32PM INF db migrated successfully
2:32PM INF start gRPC server at [::]:9090
2:32PM INF start task processor
2:32PM INF Starting processing
2:32PM INF start HTTP gateway server at [::]:8080
```

Then, in Postman, I'm gonna create 3 different users. The first one is 
`techschool`, the second one is `Alice`, and the third one is `Bob`.

![](../images/part70/3.png)

![](../images/part70/4.png)

![](../images/part70/5.png)

Since we don't explicitly set the role in the create user API, their roles 
will be set to the default value, which is "depositor".

![](../images/part70/6.png)

To test different scenarios, I'm gonna change the role of `techschool`
to "banker".

![](../images/part70/7.png)

Alright, now let's get back to Postman and hit this Login User RPC using
`techschool` account.

![](../images/part70/8.png)

It's successful, and we've got an access token here.

![](../images/part70/9.png)

Normally we would have to manually copy this value and paste it to the 
Authorization metadata header of the Update User RPC. But I'm gonna show
you a quicker way to do that automatically.

In the Login User RPC, let's open the `Scripts` tab. Choose the "After 
response" section, and click on this script template to test status
code is 0.

![](../images/part70/10.png)

You can scroll down a bit and add another script to test a property exists
for all messages. In this case, we want to check that the response message
has the `access_token` field.

```javascript
pm.test('All reponse messages have proterty "access_token"',
    function() {
    pm.response.messages.to.have.property('access_token');
});
```

And we will take the value from this field to store in a collection variable.

```javascript
pm.collectionVariables.set("access_token", pm.response.messages.all()[0].data.access_token)
```

The variable name is "access_token" and its value can be gotten by calling
`pm.response.messages.all()`, then take the first message, and get its
`data.access_token` field.

Now, if we invoke the Login User RPC again, we'll get a new access token,
and in the `Test results` tab, we can see 2 passing tests.

![](../images/part70/11.png)

So if we open the collection variable list, there will be an 
`access_token` variable with the value exactly like in the response
message.

![](../images/part70/12.png)

So now, we can use this variable in the metadata of the Update User RPC.
In the `Authorization` tab, let's select `Bearer Token` type, and in the 
`Token` box, we can use a pair of double curly bracket to refer to the
`access_token` variable we've just created.

![](../images/part70/13.png)

And with this, we can safely remove the Authorization metadata, 

![](../images/part70/14.png)

because Postman will automatically take the value from the variable, and
set it up for us when sending the request.

Alright, now as we're using an access token of `techschool`, a banker
user, we can try to update the full name of user "alice" to "Alice 1".

![](../images/part70/15.png)

Oops, we've got an error: "invalid argument".

![](../images/part70/16.png)

Ah OK, the name can only contain letters or space, so I'm gonna change 
this to the word "One" like this.

![](../images/part70/17.png)

And invoke the Update User RPC again.

![](../images/part70/18.png)

This time, the request is successful, and the full name has been changed
to "Alice One".

Next, let's try login with "Bob", a depositor user. When we do this, 
the access token variable's value will change automatically.

![](../images/part70/19.png)

So now, if we try to change the full name to "Alice Two", this request
will be sent using Bob's access token. And we will get a `Permission
denied` error, 

![](../images/part70/20.png)

because a depositor is not allowed to update other user's data.

But of course, they can update their own data.

So let's try login with Alice's account.

![](../images/part70/21.png)

And invoke the Update User method again.

![](../images/part70/22.png)

This time, the request is successful, exactly as we expected.

So that's how we implement role-based access control in Go.

I hope you have learned something interesting and useful. Thanks a lot 
for watching, happy learning, and see you in the next lecture!