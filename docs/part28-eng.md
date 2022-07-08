# How to create a production DB on AWS RDS

[Original video](https://www.youtube.com/watch?v=0EaG3T4Q5fQ)

Hello everyone and welcome to the backend master class. In this lecture, we 
will learn how to create a production database with AWS RDS.

## AWS RDS

RDS is a managed Relational Database Service on AWS. It supports several kinds
of relational database, and it is automatically deployed and managed by 
Amazon Web Service, so we don't have to care much about how to maintain or 
scale the DB cluster. It is also super easy to set up. Let's click this 
`Create database` button to create a new database!

![](../images/part28/1.png)

There are 2 creation methods: `Standard create` and `Easy create`. `Easy 
create` will use some recommended best-practice configuration for your DB,
while `Standard create` let you customize all configuration options. I'm
gonna use this method for this demo.

![](../images/part28/2.png)

Now we have to choose the DB engine type. As you can see, there are 6 different
types of engine. I'm gonna select `Postgres` for our simple bank project. For
the version, let's select `12.6`, because we're using `Postgres 12` for 
development.

Next, let's choose a config template. It can be `Production`, `Dev/Test` or 
`Free Tier`. For the demo purpose, I'm just gonna use `Free Tier`.

Next, we have to enter the name for our DB instance. Let's called it 
`simple-bank`. For the `Master username`, I'm gonna use `root` just like 
what we're using for development. Then for the password, I will check this box
(`Auto generate a password`) to let RDS auto generate a random one for me.

![](../images/part28/3.png)

Next, the `DB instance class`. As we selected the `Free Tier` template, there's
only 1 type of instance available: `db.t2.micro`. If you select `Production`
or `Dev/Test` template, then there will be many other bigger instance types
to choose.

Now for the storage we will have 20 GB of free SSD storage and there's also 
an option to enable autoscaling, which will allow the storage to increase once
the specified threshold is exceeded. We don't need it for our demo application,
so I'm gonna disable this feature.

![](../images/part28/4.png)

For production usage, you will also have the option to setup multi 
Availability Zone, or a stand-by instance in a different zone to provide data
redundancy and minimize latency. But as we're using `Free Tier`, this option
is not available. So let's move on!

Here in this `Connectivity` section, we will specify how we want our DB to 
be accessed. By default, the DB will be deployed inside the default virtual
private cloud (or VPC). We will learn more about VPC in another lecture. For
now let's just use this default one. DB `Subnet group` is a way to specify the
subnets and IP ranges the DB instance can use in the VPC. Let's leave it as 
default for now. Now comes one important setting. Do you want your DB to be
publicly accessible or not? If you choose `Yes`, then all the EC2 instances 
and devices outside of the VPC can connect to your DB. But we will have to 
setup a `VPC security group` to allow that. If you choose `No`, then RDS 
will not assign a public IP address to the DB, thus only EC2 instances 
and devices inside the same VPC can connect to it. For this lecture, I want 
to access the DB from my local computer, so I'm gonna choose `Yes`.

![](../images/part28/5.png)

This will ensure that the DB will have a public IP address, but to really have 
access to it from outside, we must setup a proper security group. There's 
already a default security group, but it doesn't give access to the DB's port 
from anywhere, so I'm gonna create a new one. Let's call it 
`access-postgres-anywhere`. Next the `Availability Zone`. You can choose 1 of 
these zones, or just leave it as `No preference`.

![](../images/part28/6.png)

Now some `Additional configurations`. We can optionally set a different port
for the DB, or just leave it as default port `5432`.

For database authentication, we're gonna use password authentication, so 
nothing to be changed here.

![](../images/part28/7.png)

Next, we have the option to create an initial database. If you don't specify 
a name here, Amazon RDS will not create a database for you. I want it to
create the `simple_bank` database for me, so I write its name in this box.

![](../images/part28/8.png)

No need to touch the `DB parameter group`. You can also `Enable automatic 
backup` for your DB. Or setup some monitoring and logging configurations.

Check this box if you want to enable auto minor version upgrade. And select
the DB maintenance window time if you want.

![](../images/part28/9.png)

If it is a production DB, you might also want to enable deletion protection.
It will help protect the DB from being deleted accidentally.

At the end of the form, we will see the estimated monthly costs of our DB 
instance. As we're using `Free Tier`, there are 750 hours of free usage, with
20 GB of general purpose storage. As well as 20 GB for automated backup 
storage and user-initiated snapshots. If you're using more than these 
thresholds, a standard service rate will be applied. You can read more 
about its pricing in this [link](https://aws.amazon.com/rds/pricing/).

OK, now let's click `Create database`.

![](../images/part28/10.png)

As you can see, the database is now being created. It will take a few minutes
to be completely ready. While waiting for it, let's click this button to see 
the credentials to access the DB.

![](../images/part28/11.png)

As I asked RDS to auto generate a random password for me before, here, in this
pop-up window, we can see and copy its `Master password`.

![](../images/part28/12.png)

Now I'm gonna open TablePlus and create a new connection to access our remote
DB. For the connection name, let's call it `AWS Postgres`. We don't have the
host URL yet, so let's leave it empty for now. The username should be `root`.
And let's paste in the password. The database name is `simple_bank`. And 
that's it.

![](../images/part28/13.png)

Now we just need to wait for the DB to be created and get its host URL. Let's
go back to the AWS console. Close this pop-up window and refresh the page.
It's still not ready yet. But here in the `Networking` table, we can see that
our DB is in the `eu-west-1b` zone, it's using the default VPC and subnet
group. One thing we must do now is to check the security group: 
`access-postgres-anywhere` that we've specified in the DB creation process. 

![](../images/part28/14.png)

If we follow the link of this security group, we can see that it has 1 inbound 
rule of type PostgreSQL.

![](../images/part28/15.png)

The protocol is `TCP`, and the port is `5432`. This `Source` IP is actually my
current IP address. I can quickly check that by searching for my IP in the 
browser. OK, so basically, this rule only allows my IP address to access
the DB at port `5432`. But my IP is not static, and I don't want to have to 
update this rule every time my IP changes. So I will change the `Source` to
`Anywhere`.

![](../images/part28/16.png)

This will make sure that all IPs can access the database. Of course, if it
is a production DB, you wouldn't want to expose it to the whole internet. 
Alright, let's save the rule, and go back to the RDS database page.

OK, now the database has been successfully created. 

![](../images/part28/17.png)

Let's check its connection details. This time, we have the endpoint to access
it. So let's copy this URL,

![](../images/part28/18.png)

and paste it to the `Host` of our new connection in TablePlus. Then 
click `Test`. All boxes are green. It means the connection is good.

![](../images/part28/19.png)

So let's click `Connect`. And voilà, we're now connected to the database. 
However, at the moment, it is completely empty. That's because we haven't
run DB migration to create the tables yet.

![](../images/part28/20.png)

## Run DB migrations

So let's do that now. In our simple bank project's repo, let's open the
Makefile. There's a `migrate up` command that we're using to migrate DB 
locally.

```makefile
migrateup:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up
```

All we have to do now is to change this URL 
`postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable` to 
the remote DB's URL. The username is still `root`, but the password is 
different. So let's copy it from the AWS console, and paste it here. The
`localhost` should also be changed to the remote host, that AWS RDS gives us
in the console. The port and DB name are the same, so we don't have to change
them. But we should remove the `sslmode=disable` parameter, because we're
connecting to a remote DB, so it's better to use a secured connection. OK, 
I think that will work.

```makefile
migrateup:
    migrate -path db/migration -database "postgresql://root:tupExr0Gp4In4Ww4WHKR@simple-bank.czutruo2pa5q.eu-west-1.rds.amazonaws.com:5432/simple_bank" -verbose up
```

Let's save the file, then open the terminal, and run `make migrate up`.

![](../images/part28/21.png)

All successful! So let's open TablePlus to check the DB. I'm gonna refresh
it. As you can see, all tables are successfully created.

![](../images/part28/22.png)

Awesome! So now you know how to set up a managed database instance using
AWS RDS. I hope you find it useful.

Thanks a lot for watching! Happy learning and see you in the next lecture.