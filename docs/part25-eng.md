# How to write Docker-compose file and control service start-up orders with wait-for.sh

[Original video](https://www.youtube.com/watch?v=VcFnqQarpjI)

Hello everyone! Welcome back to the backend master class!

In the [last lecture](part24-eng.md), we've learned how to use Docker 
network to connect 2 stand-alone Docker containers by names. Today I'm gonna 
show you how to use `docker-compose`  to automatically setup all services in 
the same Docker network and launch them all at once with a single command.

OK, let's start!

## How to create Docker-compose file

First, I'm gonna create a new file: `docker-compose.yaml` at the root of our
project. There are a lot of things you can config with a `docker-compose` 
file. You can read all about them in the documentation by going to 
[docs.docker.com](https://docs.docker.com), open `Reference`, `Compose file 
reference`, `version 3`. On the right-hand side menu you can see a list of
different syntaxes.

In this video, I will show you some of the most important one that we 
usually use. Here in the center if the page, we can see an example `compose`
file.

![](../images/part25/1.png)

Basically you define the `docker-compose` version, then a list of services
that you want to launch together. In this example, there's a `redis` 
service, a `postgres` DB service and some other web services for voting and 
computing the result.

Now I'm just gonna copy the first 2 lines of this example, and paste them
to our `docker-compose.yaml` file. Note that a `yaml` file is very sensitive
to indentation.

```yaml
version: "3.9"
services:
```

At the moment, the file is using 4 spaces indentation, but I just want to
use 2 spaces instead. So I'm gonna click on this button

![](../images/part25/2.png)

select `Indent using spaces`,

![](../images/part25/3.png)

and choose 2 as the tab size.

![](../images/part25/4.png)

OK, now if we press `Enter`, we can see that it is nicely indented with 2 
spaces. Next we have to declare a list of services we want to launch. The
first service should be `postgres` db. For this service, we've gonna use a
prebuild Docker image. Hence, the `image` keyword, followed by the `name` 
and `tag` of the image, which is `postgres:12-alpine` in this case. Now 
we will use the `environment` keyword to specify some environment 
variables for username, password and db name just like what we did
in the GitHub CI workflow.

```yaml
  postgres:
    image: postgres:12-alpine
    environment:
```

First let's copy the `POSTGRES_USER` variable. The syntax is a bit 
different, we have to use the equal operator to assign the value. 
Similarly, I gonna copy the POSTGRES_PASSWORD variable and finally the
POSTGRES_DB variable.

```yaml
    environment:
      - POSTGRES_USER=root
      - POSTGRES_PASSWORD=secret
      - POSTGRES_DB=simple_bank
```

Alright, next step, we will declare the `api` service, which will serve all
API requests to our simple bank. For this service, we should build its 
image from the golang source, so under the `build` keyword, we specify
the `context` to build the image:

```yaml
api:
    build:
      context: .
```

This dot means the current root folder. Then we use dockerfile keyword to
tell `docker-compose` where to find the Docker file to build the image. In
this case, it's just the Dockerfile at the root of the project. Next, we
should publish the port `8080` to the host machine so that we can call the 
simple bank API from outside of the container.

```yaml
  api:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
```

One of the most important thing we must do is to tell the `api` service how 
to connect to the `postgres` service. In order to do that, we will set 1 
environment variable: `DB_SOURCE`. As we've seen in the previous lecture, 
setting this environment variable will override the value we declare in the
`app.env` file and since all services in this `docker-compose` file will
run on the same network, they can communicate with each other via name.
Therefore, here, in this URL `DB_SOURCE=postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable`, 
instead of `localhost`, we will use the name of the `postgres` service.

```yaml
api:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      - DB_SOURCE=postgresql://root:secret@postgres:5432/simple_bank?sslmode=disable
```

And that's basically it! The docker-compose file is completed. Let's try
to run it!

If you have the latest `Docker CLI` on your machine, all you have to do is
to run `docker-compose up`

```shell
docker-compose up
```

Then Docker-compose will automatically search for the `docker-compose.yaml`
file in the current folder and run it for you. As you can see here, before
running the service, it has to build the Docker image for our simple 
bank API service first. Then after the image is successfully built, 
`docker-compose` will start both the `postgres` and the `api` service at
once.

![](../images/part25/5.png)

In the logs, we can see whether it comes from which service by looking at
the prefix of the line. It is either `postgres_1` or `api_1` in this case.
Now if in another terminal tab, we run 

```shell
docker images
```

we can see a new `simplebank_api` image.

![](../images/part25/6.png)

Basically, the image name is prefixed with `simplebank`, which is the name
of the folder containing our docker file. And its suffix is the name of
the service itself, which is `api`.

Now let's run

```shell
docker ps 
```

to see all running services. There are 2 services: `simple_bank_postgres_1`
and `simple_bank_api_1`. Both of them have the `simple_bank` prefix 
followed by the name of the service.

![](../images/part25/7.png)

Now if we go back to the `docker-compose` tab, and scroll up to the top, we
can see clearly here that a new network `simplebank_default` is created
before the 2 service containers are created.

![](../images/part25/8.png)

Now if we try to inspect the `simplebank_default` network, we can see that
the 2 service containers are actually running on this same network.

![](../images/part25/9.png)

That's why they can discover each other by names.

Alright, now it's time to send some real API requests to see if the service
is working well or not. As the new database is completely empty, I'm
just gonna send the create user API.

![](../images/part25/10.png)

Oops, we've got `500 Internal Server Error`, and the reason is: "relation
users does not exist". Do you know why?

## Update Docker image

Well, that's because we haven't run db migration to create the db schema yet.
To fix this, we have to update our Docker image to run db migration before
starting the API server. It would be done in a similar fashion as what 
we've done in the GitHub CI workflow: we will have to download 
golang-migrate binary to the Docker image and use it to run the migrations.

![](../images/part25/11.png)

So I'm gonna copy these 2 instructions and paste them to the `Dockerfile` in
the builder stage.

```yaml
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.14.1/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate.linux-amd64 /usr/bin/migrate
```

First we have to run this `curl` command

```
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.14.1/migrate.linux-amd64.tar.gz | tar xvz      
```

to download and extract the `migrate` binary.

```dockerfile
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.14.1/migrate.linux-amd64.tar.gz | tar xvz
```

Then I'm gonna move this second command to the run stage

```
sudo mv migrate.linux-amd64 /usr/bin/migrate
```

where we will copy from `builder` the downloaded `migrate` binary to the
final image. We have to change the path of original `migrate` file to `/app`,
because that's the working directory in the builder stage when we download
and extract the file.

```dockerfile
COPY --from=builder /app/migrate.linux-amd64 /usr/bin/migrate
```

Then I'm gonna put this file in the same `WORKDIR` folder in the final 
run stage image, which is also `/app`.

```dockerfile
COPY --from=builder /app/migrate.linux-amd64 ./migrate
```

Next, we also have to copy all migration SQL files from the `db/migration`
folder to the image. So copy `db/migration`. I'm gonna put it in the 
`migration` folder under the current working directory.

```dockerfile
COPY db/migration ./migration
```

Now, one thing we must do is, to install `curl` in the builder stage image
because by default, the base Alpine image doesn't have `curl` preinstalled.
To do that, we just have to add a `RUN apk add curl` instruction here.
Finally, we have to change the way we start the app. So that, it can run 
the db migration before running the `main` binary. I'm gonna create a new 
file: `start.sh` at the top of our project. Then let's change mod this
file to make it executable.

```shell
chmod +x start.sh
```

This file will be run by `/bin/sh` because we're using Alpine image, so
the `bash` shell is not available. We use `set -e` instruction to make sure
that the script will exit immediately if a command returns a non-zero
status. First step, we will run db migration. So we call the `/app/migrate`
binary, pass in the path to the folder containing all migration SQL files,
which is `/app/migration`. Then the database URL, which we will take
from the `DB_SOURCE` environment variable. So here, I just use `$DB_SOURCE`
to get its value. Let's also use `-versbose` option to print out all
details when the migration is run. Finally the `up` argument is used to
run migrate up all migrations.

```shell
set -e

echo "run db migration"
/app/migrate -path /app/migration -database "$DB_SOURCE" -verbose up
```

After running `migrate up`, we will start the app. All we have to do in 
this step is to call `exec "$@"`.

```shell
echo "start the app"
exec "$@"
```

It basically means: takes all parameters passed to the script and run it. 
In this case, we expect it to be `/app/main` as specified in this `CMD` 
instruction.

```dockerfile
CMD ["/app/main"]
```

To make it work, we will use the `ENTRYPOINT` instruction and specify the
`/app/start.sh` file as the main entry point of the Docker image.

```dockerfile
ENTRYPOINT ["/app/start.sh"]
```

Keep in mind that when the `CMD` instruction is used together with 
`ENTRYPOINT`, it will act as just the additional parameters that will be
passed into the entry point script. So basically, it will be similar 
to running "/app/starts.sh" with "/app/main" as the second argument. But
by separating the command from the entrypoint we have more flexibility to
replace it with other command at run time whenever we want. You can read
more about this in the Docker documentation page, in the `CMD` instruction
section.

```dockerfile
# Builds stage
FROM golang:1.16-alpine3.13 AS builder
WORKDIR /app
COPY . .
RUN go build -o main main.go
RUN apk add curl
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.14.1/migrate.linux-amd64.tar.gz | tar xvz

# Run stage
FROM alpine3.13
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/migrate.linux-amd64 /usr/bin/migrate
COPY app.env .
COPY db/migration ./migration

EXPOSE 8080
CMD ["/app/main"]
ENTRYPOINT ["/app/start.sh"]
```

OK, now the Docker file is updated, let's try to run `docker-compose` 
again. But first, we need to run `docker-compose down` to remove all
existing containers and networks. And we should also remove the 
`simplebank_api` image, because we want to rebuild the image with the 
new db migration scripts.

```shell
docker rmi simplebank_api
```

OK, everything is cleaned up. Let's run `docker-compose up` again!
It will take a while to rebuild the image.

![](../images/part25/12.png)

Oops, we've got an error: "start.sh: no such file or directory".

Oh, that's because I forgot to copy the `start.sh` file into the docker
image. So let's do that now!

Here in the run stage, we just need to add 1 more instruction:

```dockerfile
COPY start.sh .
```

And that's it! We should be good now. Let's run `docker-compose down` to
remove all containers. Run

```shell
docker rmi simplebank_api
```

to remove the old `simplebank_api` image and finally run 
`docker-compose up` to rebuild and relaunch everything! This time, looks
like the db migration script was run, however, it was still not successful.

![](../images/part25/13.png)

## Waiting for DB start

The service exited with code 1. And in the log, we can see the error: 
"connection refused". The app cannot connect to the database to run db 
migration. That's because it takes some time for Postgres server to be
ready, but the app tried to run db migration immediately when it starts.
At that time the Postgres server was not ready to accept connection yet. 
So to fix this, we have to tell the app to wait for Postgres to be ready
before trying to run the db migration script and start the API server. 
Now although in the `docker-compose.yaml` file, we can use the 
`depends_on` instructions to tell `docker-compose` that the `api` service
depends on the `postgres` service, this only makes sure that the `postgres`
will be started before the `api` service. It doesn't ensure that the
`postgres` is in ready state before starting the `api` service.

You can read more about this behavior in the `docker-compose` 
documentation page. Just search for `depends_on` on [https://docs.docker.com/compose/compose-file/compose-file-v3/](https://docs.docker.com/compose/compose-file/compose-file-v3/)
page. Then you will see there, they listed several things to be aware of
when using the `depends_on` instruction. If we want to wait for a 
service to be ready, we should follow this [link](https://docs.docker.com/compose/startup-order/) to know how to control
the start up order. There are several tools that has been written to 
solve this problem, but the one we should use in our case is the 
sh-compatible `wait-for` script, because we're using an Alpine-based 
image.

OK, so here is its [Github page](https://github.com/Eficode/wait-for). 
`Wait-for` is designed to synchronize services like Docker containers.
And its usage is pretty simple. We just need to run `wait-for`, and pass
in the `host:port` URL we want to wait.

Here's an example,

```shell
$ ./wait-for www.eficode.com:80 -- echo "Eficode site is up"
Eficode site is up
```

It's basically waiting for the web page `eficode.com` port 80 to be 
ready before executing the `echo` statement. So what I'm gonna do is
open its latest release version, and click on this link to download the
script.

![](../images/part25/14.png)

The file will be downloaded to your `Downloads` folder. So I'm gonna open
the terminal and copy that file to the root of our simple bank project. 
I will rename it to `wait-for.sh` to make it cleaner about the file type.

```shell
mv ~/Downloads/wait-for ./wait-for.sh
```

Note that this file is not executable yet, so we should run change mod +x
to make it executable.

```shell
chmod +x wait-for.sh
```

Alright, now back to our `Dockerfile`. I'm gonna add 1 more instruction
to copy the `wait-for.sh` file into the final Docker image.

```dockerfile
COPY wait-for.sh .
```

Then, in the `docker-compose` file, we should override the default entry
point and command. So that, it will wait for the `postgres` service to be
ready before trying to start the `api` service. Let's add `entrypoint` here
and its content should be: `/app/wait-for.sh`. Then the host and port to
wait for should be `postgres:5432`, followed by 2 dashes, then finally
the `start.sh` script to run after the `postgres` server is ready.

```yaml
version: "3.9"
services:
  postgres:
    image: postgres:12-alpine
    environment:
      - POSTGRES_USER=root
      - POSTGRES_PASSWORD=secret
      - POSTGRES_DB=simple_bank
  api:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      - DB_SOURCE=postgresql://root:secret@postgres:5432/simple_bank?sslmode=disable
    depends_on:
      - postgres
    entrypoint: ["/app/wait.sh", "postgres:5432", "--", "/app/start.sh"]
```

Now you should know that when the `ENTRYPOINT` is overridden like this,
it will also clear out any default command on the image, meaning that the
`CMD` instruction we wrote in the `Dockerfile` will be ignored. So in this
case, we have to explicitly specify the command we want to run in the
`docker-compose.yaml` file as well. We use the `command` keyword for that
purpose. And the command we want to run is simply: `/app/main`.

```yaml
    entrypoint: [ "/app/wait.sh", "postgres:5432", "--", "/app/start.sh" ]
    command: [ "/app/main" ]
```

And that will be it! I think now we have everything we need for the app
to launch successfully. Let's try it one more time! First let's run 
`docker-compose down` to remove all existing containers. Run `docker rmi`
to remove the `simplebank_api` image.

```shell
docker rmi simplebank_api
```

And finally run 

```shell
docker-compose up
```

to rebuild and relaunch the services. This time, you can see that the 
services are started in order.

![](../images/part25/15.png)

First the `postgres` service will run, and only when it is ready to listen
for connections, then the `api` service will start running the db 
migration. Therefore, the migrations were run successfully, first the 
`init_schema` migration, then the `add_user` migration. Finally, the API
server was started successfully after the db migrations. We can see the Gin
logs saying that the server is listening and serving HTTP requests on 
port 8080.

![](../images/part25/16.png)

## Create User API test using Postman

So let's open Postman to try it! I'm gonna send this same create user 
request as before.

![](../images/part25/17.png)

But this time, the request is successful. A new user is created! Awesome!

And with this I'm gonna conclude today's lecture about `docker-compose`.

Now you know how to use this powerful tool to launch multiple services at
once, with specific startup orders.

Thanks a lot for watching, happy coding, and see you in the next lecture!