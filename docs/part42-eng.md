# Implement gRPC API to create and login users in Go

[Original video](https://www.youtube.com/watch?v=7xiWqyZW9lE)

Hello guys, welcome back to the backend master class! In the [last
lecture](part41-eng.md), we've learned how to start a gRPC server
using the codes that `protoc` generated. We also tried to call the
RPCs on the server with Evans client. However, all the RPCs to
create & login users are still returning an error code, because 
they're not implemented yet at the moment.

```shell
call CreateUser
username (TYPE_STRING) => quang
full_name (TYPE_STRING) => Quang
email (TYPE_STRING) => quang@gmail.com
password (TYPE_STRING) => secret
command call: rpc error: code = Unimplemented desc = method CreateUser not implemented
```

So today, let's learn how to implement them!

## Implementing API to create users

If we open the `UnimplementedSimpleBankServer`, it will bring 
us to the generated gRPC service code, where we can find the 
`SimpleBankServer` interface that we need to implement. And
here, the `UnimplementedSimpleBankServer` has already given us the 
basic implementation of the 2 methods that are required by 
the interface: `CreateUser` and `LoginUser`. What we have to
do now, is implement them on our own `Server` struct.

So first, I'm gonna copy this `CreateUser` method.

```go
func (UnimplementedSimpleBankServer) CreateUser(context.Context, *CreateUserRequest) (*CreateUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateUser not implemented")
}
```

Then let's create a new file: `rpc_create_user.go` inside the 
`gapi` package, and paste in the copied codes. In order to 
add this method to our server, we have to change this function
receiver to be our own `Server` object. The `Server` struct is
already defined in the `server.go` file in the [previous 
lecture](part41-eng.md).

OK, next let's add the parameter name for the context and the
request. The `CreateUserRequest` and the `CreateUserResponse`
come from the `pb` package that `protoc` has generated for us.
Once we save the file, all the necessary packages will be 
automatically imported and there will be no errors in the file
anymore.

```go
func (server *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateUser not implemented")
}
```

Now, we have to write a real implementation for this method.
It would be very similar to what we've implemented before
in the Gin HTTP API server. So let's open the `user.go` file
inside the `api` package.

Here we can find the `createUser` handler method.

```go
func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	arg := db.CreateUserParams{
		Username:       req.Username,
		HashedPassword: hashedPassword,
		FullName:       req.FullName,
		Email:          req.Email,
	}

	user, err := server.store.CreateUser(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				ctx.JSON(http.StatusForbidden, errorResponse(err))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := newUserResponse(user)
	ctx.JSON(http.StatusOK, rsp)
}
```

As you can see, for Gin, we have to bind the input parameters
into the request object ourselves. But for gRPC, we don't 
have to do that, because it has already been taken care of
by the framework.

So let's ignore the input binding codes, and just copy the main
part of this function, where we interact with the DB to create
a new user. Paste it to our RPC method.

```go
func (server *Server) CreateUser(context.Context, *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	arg := db.CreateUserParams{
		Username:       req.Username,
		HashedPassword: hashedPassword,
		FullName:       req.FullName,
		Email:          req.Email,
	}

	user, err := server.store.CreateUser(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				ctx.JSON(http.StatusForbidden, errorResponse(err))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	
	return nil, status.Errorf(codes.Unimplemented, "method CreateUser not implemented")
}
```

OK, now let's see how we can modify the code to make it 
work with gRPC. First, for the input password parameter,
as you can see, we can easily get it from the 
`CreateUserRequest`, because gRPC has already bound all
the input data into this object for us. Here we can 
get the value directly from the `Password` field of 
the struct,

```go
hashedPassword, err := util.HashPassword(req.Password)
```

or a better way is to use a getter function:
`GetPassword()`, because this function will provide a
safety check, just in case the request object is `nil`.

```go
func (x *CreateUserRequest) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}
```

Alright, let's go back to our code. Here we're hashing the 
password because we don't want to store its plaintext value
in the database. If an error occurs, we will have to return
an internal error to the client.

```go
hashedPassword, err := util.HashPassword(req.GetPassword())
if err != nil {
    ctx.JSON(http.StatusInternalServerError, errorResponse(err))
    return
}
```

How can we do that in gRPC?

Well, if you look at the `return` statement that `protoc` generated
for us, it's pretty simple.

```go
return nil, status.Errorf(codes.Unimplemented, "method CreateUser not implemented")
```

All we have to do is to create an error using the `status.Errorf()`
function. `status` is actually a sub package of gRPC. So this
`Errorf` function allows us to pass in 2 parameters. The first 
parameter is a status code, which in our case should be 
`codes.Internal`. Again, `codes` is another sub package of gRPC,
where all the gRPC status codes are defined. And the second 
parameter is an error message, let's say: "failed to hash 
password".

```go
hashedPassword, err := util.HashPassword(req.GetPassword())
if err != nil {
    return nil, status.Errorf(codes.Internal, "failed to hash password")
}
```

We can also embed the original error into this message to make
it cleaner.

```go
hashedPassword, err := util.HashPassword(req.GetPassword())
if err != nil {
    return nil, status.Errorf(codes.Internal, "failed to hash password: %s", err)
}
```

After this step, we will call a function in the DB store to
create a new user. Here's the input argument that we need to 
pass in the function. It's possible to leave it just like 
this.

```go
arg := db.CreateUserParams{
    Username:       req.Username,
    HashedPassword: hashedPassword,
    FullName:       req.FullName,
    Email:          req.Email,
}
```

We're accessing the fields directly from the `request` object.
Or, we can do it a little bit better, by calling the getter
functions to get the value. So `request.GetUsername()`, the
hashed password is the value we've just computed. Then 
`request.GetFullName()`, and finally, `request.GetEmail()`.
Now we call `server.store.CreateUser()` and pass in the input
argument.

```go
arg := db.CreateUserParams{
    Username:       req.GetUsername(),
    HashedPassword: hashedPassword,
    FullName:       req.GetFullName(),
    Email:          req.GetEmail(),
}

user, err := server.store.CreateUser(ctx, arg)
```

If error is not `nil`, then there are 2 cases to check.
In case the error is a unique violation, it means there's
already an existing user with the same username. So we
will have to return a different error status code here.
I will copy the `return` statement from above, then change
this status code to `AlreadyExists` and the error message
should be: "username already exists". In the other case,
we don't know exactly what caused the error in the database,
so we just return an internal error, with a message saying
"failed to create user".

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

Finally, if no errors occur, this is where we should create a response
object and return it to the gRPC client. So, let's create a 
`CreateUserResponse` object here. And return that object with a `nil`
error at the end of the function.

```go
rsp := &pb.CreateUserResponse{}
return rsp, nil
```

Now, what do we have in the response? Well, all the required fields
have already been defined in the generated struct. In this case,
there's only 1 `User` field in the struct. And the type of this 
field is `pb.User`, which is another struct generated by `protoc`
from the `protobuf` definition we wrote in lecture 40.

```go
rsp := &pb.CreateUserResponse{
    User: &pb.User{

    },
}
```

We cannot just return the `db.User` object here, because it's a 
different type. And actualy, we should not mix up the DB layer 
with the API layer struct. It's better to separate them from
each other, because sometimes we don't want to return every 
field in the DB to the client. The hashed password, for 
example, is sensitive information that we want to hide. So
what we need now is a function to convert the `db.User` to 
`pb.User`.

I'm gonna create a new file called `converter.go` inside the `gapi`
package. And in this file, let's add a function: `convertUser()`,
which will take a `db.User` object as input, and returns a `pb.User`
object as output.

```go
func convertUser(user db.User) pb.User {
	
}
```

With this function, we can simply call it in the `CreateUser` RPC
to convert the internal `db.User` object to the required output 
type of gRPC.

```go
rsp := &pb.CreateUserResponse{
    User: convertUser(user),
}
```

Now let's implement the `convertUser()` function. Actually, the
return type should be a pointer to `pb.User`, because that's
what the generated response struct wants. The conversion is in 
fact pretty simple. We need to convert each and every field of
the struct. First, the `Username` should be `user.Username`, 
the `Fullname` should be `user.FullName`, `Email` should be
`user.Email`, `PasswordChangedAt` should be 
`user.PasswordChangedAt`, but this field is a bit different,
because the timestamp type in `protobuf` is not the same sa the
Golang's `time` type. So here we have to use the 
`timestamppb.New()` function to convert its value. Similar for 
the `user.CreatedAt` field. And that's basically it!

```go
func convertUser(user db.User) *pb.User {
	return &pb.User{
		Username: user.Username,
		FullName: user.FullName,
		Email: user.Email,
		PasswordChangedAt: timestamppb.New(user.PasswordChangedAt),
		CreatedAt: timestamppb.New(user.CreatedAt),
	}
}
```

The `convertUser()` function is completed.

Our `CreateUser` RPC is now also ready to handle the request.

Let's run 

```shell
make server
```

in the terminal to start the gRPC server.

Then, in another tab, let's run 

```shell
make evans
```

to open Evans client console.

Then, let's call the `CreateUser` RPC. Enter the username, full 
name, email, and password.

```shell
call CreateUser
username (TYPE_STRING) => quang1
full_name (TYPE_STRING) => Quang
email (TYPE_STRING) => quang@gmail.com
password (TYPE_STRING) => secret
{
  "user": {
    "username": "quang1",
    "fullName": "Quang",
    "email": "quang@gmail.com",
    "passwordChangedAt": "0001-01-01T00:00:00Z",
    "createdAt": "2022-04-10T10:10:37.688043Z"
  }
}
```

And voilà, a new user has been created, we've got the
response from the server, with the information of the 
created user.

Pretty cool, isn't it?

So that's how we implement the `CreateUser` API in gRPC.

## Implementing Login User API

How about the Login User API?

Well, it should be very similar.

Now is the time for you to pause the video and try
to implement it on your own.

Then we will do it together in a moment. Alright, 
have you managed to implement it successfully?

The first thing we need to do is to follow the 
`UnimplementedSimpleBankServer`.

In this file `service_simple_bank_grpc.pb.go`, let's
copy the `LoginUser()` method.

```go
func (UnimplementedSimpleBankServer) LoginUser(context.Context, *LoginUserRequest) (*LoginUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method LoginUser not implemented")
}
```

We will need add this method to our real Simple Bank
server implementation.

So next, I'm gonna create a new file called 
`rpc_login_user.go` inside the `gapi` folder. To
keep the code clean and well-structured, I recommend
you to keep each RPC in a separate file.

OK, now let's paste in the code that we've just 
copied.

Change the receiver of this function to our `server`
object. Set the variable name for the context and
request. And add package `pb` to the `LoginUserRequest`
and `LoginUserResponse`.

Then, let's open the login user API's HTTP handler
that we've implemented for Gin server before. Ignore
the request binding, and copy all the main processing
codes, up to the part where the login user response 
is created. Then paste that code to the new Login User
RPC function.

OK, the first step to handle the login user request
would be fetching the user from the database.

So here, we call `server.store.GetUser()` to find
the user with this username.

```go
func (server *Server) LoginUser(ctx context.Context, req *pb.LoginUserRequest) (*pb.LoginUserResponse, error) {
    user, err := server.store.GetUser(ctx, req.GetUsername())
    ...
}
```

If user is not found, we will return a `nil` response
together with a new error, created by the 
`status.Errorf()` function. In this case, the error
will store the status code `NotFound`, and a message
saying "user not found". I'm gonna delete this JSON
response of the Gin handler. In case an unexpected
error occurs, we return another error with status code
`Internal` and message "failed to find user". 

```go
if err != nil {
    if err == sql.ErrNoRows {
        return nil, status.Errorf(codes.NotFound, "user not found")
    }
    return nil, status.Errorf(codes.NotFound, "failed to find user")
}
```

If the user exists, and has been successfully fetched 
from the DB, then next, we have to check if the input
password matches with the one in our DB or not. Note
that we only store the hashed value of the password,
so here we call the `util.CheckPassword()` function
to compare the plaintext password with the stored 
hashed value. If error is not `nil`, then it means
the provided password is incorrect.

In this case, we return a `nil` response, and an error
with status code `NotFound` to the client.

```go
err = util.CheckPassword(req.Password, user.HashedPassword)
if err != nil {
    return nil, status.Errorf(codes.NotFound, "incorrect password")
}
```

OK, now if the password matches, we call the 
`tokenMaker.CreateToken()` function to create a
new access token. If this call returns a not `nil`
error, we will return an internal error to the client
with a message saying "failed to create access 
token".

```go
accessToken, accessPayload, err := server.tokenMaker.CreateToken(
    user.Username,
    server.config.AccessTokenDuration,
)
if err != nil {
    return nil, status.Errorf(codes.Internal, "failed to create access token")
}
```

Similarly, we do the same if the call to create refresh token
fails.

```go
refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(
    user.Username,
    server.config.RefreshTokenDuration,
)
if err != nil {
    return nil, status.Errorf(codes.Internal, "failed to create refresh token")
}
```

You can go back to watch lecture 37 if you don't know about
refresh token.

Alright, now the final step is to create a new session record 
in the DB.

```go
session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
    ID:           refreshPayload.ID,
    Username:     user.Username,
    RefreshToken: refreshToken,
    UserAgent:    ctx.Request.UserAgent(),
    ClientIp:     ctx.ClientIP(),
    IsBlocked:    false,
    ExpiresAt:    refreshPayload.ExpiredAt,
})
```

Here you can see that most of the parameters are already
available, except for the User agent and the client IP
address. We will learn how to get them from the gRPC
context metadata in another lecture. For now, let's
just set an empty string to both of them.

```go
session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
    ID:           refreshPayload.ID,
    Username:     user.Username,
    RefreshToken: refreshToken,
    UserAgent:    "",
    ClientIp:     "",
    IsBlocked:    false,
    ExpiresAt:    refreshPayload.ExpiredAt,
})
```

Again, if the call fails, we return an internal error,
saying "failed to create session".

```go
session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
    ID:           refreshPayload.ID,
    Username:     user.Username,
    RefreshToken: refreshToken,
    UserAgent:    "",
    ClientIp:     "",
    IsBlocked:    false,
    ExpiresAt:    refreshPayload.ExpiredAt,
})
if err != nil {
    return nil, status.Errorf(codes.Internal, "failed to create session")
}
```

When everything is successful, we will create a 
`pb.LoginUserResponse` object, and return it together with a
`nil` error. Now, the first field of this response object
is a `User`, so we just use the `convertUser()` function to convert
the `db.User` to `pb.User`. The session ID is gonna be 
`session.ID.String()`, the `AccessToken` should be `accessToken`,
and the same for the refresh token. Now the `AccessTokenExpiresAt`
will be `timestamppb.New()`, and we can get its value from the
`accessPayload`. Similarly, the `RefreshTokenExpiresAt` value
should come from the `refreshPayload`. And that's it!

```go
rsp := &pb.LoginUserResponse{
    User: convertUser(user),
    SessionId: session.ID.String(),
    AccessToken: accessToken,
    RefreshToken: refreshToken,
    AccessTokenExpiresAt: timestamppb.New(accessPayload.ExpiredAt),
    RefreshTokenExpiresAt: timestamppb.New(refreshPayload.ExpiredAt),
}
return rsp, nil
```

We're done with the implementation of the `LoginUser` RPC.
Let's open the terminal and restart the gRPC server. Reconnect
to it with Evans client. This time, we will call the `LoginUser`
RPC. Enter the username and password we created before.

```shell
call LoginUser
username (TYPE_STRING) => quang1
password (TYPE_STRING) => secret
{
  "user": {
    "username": "quang1",
    "fullName": "Quang",
    "email": "guang@gmail.com",
    "passwordChangedAt": "0001-01-01T00:00:00Z",
    "createdAt": "2022-04-10T10:10:37.688043Z"
  },
  "sessionId": "e71d4c75-827e-4a4c-84f3-4c40b3554439",
  "accessToken": "v2.local.uy2QOalUcpdYWXdXQrYMJCSiohbf6FX_sa6eaa6_BtWGRAfX9NYbAGSzV-0AcVOd8YxOS-Fi3jxZrIjPoLIvu3bj9hDqaL2apx9vv4noGmDxfZUTmut6YRPrQMiR2I2FRVP283ZQL1ocmDjAkI2QhXnN0IZaiWFlypmGE_xq37zThDcIjCU6WX_4lrekMNrFkRKOdky2EVyPwCsRV1J_MDeq00ePEjaA-gwx7kkwy3rmfO3ir3R2eR36HNGYKaUUQH77ZuxtTkj7YZZjclk.bnVsbA",
  "refreshToken": "v2.local.ztop-Ppht-YlqJ9myb_vPTCzIWDD2LiR3n7bVIEsKPoxJ_71379knOo7LsGAxg6cQskbn1fCzrBHAbXE-hg0-YAHfNr1Y69lRc12pKVgq8nh-odcvIpwDha03vdtkA4_DY_eeqkUvBRL928No_HnH32OfnyrBOX_yo6OZmPnOmw4HjyFSFh1U-6sBZBmgELdcBwaxo6Pci6sdYrCPOU4Qs0vh5aiJ7IfqmraGN-Yx5qVnuGyi0n8kHInL444HUSxKWUZKqf4T-kkK4B6E10.bnVsbA",
  "accessTokenExpiresAt": "2022-04-10T10:39:32.035910Z",
  "refreshTokenExpiresAt": "2022-04-11T10:24:32.036171Z"
}
```

Then voilà, we've got a successful response, with all the user
information and the access and refresh tokens.

Awesome!

Now let's try to login with a user that doesn't exist.

```shell
call LoginUser
username (TYPE_STRING) => quang2
password (TYPE_STRING) => secret
command call: rpc error: code = NotFound desc = user not found
```

This time, we've got an error with status code `NotFound`
and message: "user not found".

How about login with an incorrect password?

```shell
call LoginUser
username (TYPE_STRING) => quang1
password (TYPE_STRING) => wrong
command call: rpc error: code = NotFound desc = incorrect password
```

In this case, we still got a NotFound error, but the 
message is "incorrect password".

So everything is working well. And that brings us to the
end of this lecture.

We've successfully implemented 2 unary gRPC APIs to create
and login user.

I hope it was interesting and useful for you.

In the next video, we will learn how to use gRPC gateway 
to allow serving both gRPC and HTTP requests to these 
APIs.

Until then, happy learning, and see you in the next 
lecture!