# Why a bit of delay can be good for your async tasks

[Original video](https://www.youtube.com/watch?v=ILNiZgseLUI)

Hello everyone! Welcome to the backend master class! Continue with the
background worker topic. Today, I'll show you why it is important to add
a bit of delay to your async task.

## Why delay is important

If you still remember, in lecture 56 of the course, we've implemented
this `CreateUser` transaction

```go
func (server *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	...
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
	
	txResult, err := server.store.CreateUserTx(ctx, arg)
	...
}
```

with a callback function: `AfterCreate`. This function will be called
after a new user is created, to send an async task for email verification
to Redis. And here, you can see that we have a 10-second delay, 
`asynq.ProcessIn(10 * time.Second)`. This means that the task will only
be picked up by the worker 10 seconds after it is created. Now, to
understand why this delay is important, let's see what will happen if I
comment out this line.

```go
func (server *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	...
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
                // asynq.ProcessIn(10 * time.Second),
                asynq.Queue(worker.QueueCritical),
            }
            return server.taskDistributor.DistributeTaskSendVerifyEmail(ctx, taskPayload, opts...)
        },
    }
	
	txResult, err := server.store.CreateUserTx(ctx, arg)
	...
}
```

Then, let's open the terminal and do some tests. First, let's make sure 
both Postgres and Redis are up and running.

```shell
docker ps
CONTAINER ID   IMAGE                                                       COMMAND                  CREATED        STATUS                          PORTS                                       NAMES
1ef5fb823898   redis:7-alpine                                              "docker-entrypoint.s…"   3 weeks ago    Up 2 seconds                    0.0.0.0:6379->6379/tcp, :::6379->6379/tcp   redis
88cabd7ba6ab   postgres:12-alpine                                          "docker-entrypoint.s…"   3 weeks ago    Up 7 seconds                    0.0.0.0:5432->5432/tcp, :::5432->5432/tcp   postgres12
```

OK, they are!

So we can start our web server.

```shell
make server
go run main.go
11:11AM INF db migrated successfully
11:11AM INF start gRPC server at [::]:9090
11:11AM INF start task processor
11:11AM INF Starting processing
11:11AM INF start HTTP gateway server at [::]:8080
```

Next, I'm gonna open Postman, and send a request to create a new user:
"alice10".

![](../images/part58/1.png)

Alright, the request is successful. Now, let's look at the web server's
logs.

```shell
11:12AM INF enqueued task max_retry=10 payload="{\"username\":\"alice10\"}" queue=critical type=task:send_verify_email
11:12AM INF received an HTTP request duration=101.917294 method=POST path=/v1/create_user protocol=http status_code=200 status_text=OK
11:12AM INF processed task email=alice10@email.com payload="{\"username\":\"alice10\"}" type=task:send_verify_email
```

It received a request, and enqueued a verify email task. And the task is
processed immediately.

There was no problem at all. So does that mean we don't need to delay 
the task? Well, not really! It is OK this time, but there might be 
issues another time.

Let's take a look at the implementation of the `CreateUser` transaction.

```go
func (store *SQLStore) CreateUserTx(ctx context.Context, arg CreateUserTxParams) (CreateUserTxResult, error) {
	var result CreateUserTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.User, err = q.CreateUser(ctx, arg.CreateUserParams)
		if err != nil {
			return err
		}
		return arg.AfterCreate(result.User)
	})

	return result, err
}
```

Here, the `AfterCreate` function is called at the end of the transaction,
but since it's written inside the `execTx` function, this function is 
actually run before the transaction is committed. As you can see 
here,

```go
func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
    ...
	
	err = fn(q)
	if err != nil {
        if rbErr := tx.Rollback(); rbErr != nil {
            return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
        }
        return err
    }

    return tx.Commit()
}
```

the function is called, and if it returns an error, the transaction
will be rolled back. Otherwise, it will be committed to the database.

So what if there's high traffic on the db, that makes this commit 
statement take longer time to complete?

For example, we can add a `time.Sleep` 2 seconds here to simulate that
scenario.

```go
func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
    ...
	
	err = fn(q)
	if err != nil {
        if rbErr := tx.Rollback(); rbErr != nil {
            return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
        }
        return err
    }

    time.Sleep(2 * time.Second)
    return tx.Commit()
}
```

Let's see what will happen in this case.

I'm gonna restart the web server.

```shell
make server
go run main.go
11:13AM INF db migrated successfully
11:13AM INF start gRPC server at [::]:9090
11:13AM INF start task processor
11:13AM INF Starting processing
11:13AM INF start HTTP gateway server at [::]:8080
```

And go back to Postman to send a create-new-user request: "alice11".

![](../images/part58/2.png)

It will take about 2 seconds to complete because of the sleep statement. But
now, if we look at the logs,

```shell
11:13AM INF enqueued task max_retry=10 payload="{\"username\":\"alice11\"}" queue=critical type=task:send_verify_email
11:13AM ERR process task failed error="user doen't exist: skip retry for the task" payload="{\"username\":\"alice11\"}" type=task:send_verify_email
11:13AM WRN Retry exhausted for task id=4ec2e205-bdac-4211-912b-52afdf789f8f
11:13AM INF received an HTTP request duration=2100.460659 method=POST path=/v1/create_user protocol=http status_code=200 status_text=OK
```

we will see that the task has been enqueued, and it has already been 
processed, but failed because the user couldn't be found in the database.

Do you know why? Well, that's because it took the transaction 2 seconds
to be committed, but the send email task was enqueued without delays,
so it was processed by the worker immediately. And because of this,
when the worker tried to look for the user in the database, it could not
find the record, since the transaction was not committed yet. Also, in
the code, if the user couldn't be found, we return the `asynq.SkipRetry`
error,

```go
func (processor RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
	...

	user, err := processor.store.GetUser(ctx, payload.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user doen't exist: %w", asynq.SkipRetry)
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	...
}
```

so the task won't be retried in this case. And it's a big issue, because
user will never receive the verification email.

So now you know the importance of this 10 seconds delay. It will make 
room for the db transaction to be fully committed before the async task
is picked up by the worker.

Alright, let's restart the server and give it a try!

```shell
go run main.go
11:16AM INF db migrated successfully
11:16AM INF start gRPC server at [::]:9090
11:16AM INF start task processor
11:16AM INF Starting processing
11:16AM INF start HTTP gateway server at [::]:8080
```

I'm gonna change the `username` and `email` to "alice12". And resend the
create user request. It will take 2 seconds to finish, just like before.

![](../images/part58/3.png)

But this time, in the server log,

```shell
11:16AM INF enqueued task max_retry=10 payload="{\"username\":\"alice12\"}" queue=critical type=task:send_verify_email
11:16AM INF received an HTTP request duration=2109.752549 method=POST path=/v1/create_user protocol=http status_code=200 status_text=OK
```

we don't see any errors. That's because the task has been delayed for 10
seconds. So it's only been enqueued for now. And will only be processed
by the worker after 10 seconds.

```shell
11:16AM INF processed task email=alice12@email.com payload="{\"username\":\"alice12\"}" type=task:send_verify_email
```

At that time, the create user transaction has already been committed. So 
the task will be successfully processed, without any issues. And that's 
how a bit of delay can help us easily solve this problem. But make sure you
set the right amount of delay for your task, because for example, here

```go
func (server *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	...
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
	
	txResult, err := server.store.CreateUserTx(ctx, arg)
	...
}
```

if we just delay the task for 1 second, `asynq.ProcessIn(1 * time.Second)`,
which is lower than the 2 second that the transaction takes to commit,
then the problem will still occur.

Normally, I would say 10 seconds should be quite enough for the transaction
to commit, but we never know for sure. In some rare cases, anything could
happen. So, one way to deal with that, is to always allow the task to 
retry.

We can simply comment out this `async.SkipRetry` statement.

```go
func (processor RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
	...

	user, err := processor.store.GetUser(ctx, payload.Username)
	if err != nil {
		// if err == sql.ErrNoRows {
		//	return fmt.Errorf("user doen't exist: %w", asynq.SkipRetry)
		// }
		return fmt.Errorf("failed to get user: %w", err)
	}

	...
}
```

So even if the user is not found, the task will be retried again at 
another time, and hopefully at that time, it will be successful, when
transaction has been committed.

Of course, if the transaction fail to commit, then the user will never
be created in the DB, so the task will always fail. But it's not a big 
deal, because it will run out of retries eventually, since we have set
the maximum number of retries to 10 times in this case, `asynq.MaxRetry(10)`.

Alright, let's restart the server and do 1 more test.

```shell
make server
go run main.go
11:18AM INF db migrated successfully
11:18AM INF start gRPC server at [::]:9090
11:18AM INF start task processor
11:18AM INF Starting processing
11:18AM INF start HTTP gateway server at [::]:8080
```

In Postman, I'm gonna change the `username` and `email` to "alice13", and
resend the request!

![](../images/part58/4.png)

The request is successful after 2 seconds. But this time, in the log,

```shell
11:18AM INF enqueued task max_retry=10 payload="{\"username\":\"alice13\"}" queue=critical type=task:send_verify_email
11:18AM ERR process task failed error="failed to get user: sql: no rows in result set" payload="{\"username\":\"alice13\"}" type=task:send_verify_email
11:18AM INF received an HTTP request duration=2102.003726 method=POST path=/v1/create_user protocol=http status_code=200 status_text=OK
```

the task has been processed 1 time, and it failed. That's because we set
the delay to just 1 second.

But note that, we have removed the `async.SkipRetry` statement, so this 
task will be retried again later. OK, as you can see, after a while, the
task is retried, and it is successfully processed.

```shell
11:19AM INF processed task email=alice13@email.com payload="{\"username\":\"alice13\"}" type=task:send_verify_email
```

Exactly as we wanted.

Before we finish, I'm gonna revert this delay time to 10 seconds,

```go
func (server *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	...
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
	
	txResult, err := server.store.CreateUserTx(ctx, arg)
	...
}
```

and remove this `time.Sleep` statement in the `execTx` function.

```go
func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
    ...
	
	err = fn(q)
	if err != nil {
        if rbErr := tx.Rollback(); rbErr != nil {
            return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
        }
        return err
    }
	
    return tx.Commit()
}
```

I'll keep this `asynq.SkipRetry` here, but commented out, as a reference.

```go
func (processor RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
	...

	user, err := processor.store.GetUser(ctx, payload.Username)
	if err != nil {
		// if err == sql.ErrNoRows {
		//	return fmt.Errorf("user doen't exist: %w", asynq.SkipRetry)
		// }
		return fmt.Errorf("failed to get user: %w", err)
	}

	...
}
```

And that wraps up this lecture about the importance of delay when
processing async tasks.

I hope it was interesting and useful for you.

Thanks a lot for watching, happy learning, and see you in the next 
lecture!