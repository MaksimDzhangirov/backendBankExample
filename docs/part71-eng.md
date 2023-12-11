# Grant AWS EKS cluster access to Postgres and Redis using security group

[Original video](https://www.youtube.com/watch?v=pPXYu6QQGE8)

Hello everyone, welcome to the backend master class. In this lecture and the
next, we're gonna learn how to deploy our simple bank server, which 
includes both gRPC and HTTP APIs, to a Kubernetes cluster on AWS.

## Creating a new Kubernetes cluster using EKS

Let's start by creating a new Kubernetes cluster using EKS. I'm gonna name
it "simple-bank-eks". Choose the latest Kubernetes version. Then let's
create a cluster role for it by following 
[this instruction](https://docs.aws.amazon.com/eks/latest/userguide/service_IAM_role.html#create-service-role).

![](../images/part71/1.png)

![](../images/part71/2.png)

We have to open the IAM service, select the "Role" menu, and click "Create
role".

![](../images/part71/3.png)

Choose "AWS service", select "EKS" in the dropdown, then choose "EKS 
cluster".

![](../images/part71/4.png)

![](../images/part71/5.png)

The `AmazonEKSClusterPolicy` will be automatically added for
us. Next, we'll have to enter a name for the role, such as "eksClusterRole".

![](../images/part71/6.png)

![](../images/part71/7.png)

Finally, click "Create role", and role will be created.

![](../images/part71/8.png)

Now let's continue creating the EKS cluster. When we press this refresh 
button, 

![](../images/part71/9.png)

the "eksClusterRole" will show up.

![](../images/part71/10.png)

In the next step, I'm gonna use the default VPC and subnets. The security
group will be created automatically for us, so just leave it empty.

![](../images/part71/11.png)

For the cluster endpoint access, let's choose "Public and private".

![](../images/part71/12.png)

Feel free to configure logging if you want, 

![](../images/part71/13.png)

then make sure to select the 3 default add-ons: Proxy, DNS and CNI.

![](../images/part71/14.png)

You can also choose the version you want to use for each add-ons.

![](../images/part71/15.png)

Then, in the final step, let's review everything, and click "Create".

![](../images/part71/16.png)

It will take several minutes to complete.

Once the cluster is ready, we'll be able to add a worker node group 
to it. 

![](../images/part71/17.png)

I'm gonna call it "simple-bank-nodes". We also need to create a role
for the node group.

![](../images/part71/18.png)

So let's head over to IAM service, and click "Create role".

![](../images/part71/19.png)

Choose "EC2" service in the list, and click "Next".

![](../images/part71/20.png)

In the permission policies, let's look for "eks" 

![](../images/part71/21.png)

and choose 2 policies: `AmazonEKS_CNI` and `AmazonEKSWorkerNode`.

![](../images/part71/22.png)

Also search for "container" and choose 
`AmazonEC2ContainerRegistryReadOnly`.

![](../images/part71/23.png)

Since we're gonna need it to download the docker image from ECR.

In the last step, let's set a name for this role, such as: 
"eksWorkerRole".

![](../images/part71/24.png)

Then click "Create role" to create it.

Back to the `Configure node group` page, we can click the refresh button
to load & select the "eksWorkerRole".

![](../images/part71/25.png)

In the next step, we can select multiple configurations for the node
machine. I'm just gonna change the instance type to `t3.small` as
it's enough for this demo.

![](../images/part71/26.png)

I'm gonna change the `Desired size` and `Minimum size` to 1 node as
well.

![](../images/part71/27.png)

You can choose bigger numbers, depending on how you want to scale
your app.

Next, we can use the default subnets that have already been selected for
us.

![](../images/part71/28.png)

Finally, review everything one more time, and press "Create".

![](../images/part71/29.png)

It will also take several minutes to complete. Once the node group
is ready, we should check the status of the Add-ons. Here we can see
that there are new versions for the `CNI` and `Proxy` add-ons.

![](../images/part71/30.png)

So let's update them to the latest versions.

![](../images/part71/31.png)

![](../images/part71/32.png)

The `DNS` add-on is now degraded,

![](../images/part71/33.png)

but there's also a new version for it, so let's update it as well.

![](../images/part71/34.png)

![](../images/part71/35.png)

Hopefully it will work well after being updated to the latest version.

Sometimes it's degraded because there are no worker nodes in the 
cluster, but since the node group we created is now ready, it should
be able to run well.

![](../images/part71/36.png)

While waiting for the add-ons to be updated, let's open the terminal
and fetch the config of the EKS cluster to our local machine. Note that
in order to run following commands, you need to have `awscli`, `kubectl`
and `k9s` installed on your machine. You can watch some of the lectures
in section 3 of the course to know how to do that.

I'm gonna run

```shell
aws eks update-kubeconfig
```

then pass in the name of the EKS cluster, which is "simple-bank-eks" and
the region it's running on, which is "eu-west-1".

```shell
aws eks update-kubeconfig --name simple-bank-eks --region eu-west-1
Added new context arn:aws:eks:eu-west-1:760486049168:cluster/simple-bank-eks to /Users/quangpham/.kube/config
```

OK, so the context has been added to the kube config file.

We can show its content using the `cat` command.

```shell
cat ~/.kube/config
```

You should verify that the cluster ARN string matches the one on the AWS 
console.

![](../images/part71/37.png)

![](../images/part71/38.png)

Then, we can run the

```shell
kubectl config use-context arn:aws:eks:eu-west-1:760486049168:cluster/simple-bank-eks
Switched to context "arn:aws:eks:eu-west-1:760486049168:cluster/simple-bank-eks".
```

command with that ARN, to make sure that `kubectl` will always connect to
this EKS cluster from now on.

Now let's open the `k9s` console.

Et voilà, we're now connected to our kubernetes cluster.

![](../images/part71/39.png)

It already has some system pods up and running. So now if we go back to
the AWS console, and refresh the page, we'll see that all 3 add-ons are
active. Perfect!

![](../images/part71/40.png)

If we open the networking tab, you will see that a security group has also
been created for us.

![](../images/part71/41.png)

Later, we will use this security group to allow our EKS cluster to access
the database and Redis. And that brings us to the next step, which is
creating a database.

## Creating a database

Let's open the RDS service, and create a new database.

![](../images/part71/42.png)

Choose `PostgreSQL`, and select the newest version.

![](../images/part71/43.png)

I'm gonna use the Free tier template for this demo. Then, let's set a name
for this instance: "simple-bank-db". I'll use "root" as the master 
username, and check this box to manage the password in Secret manager.

![](../images/part71/44.png)

For the instance type, `t3.micro` is enough for now. Most of the 
configurations are already well set, so we don't have to touch them.
But make sure the VPC is the same as the one we use for our EKS cluster.

![](../images/part71/45.png)

Normally we don't need access to the DB from outside of the VPC, so let's
choose "No" for public access.

Now, for the VPC security group, let's create a new one. I'm gonna name
it "db-access". We'll config this security group later to allow access
from the EKS cluster.

![](../images/part71/46.png)

The next thing we have to change is in the "Additional configuration" 
section, where we have to enter the initial database name, let's say
"simple_bank".

![](../images/part71/47.png)

If you don't specify this, Amazon won't create a database for you.

The rest of the options are for backup, encryption, and maintenance. At 
the end, it shows the estimated monthly cost for this DB instance.

![](../images/part71/48.png)

But if you're on a free tier, it should be free if you keep your usage
under the limit.

Let's click "Create database" to submit the form.

![](../images/part71/49.png)

We need to wait a few minutes for the DB to be ready. Once the DB is 
successfully created, we can view its connection details.

![](../images/part71/50.png)

![](../images/part71/51.png)

The username is "root" as we set earlier. But for the password, we have to 
go to Secrets Manager service to see it. Press this "Retrieve secret 
value", and you'll see the username and password.

![](../images/part71/52.png)

![](../images/part71/53.png)

If you still remember, in one of the previous lecture, when building the
Docker image on GitHub action, we fetch the secret values from this 
"simple-bank" secrets,

![](../images/part71/54.png)

and create the `app.env` file containing all required environment 
variables.

Now we'll have to set a new value for the DB source.

![](../images/part71/55.png)

It must look like this value

```
DB_SOURCE=postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable
```

we're using to connect to a local DB. But we'll have to replace the 
credentials and the endpoint of the DB. The username "root" is the same,
so no need to touch it. But the password contains a lot of special 
characters, which is not URL-safe, so we cannot just copy and paste it
to the `DB_SOURCE` string.

In the `golang-migrate` 
[GitHub repo](https://github.com/golang-migrate/migrate), they also mention
that any [reserved URL characters](https://github.com/golang-migrate/migrate#database-urls)
need to be escaped. And they even give us a simple Python script to do the
job. Let's copy the script, and run it in the terminal.

```shell
python3 -c 'import urllib.parse; print(urllib.parse.quote(input("String to encode: "), ""))'
```

Then, we can copy and paste in the DB password string

```shell
python3 -c 'import urllib.parse; print(urllib.parse.quote(input("String to encode: "), ""))'
String to encode: q()L$qa4*9d.%s!rbNGb612aGgF_
q%28%29L%24qa4%2A9d.%25s%21rbNGb612aGgF_
```

and the script will return a URL-safe version of it. All we have to do now
is, copy this string, and paste it to the DB_SOURCE. Next, we have to 
replace this "localhost" with the real DB's endpoint.

![](../images/part71/56.png)

You can find it in the `Connectivity and security` section on AWS RDS.

![](../images/part71/57.png)

The default port is the same, so we don't need to change it. The DB name
is also the same, "simple_bank" as we set it when creating the instance.
But we should remove the argument at the end that disable SSL, to make it
more secured to communicate with the DB.

```
DB_SOURCE=postgresql://root:q%28%29L%24qa4%2A9d.%25s%21rbNGb612aGgF_@simple_bank-db.ckm4thhx3nyn.eu-west-1.rds.amazonaws.com:5432/simple_bank
```

OK, next let's learn how to allow the EKS cluster to access the DB. Remember
we've created the "db-access" security group before? Let's open it, and
check the `Inbound rules`.

![](../images/part71/58.png)

![](../images/part71/59.png)

For now there's only 1 rule that allow access from my current IP address.
But I don't plan to access the DB directly from my home, so let's delete 
this one.

![](../images/part71/60.png)

Then, let's create a new rule. And for the type, let's select PostgreSQL.
It will fill in the default port `5432` automatically. For the `Source`, 
choose "Custom", and in the next box, we'll select the security group
of our `simple-bank` EKS cluster. So this rule basically means that we'll
allow any resources in the EKS cluster to connect to the DB.

![](../images/part71/61.png)

Alright, now we have the DB fully configured, the next step would be 
creating Redis using `Amazon ElastiCache`.

## Creating Redis

To remind, we have an async worker that sends email to users in the 
background. And it uses Redis as a message queue to store the async tasks.

![](../images/part71/62.png)

![](../images/part71/63.png)

![](../images/part71/64.png)

OK, I'm gonna configure and create a new cluster.

![](../images/part71/65.png)

You can choose to enable or disable cluster mode. Enabling cluster mode 
is better in term of scalability and availability. For demo purpose, I'm
gonna leave it as disabled. Let's name this cluster "simple-bank-redis",
and write a simple description for it.

![](../images/part71/66.png)

We can keep the default engine version, the port, and parameter groups.
But I'm gonna change the node type to `t3.micro`, because it's included
in the free tier limit. Let's also reduce the number of replicas to 1 
for now.

![](../images/part71/67.png)

Next, we have to create a new subnet group, I'm gonna set its name to
"redis-subnets", and make sure it stays in the same VPC as our EKS 
cluster.

![](../images/part71/68.png)

OK, next, there are some advanced settings, for security, backup, 
maintenance, logs, and tags. If you don't need to change anything, just
skip them.

![](../images/part71/69.png)

In the last step, let's review all the configurations. OK, everything
looks good, except for 1 thing. I forgot to create a security group for
the Redis cluster.

![](../images/part71/70.png)

This is important to enable access from EKS cluster. So let's go back to
this "Advanced Settings" section to add it.

![](../images/part71/71.png)

But, we have to create the security group first, using the EC2 service. 
Let's create a new one, named "redis-access", with any description you 
want.

![](../images/part71/72.png)

Make sure to select the same VPS as our EKS cluster.

Then, I'm gonna add a new "Inbound rules", with port 6379 - the default 
port of Redis. For the "Source", we'll select the security group of our
simple-bank EKS cluster, just like what we did for the "db-access" security
group before.

![](../images/part71/73.png)

So with this, it will allow any resources in the EKS cluster to connect 
to Redis.

![](../images/part71/74.png)

Now as the security group has been created, let's go back to the settings,
click "Manage" and refresh the list. Tick the box of "redis-access" 
security group and click "Choose".

![](../images/part71/75.png)

So now, we're good to go. Let's create the Redis cluster.

![](../images/part71/76.png)

As usual, it will take a few minutes to finish. Once it's done, we'll 
be able to go to this page and see the endpoints of the Redis cluster.

![](../images/part71/77.png)

Actually there are 2 endpoint, 1 for read only and the primary one for
both read and write. We're gonna copy the primary endpoint, and paste
it to the environment variable lists in "Secrets manager".

![](../images/part71/78.png)

And that should be it!

All the infrastructures including the EKS cluster, the DB, and Redis 
are now ready. In the next lecture, we'll learn how to use them to deploy
"Simple Bank" server to AWS.

I hope you enjoyed this video. Thanks a lot for watching, happy learning
and see you in the next lecture!
