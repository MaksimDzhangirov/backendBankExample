# Email verification in Go: design DB and send email

[Original video](https://www.youtube.com/watch?v=lEHkwDPHrcc)

Hello everyone. Welcome to the backend master class! In the previous lecture,
we're learned how to send emails using Gmail. However, we haven't integrated
it into our async worker yet. To recall, what we're trying to do is: 
sending a welcome email to a new user and request them to verify their
email address.

![send welcome email to new users](../images/part61/1.png)

And that's exactly what I'm gonna show you in this video.

The idea is simple: for each new user, we'll generate a random secret code
and store it in a DB table. This code will also be sent to the user's
registered email, together with a link to a page on our website for 
verification.

![store random secret code in the DB](../images/part61/2.png)

If the user really owns the email address, they would be able to access
the page with the correct secret code, and thus, their email will be
verified.

![email verified](../images/part61/3.png)

Alright, let's get started!

## Update database

First, we need to update the database schema. So I'm gonna open the 
`db.dbml` file inside the `doc` folder. In the definition of the "users"
table, let's add a new boolean field called "is_email_verified". It will
tell us whether the email has been verified or not. This field should
not be null, and its default value should be false.

```shell
email varchar [unique, not null]
is_email_verified bool [not null, default: false]
password_changed_at timestamptz [not null, default: '0001-01-01']
```

Having a default value is very important for a new column, because if you 
already have some existing records in the database, their values must be
filled so that the not-null constraint can be satisfied.

Next, I'm gonna add a new table called "verify_emails". This table will
store the secret code and all necessary information about the email we send
to the user. It will have an `id` column of type `bigserial`, which will
be the primary key of this table. A `username` column of type `varchar`,
which is a foreign key that references the corresponding column in the
"users" table. And it should also be not null. There's also a column to 
keep track of the user's email address. Although it seems to be duplicate
since the email is already stored in the "users" table, it actually not,
because we might allow users to change their email address in the future.
We don't need this unique constraint in this table, because we can send 
multiple verification emails to the same address for different purposes.
OK, now comes the most important field: the secret code. Its type should
be `varchar`, and it should not be null. We also need a boolean field to 
tell whether this secret code has been used or not. For security reasons,
we don't want a secret code to be reused multiple times. This field should
not be null, and its default value is false. Nexy, I'm gonna add a 
`created_at` field to this table. To keep track of the time when this 
email record is created. And finally, an expired_at field to tell when it
is expired. This is also for security reasons, because we don't want the
secret code to be valid for too long, which might increase th chance that
it can be leaked. So I'll set its default value to `now() + interval '15 
minutes'`. This will make sure that the code will only be valid for 
15 minutes after creation.

```
Table verify_emails {
  id bigserial [pk]
  username varchar [ref: U.username, not null]
  email varchar [not null]
  secret_code varchar [not null]
  is_used bool [not null, default: false]
  created_at timestamptz [not null, default: `now()`]
  expired_at timestamptz [not null, default: `now() + interval '15 minutes'`]
}
```

Alright, I think this DB design should be good enough for our feature.

Let's open the terminal and run

```shell
make db_schema
dbml2sql --postgres -o doc/schema.sql doc/db.dbml
  ERROR: You have a syntax error at "db.dbml" line 20 column 26. Expected " " or <>, >, - or < but "U" found.

  A complete log can be found in:
     /Users/quangpham/Projects/techschool/simplebank/dbml-error.log
```

to generate SQL codes for the new schema. Oops, we've got a syntax error
on line 20, let's go back to the code to fix it. OK, I see! Here, when
defining the foreign key reference I forgot this > character.

```
username varchar [ref: > U.username, not null]
```

Now, this should be fixed. Let's go back to the terminal and run 

```shell
make db_schema
dbml2sql --postgres -o doc/schema.sql doc/db.dbml
  ✔ Generated SQL dump file (PostgreSQL): schema.sql
```

again.

This time, the SQL file is generated successfully. And in VS Code, we 
will see that there are new changes in the `schema.sql` file.

```postgresql
CREATE TABLE "users" (
  "username" varchar PRIMARY KEY,
  "hashed_password" varchar NOT NULL,
  "full_name" varchar NOT NULL,
  "email" varchar UNIQUE NOT NULL,
  "is_email_verified" bool NOT NULL DEFAULT false,
  "password_changed_at" timestamptz NOT NULL DEFAULT '0001-01-01',
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "verify_emails" (
  "id" bigserial PRIMARY KEY,
  "username" varchar NOT NULL,
  "email" varchar NOT NULL,
  "secret_code" varchar NOT NULL,
  "is_used" bool NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "expired_at" timestamptz NOT NULL DEFAULT (now() + interval '15 minutes')
);
```

A new column is added to the "users" table, and there's a new statement
to create the "verify_emails" table. At the bottom of the file, there's
an "alter table" statement to add a foreign key constraint to that new
table as well.

```postgresql
ALTER TABLE "verify_emails" ADD FOREIGN KEY ("username") REFERENCES "users" ("username");
```

Now, as our project has been deployed to production, we can't just drop
the whole DB and recreate it with the new schema. So in the next step,
we'll have to write a DB migration script to update the schema with only
the new changes.

In the `README` file, at the bottom, you'll find a command that helps us
create a new DB migration file.

```shell
migrate create -ext sql -dir db/migration -seq <migration_name>
```

Since this command will be used a lot in development, I'm gonna add it
to the Makefile to make it easier to run.

Let's add a new make command called "new_migration" and paste in the
"migrate create" statement that we've just copied.

```makefile
...
migratedown1:
	migrate -path db/migration -database "$(DB_URL)" -verbose down 1

new_migration:
	migrate create -ext sql -dir db/migration -seq <migration_name>
	
...
```

Note that we need to replace this `<migration_name>` with a command-line
argument. So that we can set different names for it when running in the
terminal. It can be done easily using this `$` character, followed by the
argument's name inside a pair of brackets.

```makefile
new_migration:
	migrate create -ext sql -dir db/migration -seq $(name)
```

In this case, I'm using "name" as the argument.

Then, let's add the `new_migration` command to the PHONY list.

```makefile
.PHONY: postgres createdb dropdb migrateup migratedown migrateup1 migratedown1 new_migration db_docs db_schema sqlc test server mock proto evans redis
```

Save the file, and go back to the terminal. We can now run 
"make new_migration" and set the "name" argument to "add_verify_emails".

```shell
make new_migration name=add_verify_emails
migrate create -ext sql -dir db/migration -seq add_verify_emails
/Users/quangpham/Projects/techschool/simplebank/db/migration/000004_add_verify_emails.up.sql
/Users/quangpham/Projects/techschool/simplebank/db/migration/000004_add_verify_emails.down.sql
```

As you can see, 2 new migration files (`up` and `down`) have been created.
They both have the "add_verify_emails" prefix in their names. We can find 
these new files in the "db/migration" folder.

Now we have to fill them with the SQL codes. First, the migration `up`. 
It's pretty easy for the new table "verify_emails", because we just need
to copy the generated codes from the `schema.sql` file. We must also 
copy the statement that adds foreign key constraints to this new table.
For the new column "is_email_verified", we can also copy its code from
`schema.sql`,

```postgresql
"is_email_verified" bool NOT NULL DEFAULT false,
```

but we have to manually write the first part of the statement, which is:
`ALTER TABLE "users" ADD COLUMN`. Then paste in the rest of the column 
definition.

```postgresql
ALTER TABLE "users" ADD COLUMN "is_email_verified" bool NOT NULL DEFAULT false;
```

And that's it!

The migration `up` file is completed.

Next, the migration `down`. There's no generated code for it, so we'll have
to write everything on our own. But it's pretty simple. We just have to
reverse what we did in the migration `up`. First, `DROP TABLE 
"verify_emails"` if it exists. The `CASCADE` keyword at the end will make
sure that if there are records in other tables that reference this table,
they will all be deleted. Then, for the "is_email_verified" column, we can
use the same `ALTER TABLE "users"` statement as in the migration `up`, but
this time, we're gonna `DROP COLUMN` instead of `ADD COLUMN`. And we're
done with the migration scripts.

```postgresql
DROP TABLE IF EXISTS "verify_emails" CASCADE;

ALTER TABLE "users" DROP COLUMN "is_email_verified";
```

Let's take a look at the current state of the DB before we run them. As you
can see, the current migration version is 3.

![](../images/part61/4.png)

Now, let's run "make migrateup" in the terminal.

```shell
make migrateup
migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up
2023/02/25 11:36:26 Start buffering 4/u add_verify_emails
2023/02/25 11:36:26 Read and execute 4/u add_verify_emails
2023/02/25 11:36:26 Finished 4/u add_verify_emails (read 9.04674ms, ran 57.574636ms)
2023/02/25 11:36:26 Finished after 72.96994ms
2023/02/25 11:36:26 Closing source and database
```

It's successful. So if we refresh TablePlus, we'll see that the version
has been changed to 4.

![](../images/part61/5.png)

The new "verify_emails" table has been created.

![](../images/part61/6.png)

And in the "users" table, there's a new column: "is_email_verified", whose
value has been set to `FALSE` for all existing records.

![](../images/part61/7.png)

That's exactly what we wanted. So the migration `up` script is working
well. Now let's try running `make migratedown1` in the terminal to run
only the last migration `down` scripts.

```shell
make migratedown1
migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose down 1
2023/02/25 11:36:58 Start buffering 4/d add_verify_emails
2023/02/25 11:36:58 Read and execute 4/d add_verify_emails
2023/02/25 11:36:58 Finished 4/d add_verify_emails (read 9.216659ms, ran 24.391741ms)
2023/02/25 11:36:58 Finished after 40.384528ms
2023/02/25 11:36:58 Closing source and database
```

This time, if we refresh TablePlus, the "verify_emails" table will be gone.
And the current migration version will be set back to 3.

![](../images/part61/8.png)

The "is_email_verified" column also disappeared from the "users" table.

![](../images/part61/9.png)

So the migration down script is working well too. We can now run 

```shell
make migrateup
migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up
2023/02/25 11:37:16 Start buffering 4/u add_verify_emails
2023/02/25 11:37:16 Read and execute 4/u add_verify_emails
2023/02/25 11:37:16 Finished 4/u add_verify_emails (read 8.722009ms, ran 31.07725ms)
2023/02/25 11:37:16 Finished after 46.569695ms
2023/02/25 11:37:16 Closing source and database
```

again to have the latest version of the database before continuing with
the implementation of our feature.

Alright, now the DB schema is ready.

## Write a query to create a new verify email

Let's go back to the code and write a query to create a new verify email.
In the `db/query` folder, I'm gonna create a new file called 
`verify_email.sql`. As we're gonna generate codes with `sqlc`, I will 
write an instruction comment for it here, in `verify_email.sql`. The
name of the generated function should be `CreateVerifyEmail` and it should
return one record.

```postgresql
-- name: CreateVerifyEmail :one
```

Under this instruction, let's write our query:

```postgresql
-- name: CreateVerifyEmail :one
INSERT INTO verify_emails (
                           
)
```

Now let's see which columns we have to set in this query. There's no need
to set the "id" column because it's an auto-increment key. We'll surely
have to provide the username, email and secret code. The last 3 columns
can take their default values, so we don't need to care about them
in our `INSERT` query. Let's head back to our query and add the names of 
the 3 columns: `username`, `email` and `secret_code`, followed by the list 
of `VALUES` for them, which should be provided via 3 corresponding 
input arguments. Finally, we add `RETURNING *` keyword at the end to return
all columns of the newly inserted record.

```postgresql
-- name: CreateVerifyEmail :one
INSERT INTO verify_emails (
    username,
    email,
    secret_code
) VALUES (
    $1, $2, $3
) RETURNING *;
```

Now we can run 

```shell
make sqlc
sqlc generate
```

in the terminal to generate Golang code for this query.

Alright, now if we take a look at the `sqlc` package,

```go
CreateUser(ctx context.Context, arg CreateUserParams) (User, error)
CreateVerifyEmail(ctx context.Context, arg CreateVerifyEmailParams) (VerifyEmail, error)
DeleteAccount(ctx context.Context, id int64) error
```

we will see that a new method: `CreateVerifyEmail()` has been added to the
`Querier` interface. And its implementation has also been generated to this
`verify_emails.sql.go` file. Note that the change we've just made will
break some tests in the `api` package, because when a new method is added,
the `MockStore` doesn't implement that method yet. So we'll have to run

```shell
make mock
mockgen -package mockdb -destination db/mock/store.go github.com/techschool/simplebank/db/sqlc Store
```

in the terminal to regenerate the MockStore as well.

After doing so, all the errors will be gone.

Awesome!

## Create a new verify email record in the database

Now it's time to get back to the task processor, and use the method we've
just added to create a new verify email record in the database. I'm gonna
call `processor.store.CreateVerifyEmail`, pass in the context and a 
`db.CreateVerifyEmailParams` object. We will have to provide 3 input 
fields: the `Username` will be set to `user.Username`, the `Email` will
be set to `user.Email` and for the `SecretCode`, I'm gonna use 
`util.RandomString()` function to generate a random code of 32 characters.

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
    
    processor.store.CreateVerifyEmail(ctx, db.CreateVerifyEmailParams{
        Username:   user.Username,
        Email:      user.Email,
        SecretCode: util.RandomString(32),
    })
    
    ...
}
```

This is a function we wrote a long time ago. The string that it generates
will contain only lowercase English letters.

```go
func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet)

	for i := 0; i < n; i++ {
		c := alphabet[rand.Intn(k)]
		sb.WriteByte(c)
	}

	return sb.String()
}
```
You can always write a different function that's suitable to your system, 
but make sure the secret code is long enough to avoid brute-force attacks.
OK, now, this function will return a `verifyEmail` object and an error.
If the error is not `nil`, we simply wrap and return it with a message 
saying "failed to create verify email".

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

    verifyEmail, err := processor.store.CreateVerifyEmail(ctx, db.CreateVerifyEmailParams{
        Username:   user.Username,
        Email:      user.Email,
        SecretCode: util.RandomString(32),
    })
    if err != nil {
        return fmt.Errorf("failed to create verify email: %w", err)
    }
	
    ...
}
```

Otherwise, the email record has been created successfully, then, it's time
to send a real email to the user.

## Send a real email to the user

I'm gonna use the code from the `mail` package that we wrote in the last
lecture. So, in the `RedisTaskProcessor` struct, I'm gonna add a new `mailer`
of type `EmailSender` to this struct.

```go
type RedisTaskProcessor struct {
	server *asynq.Server
	store  db.Store
	mailer mail.EmailSender
}
```

As you can see, it's the interface that we wrote in the previous lecture,
which provides this method

```go
type EmailSender interface {
    SendEmail(
        subject string,
        content string,
        to []string,
        cc []string,
        bcc []string,
        attachFiles []string,
    ) error
}
```

for sending emails.

Back to the task processor, we have to add a new mailer argument to 
this `NewRedisTaskProcessor` function, and then add it here as well, 
when initializing the `RedisTaskProcessor`.

```go
func NewRedisTaskProcessor(redisOpt asynq.RedisClientOpt, store db.Store, mailer mail.EmailSender) TaskProcessor {
	...

	return &RedisTaskProcessor{
		server: server,
		store:  store,
		mailer: mailer,
	}
}
```

Alright, now with the `mailer` ready inside the processor, we can go back
to the send verify email task and use it. After creating the email record 
in the database, I'm gonna call `processor.mailer.SendEmail()`. We have
to pass in the subject, the content, and the address, where the email 
should be sent to. We don't have any `cc`, `bcc` or attach files in this 
case, so I use `nil` for those fields.

```go
func (processor RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
	...
    if err != nil {
        return fmt.Errorf("failed to create verify email: %w", err)
    }
    
    processor.mailer.SendEmail(subject, content, to, nil, nil, nil)
    
    log.Info().Str("type", task.Type()).Bytes("payload", task.Payload()).
        Str("email", user.Email).Msg("processed task")
    return nil
}
```

Now, let's define the subject of the email, let's say "Welcome to Simple 
Bank". Then, the content will be a multiple string. I want to say "Hello",
followed by the name of the user, so let's add an `fmt.Sprintf()` statement
here to format the string, and pass in `user.FullName` as one of its 
input argument. Next, I'm gonna add a line break tag (`<br>`) here 
to create a new line. Then, let's say "Thank you for registering with us!",
and a line break. Now comes the most important part of the message. We'll
provide a link for the user to click on to verify their email address.
Here I use the `<a href>` tag to specify the verification URL. So we'll
need to declare this new variable before writing the content. Normally
it should point to a front-end page, whose responsibility is to parse the
input argument from the URL, and call the corresponding API on the backend
side for verification. For this demo, I'll set it to 
`http://simple-bank.org` followed by 2 query parameters: the first param 
is the id of the verification email record, and the second one is the 
secret code of that email. We can get their values from the `verifyEmail`
object we created above.

```go
func (processor RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
    ...
	if err != nil {
        return fmt.Errorf("failed to create verify email: %w", err)
    }
    
    subject := "Welcome to Simple Bank"
    verifyUrl := fmt.Sprintf("http://simple-bank.org?id=%d&secret_code=%s",
        verifyEmail.ID, verifyEmail.SecretCode)
    content := fmt.Sprintf(`Hello %s,<br>
    Thank you for registering with us!<br>
    Please <a href="%s">click here</a> to verify your email address.<br>
    `, user.FullName, verifyUrl)
    processor.mailer.SendEmail(subject, content, to, nil, nil, nil)
    
    ...
}
```

OK, next, we have to declare a variable that stores the address to which
we want to send the email. Note that the `SendEmail()` function can return
an error. So we need to check whether that error is `nil` or not. If it is
not `nil`, we'll just wrap and return it with this message: "failed to 
send verify email".

```go
func (processor RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
    ...
    content := fmt.Sprintf(`Hello %s,<br>
    Thank you for registering with us!<br>
    Please <a href="%s">click here</a> to verify your email address.<br>
    `, user.FullName, verifyUrl)
    to := []string{user.Email}
    err = processor.mailer.SendEmail(subject, content, to, nil, nil, nil)
    if err != nil {
        return fmt.Errorf("failed to send verify email: %w", err)
    }
    
    ...
}
```

And that's it! We're done with the implementation of the send verification
email task.

Now, the last thing we need to do is update the `main.go` file, because
there's an error here.

```go
func runTaskProcessor(redisOpt asynq.RedisClientOpt, store db.Store) {
	taskProcessor := worker.NewRedisTaskProcessor(redisOpt, store)
	...
}
```

Since we've added a new mailer object to the `NewRedisTaskProcessor()` 
function, so at the beginning of the `runTaskProcessor()` function, I'm
gonna call `mail.NewGmailSender()`. It requires 3 input arguments:
`name`, `fromEmailAddress` and `fromEmailPassword`, which we can get
from the environment variables via the config object that we updated in 
the previous lecture. So, let's add the `config` object as 1 input 
argument of this function. And we can pass in its value when we call
`runTaskProcessor` in the `main()` function.

```go
func main() {
    ...
    taskDistributor := worker.NewRedisTaskDistributor(redisOpt)
    go runTaskProcessor(config, redisOpt, store)
    go runGatewayServer(config, store, taskDistributor)
    runGrpcServer(config, store, taskDistributor)
    //runGinServer(config, store)
}
```

OK, now with the `config` object, we can get the values of the email 
sender's name, email sender's address, and email sender's password. 
They're already provided in the `app.env` file. So we can use them to 
create a `NewGmailSender` mailer. And use this mailer to create the
`RedisTaskProcessor`.

```go
func runTaskProcessor(config util.Config, redisOpt asynq.RedisClientOpt, store db.Store) {
    mailer := mail.NewGmailSender(config.EmailSenderName, config.EmailSenderAddress, config.EmailSenderPassword)
    taskProcessor := worker.NewRedisTaskProcessor(redisOpt, store, mailer)
    ...
}
```

And that's basically it! The send verification email task is now ready to
be tested.

Let's run 

```shell
make server
go run main.go
11:52AM INF db migrated successfully
11:52AM INF start gRPC server at [::]:9090
11:52AM INF start task processor
11:52AM INF Starting processing
11:52AM INF start HTTP gateway server at [::]:8080
```

in the terminal to start the server.

Then in Postman, I'm gonna change the body of the `create_user` request
a bit, the username is gonna be "techschool", the full name is also 
"Tech School", and finally, the email of this user is gonna be 
"techschool.guru@gmail.com".

OK, let's send this request to the server.

![](../images/part61/10.png)

It's successful. A new user has been created. And if we look at the server's
logs, we will see that a new task to send a verification email has been
enqueued.

```shell
11:53PM INF enqueued task max_retry=10 payload="{\"username\":\"techschool\"}" queue=critical type=task:send_verify_email
11:53PM INF received an HTTP request duration=101.511666 method=POST path=/v1/create_user protocol=http status_code=200 status_text=OK
```

After a few seconds, the task will be processed.

```shell
9:39PM INF processed task email=techschool.guru@gmail.com payload="{\"username\":\"techschool\"}" type=task:send_verify_email
```

This means that the verification email has been sent to the registered
email address.

So let's open Tech School's gmail inbox and refresh it.

And voilà, we have a new email here with the title: "Welcome to Simple
Bank".

![](../images/part61/11.png)

Inside, the content of this email is exactly as we wrote before:
"Hello Tech School, Thank you for registering with us! Please click
here to verify your email address."

![](../images/part61/12.png)

And if we click on the link, it will bring us to `simple-bank.org`.

Of course, this page doesn't exist yet, but we can find in the URL the 
2 parameters: `id` and `secret_code`.

![](../images/part61/13.png)

In fact, I think we should change the URL a bit, to make it point to a 
specific page instead of the home page, for example, we can add a 
`/verify_email` path to the URL like this.

```go
func (processor RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
    ...
    
    subject := "Welcome to Simple Bank"
    verifyUrl := fmt.Sprintf("http://simple-bank.org/verify_email?id=%d&secret_code=%s",
        verifyEmail.ID, verifyEmail.SecretCode)
    content := fmt.Sprintf(`Hello %s,<br>
    Thank you for registering with us!<br>
    Please <a href="%s">click here</a> to verify your email address.<br>
    `, user.FullName, verifyUrl)
    ...
}
```

And now, if we look at the database, we will see a new `verify_email` 
record here, with the same `id` and `secret_code` as in the URL query
parameters.

![](../images/part61/14.png)

Also, ib the "users" table, there's a new record for "tech school" user,
whose email is not yet verified.

![](../images/part61/15.png)

We will learn how to write a new API for email verification in the next 
lecture.

Until then, you can try to do it on your own to see how it goes.

And that brings us to the end of this lecture. I hope you find it
interesting and useful. Thanks a lot for watching, happy learning, and
see you in the next one!