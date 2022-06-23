# How to use Docker network to connect 2 stand-alone containers

[Original video](https://www.youtube.com/watch?v=VcFnqQarpjI)

Hello everyone, and welcome back to the backend master class!

In the previous lecture, we've learned how to build a minimal Docker image
for our Golang application. The image we built was pretty small, just `17.7MB`.
However, I'm not sure if it gonna work well or not because we haven't tried
to run it yet. So let's do that now!

## How to run Docker containers

To start a container from this image we will use `docker run` command. The
`--name` option is used to specify the name of the container we're gonna 
create. Let's say `simplebank`. Then the `-p` option is used to publish 
the port `8080` to the external host machine. Finally, the last argument we
must provide is the name and tag of the image which is `simplebank:latest`.

```shell
docker run --name simplebank -p 8080:8080 simplebank:latest
```

Oops, we've got an error! "Cannot load config: Config file `app` not found."
This is what I expected! Do you remember that we use `Viper` to load the 
`app.env` config file in [lecture 12](part12.md)? But this file doesn't exist
in the Docker image that we built. In the last lecture, we just copied the
binary executable file, but we didn't copy this config file to the final run
stage image. So to fix the error, we have to copy the `app.env` file to that
image as well. Now, I know this is just a config file we used for 
development, which connects to the local Postgres DB, and uses a trivial 
token symmetric key. So it should not be used in the production image, right?
Don't worry, we will learn how to replace this file with the real production
config file in another lecture, when we update our CI/CD workflow to 
automatically build the image for us. For now, let's just use this 
development config to test our Docker image first. Alright, since the last
`docker run` command fail, if we run `docker ps -a`, we can see that it left
behind an exited `simplebank` container. So let's run

```shell
docker rm simplebank
```

to remove this container. Now before we rebuild the Docker image, let's run
`docker rmi` to remove old `simplebank` image.

```shell
docker rmi 58c7252992ba
```

Then let's run `docker build` command to rebuild it.

```shell
docker build -t simplebank:latest .
```

OK, now we have the new image. Let's run it!


This time the error is gone! The server has started successfully! Here 
you can see a warning that Gin is running in debug mode.

![](../images/part24/1.png)

In production, we can set this `GIN_MODE` environment variable to `release` to
instruct Gin that it should run the server in `release` mode instead. So 
let's stop the server and try it! I'm gonna remove the old `simplebank`
container,

```shell
docker rm simplebank
```

and start a new one with the same `docker run` command, but this time, we will
use the `-e` option to set `GIN_MODE` environment variable to `release`. 

```shell
docker run --name simplebank -p 8080:8080 -e GIN_MODE=release simplebank:latest
```

And voilà, the server has started, but now we don't see any debug logs as 
before. So it worked! If we open another terminal tab and run `docker ps`,
we can see that `simplebank` container is up and running, together with the
`postgres12` database container.

![](../images/part24/2.png)

OK, now let's open Postman and send a request to make sure that it is working
properly. I'm gonna call this Login user API!

![](../images/part24/3.png)

Oops, we've got `500 Internal Server Error`,  and the reason is the server 
cannot connect to local host, port 5432, which is the address of our Postgres
database. In the terminal, we can see the log of this request here.

![](../images/part24/4.png)

So what happened? We've already confirmed that the postgres db container is
also running, right? Yes, but the problem is that, as stated in the `app.env`
config file, we're connecting to the database via the `localhost` address. But
the `simplebank` and `postgres` containers are 2 separated containers, so
actually they don't have the same network IP address. We can run this `docker
container inspect` command

```shell
docker container inspect postgres12 
```

to see the network settings of the `postgres12` container.

![](../images/part24/5.png)

As you can see here, it's running on the default `bridge` network, and its 
IP address is `172.17.0.3`. Now let's run

```shell
docker container inspect simplebank 
```

![](../images/part24/6.png)

Voilà, the IP address of the `simplebank` container is different: 
`172.17.0.2`, although it's running on the same default `bridge` network with 
the `postgres12` container. So now you know why we cannot use `localhost`
to connect to Postgres container.

How can we fix it? Well, the easiest way is to just replace `localhost` with
the real IP address of the `postgres12` container in the `app.env` config
file. Like this:

```
DB_SOURCE=postgresql://root:secret@172.17.0.3:5432/simple_bank?sslmode=disable
```

and rebuild the Docker image.

However, it's not very convenient, because we will have to rebuild the image
everytime the IP address of Postgres container changes. In fact, there's 
a better way to change the `DB_SOURCE` without rebuilding the image. Keep
in mind that we're using `Viper` to read the config, and it has the ability
to override the config written in `app.env` file with the value we provide
via environment variables. So I'm gonna stop the current `simplebank` 
container, remove it

```shell
docker rm simplebank
```

and start a new `simplebank` container, but this time, I will add 1 more
environment variable to override the `DB_SOURCE` value with the real IP
of the `postgres` container.

```shell
docker run --name simplebank -p 8080:8080 -e GIN_MODE=release DB_SOURCE=postgresql://root:secret@172.17.0.3:5432/simple_bank?sslmode=disable simplebank:latest
```

Oh, we have an error, because this `DB_SOURCE` URL contains several special 
characters. So we should wrap it inside a pair of double quotes. Just
like that

```shell
docker run --name simplebank -p 8080:8080 -e GIN_MODE=release DB_SOURCE="postgresql://root:secret@172.17.0.3:5432/simple_bank?sslmode=disable" simplebank:latest
```

and voilà, the server is now started successfully.

Now let's send the login request again to see how it goes!

![](../images/part24/7.png)

Yee! It's successful! We've got a new access token. So now you know how to
connect your container to another stand-alone container via its IP address.
However, IP address can change if we restart the container, so using it is
not really ideal.

Let's stop the `simplebank` container and remove it.

```shell
docker rm simplebank
```

Then, I'm gonna show you a better way to connect to Postgres without using
IP address, but using user-defined network instead.

## How to create user-defined network

Now, if we run `docker network ls`, we can see the default `bridge` network
that our `simplebank` and `postgres` containers were using.

![](../images/part24/8.png)

We can see more details about this network by running `docker network 
inspect`, and pass in the network name: `bridge`

```shell
docker network inspect bridge
```

![](../images/part24/9.png)

Here, in the `Containers` block, we can see the list of containers that are
running on this network. Right now, only `postgres12` container is using it
because we've just removed the `simplebank` container. Normally containers
running on the same network can discover each other via name instead of IP
address. However, it doesn't work for the default `bridge` network. So what
we need to do is: to create our own network, and put both `postgres` container
and `simplebank` container on this network. Then they would be able to call
each other using names.

Docker provides several commands to manage networks. Let's use the 
`docker network create` command to create a new network. I'm gonna call it
`bank-network`.

```shell
docker network create bank-network
```

## Connect 2 stand-alone containers

Then we can use the `docker network connect` command to connect the existing
`postgres` container to it. Here's the syntax.

![](../images/part24/10.png)

So `docker network connect`, then the name of the network: `bank-network` and
finally the name of the container: `postgres12`. Now if we run

```shell
docker network inspect bank-network
```

we can see that the `postgres12` container has been connected to this new
network. 

![](../images/part24/11.png)

And if we inspect the `postgres12` container,

```shell
docker container inspect postgres12
```

![](../images/part24/12.png)

we can see that it is now connected to 2 different networks: the 
`bank-network` that we've just created, and the default `bridge` network that
it was originally connected to. It's totally OK for a container to be connected
to multiple networks at the same time.

Alright, now it's time to recreate the `simplebank` container, that will
connect to this new network. We will use the same `docker run` command as 
before, but we will add 1 more option: `--network`, and pass in the name
of the network: `bank-network`.

```shell
docker run --name simplebank --network bank-network -p 8080:8080 -e GIN_MODE=release DB_SOURCE="postgresql://root:secret@172.17.0.3:5432/simple_bank?sslmode=disable" simplebank:latest
```

This time, the `simplebank` container will be running on the same network
with Postgres, so it can discover the Postgres container's address by its
name, so let's replace this IP address with `postgres12`.

```shell
docker run --name simplebank --network bank-network -p 8080:8080 -e GIN_MODE=release DB_SOURCE="postgresql://root:secret@postgres12:5432/simple_bank?sslmode=disable" simplebank:latest
```

And that's it! The server is now started. Let's resend the login request to
see if it's working or not. 

![](../images/part24/13.png)

Yee! The request is successful. Awesome!

So the `simplebank` container has successfully talked to the `postgres` 
container using its name instead of the IP address. That's pretty cool, isn't
it? Now if we inspect the `bank-network`, we can see that there are 2 
containers connecting to it: `simplebank` and `postgres12`.

![](../images/part24/14.png)

OK, so now you know how to take advantages of the user-defined network to 
allow 2 stand-alone containers to talk to each other by names.

Before we finish, I'm gonna update the `make postgres` command to connect
itself to the `bank-network` upon creation. Then let's add all the changes
we've made so far, create a new commit, and push it to GitHub.

![](../images/part24/15.png)

OK, here we can see the new commit in this pull request.

![](../images/part24/16.png)

When you work on a team, your teammates can go to this PR to review your
codes. 

![](../images/part24/17.png)

You can click on any line to write a comment like this and click `Add single
comment` or `Start a review`. You can also select multiple lines to leave
a comment that involves all of them. And once the code is properly reviewed,
and the unit tests passed, we can merge it to the base branch, which is 
`master` in this case. Let's click `Merge pull request`! `Confirm merge` and
finally delete the feature branch.

![](../images/part24/18.png)

![](../images/part24/19.png)

![](../images/part24/20.png)

And that's it! We're done! The pull request is successfully merged to 
`master`. And that wraps up today's lecture about Docker networking.

I hope you find it useful. Thanks a lot for watching. Happy learning and see 
you in the next lecture!