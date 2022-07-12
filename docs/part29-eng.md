# Store & retrieve production secrets with AWS secrets manager

[Original video](https://www.youtube.com/watch?v=3i1mQ_Ye8jE)

Hello everyone, and welcome to the backend master class.

In this lecture, we will learn how to use another service: AWS Secret Manager
to manage the environment variables and secrets for our application.

If you remember, in the [previous lecture](part28-eng.md), we have set up a 
production database on AWS RDS, and this is the URL

```
postgresql://root:tupExr0Gp4In4Ww4WHKR@simple-bank.czutruo2pa5q.eu-west-1.rds.amazonaws.com:5432/simple_bank
```

we used to access it. When we deploy the simple bank app to production, we
would want it to connect to this database. So basically, we must replace the
`DB_SOURCE` variable in the `app.env` file with the real production DB URL. Also,
we need to generate a stronger token symmetric key. We should not use this
`TOKEN_SYMMETRIC_KEY=12345678901234567890123456789012` trivial and 
easy-to-guess value, right?

So, the idea is, in the GitHub Actions `deploy` workflow, before building and
pushing the Docker image to ECR, we will replace all environment variables
in the development `app.env` file with the real production values. By doing so,
when we run the Docker container on the server later, it will have all the 
correct settings for production environment. OK, but the question is, where
do we store the values of these environment variables?

Of course, we cannot put them directly in our GitHub repository, because it
would be very insecure, right? One good solution is to use AWS secret manager
service.

![](../images/part29/1.png)

## AWS Secret Manager service

This service allows us to store, manage, and retrieve any kind of secrets for
our application. It is pretty cheap, just 0.4 dollars per secret per month,
and only 0.05 dollars per 10000 API calls, and we also have 30-day free trial
period. As we have planed to use AWS to deploy our app, it totally makes sense
to take advantages of this secret manager service. OK, let's create a new 
secret!

![](../images/part29/2.png)

There are several types of secret, such as credentials of RDS, DocumentDB, 
Redshift, or other databases. In our case, we want to store not only the DB
credentials, but other environment variables as well, so I'm gonna choose 
"Other type of secrets".

![](../images/part29/3.png)

Then in the next section, we can add as many key-value pairs as we want. 
First, let's add the `DB_SOURCE`. I'm gonna copy the production RDS database
URL from the Makefile and paste it to this input textbox.

![](../images/part29/4.png)

Then let's click `Add row` to add a new pair of key-value. The key will be 
`DB_DRIVER`, and the value will be `postgres`. Next, a new row for the 
`SERVER_ADDRESS`. Its value will be the same as in development: `localhost`
port `8080`. Note that this is just the internal address of the container.
Later, when we actually deploy the app to kubernetes, we will learn how to
setup a load balancer and domain name that will route the API requests to
the correct container's internal address. OK, now let's add one more row for
the `ACCESS_TOKEN_DURATION`. Its value will be 15 minutes. And finally, the 
last row for the `TOKEN_SYMMETRIC_KEY`. Its value should be a 32-character 
string. There are many ways to generate a random string of 32 characters.
Today I'm gonna show you how to do it using `openssl` command. It's pretty
simple, we just run:

```shell
openssl rand -hex 64
```

Use the `-hex` flag to tell it to output a string of only hexdecimal digits. 
And finally the number of bytes, let's say 64 bytes to make a very long 
string. Actually this string has 128 characters because each hex digit only
takes 4 bits, or half a byte. In our case, we only need 32 characters, so 
here we use the `pipe` to chain its output to the next command:

```shell
openssl rand -hex 64 | head -c 32 
```

which means, just take the first 32 characters. And voila, we now have a 
random string for the token symmetric key. Let's copy and paste it to the 
secret manager. OK, now we have added all necessary variables to the secret. 
There's an option to select a custom AWS KMS key to encrypt the data, but
it's not mandatory. We can just go with the default encryption key for now.

![](../images/part29/5.png)

In the next step, we have to give our secret a name, so that we can easily 
refer to it later. Let's call it `simple_bank`. You can also write some 
short description to help remind you about what values this secret stores. 
Optionally, we can add some tags to make it easier to manage, search or filter
AWS resources. And an option to set permissions to access the secret. But we
can skip all of them for now.

![](../images/part29/6.png)

So let's click `Next`!

In this step, we're able to enable automatic rotation for our secret values.
Simply put, we can set a schedule and a lambda function, then the secret 
manager run that function to change the secret values when the time comes. 
You can use this to frequently update your DB password, or token symmetric key
if you want. To keep this lecture simple, I'm just gonna disable automatic 
rotation.

![](../images/part29/7.png)

In the last step, we can review all the settings of the secret. AWS also give 
us some sample code for several languages. In case you want to fetch the 
secret value directly from your code for example, you can use this template,
and download the appropriate SDK to do so. We don't need to do that in our
case, so let's click `Store` to save the secret.

![](../images/part29/8.png)

And voila, the secret is successfully created. In this page, we can click 
`Retrieve secret value` button to show all the content stored in the secret.

![](../images/part29/9.png)

Now the secret is ready, we will learn how to update the GitHub `deploy` 
workflow to retrieve the secret values and save them to the `app.env` file. 
To develop this feature, I think we will need to install the AWS CLI. It is a
very powerful tool to help us easily interact with the AWS services via API
call from the terminal. You can choose the suitable package depending on your
OS. I'm on macOS, so I will click on this [link](https://awscli.amazonaws.com/AWSCLIV2.pkg)
to download the installer package. Then open it to start the installation, and
follow the instructions on the UI.

![](../images/part29/10.png)

OK, now the AWS CLI package is successfully installed, we can run these 2
commands to verify that it is working properly:

```shell
which aws
```

and

```shell
aws --version
```

All looking good. Next we have to setup the credentials to access our AWS
account. To do that, just run

```shell
aws configure
```

We will be asked for the access key ID and secret. So let's go back to the 
AWS console and open IAM service.

![](../images/part29/11.png)

In the left menu, click on `Users`. Here, we can see the github-ci user that 
we've set up in one of the previous lecture. On the security credentials tab, 
we can see the `Access Key ID` that's being used by GitHub Action.

![](../images/part29/12.png)

But for security reason, we cannot see its `Secret Access Key`.

So we have to create a new one to use locally. Click on `Create access key` 
button.

![](../images/part29/13.png)

Let's copy this access ID and paste it to the terminal.

![](../images/part29/14.png)

Next, it will ask for the `Secret Access Key`. Let's show the value and copy 
it. Then paste it to the terminal. It will ask for a default region name. I'm
gonna put `eu-west-1`, because it's the main region I'm currently using. And
finally, the output format. It's the data format we want AWS to return when we
use the CLI to call its API. For me, I'm gonna use JSON format.

![](../images/part29/15.png)

Alright, now if we look at the `.aws` folder, we will see 2 files: `credentials`
and `config`. The credentials file contains the access key id and secret that 
we've just entered before. The `default` at the top of the file is the name of
the AWS profile. You can add multiple AWS profiles with different access key
to this file if you want.

![](../images/part29/16.png)

The `default` profile is, of course, the one you will use by default, which 
means, if you don't explicitly specify a profile name when running a command,
then this `default` profile's credentials will be used.

Similarly, the `config` file contains some default configurations, in our case,
it's the default region and output format that we've entered before.

![](../images/part29/17.png)

OK, now we can call the secret manager's API to retrieve our secret values.
You can run

```shell
aws secretsmanager help
```

to read its manual.

The sub command we're gonna use is `get-secret-value`. So let's run the
help command again with `get-secret-value` to see its syntax.

```shell
aws secretsmanager get-secret-value help
```

Basically, we will need to pass in the secret ID of the secret we want to get.
It can be either an ARN (Amazon resource name), or friendly name of the 
secret. You can find the ARN of the secret in its AWS console page.

![](../images/part29/18.png)

The friendly name is the name we set when creating this secret, which is
`simple_bank`.

Alright, now get back to the terminal and run

```shell
aws secretsmanager get-secret-value --secret-id simple_bank
```

Oops, we've got an error: the github-ci user is not authorized to perform
this request. That's expected, because we haven't grant permissions to allow
this user to get the secret value yet.

In the IAM page, we can see the `github-ci` user is ib the `deployment` group, 
which only has permission to access the Amazon ECR service. What we need to
do now is to give this group access to the Secret Manager service as well. So,
in this user group's page, let's open the `Permissions` tab, then click 
`Add permissions`, `Attach Policies`.

![](../images/part29/19.png)

Search for "Secret" in this filter box.

![](../images/part29/20.png)

Here it is! Let's select this `SecretManagerReadWrite` policy and click 
`Add permissions`.

![](../images/part29/21.png)

Alright, now all users in this group should have access to secret manager 
service. Let's go back to the terminal and run the `get-secret-value` command.
We still get access denied exception. I think the permission we've just added
needs some time to be effective. So let's wait a bit, and let's try using the
secret ARN instead of the friendly name:

```shell
aws secretsmanager get-secret-value --secret-id arn:aws:secretsmanager:eu-west-1:095420225348:secret:simple_bank-DI3Vdk
```

OK, now the call is successful. I'm gonna try again with the friendly name: 
`simple_bank`.

```shell
aws secretsmanager get-secret-value --secret-id simple_bank
```

It's also successful this time. Cool!

As you can see, the result is in JSON format. And we have several more 
information than just the values stored in the secret. What we need is only 
the value stored in this "SecretString" field.

![](../images/part29/22.png)

To get only this field, we just have to add a `--query` argument to the 
previous command and pass in the name of the field: "SecretString".

```shell
aws secretsmanager get-secret-value --secret-id simple_bank --query SecretString
```

Voilà, now we see only the data stored in the secret. However, its value is 
in the form of a string, not a JSON object. We have to add one more argument:
"--output text" to the command in order to get the output value in JSON format
as you can see here.

```shell
aws secretsmanager get-secret-value --secret-id simple_bank --query SecretString --output text
```

## Transform JSON object into environment variables

OK, but now, how can we transform this JSON object into environment variable
format to store in the `app.env` file? Well, there's a very nice tool called
"jq" that will help us with this problem. [jq](https://stedolan.github.io/jq/) is a lightweight and flexible
command-line JSON processor. Let's see how to install [it](https://stedolan.github.io/jq/download/).
On Linux, `jq` is already available in the official Debian and Ubuntu, so we
don't need to do anything. But it's not available on MacOS by default, so we
have to install it. In the terminal, let's run:

```shell
brew install jq
```

While waiting for `brew` to install the package, let's take a look at its 
[manual](https://stedolan.github.io/jq/manual/). There are many built-in filters
and operators that we can use to transform the input JSON data. The most basic
one is `identity`, represented by just a dot. This filter just returns whatever
it takes as input unchanged. Then we have the object identifier index, or a dot
followed by the name of the field we want to get. For example, here we run 
`jq '.foo'` so with this input JSON,

```shell
        jq '.foo?'
Input	{"foo": 42, "bar": "less interesting data"}
Output	42
```

it will return the value of the field "foo", which is 42. If field doesn't
exist as in this example, it will return `null`.

```shell
	    jq '.foo?'
Input	{"notfoo": true, "alsonotfoo": false}
Output	null
```

There are many many other filters, commands, and syntaxes, which I think you 
can discover on your own if you want. OK, back to the terminal. Looks like
`jq` has been installed successfully. Let's run 

```shell
jq --version
```

to verify. OK, it's version 1.6.

![](../images/part29/23.png)

Now let's run the command to fetch the secret values as JSON object, we will
have to chain the output of this command with `jq` to produce the final output
file. First, I want to convert this key-value JSON object into an array, where
each object will be 1 separate environment variable. To do that, we will use
the `to_entries` operator of `jq`.

```shell
	    jq 'to_entries'
Input	{"a": 1, "b": 2}
Output	[{"key":"a", "value":1}, {"key":"b", "value":2}]
```

As you can see in this example, it transforms 1 object with 2 keys a, b into
1 array of 2 objects. Each object has a `key` and `value` field. That's exactly 
what we want, so here I'm gonna chain the get secret value command with `jq`
`to_entries`.

```shell
aws secretsmanager get-secret-value --secret-id simple_bank --query SecretString --output text | jq 'to_entries'
```

![](../images/part29/24.png)

Voilà, now we have 5 different objects, each stored 1 separate environment 
variable. Next, we have to iterate through them and transform each object into
the form of `key=value`, since that's the final format we want to store in the
`app.env` file. For this kind of transformation, we're gonna use the `map`
operator. The way it works is very similar to the `map` function in Python
or Ruby. Basically, it iterates through the list of values, apply a transform
function on each of them and return a new list of the transformed values.

```shell
	    jq 'map(.+1)'
Input	[1,2,3]
Output	[2,3,4]
```

So here, in our command, we can chain this `to_entries` operator with `map` 
and let's say if we only want to get the key of the object, we will use `.key`
as the transform function.

```shell
aws secretsmanager get-secret-value --secret-id simple_bank --query SecretString --output text | jq 'to_entries|map(.key)'
```

Then voilà, we've got a new array of strings with all the keys.

![](../images/part29/25.png)

Similarly, we can get an array of strings with only the values by using 
`.value` as the transform function. OK, but what we want is a string contains
both key and value, separated by an equal sign. For that, we will need the 
string interpolation operator. Basically, it allows us to put an expression 
inside a string, by using a backslash followed by a pair of parenthesis.

```shell
	    jq '"The input was \(.), which is one less than \(.+1)"'
Input	42
Output	"The input was 42, which is one less than 43"
```

As in this example, the first expression will just be replaced by the input
value, while the second one will be the input value + 1. In our case, for this
`map` function we will first wrap it as a string then the first interpolation 
expression should be `.key` followed by an equal sign, and then the second 
interpolation expression will be `.value`.

```shell
aws secretsmanager get-secret-value --secret-id simple_bank --query SecretString --output text | jq 'to_entries|map("\(.key)=\(.value)")'
```

![](../images/part29/26.png)

OK, now we've got an array of 5 strings, each store 1 environment variable in
the desired format. But, in the final output, we must get rid of the array, so
to do that, we will use the array object value iterator, or a dot followed by
a pair of square brackets.

```shell
	    jq '.[]'
Input	[{"name":"JSON", "good":true}, {"name":"XML", "good":false}]
Output	{"name":"JSON", "good":true}
{"name":"XML", "good":false}
```

As you can see in this example, by using this, the array will be gone, and 
only the objects are printed. Let's try it! I'm gonna add a pipe chain here,
followed by the array iterator.

```shell
aws secretsmanager get-secret-value --secret-id simple_bank --query SecretString --output text | jq 'to_entries|map("\(.key)=\(.value)")|.[]'
```

![](../images/part29/27.png)

Now you can see, only the 5 strings remained, the array square brackets and 
the commas are gone. The last thing we need to do is to get rid of the double
quote characters surrounding the string. For that, we just need to pass in
the `-r` (or `--raw_output`) option to the `jq` command. With this option, the
result strings will be written out without quotes. Let's try it!

```shell
aws secretsmanager get-secret-value --secret-id simple_bank --query SecretString --output text | jq -r 'to_entries|map("\(.key)=\(.value)")|.[]'
```

![](../images/part29/28.png)

Awesome, now the output looks exactly as we wanted it to be. All we have to do
is to redirect this output to overwrite the `app.env` file just like that!

```shell
aws secretsmanager get-secret-value --secret-id simple_bank --query SecretString --output text | jq -r 'to_entries|map("\(.key)=\(.value)")|.[]' > app.env
```

OK, let's check the file to see how it goes.

![](../images/part29/29.png)

Excellent! The whole file content has been replaced with the production 
environment variables. Exactly as we stored in our secret.

## Update `deploy` workflow

The next step we must do is to plug this command to the GitHub CI `deploy` 
workflow before building the Docker image. But first, I need to reset all the
changes we've made to the `app.env` and `Makefile`. Let's run `git checkout .`
in the terminal. OK, now all the content of the files has been reset to the 
original version on `master` branch.

I'm gonna create a new branch `ft/secrets_manager` to make new changes.

```shell
git checkout -b ft/secrets_manager
```

In the `deploy.yml` file, let's add a new step. Its name will be: `Load secrets
and save to app.env`.

```yaml
- name: Load secrets and save to app.env
  run: aws secretsmanager get-secret-value --secret-id simple_bank --query SecretString --output text | jq -r 'to_entries|map("\(.key)=\(.value)")|.[]' > app.env
```

And it will run commands that we've prepared before to fetch the secret values
from AWS, transform them, and store in the `app.env` file. We don't have to
install `jq` because it's already available in the Ubuntu image, we don't have
to setup AWS CLI credentials either, because it's already been set up in the 
previous step of the workflow. So let's commit this change.

```shell
git commit -m "load secrets and save to app.env"
```

Push it to GitHub.

```shell
git push origin ft/secrets_manager
```

And open this [URL](https://github.com/techschool/simplebank/pull/new/ft/secrets_manager)
in the browser to create a new pull request. This is a very simple change, so
there's not much to be reviewed. Let's wait a bit for the unit tests workflow
to finish. OK, now it's completed.

![](../images/part29/30.png)

I'm gonna merge this PR, and delete the feature branch. Let's open the 
`master` branch of this repo, the workflows are running. Let's check the 
details of the `deploy` workflow to see how it goes. 

![](../images/part29/31.png)

OK, looks like the load secrets step was already successful. And the image is
being and pushed to ECR. Alright, everything finishes without any errors.

![](../images/part29/32.png)

Let's open the AWS console, ECR service. In the `simplebank` repository, we see
a new image that has just been pushed.

![](../images/part29/33.png)

So it works! But, I want to make sure that the image we built can actually talk
to the production DB when we run it. So I will try to download this image
to my local machine and run it. Let's copy this image's URL `095420225348.dkr.ecr.eu-west-1.amazonaws.com/simplebank:5750a6ef812d5775e9adc07708428dead6a54ceb` 
and run `docker pull` this URL in the terminal.

```shell
docker pull 095420225348.dkr.ecr.eu-west-1.amazonaws.com/simplebank:5750a6ef812d5775e9adc07708428dead6a54ceb
```

## Pull Docker image and run locally

We got an error because Docker cannot pull image from this private repository.
We have to log in to the AWS ECR registry first in order to pull or push image.

To do that, let's search for `aws ecr get login password`. We're using AWS CLI
version 2, so let's open this [page](https://awscli.amazonaws.com/v2/documentation/api/latest/reference/ecr/get-login-password.html).
This command will allow us to call the AWS API to retrieve an authentication
token that Docker can use to login to our private registry. So let's copy this
`aws ecr get-login-password` command

```
aws ecr get-login-password
```

and run it in the terminal.

![](../images/part29/34.png)

Voilà, the authentication token is successfully returned. Now we will pipe this 
token to the `docker login` command, pass in the username AWS, and password 
from `stdin` argument. Finally, the URL of our private Docker registry.

```shell
aws ecr get-login-password | docker login --username AWS --password-stdin 095420225348.dkr.ecr.eu-west-1.amazonaws.com
```

We must remove the name of the `simplebank` repository from the URL.

![](../images/part29/35.png)

Alright, login succeeded. Now we can pull the production `simplebank` image to 
local.

```shell
docker pull 095420225348.dkr.ecr.eu-west-1.amazonaws.com/simplebank:5750a6ef812d5775e9adc07708428dead6a54ceb
```

It's done. Let's check the images.

```shell
docker images
```

![](../images/part29/36.png)

Here it is! Now, I'm gonna run this image to see if it's gonna work well or 
not.

```shell
docker run 095420225348.dkr.ecr.eu-west-1.amazonaws.com/simplebank:5750a6ef812d5775e9adc07708428dead6a54ceb 
```

![](../images/part29/37.png)

Oops, it failed at the `run db migration` step. `Error: URL cannot be empty`.
If we look at the `starts.sh` file, we can see that in the db migration step,
we're using the `DB_SOURCE` environment variable. But, this variable is only
defined in the `app.env` file that our app will read when it starts. It was
not set as the real environment variable of the container before the db
migration step is run. So that's why migrate complains that the URL is empty.
To fix this, we have to load the variables from the `app.env` file into
the container's environment before running db migrate.

If I echo `DB_SOURCE` now, it's still empty.

![](../images/part29/38.png)

To load the variables to the current shell's environment, I will use the 
`source` command. So 

```shell
source app.env
```

Now if I echo `DB_SOURCE` again, it's not empty anymore.

![](../images/part29/39.png)

OK, so let's copy this `source` command, and paste it to the `start.sh` file,
right before the db migrate command.

```shell
echo "run db migration"
source app.env
/app/migrate -path /app/migration -database "$DB_SOURCE" -verbose
```

Note that inside the image, the `app.env` file is stored in the working 
directory `/app`. So here, we have to change the path to: `/app/app.env`.

```shell
echo "run db migration"
source /app/app.env
/app/migrate -path /app/migration -database "$DB_SOURCE" -verbose
```

And that's it! I think this will fix the issue.

Let's check out the `master` branch, pull latest change from GitHub. And 
create a new feature branch for the fix. I'm gonna call it `ft/load_env`. Add
the `start.sh` file that we've just updated, commit it with a message: "load
environment variable before running db migration". Then push it to GitHub.

![](../images/part29/40.png)

Open this [link](https://github.com/techschool/simplebank/pull/new/ft/load_env) 
in the browser to create a new pull request. Wait a bit for the unit tests
to complete. Then merge the pull request to `master`. And delete the feature 
branch. Now we have to wait for the `deploy` workflow to finish. While 
waiting, I'm gonna check out `master` branch on local. Pull the latest change
that we've just merged from GitHub.

![](../images/part29/41.png)

OK, now let's see if the workflow has completed or not. It's still running.
OK, it completed.

![](../images/part29/42.png)

In the AWS console, we can see the new image.

![](../images/part29/43.png)

I'm gonna copy its URL. Then run `docker pull` to pull this image to local.

```shell
docker pull 095420225348.dkr.ecr.eu-west-1.amazonaws.com/simplebank:059c54e72ea72b76d1539c1627a958940b09da1
```

Alright, now let's run it, with a port mapping 8080 to 8080.

```shell
docker run -p 8080:8080 095420225348.dkr.ecr.eu-west-1.amazonaws.com/simplebank:059c54e72ea72b76d1539c1627a958940b09da1
```

So that we can send an API request to make sure it's working well. This time,
the db migration was successful, and the server has started listening and 
serving request on port 8080. I'm gonna open Postman, and send the first API
request to create a new user.

![](../images/part29/44.png)

It's successful! The user has been created. Let's connect to the production
DB on AWS using Table Plus and check the data of the `users` table.

![](../images/part29/45.png)

Here we go, the user Alice has been inserted as the first record in this 
table. So now we can say that our production docker image is good and ready
to be deployed.

And with that, I conclude today's lecture about storing and retrieving 
production configurations using AWS secret manager. I hope it was interesting
and useful for you.

In the next lecture, we will learn how to deploy the production image that 
we've prepared today to a kubernetes cluster. Until then, happy learning and
I'll catch you guys later!