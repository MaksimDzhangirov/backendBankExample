# Implement email verification API in Go

[Original video](https://www.youtube.com/watch?v=50ZN-4UNwnY)

Hello everyone, welcome to the backend master class! In the last lecture, we
learned how to create and send a verification email to a newly registered
user. In the email, we ask them to click on this link 

```go
verifyUrl := fmt.Sprintf("http://simple-bank.org/verify_email?id=%d&secret_code=%s",
        verifyEmail.ID, verifyEmail.SecretCode)
```

to verify their email address. But we haven't implemented the API to handle
it yet. So that's exactly what I'm gonna show you in this lecture.

## Writing email verification API

The idea is, when receiving an API request to verify email, we will look
for the `VerifyEmail` record with the provided `id` and `secret_code`
in our database. If it exists and is not expired or already used, then we
can mark it as used, and update the `is_email_verified` field of the user
to true. Pretty simple, isn't it?

Alright, let's open the send verify email task. In the previous lecture,
I used this fake URL

```go
verifyUrl := fmt.Sprintf("http://simple-bank.org/verify_email?id=%d&secret_code=%s",
        verifyEmail.ID, verifyEmail.SecretCode)
```

to demonstrate that. We should send users a link to a front-end page, which
will parse the input query parameters, then use them to call the verify
email API on the backend side. And of course, on production, we should use
a secured HTTPS protocol here `https://simple-bank.org/`. But for the
purpose of this demo, I'm not gonna implement a real front-end page, so I
will replace this with a link that points directly to our local backend 
API: `http://localhost:8080/v1/verify_email`.

```go
verifyUrl := fmt.Sprintf("http://localhost:8080/v1/verify_email?id=%d&secret_code=%s",
    verifyEmail.ID, verifyEmail.SecretCode)
```

The `v1` here indicates the API version, which we're also using for all 
other routes. And to make it clearer, I'm gonna change this `id` parameter
to `email_id`.

```go
verifyUrl := fmt.Sprintf("http://localhost:8080/v1/verify_email?email_id=%d&secret_code=%s",
    verifyEmail.ID, verifyEmail.SecretCode)
```

since it's referring to the id of the `verify_email` record in our DB. OK,
so now, let's start building the API. As we're using gRPC gateway, I'm
gonna define it in `protobuf`. It's pretty similar to what we did for other
APIs, so let's duplicate this `rpc_create_user.proto` file and rename
it to `rpc_verify_email.proto`.

```shell
cd proto
cp rpc_create_user.proto rpc_verify_email.proto
```

In this file, let's replace all "CreateUser" to "VerifyEmail". We will have
2 fields for the request: `email_id` and `secret_code`. The `email_id`
field should be an `int64`, and the `secret_code` field will be a 
`string`.

```protobuf
message VerifyEmailRequest {
  int64 email_id = 1;
  string secret_code = 2;
}
```

For the response, we only need 1 boolean field: `is_verified`.
This will tell us if the email has been successfully verified or not.

```protobuf
message VerifyEmailResponse {
  bool is_verified = 1;
}
```

OK, we can now get rid of the redundant import statement at the top.

```protobuf
import "user.proto";
```

And in the next step, we'll have to add a new RPC in the 
`service_simple_bank.proto` file. I'm gonna import the 
`rpc_verify_email.proto` file, and at the bottom, let's duplicate the
`LoginUser` RPC, then change its name to `VerifyEmail`, its input to 
`VerifyEmailRequest` and its output to `VerifyEmailResponse`. Now, for 
the HTTP option `option (google.api.http)`, we'll make it a `GET` 
request, because when the user clicks on the link we sent, the browser 
will make a GET request to that link. And so, this path should be updated 
to `/v1/verify_email`. There's no "body" in a GET request, so let's remove
it. This gRPC gateway option 
`option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation)`
is for generating the documentation, so let's update the description to:
"Use this API to verify user's email address" and the summary to: "Verify
email". And that should be it.

```protobuf
...
import "rpc_login_user.proto";
import "rpc_verify_email.proto";
import "protoc-gen-openapiv2/options/annotations.proto";
...

service SimpleBank {
  ...
  rpc LoginUser(LoginUserRequest) returns (LoginUserResponse) {
    option (google.api.http) = {
      post: "/v1/login_user"
      body: "*"
    };
    option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
      description: "Use this API to login user and get access token & refresh token";
      summary: "Login user";
    };
  }
  rpc VerifyEmail(VerifyEmailRequest) returns (VerifyEmailResponse) {
    option (google.api.http) = {
      get: "/v1/verify_email"
    };
    option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
      description: "Use this API to verify user's email address";
      summary: "Verify email";
    };
  }
}
```

We can now run

```shell
make proto
rm -f pb/*.go
rm -f doc/swagger/*.swagger.json
protoc --proto_path=proto --go_out=pb --go_opt=paths=source_relative \
        --go-grpc_out=pb --go-grpc_opt=paths=source_relative \
        --grpc-gateway_out=pb --grpc-gateway_opt=paths=source_relative \
        --openapiv2_out=doc/swagger --openapiv2_opt=allow_merge=true,merge_file_name=simple_bank \
        proto/*.proto
statik -src=./doc/swagger -dest=./doc
```

in the terminal to generate the codes for this new API.

Voilà, in the `pb` folder, we can see a new file `rpc_verify_email.pb.go`.
However, we can notice some red files in the `gapi` package. It seems
like there's something wrong with the import `pb` package. Actually, I
notice that this sometimes happens, because we remove all existing files
in the `pb` package `rm -f pb/*.go` before generating new codes, so VS Code
has some problems reloading the package after that. If this happens, we can
simply quit VS Code, then reopen it, and everything should be OK.

Alright, now we have the generated stubs for the new API. We can implement
its handler in the `gapi` package. I'm gonna duplicate the 
`rpc_create_user.go` file, and rename it to `rpc_verify_email.go`.

```shell
cd gapi
cp rpc_create_user.go rpc_verify_email.go
```

In this file, let's replace all occurrences of "CreateUser" to 
"VerifyEmail". Now we have the `VerifyEmail` handler, that takes a
`VerifyEmailRequest` as input and returns a `VerifyEmailResponse` as
output.

```go
func (server *Server) VerifyEmail(ctx context.Context, req *pb.VerifyEmailRequest) (*pb.VerifyEmailResponse, error) {
	...
}
```

I'm gonna delete all these codes that handle user creation, since we're 
gonna have to implement a new logic to verify email. OK, next, let's update
the validate request function. The `VerifyEmailRequest` has 2 fields: email 
ID and secret code. So we will need to open the `validator` package to add
new functions to verify them.

First, a function to validate email ID. It will take an input value of 
type `int64` and return an error. If the provided value is less than or
equal to 0, we'll return an error saying that it must be a positive 
integer. Otherwise, just return `nil`.

```go
func ValidateEmailId(value int64) error {
	if value <= 0 {
		return fmt.Errorf("must be a positive integer")
	}
	return nil
}
```

Then, the function to validate the secret code will take a string value
as input and also return an error as output. For this, I'm just gonna 
check its length, so we simply call `ValidateString()` function with
the input value, and set the minimum length to 32, and the maximum length
to 128. You can do more checks here if you want, but it's not the main
focus of this lecture.

```go
func ValidateSecretCode(value string) error {
	return ValidateString(value, 32, 128)
}
```

Now I'm gonna head back to the `rpc_verify_email` handler. Here, in
`validateVerifyEmailRequest()`, let's call `val.ValidateEmailId()` with 
the input request's email ID and if this function returns an error, we'll
add `email_id` to the list of field violations. Similarly, we call 
`val.ValidateSecretCode()` and pass in the secret code value provided by
the input request. If the returned error is not `nil`, we add `secret_code`
to the field violation list. Finally, we can just return the `violations`
list to the caller of this function.

```go
func validateVerifyEmailRequest(req *pb.VerifyEmailRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	if err := val.ValidateEmailId(req.GetEmailId()); err != nil {
		violations = append(violations, fieldViolation("email_id", err))
	}

	if err := val.ValidatePassword(req.GetSecretCode()); err != nil {
		violations = append(violations, fieldViolation("password", err))
	}

	return violations
}
```

Alright, now it's time to use the input `email_id` and `secret_code` to
query and update the database. Since we want to update both the 
`verify_emails` and `users` tables, we will need to do everything inside
1 single DB transaction. Therefore, in the `sqlc` folder, I'm gonna
duplicate the `tx_create_user.go` file and rename it to 
`tx_verify_email.go`.

```go
cd db/sqlc
cp tx_create_user.go tx_verify_email.go
```

Then, in this file, let's rename all occurrences of "CreateUserTx"
to "VerifyEmailTx". The transaction params struct will contain 
only 2 fields: an email ID of type `int64`, and a secret code of
type `string`. Then, for the transaction result struct, we also have
2 fields to store the updated user and updated email records.

```go
type VerifyEmailTxParams struct {
	EmailId    int64
	SecretCode string
}

type VerifyEmailTxResult struct {
	User        User
	VerifyEmail VerifyEmail
}
```

OK, in the body of the transaction, I'm gonna remove everything,
just keep the declaration of the error variable and return it at 
the end.

```go
func (store *SQLStore) VerifyEmailTx(ctx context.Context, arg VerifyEmailTxParams) (VerifyEmailTxResult, error) {
	var result VerifyEmailTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error
		
		return err
	})

	return result, err
}
```

Next, we'll need to write a new query to find the verification email by
id and secret code and update its `is_used` to `true`. Then we also need
to update the `is_email_verified` field of the corresponding user to 
`true`.

First, let's open the `verify_email.sql` file inside the `query` folder.
I'm gonna add a new query named `UpdateVerifyEmail`, which will return
one row of the updated email record. The SQL query will be like this:

```postgresql
-- name: UpdateVerifyEmail :one
UPDATE verify_emails
SET
    is_used = TRUE
WHERE
    id = @id
    AND secret_code = @secret_code
    AND is_used = FALSE
    AND expired_at > now()
RETURNING *;
```

This (`@id`) is another way of naming input parameters in `sqlc`. You can 
also use `$1` parameter here, it would also work. But for me, I prefer a 
named parameter since it looks clearer. We also need to check the secret 
code of this email matches the one provided via input argument and that
the code is not already used before, and most importantly, it must not
be expired, or expired time must be greater than the current timestamp.
Finally, we use a `RETURNING *` clause at the end of the query to return
the updated email record.

Now, let's move to the `user.sql` file. We already have a `UpdateUser` here,
so all we have to do is, adding a new clause here to set the 
`is_email_verified` field to a new value provided by the input argument.
To remind you, we're using a nullable argument feature of `sqlc`, so that
each field can be updated independently without affecting the other. You can
go back to watch lecture 49 to understand how it works.

```postgresql
-- name: UpdateUser :one
UPDATE users
SET
    hashed_password = COALESCE(sqlc.narg(hashed_password), hashed_password),
    password_changed_at = COALESCE(sqlc.narg(password_changed_at), password_changed_at),
    full_name = COALESCE(sqlc.narg(full_name), full_name),
    email = COALESCE(sqlc.narg(email), email),
    is_email_verified = COALESCE(sqlc.narg(is_email_verified), is_email_verified)
WHERE
    username = sqlc.arg(username)
RETURNING *;
```

Alright, both update queries are completed.

We're now ready to run

```shell
make sqlc
sqlc generate
```

in the terminal to generate new codes for them. As you can see, in the 
`verify_email.sql.go` file,

```go
func (q *Queries) UpdateVerifyEmail(ctx context.Context, arg UpdateVerifyEmailParams) (VerifyEmail, error) {
	row := q.db.QueryRowContext(ctx, updateVerifyEmail, arg.ID, arg.SecretCode)
	var i VerifyEmail
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Email,
		&i.SecretCode,
		&i.IsUsed,
		&i.CreatedAt,
		&i.ExpiredAt,
	)
	return i, err
}
```

we have a new function to update verify-email here.

And in the `user.sql.go` file, the update user function also has a new input
argument: `is_email_verified`. Awesome!

```go
type UpdateUserParams struct {
	HashedPassword    sql.NullString `json:"hashed_password"`
	PasswordChangedAt sql.NullTime   `json:"password_changed_at"`
	FullName          sql.NullString `json:"full_name"`
	Email             sql.NullString `json:"email"`
	IsEmailVerified   sql.NullBool   `json:"is_email_verified"`
	Username          string         `json:"username"`
}
```

Now let's go back to the verify-email transaction. In the body of the 
`execTx` function, I'm gonna call `q.UpdateVerifyEmail()` and pass in the
context variable, together with a `UpdateVerifyEmailParams` object. The `ID`
field should be set to the input argument's `EmailId` and the same applies
to the secret code. This function will return the updated email record 
and an error. So I'm gonna save the updated record to `result.VerifyEmail`.
Then check whether the error is `nil` or not. If it is not `nil`, we simply
return it. Otherwise, the provided secret code must be correct, so it's 
time to call `q.UpdateUser()` to update the `IsEmailVerified` field to
`true`. This is a very general function that allows us to update multiple 
fields of the user record, but in our case, we only need to update 1 field.
So we just need to provide the required username, and a valid value for
the `IsEmailVerified` column. We can easily get the username from the
`result.VerifyEmail` record, as for the `IsEmailVerified` field, we have to
assign it to a `sql.NullBool` object, and both `Bool` and `Valid` fields
must be set to `true`. OK, now we can store the output of this `UpdateUser`
function to `result.User`, and if it returns a not `nil` error, that error
will be propagated to the outer function.

```go
func (store *SQLStore) VerifyEmailTx(ctx context.Context, arg VerifyEmailTxParams) (VerifyEmailTxResult, error) {
	var result VerifyEmailTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.VerifyEmail, err = q.UpdateVerifyEmail(ctx, UpdateVerifyEmailParams{
			ID:         arg.EmailId,
			SecretCode: arg.SecretCode,
		})
		if err != nil {
			return err
		}

		result.User, err = q.UpdateUser(ctx, UpdateUserParams{
			Username: result.VerifyEmail.Username,
			IsEmailVerified: sql.NullBool{
				Bool:  true,
				Valid: true,
			},
		})

		return err
	})
	return result, err
}
```

That's basically it! The verify-email transaction is now completed.

We can go back to the `rpc_verify_email.go` file to finish the 
implementation of the handler function. But before that, I forgot 1 
important thing, we have to add the `VerifyEmailTx()` function signature
to the `Store` interface, because in the API layer, we only have access 
to this general interface, we don't know anything about its detailed 
implementation.

```go
type Store interface {
	Querier
	TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error)
	CreateUserTx(ctx context.Context, arg CreateUserTxParams) (CreateUserTxResult, error)
	VerifyEmailTx(ctx context.Context, arg VerifyEmailTxParams) (VerifyEmailTxResult, error)
}
```

Alright, now we can call `server.store.VerifyEmailTx()`, pass in the 
context, and a `db.VerifyEmailTxParams` object. Its `EmailId` field should
be set to `req.GetEmailId()`, and the `SecretCode` field should be set to 
`req.GetSecretCode()`. This function will return a transaction result and
an error, so I save them to the corresponding variables. Then check if the
error is `nil` or not. If it is not `nil`, we simply return a `nil` 
response together with a gRPC status error code `Internal` and this
message: "failed to verify email". You can also further check the type of
the error to return a different status code in case the email is not 
found, for example. Here I choose to make it simple by always returning
the same error code. Finally, we can set the `IsVerified` field of the
response to `txResult.User.IsEmailVerified`. And we're done with the 
implementation of the verify-email API.

```go
func (server *Server) VerifyEmail(ctx context.Context, req *pb.VerifyEmailRequest) (*pb.VerifyEmailResponse, error) {
	violations := validateVerifyEmailRequest(req)
	if violations != nil {
		return nil, invalidArgumentError(violations)
	}

	txResult, err := server.store.VerifyEmailTx(ctx, db.VerifyEmailTxParams{
		EmailId:    req.GetEmailId(),
		SecretCode: req.GetSecretCode(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to verify email")
	}

	rsp := &pb.VerifyEmailResponse{
		IsVerified: txResult.User.IsEmailVerified,
	}
	return rsp, nil
}
```

You might notice that there are some errors in the `api` package. The
reason for that is, the mock store no longer satisfies the `Store` 
interface, since we've just added a new method to that interface. So to
fix this, we'll have to run

```shell
make mock
mockgen -package mockdb -destination db/mock/store.go github.com/techschool/simplebank/db/sqlc Store
```

in the terminal to regenerate the `MockStore`. And everything should be 
back to normal.

Now, we're ready to do some tests.

But first, I want to clean up the DB to make it easier to track the new
records. One easy way to do that is, simply run 

```shell
make migratedown
migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose down
2023/03/18 11:23:22 Are you sure you want to apply all down migrations? [y/N]
y
2023/03/18 11:23:24 Applying all down migrations
2023/03/18 11:23:24 Start buffering 4/d add_verify_emails
2023/03/18 11:23:24 Start buffering 3/d add_sessions
2023/03/18 11:23:24 Start buffering 2/d add_users
2023/03/18 11:23:24 Start buffering 1/d init_schema
2023/03/18 11:23:24 Read and execute 4/d add_verify_emails
2023/03/18 11:23:24 Finished 4/d add_verify_emails (read 10.285017ms, ran 37.251805ms)
2023/03/18 11:23:24 Read and execute 3/d add_sessions
2023/03/18 11:23:24 Finished 3/d add_sessions (read 57.570959ms, ran 17.475981ms)
2023/03/18 11:23:24 Read and execute 2/d add_users
2023/03/18 11:23:24 Finished 2/d add_users (read 85.049288ms, ran 19.544725ms)
2023/03/18 11:23:24 Read and execute 1/d init_schema
2023/03/18 11:23:24 Finished 1/d init_schema (read 113.168821ms, ran 35.883769ms)
2023/03/18 11:23:24 Finished after 2.0460477s
2023/03/18 11:23:24 Closing source and database
```

to apply all `down` migrations. This will remove all tables from the 
database.

Then we run 

```shell
make migrateup
migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up
2023/03/18 11:23:30 Start buffering 1/u init_schema
2023/03/18 11:23:30 Start buffering 2/u add_users
2023/03/18 11:23:30 Start buffering 3/u add_sessions
2023/03/18 11:23:30 Start buffering 4/u add_verify_emails
2023/03/18 11:23:30 Read and execute 1/u init_schema
2023/03/18 11:23:30 Finished 1/u init_schema (read 11.11641ms, ran 86.271254ms)
2023/03/18 11:23:30 Read and execute 2/u add_users
2023/03/18 11:23:30 Finished 2/u add_users (read 112.329252ms, ran 30.488839ms)
2023/03/18 11:23:30 Read and execute 3/u add_sessions
2023/03/18 11:23:30 Finished 3/u add_sessions (read 150.083295ms, ran 24.135692ms)
2023/03/18 11:23:30 Read and execute 4/u add_verify_emails
2023/03/18 11:23:30 Finished 4/u add_verify_emails (read 184.733063ms, ran 27.778521ms)
2023/03/18 11:23:30 Finished after 221.804983ms
2023/03/18 11:23:30 Closing source and database
```

to recreate all of them.

Now, if we refresh Table Plus, all existing data are gone.

![](../images/part62/1.png)

## Testing email verification API

I'm gonna run 

```shell
make server
go run main.go
11:23AM INF db migrated successfully
11:23AM INF start gRPC server at [::]:9090
11:23AM INF start task processor
11:23AM INF Starting processing
11:23AM INF start HTTP gateway server at [::]:8080
```

in the terminal to start the server and open Postman to send this 
`create_user` request. It will create a new `techschool` user with this 
`techschool.guru@gmail.com` email address.

![](../images/part62/2.png)

Voilà, the user has been successfully created. In the server logs,

```shell
11:24AM INF enqueued task max_retry=10 payload="{\"username\":\"techschool\"}" queue=critical type=task:send_verify_email
11:24AM INF received an HTTP request duration=98.256762 method=POST path=/v1/create_user protocol=http status_code=200 status_text=OK
```

we can see that a task to send a verification email has been enqueued. And
after a few seconds, this task will be processed.

```shell
11:24AM INF processed task email=techschool.guru@gmail.com payload="{\"username\":\"techschool\"}" type=task:send_verify_email
```

Now, if we look at the Tech School Gmail inbox, we will see a new "Welcome 
to Simple Bank" email.

![](../images/part62/3.png)

Inside it, there's a link that points to our local backend's API to 
verify email. Let's click on it to see what happens.

![](../images/part62/4.png)

As you can see, it sends a GET request to the `verify_email` API. And gets
back a JSON response with `is_verified` equals to `true`. This means that
the email address has been successfully verified. Again, I have to emphasize
that, this is just for demo purposes. In a real-world application, we have
to send users to a proper front-end page to display this info nicely with
text and images.

Alright, before we finish, let's check the DB to make sure everything is
correctly updated. In the `users` table, we have a new user: `techschool`.
And its `is_email_verified` field is `TRUE`, as you might expect.

![](../images/part62/5.png)

In the `verify_emails` table, there's also 1 record that belongs to the
"techschool" user. And its `is_used` field has been set to `TRUE`, so 
that means, it cannot be reused anymore in the future.

![](../images/part62/6.png)

We can test that by going back to the browser and refreshing this page.

![](../images/part62/7.png)

It will send another `verify_email` request to the server, but this time,
we got an error: "failed to verify email".

That's exactly what we wanted! And that also wraps up today's lecture 
about implementing email verification API.

I hope it was interesting and useful for you.

Thanks a lot for watching, happy learning, and see you in the next 
lecture!



