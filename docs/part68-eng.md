# Docker compose: port + volume mapping

[Original video](https://www.youtube.com/watch?v=nJBT5SKENAw)

Hello guys, welcome to the backend master class. Recently we introduced 
Redis to the project to support running async workers. However, we didn't
add it to the `docker compose` yet, so that's why when we run 

```shell
docker compose up
```

there are some errors saying Redis connection refused.

![](../images/part68/1.png)

Today I'm gonna show you how to fix this, and we'll also explore some more 
features of `docker` and `docker compose`.

## Upgrading Go to version 1.21

But first, I want to tell you that I've just upgraded Go to the latest 
version 1.21, so if you have an older version, you should do that as well.
After downloading and installing the latest Go version, you should update
the Docker file with the Golang image of the same version.

```go
# Builds stage
FROM golang:1.21-alpine3.18 AS builder
WORKDIR /app
```

In my case, I use `golang:1.21-alpine3.18`, since this is the version
listed on [Docker Hub](https://hub.docker.com/_/golang). You must also 
update the version in `go.mod` file.

```go
module github.com/techschool/simplebank

go 1.21
```

Then run 

```shell
go mod tidy
```

in the terminal to update the `go.sum` file. And don't forget to update
the go-version in your GitHub workflow.

```yaml
...
    steps:

      - name: Set up Go 1.x
        uses: actions/setup-go@v2
        with:
          go-version: ^1.21
        id: go
...
```

Alright, now let's get back to the `docker-compose.yaml` file to add Redis.

## Adding Redis to `docker-compose.yaml` file

It's pretty similar to what we've done for Postgres. We just need to insert
a new service named "redis" here.

```yaml
services:
  postgres:
    image: postgres:12-alpine
    environment:
      - POSTGRES_USER=root
      - POSTGRES_PASSWORD=secret
      - POSTGRES_DB=simple_bank
  redis:
    image: redis:7-alpine
  api:
  ...
    depends_on:
      - postgres
      - redis
```

And specify its Docker image name, which we can easily find on [Docker Hub](https://hub.docker.com/_/redis).
I'm gonna use version `7-alpine` since it is a very light-weight image,
so, the name of the image is gonna be `redis:7-alpine`. We should also add
"redis" to the "depends_on" list of the "api" service. This is to control
the start-up order of the services, since we want both Postgres and Redis
to start before the API server. OK, now let's run

```shell
docker compose up
```

in the terminal to see what happens.

![](../images/part68/2.png)

The error is still there. That's because the API server is trying to 
connect to Redis via a local IP address, but since we're running Redis
inside `docker compose` container, any internal service should connect to it
via its service name instead, just like how we connect to Postgres server
via its name here.

```yaml
environment:
  - DB_SOURCE=postgresql://root:secret@postgres:5432/simple_bank?sslmode=disable
```

The Redis address is defined via an environment variable, so let's copy
it from the `app.env` file, and paste it to the environment section of the
`api` service.

```yaml
environment:
  - DB_SOURCE=postgresql://root:secret@postgres:5432/simple_bank?sslmode=disable
  - REDIS_ADDRESS=0.0.0.0:6379
```

Then I'm gonna change this IP address to the service name "redis", that 
should solve our problem.

```yaml
environment:
  - DB_SOURCE=postgresql://root:secret@postgres:5432/simple_bank?sslmode=disable
  - REDIS_ADDRESS=redis:6379
```


Let's re-run 

```shell
docker compose up
```

to confirm.

![](../images/part68/3.png)

And voilà, this time the server is up and running, serving both gRPC and
HTTP requests on these 2 different ports.

Now let's head over to Postman to test them.

First, I'm gonna send this HTTP request to create a new user.

![](../images/part68/4.png)

It worked! We've got a `200 OK` status code, and the user `techschool` 
has been created.

```shell
7:14AM INF received a HTTP request duration=90.678899 method=POST 
path=/v1/create_user protocol=http status_code=200 status_text=OK 
```

Next, let's try calling a gRPC method to login user.

But we've got an issue here that says "Could not load server reflection".

![](../images/part68/5.png)

And it stays the same even when I click "Try Again".

This means Postman couldn't connect to the gRPC server to get the 
reflection. In the logs, we clearly see that the gRPC server has started
on port 9090.

```shell
7:14AM INF start gRPC server at [::]:9090
```

So what can be the problem?

And by the way, don't worry about this process task failed error.

![](../images/part68/6.png)

This error only shows up here because I deleted the Gmail app password 
that we used to send emails in lecture 59.

The real problem that caused Postman not able to connect to the gRPC
server is that we haven't added the port mapping for the gRPC port 9090
to the `docker-compose.yaml` file.

```yaml
ports:
  - "8080:8080"
  - "9090:9090"
```

Once we've done that, and restart the `docker compose`,

```shell
docker-compose down
docker-compose up
```

Postman should be able to talk to the gRPC server via that port.

So now I'm gonna change this username to `techschool`, and invoke the Login
User method.

![](../images/part68/7.png)

The request has been sent, but we've got status `5 NOT_FOUND`. Let's open
the database using TablePlus to see why the user is not there.

![](../images/part68/8.png)

Oops, we couldn't connect to the database. OK, so it's the same issue: 
we haven't added the port mapping for Postgres yet. Let's add `5432:5432`
to the ports list, and restart the `docker-compose` again.

```yaml
  postgres:
    image: postgres:12-alpine
    environment:
      - POSTGRES_USER=root
      - POSTGRES_PASSWORD=secret
      - POSTGRES_DB=simple_bank
    ports:
      - "5432:5432"
```

```shell
docker-compose down
docker-compose up
```

Now we should be able to connect to the database from TablePlus.

![](../images/part68/9.png)

Alright, so the `users` table is indeed empty.

![](../images/part68/10.png)

Why? Didn't we just create a new user before? Yes, but we also run 
`docker compose down`, so all the data of the Postgres service is gone 
together with it. We'll learn how to keep that data in a moment.

For now, let's try to continue testing the gRPC API. I'm gonna create a
new gRPC request named Create User RPC. Enter the URL of the gRPC in 
this box,

![](../images/part68/11.png)

and select the corresponding method here.

![](../images/part68/12.png)

Let's copy the create user payload from the HTTP call,

```json
{
    "username": "techschool",
    "password": "secret",
    "full_name": "Tech School",
    "email": "techschool.guru@gmail.com"
}
```

paste in the message, and Invoke the CreateUser RPC.

![](../images/part68/13.png)

OK, the user is now successfully created.

So if we invoke the LoginUser again,

![](../images/part68/14.png)

the request will be successful this time.

Alright, so now both the HTTP and gRPC servers are working properly.

Let's deal with the Postgres data lost issue. To fix it, we'll store the
data inside a docker volume, so that it won't get destroyed when we run
`docker-compose down`. [This is an example](https://docs.docker.com/compose/compose-file/compose-file-v3/#volume-configuration-reference) of 
how we can define the volume for a db service.

```yaml
version: "3.8"

services:
  db:
    image: db
    volumes:
      - data-volume:/var/lib/db
  backup:
    image: backup-service
    volumes:
      - data-volume:/var/lib/backup/data

volumes:
  data-volume:
```

In this example, `data-volume` is the name of the docker volume, and it 
will be mapped to a folder inside the db service's container. We'll have
to figure out the exact location of this folder in Postgres' image. So in
the Docker Hub page of [Postgres](https://hub.docker.com/_/postgres), let's 
search for "volume".

![](../images/part68/15.png)

Here it is, by default `/var/lib/postgresql/data` is the location we're
looking for.

So let's copy this volume setting, and paste it to our 
`docker-compose.yaml` file, inside the `postgres` service.

```yaml
  postgres:
    image: postgres:12-alpine
    environment:
      - POSTGRES_USER=root
      - POSTGRES_PASSWORD=secret
      - POSTGRES_DB=simple_bank
    ports:
      - "5432:5432"
    volumes:
      - data-volume:/var/lib/db
```

Then I'm gonna replace this `/var/lib/db` folder with the right one for 
Postgres.

```yaml
  postgres:
    image: postgres:12-alpine
    environment:
      - POSTGRES_USER=root
      - POSTGRES_PASSWORD=secret
      - POSTGRES_DB=simple_bank
    ports:
      - "5432:5432"
    volumes:
      - data-volume:/var/lib/postgresql/data
```

One more thing we must do is, declaring the name of the volume at the end 
of the `docker-compose.yaml` file. This section is where you can specify
all volumes you used in your services.

```yaml
version: "3.9"
services:
  postgres:
    image: postgres:12-alpine
    environment:
      - POSTGRES_USER=root
      - POSTGRES_PASSWORD=secret
      - POSTGRES_DB=simple_bank
    ports:
      - "5432:5432"
    volumes:
      - data-volume:/var/lib/postgresql/data
  redis:
    image: redis:7-alpine
  api:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - DB_SOURCE=postgresql://root:secret@postgres:5432/simple_bank?sslmode=disable
      - REDIS_ADDRESS=0.0.0.0:6379
    depends_on:
      - postgres
      - redis
    entrypoint: [ "/app/wait.sh", "postgres:5432", "--", "/app/start.sh" ]
    command: [ "/app/main" ]
volumes:
  data-volume:
```

In our case, we just have 1 volume for now, but you can also add 1 more
volume for Redis if you want.

Alright, now let's run 

```shell
docker-compose down
```

and then 

```shell
docker-compose up
```

again.

For the moment, the stored data will still be gone, since we didn't 
have volume in the previous run.

![](../images/part68/16.png)

I'm gonna invoke the Create User method to insert a new record to the DB.

![](../images/part68/17.png)

Now, let's see if the data will be kept when we restart `docker-compose`.

```shell
docker-compose down
docker-compose up
```

This time, you can see that the created user still exists in the database.

![](../images/part68/18.png)

And we can still login successfully. So it worked! Awesome.

Now, if you want to go deeper, we can run 

```shell
docker volume ls
```

to list all existing volumes. And you can see the `simplebank_data-volume`
at the bottom.

![](../images/part68/19.png)

Let's see what happens when we run 

```shell
docker compose down
```

and list the volumes again.

![](../images/part68/20.png)

Here we go, the `simplebank_data-volume` is still there. So when we run

```shell
docker compose up
```

this volume will be attached to the new Postgres container, and therefore,
the existing data won't be lost.

But if you run 

```shell
docker volume rm simplebank_data-volume
simplebank_data-volume
```

and pass in the name of the volume, it will be completely removed,

![](../images/part68/21.png)

So now, if we run

```shell
docker compose up
```

a new volume will be created for the Postgres container.

And as a result, all existing data of the previous run will be gone.

![](../images/part68/22.png)

And that's all I wanted to share with you in this video. I hope it was 
interesting and useful for you.

Thanks a lot for watching! Happy learning, and see you in the next 
lecture. 