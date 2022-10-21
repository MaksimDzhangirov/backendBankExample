# Run DB migrations directly inside Golang code

[Original video](https://www.youtube.com/watch?v=TG43cMpaxlI)

Hello everyone, welcome to the backend master class. Today we will get
back to an old topic: running DB migrations. But this time, we're
gonna do it differently than what we did before. If you still remember, 
in lecture 25, we've learned how to use `docker-compose` to run both
the database and the backend server in 1 single command. And in that
lecture, we had to download the `migrate` binary to the Docker image,
then used it to run the DB migrations inside the `start.sh` file, right
before starting the server. It was working well at that time. However,
I recently found out that this approach is no longer working as expected.

## Problem while running DB migrations

Let me show you by running 

```shell
docker-compose up
```

in the terminal.

![](../images/part48/1.png)

As you can see, the `simple-bank` API service couldn't start this time,
and the error message is saying that it cannot connect to the database 
on this `localhost` IP address. This is strange because in the 
`docker-compose.yaml` file, we've made sure to replace the `DB_SOURCE`
environment variable with a URL that points to the Postgres container.
So it should connect to the Postgres container instead of `localhost`,
right?

Well, yes, but let's take look at the `start.sh` file. Here, right before
running the migration, there's a `source` command that reloads the 
environment variables from the `app.env` file. So this is the reason why
the `DB_SOURCE` variable's value we set in the `docker-compose` file has
been replaced by the one in the `app.env` file, which is pointing to 
`localhost` instead of the Postgres container as we wanted. But why?

It was working well in lecture 25, wasn't it? What has changed since that
lecture? Well, if we look into the Git blame history of line 

```shell
source /app/app.env
```

by following this link, we will find out that the `source` command was
only added later.

![](../images/part48/2.png)

In the commit histories of our GitHub repo, we see that this commit is 
pushed to GitHub when we're trying to deploy our app to production. And if
I remember correctly, this change was made in lecture 29, since we wanted
the migration to connect to the production database URL, which was stored
inside the `app.env` file instead of a real environment variable.

![](../images/part48/3.png)

![](../images/part48/4.png)

That's why the `migrate` command can't pick up the right value of the 
`DB_SOURCE`, and we must add the `source` command to set the environment
variables to the ones inside the `app.env` file.

But it's not good when a change in production breaks our development 
environment, right?

So how can we fix it?

## Fix the problem

Well, the problem comes from the `source` command, so if we can get rid of
it but still get the migration to run in production, then it would be
perfect. Note that once we start the Golang API server, it will load
all the variables from the `app.env` file,

```go
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}
```

So if we can run the migration at this point, it will have the correct 
value of the `DB_SOURCE`.

```go
func main() {
    ...
	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	// run db migration
	
	store := db.NewStore(conn)
	...
}
```

This means we have to run the DB migration inside our Golang server code.
And that's exactly what we're gonna do in this video.

Alright, first let's delete this chunk of codes that runs the migration 
inside the `start.sh` file.

```shell
echo "run db migration"
source /app/app.env
/app/migrate -path /app/migration -database "$DB_SOURCE" -verbose
```

Since we're gonna run DB migrations from our Golang codes, there's no need 
to download the `migrate` binary to the Docker image anymore, so I'm 
gonna delete these 2 commands from the `Dockerfile`, 

```dockerfile
RUN apk add curl
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.14.1/migrate.linux-amd64.tar.gz | tar xvz
```

and this command, which copies the `migrate` binary to the final image, as 
well.

```dockerfile
COPY --from=builder /app/migrate.linux-amd64 /usr/bin/migrate
```

So now, our `Dockerfile` is much simpler than before. We can even remove 
the `start.sh` entry point if we want, but I don't want to make it 
complicated by changing too many things at once, and besides, it can be
a good reference for how to use Docker entrypoint. So I'm gonna keep it
in the code base for now.

OK, now let's go back to the `main.go` file and add some codes to run the
DB migrations!

If we look at the documentation of `golang-migrate` on its GiHub repo, 
we will find a [section](https://github.com/golang-migrate/migrate#use-in-your-go-project)
that shows us how to run DB migration using Golang.

Basically, we will have to create a new `migrate` object, where we must 
pass in the location of the migration files, as well as the URL of the
target database server. But first, let's copy this import `migrate` package
statement and paste it to our `main.go` file.

```go
"github.com/golang-migrate/migrate/v4"
```

Then, I'm gonna declare a new function called `runDBMigration()`. This 
function will take a migration URL, and a DB source string as input.

```go
func runDBMigration(migrationURL string, dbSource string) {
	
}
```

OK, now we can call `runDBMigraion()` here,

```go
func main() {
	...
	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	runDBMigration()

	store := db.NewStore(conn)
	...
}
```

right before starting the servers, and after we've loaded the environment 
variables into the `config` object. Note that the migration URL should
point to the location of the migration files, which can be either on the
local machine or a remote host. In our case, it will be the `migration` 
folder inside the Docker container. Because in the `Dockerfile`, we've
copied the whole content of the `db/migration` folder to the Docker
image.

```dockerfile
COPY db/migration ./migration
```

To make it more flexible, I will define this location as an environment 
variable. So, inside the `app.env` file, let's add a `MIGRATION_URL`
variable. And as you've seen before in the documentation, they use the 
"github" scheme to point to a remote location on GitHub. But in our case,
we're gonna use a local file system instead, so the scheme should be 
"file", and the location is `db/migration`.

```
MIGRATION_URL=file://db/migration
```

Alright, now we have to add this new variable to the `Config` struct, so
I'm gonna duplicate this line,

```go
DBSource             string        `mapstructure:"DB_SOURCE"`
```

and change the `mapstructure` tag, as well as the field name to 
Migration Url.

```go
MigrationURL         string        `mapstructure:"MIGRATION_URL"`
```

OK, then let's get back to the `main.go` file. Here, we will pass the
`config.MigrationUrl` and the `config.DBSource` to the `runDBMigration()`
function.

```go
runDBMigration(config.MigrationURL, config.DBSource)
```

Then in this function, we will call `migrate.New()` function, and pass in 
those 2 input parameters.

```go
func runDBMigration(migrationURL string, dbSource string) {
	migrate.New(migrationURL, dbSource)
}
```

Now if we try to see the implementation of this function, we're not seeing
any suggestions from Visual Studio Code. That's because I haven't installed
this package

```go
"github.com/golang-migrate/migrate/v4"
```

on my local machine yet. So there's a warning line here under the import
migrate statement.

![](../images/part48/5.png)

To fix this, let's open the terminal, stop the server, and run 

```shell
go mod tidy
```

Now the warning line is gone, and we can see the documentation of the 
`migrate.New()` function, as you can see, this function will return a 
migration object and an error.

![](../images/part48/6.png)

So we should check if the error is not `nil`. In that case, we will write
a fatal log: "cannot create a new migrate instance". Otherwise, we will 
call migration.Up() to run all the migrations `up` files.

```go
func runDBMigration(migrationURL string, dbSource string) {
    migration, err := migrate.New(migrationURL, dbSource)
    if err != nil {
        log.Fatal("cannot create a new migrate instance")
    }
    
    migration.Up()
}
```

This function will also return an error, so we have to check if it is `nil`
or not. If the error is not `nil`, we're gonna write another fatal log:
"failed to run migrate up".

```go
func runDBMigration(migrationURL string, dbSource string) {
	...
	if err = migration.Up(); err != nil {
		log.Fatal("failed to run migrate up:", err)
	}
}
```

And add the original error at the end of the message. By the way, let's add
the original error to the end of the previous fatal log as well.

```go
log.Fatal("cannot create a new migrate instance:", err)
```

OK, then at the bottom of the function, when no errors occur, we will
write an info log with a message saying: "db migrated successfully".

```go
log.Println("db migrated successfully")
```

And that's basically it.

Let's test it out!

In the terminal, I'm gonna check if the Postgres container is running or 
not.

```shell
docker ps
```

It's not running yet, so let's run

```shell
docker start postgres12
```

Then now, when we run

```shell
make server
```

we expect to see the DB migrations running.

![](../images/part48/7.png)

And yes, it seems the server is trying to do so, but it cannot create a 
new `migrate` instance because of this error: "unknown driver file 
(forgotten import?)".

So let's check their documentation again.

Here, in this example, 

```go
import (
    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/github"
)

func main() {
    m, err := migrate.New(
        "github://mattes:personal-access-token@mattes/migrate_test",
        "postgres://localhost:5432/database?sslmode=enable")
    m.Steps(2)
}
```

we can see that they have a statement to import the `source/github` 
subpackage of the `migrate` module.

So let's copy this statement, paste it to the import list of our 
`main.go` file, and change the subpackage name from "github" to 
"file"

```go
_ "github.com/golang-migrate/migrate/v4/source/file"
```

since our migration source is from the local file system.

OK, done!

Let's try to run

```shell
make server
```

again in the terminal.

![](../images/part48/8.png)

We still got an error, but it's a different one this time: "unknown driver
postgresql (forgotten import?)".

Look at the doc again, we will see that we must add one more blank import
that points to the `database/postgres` subpackage of the `migrate` module.
So let's copy and paste it to our `main.go` file, just like before.

```go
_ "github.com/golang-migrate/migrate/v4/database/postgres"
```

OK, now I'm pretty sure it should fix the problem.

Let's try to run the server again in the terminal!

![](../images/part48/9.png)

Oh no, it still failed! But luckily, this time, we've got a different 
error than before. It says: "failed to run migrate up: no change". So
it already ran past the new migrate step, and only fail when running
`migration.Up()`. And actually, it's not a real failure, since the 
error is saying "no change". This means, that there are no new changes
in the DB schema. I don't know why it's returning an error in this case,
but since the package has already been implemented that way, let's 
accept it for now, as we can easily deal with this by checking that error
is not `migrate.ErrNoChange` here,

```go
func runDBMigration(migrationURL string, dbSource string) {
	...
	if err = migration.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal("failed to run migrate up:", err)
	}
	...
}
```

before writing the fatal log.

Alright, let's rerun the server one more time!

Yee, this time, the db is migrated successfully, and both the gRPC & HTTP
servers are up and running, with no problems.

![](../images/part48/10.png)

Awesome!

Now, before we finish, I want to make sure that it's really working well
if we run everything from scratch using `docker-compose`.

So first, let's stop the `postgress12` container,

```shell
docker stop postgres12
```

run 

```shell
docker-compose down
```

to remove all existing services.

Check the existing Docker images,

```shell
docker images
```

and remove the old `simplebank_api` image.

```shell
docker rmi simplebank_api
```

OK, now everything is clean, I'm gonna run

```shell
docker compose up
```

It's gonna rebuild the Docker image, then start the Postgres database,
and then try to start the `simplebank-api`.

![](../images/part48/11.png)

But wait, it failed with a message saying: "open app/db/migration: no such
file or directory". This means that `golang-migrate` cannot find the 
migration files inside the container.

So what happened? Let's check the `Dockerfile`.

```dockerfile
# Builds stage
FROM golang:1.16-alpine3.13 AS builder
WORKDIR /app
COPY . .
RUN go build -o main main.go

# Run stage
FROM alpine3.13
WORKDIR /app
COPY --from=builder /app/main .
COPY app.env .
COPY start.sh .
COPY wait-for.sh .
COPY db/migration ./migration

EXPOSE 8080
CMD ["/app/main"]
ENTRYPOINT ["/app/start.sh"]
```

OK, can you spot my bug? Well, it's pretty easy to see, right? Here,

```dockerfile
COPY db/migration ./migration
```

we're copy `db/migration` folder to the `migration` folder inside the image
while the target folder should be `db/migration` instead.

```dockerfile
COPY db/migration ./db/migration
```

With this fixed, I'm 100% sure that it will work well this time.

Let's give it a try!

In the terminal, I'm gonna run

```shell
docker compose down
```

then remove the current `simplebank_api` image,

```shell
docker rmi simplebank_api
```

and run 

```shell
docker-compose up
```

again.

![](../images/part48/12.png)

OK, this time, the db has been migrated successfully, and the servers are 
up and running as we expected. Excellent!

We can try to send the Create User request using Postman.

![](../images/part48/13.png)

![](../images/part48/14.png)

It is indeed successful.

And that wraps up today's lecture about running DB migration using Golang.

I hope it was interesting and useful for you! Thanks a lot for watching!
Happy learning, and see you in the next lecture!
