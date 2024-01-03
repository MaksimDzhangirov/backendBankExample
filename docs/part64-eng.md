# How to test a gRPC API that requires authentication

[Original video](https://www.youtube.com/watch?v=MI7ucbAlZPM)

Hello everyone, welcome to the backend master class! In the last lecture, 
we're written some unit tests for the CreateUser RPC. However, this API
doesn't require any authentication, anyone cal call it without logging in
to get an access token. So today, I want to show you how to write unit
tests for the UpdateUser RPC, which has this `authorizeUser()` method

```go
authPayload, err := server.authorizeUser(ctx)
```

to ensure that a valid access token must be provided. The definition of this
API is written in the `rpc_update_user.proto` file.

```proto
message UpdateUserRequest {
  string username = 1;
  optional string full_name = 2;
  optional string email = 3;
  optional string password = 4;
}
```

Its request contains the `username` and some optional fields, such as 
`full_name`, `email`, and `password`, that can be updated. Alright, let's
write the tests!

## Write unit test for gRPC UpdateUser API

I'm gonna create a new file called `rpc_update_user_test.go` inside the 
`gapi` package. Then, let's copy the unit tests that we wrote in the last
lecture, paste them into the new file and change the name of the function
to `TestUpdateUserAPI`. OK, next, let's change the type of the request
to `pb.UpdateUserRequest`. For this API, we don't have an async task, so
let's remove the mock task distributor from the `buildStubs` method. Then
let's change the response type to `pb.UpdateUserResponse`.

```go
func TestUpdateUserAPI(t *testing.T) {
	user, password := randomUser(t)

	testCases := []struct {
		name          string
		req           *pb.UpdateUserRequest
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, res *pb.UpdateUserResponse, err error)
	}{
        ...
    }
    ...
}
```

Now let's update the parameters of the `OK` case. First, the request type
must be `UpdateUserRequest`. And since the `FullName`, `Email`, and 
`Password` fields are optional, their types are pointers to string. I'm 
gonna define a new variable for the new name, and use `util.RandomOwner()`
function to generate a random value for it. Similarly, a new email is 
generated with `util.RandomEmail()`.

```go
newName := util.RandomOwner()
newEmail := util.RandomEmail()
```

To keep it simple, I'm not gonna update the password for now. So let's remove
that field from the request. Then set the `FullName` field to `newName`.
As you can see, VSCode automatically adds an ampersand character to the 
beginning of the variable, since we need a pointer for this field. The same
thing applies to the `Email` field, we'll set it to `newEmail`.

```go
name: "OK",
req: &pb.UpdateUserRequest{
    Username: user.Username,
    FullName: &newName,
    Email:    &newEmail,
},
```

OK, now let's remove the mock task distributor from the `buildStubs` 
function. In this function, we'll have to mock the `UpdateUser` query, 
so let's change this `db.CreateUserTxParams` to `db.UpdateUserParams`.
The only required field in this struct is `Username`. All other fields
are optional, as you can see from their nullable types. In this case, we 
only want to update the `FullName` and `Email`, so let's set the `Username`
field to `user.Username`, `FullName` to `sql.NullString`, with a `String`
value of `newName`, and the `Valid` field is set to `true`. Similar for
the `Email` field, it's also a `sql.NullString`, with the `String` value
is `newEmail` and `Valid` is `true`.

```go
arg := db.UpdateUserParams{
    Username: user.Username,
    FullName: sql.NullString{
        String: newName,
        Valid:  true,
    },
    Email: sql.NullString{
        String: newEmail,
        Valid:  true,
    },
}
```

Now, we should `EXPECT` the `UpdateUser` function to be called with the 
arguments we've just declared. 

```go
store.EXPECT().
    UpdateUser(gomock.Any(), gomock.Eq(arg)).
```

Here I use the `gomock.Eq` matcher to compare the input argument. For the
successful case, this method should be called exactly once, and it should
return the updated user together with an error object. So we'll declare
the `updatedUser` and return it here with a `nil` error. The updated user
will be a `db.User` object, where all fields stay the same as the old user,
except for the `FullName` and `Email`, which should be set to `newName`
and `newEmail`.

```go
updatedUser := db.User{
    Username:          user.Username,
    HashedPassword:    user.HashedPassword,
    FullName:          newName,
    Email:             newEmail,
    PasswordChangedAt: user.PasswordChangedAt,
    CreatedAt:         user.CreatedAt,
    IsEmailVerified:   user.IsEmailVerified,
}
store.EXPECT().
    UpdateUser(gomock.Any(), gomock.Eq(arg)).
    Times(1).
    Return(updatedUser, nil)
```

Alright, now we can remove the `EXPECT` command of the mock task 
distributor. And move on to the `checkResponse` function.

```go
buildStubs: func(store *mockdb.MockStore) {
    arg := db.UpdateUserParams{
        Username: user.Username,
        FullName: sql.NullString{
            String: newName,
            Valid:  true,
        },
        Email: sql.NullString{
            String: newEmail,
            Valid:  true,
        },
    }
    updatedUser := db.User{
        Username:          user.Username,
        HashedPassword:    user.HashedPassword,
        FullName:          newName,
        Email:             newEmail,
        PasswordChangedAt: user.PasswordChangedAt,
        CreatedAt:         user.CreatedAt,
        IsEmailVerified:   user.IsEmailVerified,
    }
    store.EXPECT().
        UpdateUser(gomock.Any(), gomock.Eq(arg)).
        Times(1).
        Return(updatedUser, nil)
},
```

For this test, it will receive a `UpdateUserResponse` and an error object 
as the output of the API. Since this is a happy case, we expect to see
no errors, and the response should be not `nil`. We'll get the updated
user from the response, and check that the username stays the same, but
the full name should be changed to `newName`, and the email should be
changed to `newEmail`. And that's it!

```go
checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
    require.NoError(t, err)
    require.NotNil(t, res)
    updatedUser := res.GetUser()
    require.Equal(t, user.Username, updatedUser.Username)
    require.Equal(t, newName, updatedUser.FullName)
    require.Equal(t, newEmail, updatedUser.Email)
},
```

We're done with the setup of the `OK` case. I'm gonna remove all other 
cases for now, so that we can focus on making this only case pass first.

OK, let's update the main content of the `Run` function!

We can get rid of the task controller and task distributor. Remove the
task distributor, when building stubs, and just use a `nil` task 
distributor, when creating a new test server. Here, we'll have to call
`server.UpdateUser` instead of `CreateUser`. And that will be it!

```go
t.Run(tc.name, func(t *testing.T) {
    storeCtrl := gomock.NewController(t)
    defer storeCtrl.Finish()
    store := mockdb.NewMockStore(storeCtrl)

    tc.buildStubs(store)
    server := newTestServer(t, store, nil)

    res, err := server.UpdateUser(context.Background(), tc.req)
    tc.checkResponse(t, res, err)
})
```

Oh, the compiler is still complaining about this `password` variable

```go
user, password := randomUser(t)
```

is unused, so I'm gonna replace it with a blank identifier.

```go
user, _ := randomUser(t)
```

And we're good to go! Let's run the test to see what happens!

```shell
go test -timeout 30s -run ^TestUpdateUserAPI$ github.com/techschool/simplebank/gapi -v count=1
=== RUN   TestUpdateUserAPI
=== RUN   TestUpdateUserAPI/OK
    /Users/quangpham/Projects/techschool/simplebank/gapi/rpc_update_user_test.go:62: 
            Error Trace:    rpc_update_user_test.go:62
                                                    rpc_update_user_test.go:84
            Error:          Received unexpected error:
                            rpc error: code = Unauthenticated desc = unauthorized: missing metadata
```

It failed! And the error is `Unauthenticated: missing metadata`. This is 
totally expected, because the UpdateUser RPC requires a valid access
token to be sent via the metadata of the request's context, but we haven't
added any access token to the context yet. Here in the `authorizeUser` 
function,

```go
func (server *Server) authorizeUser(ctx context.Context) (*token.Payload, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing metadata")
	}

	values := md.Get(authorizationHeader)
	if len(values) == 0 {
		return nil, fmt.Errorf("missing authorization header")
	}

	authHeader := values[0]
	fields := strings.Fields(authHeader)
	if len(fields) < 2 {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	authType := strings.ToLower(fields[0])
	if authType != authorizationBearer {
		return nil, fmt.Errorf("unsupported authorization type: %s", authType)
	}

	accessToken := fields[1]
	payload, err := server.tokenMaker.VerifyToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("invalid access token: %s", err)
	}

	return payload, nil
}

```

we're trying to get the metadata from the incoming context, then we get 
the authorization header from the metadata, and this header's value
should contain the `Bearer` access token. We then use the token maker
to verify that the token is valid. So, in the unit test, we'll have to
add a valid access token to the context's metadata before sending the 
request.

In order to do that, I'm gonna add a `buildContext` method to the 
test case struct. It will take a `testing.T` object and a token maker
interface as input arguments, and it will return a context object 
containing the metadata with the access token.

```go
testCases := []struct {
    name          string
    req           *pb.UpdateUserRequest
    buildStubs    func(store *mockdb.MockStore)
    buildContext  func(t *testing.T, tokenMaker token.Maker) context.Context
    checkResponse func(t *testing.T, res *pb.UpdateUserResponse, err error)
}{
    ...
}
```

The reason for adding this as part of the test case struct is, for 
different cases, we might have different metadata, perhaps with an invalid
or expired access token, for example. Or even return a 
`context.Background()` if we don't want to send the token.

```go
buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
    return context.Background()
},
```

In the `Run` test function, we'll call `tc.buildContext()`, and pass in 
the `t` object and `server.tokenMaker` to get back a context with added
metadata. Then, use that context as the first argument of the `UpdateUser`
call.

```go
ctx := tc.buildContext(t, server.tokenMaker)
res, err := server.UpdateUser(ctx, tc.req)
```

OK, now, for the successful test case, we'll have to add a valid access
token to the metadata of the context. So first, I'm gonna assign this 
`context.Background()` to this `ctx` variable. Then, we can use the 
`NewIncomingContext` function of the `grpc` metadata package to add a
metadata (`md`) object to the background context.

```go
ctx := context.Background()
return metadata.NewIncomingContext(ctx, md)
```

We can create the metadata object with the `metadata.MD` struct. What we're
gonna add to this object is the authorization header with the access token.
Note that the `md` object is just a map of string key and string slice 
values, so here we have to declare a string slice, with only 1 element: a 
`bearerToken`.

```go
md := metadata.MD{
    authorizationHeader: []string{
        bearerToken,
    },
}
```

Next step, I'm gonna create the bearer token using `fmt.Sprint()` function. 
It should have 2 substrings separated by a space. The first one is the 
authorization type, which is "bearer" in this case. And the second one is 
the access token that we're gonna create in a moment. We'll use the 
`tokenMaker` here to create a valid access token.

```go
tokenMaker.CreateToken()
bearerToken := fmt.Sprintf("%s %s", authorizationBearer, accessToken)
```

I pass in this `CreateToken` method the user's username and a valid 
duration of 1 minute. This method will return the access token string,
a token payload, and an error. We don't use the token payload, so I leave
it as a blank identifier. Then check that the returned error is `nil`
with `require.NoError()` function.

```go
accessToken, _, err := tokenMaker.CreateToken(user.Username, time.Minute)
require.NoError(t, err)
```

We can make the code a bit shorter by putting this background context 
(`ctx := context.Background()`) directly in this 
`metadata.NewIncomingContext(ctx, md)` call. And that's basically it. 
We're done adding an access token to the context's metadata.

```go
buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
    ctx := context.Background()
    accessToken, _, err := tokenMaker.CreateToken(user.Username, time.Minute)
    require.NoError(t, err)
    bearerToken := fmt.Sprintf("%s %s", authorizationBearer, accessToken)
    md := metadata.MD{
        authorizationHeader: []string{
            bearerToken,
        },
    }
    return metadata.NewIncomingContext(ctx, md)
},
```

Let's rerun the test to see how it goes!

```shell
go test -timeout 30s -run ^TestUpdateUserAPI$ github.com/techschool/simplebank/gapi -v count=1
=== RUN   TestUpdateUserAPI
=== RUN   TestUpdateUserAPI/OK
--- PASS: TestUpdateUserAPI (0.07s)
    --- PASS: TestUpdateUserAPI/OK (0.00s)
PASS
ok      github.com/techschool/simplebank/gapi     0.410s
```


This time, the test passed! Excellent!

Next, let's try to add some more cases to test other scenarios, such as 
when the user is not found. I'm gonna duplicate the `OK` case, and change
its name to `UserNotFound`.

```go
name: "UserNotFound",
req: &pb.UpdateUserRequest{
    Username: user.Username,
    FullName: &newName,
    Email:    &newEmail,
},
```

Since this is an error case, in the `buildStubs` function, we don't need 
real parameters or the updated user, so let's get rid of them.

```go
buildStubs: func(store *mockdb.MockStore) {
    store.EXPECT().
        UpdateUser(gomock.Any(), gomock.Any()).
        Times(1).
        Return(db.User{}, sql.ErrNoRows)
},
```

Here, we can just use `gomock.Any()` to match any parameters, and return 
an empty user object, with an error: `sql.ErrNoRows`. The DB driver will 
return this error if the user is not found in the database.

Now, since the `buildContext` function is exactly the same as the `OK` 
case, let's refactor the code, so we don't have a duplicate one here. This 
method can actually be made as a global function, that can be reused in 
other RPC's unit tests as well. So I'm gonna put it in the `main_test.go` 
file, and name it `newContextWithBearerToken()`. We'll have to add a 
`username` parameter here since the `user` object is not available in 
this function. Then pass it into the `tokenMaker.CreateToken()` call. By
the way, I think we should also change this `time.Minute` hard-coded
value into a duration parameter, so that we can pss in a negative duration
to get an expired access token when we test that case. Alright, now let's
get back to the unit tests!

```go
func newContextWithBearerToken(t *testing.T, tokenMaker token.Maker, username string, duration time.Duration) context.Context {
	ctx := context.Background()
	accessToken, _, err := tokenMaker.CreateToken(username, duration)
	require.NoError(t, err)
	bearerToken := fmt.Sprintf("%s %s", authorizationBearer, accessToken)
	md := metadata.MD{
		authorizationHeader: []string{
			bearerToken,
		},
	}
	return metadata.NewIncomingContext(ctx, md)
}
```

I'm gonna delete the whole body of this `buildContext` function, and 
replace it with only 1 return: `newContextWithBearerToken()`. Pass in the
`t`, `tokenMaker`, `user.Username` and `time.Minute` as its input 
arguments.

```go
buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
    return newContextWithBearerToken(t, tokenMaker, user.Username, time.Minute)
},
```

This same statement can be used in the `buildContext` of the `UserNotFound`
case. Looks pretty neat, isn't it?

Next, in the `checkResponse` function, instead of require no errors, we 
will require a non-nil error to be returned. We should also check the 
status code of the call, so let's copy it from a failure case, that we
wrote in the `CreateUser` test in the previous lecture.

```go
st, ok := status.FromError(err)
require.True(t, ok)
require.Equal(t, codes.NotFound, st.Code())
```

Basically, we convert the returned error into a status object, and require
the `Code` field of that object to be `NotFound`, since it's the value should
be returned by the server in this case.

Alright, all done. Now it's time to rerun the test!

```shell
go test -timeout 30s -run ^TestUpdateUserAPI$ github.com/techschool/simplebank/gapi -v count=1
=== RUN   TestUpdateUserAPI
=== RUN   TestUpdateUserAPI/OK
=== RUN   TestUpdateUserAPI/UserNotFound
--- PASS: TestUpdateUserAPI (0.07s)
    --- PASS: TestUpdateUserAPI/OK (0.00s)
    --- PASS: TestUpdateUserAPI/UserNotFound (0.00s)
PASS
ok      github.com/techschool/simplebank/gapi     0.385s
```

It passed. Awesome! The rest of the test cases can be written in a pretty 
similar way, so let's duplicate this "UserNotFound" case, and test the 
expired token scenario. 

```go
name: "ExpiredToken",
req: &pb.UpdateUserRequest{
    Username: user.Username,
    FullName: &newName,
    Email:    &newEmail,
},
```

For this case, we expect the `UpdateUser` function to not be called (or 
called 0 times).

```go
buildStubs: func(store *mockdb.MockStore) {
    store.EXPECT().
        UpdateUser(gomock.Any(), gomock.Any()).
        Times(0)
},
```

Since the request should have already been rejected by the `authorizeUser()`
call.

And as I said before, we can build a context with an expired token. Just
by passing in a negative value for the `duration` parameter.

```go
buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
    return newContextWithBearerToken(t, tokenMaker, user.Username, -time.Minute)
},
```

Finally, in the `checkResponse` function, we expect the status code to be
`Unauthenticated`. And we're good to go!

```go
checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
    require.Error(t, err)
    st, ok := status.FromError(err)
    require.True(t, ok)
    require.Equal(t, codes.Unauthenticated, st.Code())
},
```

Let's rerun the tests!

```shell
go test -timeout 30s -run ^TestUpdateUserAPI$ github.com/techschool/simplebank/gapi -v count=1
=== RUN   TestUpdateUserAPI
=== RUN   TestUpdateUserAPI/OK
=== RUN   TestUpdateUserAPI/UserNotFound
=== RUN   TestUpdateUserAPI/ExpiredToken
--- PASS: TestUpdateUserAPI (0.07s)
    --- PASS: TestUpdateUserAPI/OK (0.00s)
    --- PASS: TestUpdateUserAPI/UserNotFound (0.00s)
    --- PASS: TestUpdateUserAPI/ExpiredToken (0.00s)
PASS
ok      github.com/techschool/simplebank/gapi     0.380s
```

They all passed!

Can you try adding other cases on your own? Let's say a test, when No
Authorization (or access token) is provided.

```go
name: "NoAuthorization",
req: &pb.UpdateUserRequest{
    Username: user.Username,
    FullName: &newName,
    Email:    &newEmail,
},
```

It's pretty easy, isn't it? All we have to do is change the return 
statement of `buildContext` function to `context.Background()`. Then
rerun the tests.

```shell
go test -timeout 30s -run ^TestUpdateUserAPI$ github.com/techschool/simplebank/gapi -v count=1
=== RUN   TestUpdateUserAPI
=== RUN   TestUpdateUserAPI/OK
=== RUN   TestUpdateUserAPI/UserNotFound
=== RUN   TestUpdateUserAPI/ExpiredToken
=== RUN   TestUpdateUserAPI/NoAuthorization
--- PASS: TestUpdateUserAPI (0.07s)
    --- PASS: TestUpdateUserAPI/OK (0.00s)
    --- PASS: TestUpdateUserAPI/UserNotFound (0.00s)
    --- PASS: TestUpdateUserAPI/ExpiredToken (0.00s)
    --- PASS: TestUpdateUserAPI/NoAuthorization (0.00s)
PASS
ok      github.com/techschool/simplebank/gapi     0.383s
```

And voilà, all passed again!

Before we finish, I'm gonna show you one last case, when an invalid email
address is provided. In this case, we cannot simply use the ampersand
character to get the address of a string constant like this 
`&invalid-email`. So we'll have to use a variable named `invalidEmail`,
and declare it at the top of the function like this.

```go
newName := util.RandomOwner()
newEmail := util.RandomEmail()
invalidEmail := "invalid-email"
```

```go
name: "InvalidEmail",
req: &pb.UpdateUserRequest{
    Username: user.Username,
    FullName: &newName,
    Email:    &invalidEmail,
},
```

Now the `UpdateUser` function should be called 0 times, and the context
should contain a valid access token.

```go
buildStubs: func(store *mockdb.MockStore) {
    store.EXPECT().
        UpdateUser(gomock.Any(), gomock.Any()).
        Times(0)
},
buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
    return newContextWithBearerToken(t, tokenMaker, user.Username, time.Minute)
},
```

But this time, we expect an `InvalidArgument` status code to be returned
by the server. 

```go
checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
    require.Error(t, err)
    st, ok := status.FromError(err)
    require.True(t, ok)
    require.Equal(t, codes.InvalidArgument, st.Code())
},
```

And that's it! We done! Let's rerun the test one last time!

```shell
go test -timeout 30s -run ^TestUpdateUserAPI$ github.com/techschool/simplebank/gapi -v count=1
=== RUN   TestUpdateUserAPI
=== RUN   TestUpdateUserAPI/OK
=== RUN   TestUpdateUserAPI/UserNotFound
=== RUN   TestUpdateUserAPI/ExpiredToken
=== RUN   TestUpdateUserAPI/NoAuthorization
=== RUN   TestUpdateUserAPI/InvalidEmail
--- PASS: TestUpdateUserAPI (0.07s)
    --- PASS: TestUpdateUserAPI/OK (0.00s)
    --- PASS: TestUpdateUserAPI/UserNotFound (0.00s)
    --- PASS: TestUpdateUserAPI/ExpiredToken (0.00s)
    --- PASS: TestUpdateUserAPI/NoAuthorization (0.00s)
    --- PASS: TestUpdateUserAPI/InvalidEmail (0.00s)
PASS
ok      github.com/techschool/simplebank/gapi     0.407s
```

All cases passed.

So now you know how to write unit tests for a gRPC API, that requires
authentication. You can base on this implementation to add more cases
for this update API, or write tests for other APIs in the project by 
yourself!

And that brings us to the end of this video.

I hope it was interesting and useful for you. Thanks a lot for watching!
Happy learning, and see you in the next lecture!
