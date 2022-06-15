# How to manage user session with refresh token - Golang

[Original video](https://www.youtube.com/watch?v=rT20ylRLm5U)

Hello guys!

Welcome back to the Backend Master Class. It might be a surprise
to you because, in the [previous video](part36-eng.md), I said
it was the end of the course.

My plan wsa to start a new course with more advanced backend
topics. However, after giving it some thought, I think it's more
suitable to continue posting them in this course, since the 
main purpose of the course is to help you become a master in 
backend development.

It wouldn't be a master class without advanced topics, right?

Alright, let's start with the first topic: how to manage user
sessions?

## How to manage user sessions

If you still remember, from lecture 19 to lecture 22, we've 
learned how to use PASETO or JWT as a token-based authentication.

However, I must emphasize that you should not use them for a 
long session.

![](../images/part37/1.png)

Because of the stateless design, those access tokens are not stored
in the database, and so there's no way to revoke them in case they
got leaked.

Therefore, their lifetime should be very short, such as 10 or 15 
minutes.

But if we only use access tokens, then when they're expired, users
will need to log in again with their username and password.

![](../images/part37/2.png)

It is definitely not a good user experience to ask them to log in
every 10 or 15 minutes, right?

So this is where another type of token called refresh token comes
into play. The main idea is, we will use it to maintain a stateful
session on the server, and the client can use the refresh token 
with a long valid duration, to request a new access token when 
it's expired.

![](../images/part37/3.png)

The refresh token can be as simple as a random string, or we can 
also use PASETO if we want. But it will be stored in a sessions
table in the database, with 1 additional boolean field to allow
blocking the token in case it is compromised.

With the ability to revoke the refresh token, its lifetime can be
much longer than the access token, such as several days or even 
weeks.

Alright, it's time to go into coding.

First, let's start the simple bank API server. Then try sending
the login request in Postman.

![](../images/part37/4.png)

As you can see, right now, the server only returns a PASETO 
access token. And if we look at the `app.env` file in the 
code,

```
DB_DRIVER=postgres
DB_SOURCE=postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable
SERVER_ADDRESS=0.0.0.0:8080
TOKEN_SYMMETRIC_KEY=12345678901234567890123456789012
ACCESS_TOKEN_DURATION=15m
```

We can see that the duration of the access token is very short, 
only 15 minutes. What we want to have is a refresh token in the 
login API response, with a longer duration. So, here I'm gonna
add a new variable called `REFRESH_TOKEN_DURATION` and let's 
set it to 24 hours.

```
REFRESH_TOKEN_DURATION=24h
```

Then in the `config.go` file, we have to update the `Config`
struct to load this new variable.

The new field is gonna be `RefreshTokenDuration` of type 
`time.Duration` and we must add a `mapstructure` tag for it with
the same name as the environment variable.

OK, done!

Next step, we will need to create a new sessions table in the
database.

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

So let's use this command in the README file to create a new DB
migration.

```shell
migrate create -ext sql -dir db/migration -seq <migration_name>
```

I'm gonna copy, and paste it to the terminal. Then change
the migration name to `add_sessions`.

```shell
migrate create -ext sql -dir db/migration -seq add_sessions
```

![](../images/part37/5.png)

Alright, 2 migration files, `up` and `down` have been generated.

The `up` script will be similar to that  of the `users` table,
so let's just copy it from here, and paste it to the new 
migration file.

I'm gonna change the table name to `sessions`. The first column
to this table will be the ID of the session, which can be the
same as the ID of the refresh token that we define in the 
token payload. So its type should be `uuid`, and it will be
the primary key of this table. We will also store the username
of the user in this table, but of course, it's not the primary
key anymore. The third column will store the refresh token of
this session if we want to keep track of the client type, and
where the user is connecting to the server, we can store the 
user agent, and the client IP address in this table as well.
Next, we need 1 important boolean column to block the session
in case the refresh token is compromised. Finally, let's add
an `expires_at` column to tell us the time when the refresh
token will be expired. And there's also a foreign key constraint
on the `username` column that references the same column of
the `users` table. That's it for the migration `up` script.

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

For the migration `down`, similar as in the add users migration
down script, all we have to do is to drop the `sessions` table.

```postgresql
DROP TABLE IF EXISTS "sessions";
```

Alright, now let's open the terminal and run `make migrateup`
to apply the new migration.

```shell
make migrateup
```

OK, it runs successfully! Let's check the new db schema in 
Table Plus. Here we can see the new `sessions` table.

![](../images/part37/6.png)

It has the `username` column that links to the `users` table.
The `refresh_token`, and `is_blocked` columns, exactly as we 
defined in the migration script. Awesome!

Next step, we will add some new SQL queries to create and
retrieve a session.

I'm gonna create a new file `session.sql` inside the `query`
folder. Then let's copy the content of the `user.sql` file
here. The first query is to create a new session so, insert
into sessions, the columns are id, username, and you know 
what, it's faster to just copy the rest of the columns from
the migration `up` file. Then reformat all columns to the
valid syntax.

In Visual Studio Code, we can easily create multiple cursors
by pressing the option (or Alt) key while clicking at 
different positions in the code editor. This allows us to edit
multiple lines at the same time, which is much faster than 
doing it one by one.

OK, so we've just added all necessary columns: the 
`refresh_token`, `user_agent`, `client_ip`, `is_blocked` and
`expires_at`. In total, we have 7 columns in this `INSERT` 
statement. So we must add 3 more parameters to the `VALUES`
list. And the `INSERT` query is done!

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

Now let's move to the next query: `GetSession`. For this one, 
we will look for a specific session by its ID. So I'm gonna
change this clause

```postgresql
-- name: GetUser :one
SELECT * FROM users
WHERE username = $1 LIMIT 1;
```

to where `id` equals the first parameter.

```postgresql
--name: GetSession :one
SELECT * FROM sessions
WHERE id = $1 LIMIT 1;
```

That's it! Now let's run 

```shell
make sqlc
```

in the terminal to generate Golang codes for the 2 new 
queries we've just written.

It's successful. So if we open the `session.sql.go` file
inside the `sqlc` folder, we will see that the code has
been generated. `sqlc` is using the `google/uuid` for
the session ID column. And there's one function to create
a new session, and another function to get a session by
its ID. Cool!

Now as 2 more functions have been added to the `Store` 
interface, you will notice that some of our API unit tests
are showing some errors. That's because those 2 new functions
are not implemented by the mock `Store` yet.

To fix this, we must run

```shell
make mock
```

in the terminal to regenerate the mock `Store`.

Only after that, then the errors will be gone.

We can run 

```shell
make test
```

to make sure that everything is working as expected.

![](../images/part37/7.png)

And it really is! All tests passed. Excellent!

Alright, now comes the important part, we will modify the login
API to create and return the refresh token together with the access
token.

So let's open the `api/user.go` file. In the `loginUser` function
after creating the access token here,

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

we will call `server.tokenMaker.CreateToken` one more time to 
create the refresh token. We still pass in the same `user.Username`,
but the duration of the refresh token should be different. Its
value is taken from the `config.RefreshTokenDuration` variable.
This function will return a `refreshToken` string and an error.
If error is not `nil`, we just return internal server error, the 
same as for the access token above.

```go
	refreshToken, err := server.tokenMaker.CreateToken(
		user.Username,
		server.config.RefreshTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
	}
```

If no error occurs, we will call `server.store.CreateSession` to 
insert a new session into the database. We pass in the context, 
and a `db.CreateSessionParam` object. I'm gonna copy all the 
required fields of this struct

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

and paste them inside this params object.

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

In this object, we must have the ID of the session, and as I said
before, we will use the refresh token's ID for this field. But
the problem is, the `createToken` function doesn't return the 
token payload. It just returns an encrypted token string, so we
don't know what the token's ID is. Therefore, I'm gonna add the
token payload to the list of returning values of this function.
Now as we've changed the `TokenMaker` interface, we must change
the `PasetoMaker` and `JwtMaker` as well, because they're 
implementing the `TokenMaker` interface. First, for `PasetoMaker`,
I'm gonna add `Payload` to the output of this function.

```go
func (maker *PasetoMaker) CreateToken(username string, duration time.Duration) (string, *Payload, error) {
	...
}
```

Then change this `return` statement to include `payload` as well. 
Now, for this `Encrypt` function call, we must store its output
value in 2 variables: token and error. Then we return all 3 values:
token, payload, and error at the end of the function.

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

OK, the PasetoMaker's implementation is now fixed.

Now I'm gonna do the same for the `JwtMaker`. Normally in a real
project, we just choose either 1 of them, but not both. And I highly
recommend using Paseto instead of JWT. If you don't know why,
please go back to watch lecture 19 of the course.

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

OK, now the `JwtMaker` is fixed.

But some of the unit tests are still showing errors, so let's fix
them as well. In the `TestPasetoMaker` function, we must add a 
payload variable here.

```go
	token, payload, err := maker.CreateToken(util.RandomOwner(), -time.Minute)
```

and remove this colon because `payload` is not a new variable
here anymore.

```go
	payload, err = maker.VerifyToken(token)
```

Then I will also require it to be not empty.

```go
    require.NotEmpty(t, payload)
```

You can do more checks here to make the test more robust if you
want. Let's do the same for the `TestExpiredPasetoMaker`.

Then I'm gonna fix the rest of the JWT maker unit tests in the same
manner.

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

OK, it's done!

Let's rerun the whole token package tests. They're all passed!
Excellent!

I'm gonna close all of these files.

But wait, there's still some errors in the `api/middleware_test.go`
file. Let's check it out!

OK, so here in the `addAuthorization` function, we also have to add
the `payload` object and require it to be not empty.

```go
	token, payload, err := tokenMaker.CreateToken(username, duration)
    require.NoError(t, err)
    require.NotEmpty(t, payload)
```
OK cool. The errors are gone.

Now let's go back to the `loginUser` handler function!

As we've changed the `TokenMaker` interface, there should be a 
lot of errors here. But somehow they're not showing up, maybe
that's because of this incompleted piece of codes.

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

So first, I'm gonna comment out all these fields of the 
`CreateSessionParams` object.

OK cool, now the errors have shown up. Let's fix them!

Here, when creating the access token, we need to add 1 more 
variable to store the access token payload.

```go
accessToken, accessPayload, err := server.tokenMaker.CreateToken(
    user.Username,
    server.config.AccessTokenDuration,
)
```

Then, let's move up all the way to the top.

I'm gonna add a few more fields to the `loginUserResponse`
struct. First, a field to know when the access token will 
expire. This will be useful for the client to set up a 
schedule to renew the access token later. I'm gonna move
this field up, next to the `AccessToken`. Then we will add 2
more similar fields for the refresh token and its expiration
time.

```go
type loginUserResponse struct {
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  userResponse `json:"user"`
}
```

Alright, now back to the `loginUser` handler.

Let's add the refresh payload variable to this `CreateToken` call.

```go
	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		server.config.RefreshTokenDuration,
	)
```

OK, now I'm gonna uncomment these parameters.

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

We're gonna set the session `ID` to be `refreshPayload.ID`, the
`Username` is gonna be `user.Username`, `RefreshToken` is simply
the value returned by the `CreateToken` call above. For the
`UserAgent` and `ClientIP`, let's temporarily set them as empty
string for now. We will come back a bit later to fill in the
correct values. Next, `IsBlocked` field should be `false`, of
course. And finally `ExpiresAt` will be set to 
`refreshPayload.ExpiredAt`. OK, this `CreateSession` statement 
will return a `session` object and an error.

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

If error is not `nil`, then we just return internal server error
as in other cases. Otherwise, we're ready to send the response 
to the client.

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

But I want to add 1 more field to the response struct which is
the ID of the session (or it's also the ID of the refresh token).

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

Alright, now in this `loginUserResponse` object, I'm gonna set
`SessionID` to be `session.ID`, `AccessTokenExpiresAt` to be
`accessPayload.ExpiredAt`, `RefreshToken` to be `refreshToken`,
and finally, `refreshTokenExpiresAt` to be 
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

That will be it!

The login user API with refresh token support is now completed.

Let's open `user_test.go` file, and try to run the 
`TestLoginUserAPI` to see if it's still working well or not.

![](../images/part37/8.png)

OK, so 1 unit test failed. And the reason is: "there are no 
expected calls of the method `CreateSession`". This is reasonable,
since we haven't added a stub call requirement for this new method
yet.

Fixing this is very simple. In the `buildStubs` function of the OK case,
I'm gonna add 1 more `store.EXPECT()` statement but this time, we're
gonna expect the `CreateSession` function to be called with any context,
and any parameters. And it should be called exactly 1 time. Just like 
that!

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

And we're done! Of course, you can make the test stronger by replacing
`gomock.Any()` with a more specific object. Please watch lecture 18 if
you don't know how to do it.

Alright, let's rerun the unit tests.

![](../images/part37/9.png)

Now all of the tests have passed. Awesome!

Let's go ahead and run

```shell
make server
```

in the terminal to start the server.

Then, I'm gonna send the login API request again using Postman.

![](../images/part37/10.png)

This time, we have several more fields in the response.

First the `session_id`, the `access_token` and its expiration time,
then the refresh token together with its expiration time. You can see
that it lasts much longer, 1 day, compared to just 15 minutes of the 
access token.

Now let's check the database. In the `sessions` table, we see 1 
record. We can open this right hand side section to see more details.

![](../images/part37/11.png)

As you can see, it has the same ID as the session we received in Postman.
And with the `is_blocked` field, we can easily block this session in case
its refresh token get hacked. Here we also see its cretion and expiration
time. But the `user_agent` and client IP address are still empty. Now 
let's go back to the code to fill in their correct values.

It's actually pretty simple because we're using Gin framework. For the
`UserAgent`, we can just call `ctx.Request.UserAgent()`. This information
is already available inside the Gin context object. And similarly, for the
client IP, we can just call `ctx.ClientIP()`. And that's it! Super easy,
isn't it?

You can also get other metadata from the context, such as the content 
type, if you want. OK, now let's restart the server and test it out! 

![](../images/part37/12.png)

I'm gonna resend this login user request. It's successful! So let's
checkout the `sessions` table.

![](../images/part37/13.png)

This time, we have a new session record, and look at the `user_agent`
and `client_ip` fields! They have been saved with the correct value.
Pretty cool, right?

Alright, so now as the login user API with refresh token and session
record has been working very well, the only thing that is still
missing is an API to renew the access token when it expires.

As this API is very similar to the login user API, I'm gonna copy
the content of this `loginUser` handler, and create a new file
called `token.go` inside the `api` package. Then let's paste in the
content that we've just copied. Now we must change all the request,
response and handler function name from `loginUser` to 
`renewAccessToken`.

As you can see, in Visual Studio Code, I use the `Command + Shift + L`
key combination to edit this value at multiple places at the same
time.

Alright, now for the `renewAccessTokenRequest`, the only field we
would need is the `refreshToken`.

```go
type renewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
```

So I'm gonna change this `Username` field to `refreshToken` and 
remove the `Password` field.

Similarly, for the `renewAccessTokenResponse`, we will just return
2 fields: `AccessToken` and its expiration time. So let's get rid
of all other redundant fields.

OK, then in the `renewAccessToken` function, first we bind the 
input json into the request object. Then, here, we need to verify
if the refresh token is valid or not. So I'm gonna call 
`server.tokenMaker.VerifyToken`, and pass in `req.RefreshToken`.
The output of this function call is a `refreshPayload` and an
error. 

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

If error is not `nil`, then it means the refresh token is invalid
or expired. In that case, we simply return an unauthorized status
code to the client. Otherwise, we will find the session in the 
database by calling `server.store.GetSession` and pass in the
refresh token's ID as session ID.

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

It will return a session object, or an error. If error is not `nil`,
then we just handle it just like when user is not found in the login
API. Return 404 `Not found` if session doesn't exist, or internal 
server error in other cases.

We don't have to compare password in this `renewAccessToken` API, but
instead, we must check if the session is blocked or not.

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

If it is blocked, then we will create a new error object saying 
"blocked session" and return it to the client together with a 
status code `401 Unauthorized`. We should also check if the
`session.Username` is the same as the one stored in the refresh
token.

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

If they're not the same, then we also return an error to the client
saying "incorrect session user". Let's also check if the 
`session.RefreshToken` is the same as `req.RefershToken` or not.

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

Normally they should be the same. But, just in case they're not, we 
return an error saying "mismatched session token".

Finally, although the expiration time of the token has been checked
in the `VerifyToken` function, we can check it one more time here,
because in some rare cases, we might want to force it to expire 
before its actual expiration time.

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

So if current time is after `session.ExpiresAt` we return an error
saying "expired session". And I think that should be enough for
the checking part. Now if everything goes well, we can issue a new
access token.

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

This time, we can get the `username` from the `refreshPayload` 
object. And that's it! The error handling part should stay the same.
And we can remove the rest of the code that creates a new refresh 
token and session.

Finally, we will delete all redundant fields in the response object,
just keep 2 fields: `accessToken`, and `accessTokenExpiresAt`.

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

And we're done with the `renewAccessToken` handler function. Next,
we must register this handler as a new route in the `server.go`
file.

I'm gonna add here a new `router.POST`, where the path is 
`/token/renew_access`, and the handler is `server.renewAccessToken`.

```go
func (server *Server) setupRouter() {
	router := gin.Default()

	router.POST("/users", server.createUser)
	router.POST("/users/login", server.loginUser)
	router.POST("/token/renew_access", server.renewAccessToken)
	...
}
```

And we're done!

Let's open the terminal and restart the server!

```shell
make server
```

Then open Postman, I'm gonna create a new request. Change the method
to POST, then the request URL should be 
`localhost:8080/tokens/renew_access`. Then in the `Body` tab, let's 
select Raw, and JSON format. The only field we would need is 
`refresh_token`. I'm gonna copy its value from the login API's
response. Alright, now let's send the request!

![](../images/part37/14.png)

Yee! It's successful!

We've got a new access token that will last for the next 15 minutes!
Awesome!

OK, now I'm gonna save this new request into my Simple Bank Postman
collection.

![](../images/part37/15.png)

And before we finish, I will try 1 edge case, where we use the access
token instead of the refresh token in the API.

Let's see if we will be able to get a new access token or not.

OK, I'm gonna send the request!

![](../images/part37/16.png)

Oh, the access token has already expired. So let's go back and login
again. Then copy the new access token, paste it to the renew access 
API, and send the request one more time!

![](../images/part37/17.png)

This time, we've got error: "no rows in result set", that's because
there's no session record in the database with the ID of the access
token we've used. That's exactly what we wanted: only the refresh 
token should be able to issue a new access token.

OK, how about trying an invalid refresh token, such as "abc"? We've
got an error: "token is invalid".

![](../images/part37/18.png)

And this is a very nice bonus when we use PASETO as the refresh token,
because for cases like this, the server doesn't have to query the 
database at all in order to know that the refresh token is invalid.

That definitely will reduce a lot of load to the database!

And this brings us to the end of this video. I hope it was interesting
and you've learned something useful.

Thanks a lot for watching, and see you in the next lecture!