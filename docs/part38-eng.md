# Generate DB documentation page and schema SQL dump from DBML

[Original video](https://www.youtube.com/watch?v=dGfVwsPr-IU)

Hello guys, welcome to the backend master class!

If you still remember, in the first lecture of the course, we've
learned about dbdiagram.io, a free service built by Holistics that
helps us design the database using DBML, or database markup 
language, visualize the entity-relationship diagrams of the table,
and then automatically generate and export SQL codes of the 
schema.

Recently, I just found out that Holistic has built another super 
cool service called [dbdocs](https://dbdocs.io/).

## Creating db documentation using dbdocs

This service helps us create a beautiful web-based database 
documentation using the same DBML file that we wrote before.
Although it's nothing advanced, it is so cool that I want 
to show you how to use it in our simple bank project. I'm sure
it will make your development much more fun and productive.

So, in the lecture, we'll learn how to generate both the DB 
documentation page as well as the SQL schema codes directly 
from the terminal using `dbdocs` and DBML's CLI tool.

First, let's open [dbdocs.io](https://dbdocs.io/) website and
access the `Get Started` [page](https://dbdocs.io/docs). In order
to install `dbdocs` tool, we need to have `NodeJS` and `npm` on our 
machine.

As I'm on a Mac, I will install them with `Homebrew`.

```shell
brew install node
```

After it is installed successfully, we can run `node -v` and `npm -v` 
to see their versions.

Alright, now we can install `dbdocs` with this command:

```shell
npm install -g dbdocs 
```

Then let's run `dbdocs` command to check that it has been installed
correctly.

```shell
dbdocs
```

![](../images/part38/1.png)

Here we can also see some available commands to help us build the 
docs or managing the login and password of the documentation page.

The next step is to define the schema using DBML, which we've already
done in the first lecture of the course.

So I'm just copy its content from [dbdiagram.io](https://dbdiagram.io/home).
Then in our simple bank's repo, I'm gonna create a new folder called
`doc` and create a new file called `db.dbml` inside this folder. Then
let's paste in the schema's DBML code. Right now there's no syntax
highlighting yet.

In Visual Studio Code, we can look for `dbml` and install this 
`vscode-dbml` extension.

![](../images/part38/2.png)

After it is installed, go back to the DBML file, we can see that code
highlighting has been enabled.

We can also add some more metadata and description about the project.
For example, I'm gonna change this name to `Simple Bank Database`.

```
Project first_project {
  database_type: 'PostgreSQL'
  Note: '''
    # Simple Bank Database
    **markdown content here**
  '''
}
```

You can also write more details in markdown here if you want. For
now I'm just gonna remove it.

```
Project first_project {
  database_type: 'PostgreSQL'
  Note: '''
    # Simple Bank Database
  '''
}
```

OK, now with this DBML file, we can generate the DB documentation
page. But first, we need to login to `dbdocs`.

To do that, we just have to run `dbdocs login` command in the 
terminal and provide an email address for authentication. So let's 
do it!

```shell
dbdocs login
Choose a login method:
  1) Email
  2) GitHub
  Answer:
```

We can choose to login with either an email or a GitHub account. I'm
gonna choose email.

Then enter the email address: `techschool.guru@gmail.com`.

Now it says `Request email authentication`, we will have to check the
email they sent and input the provided OTP here.

So I'm gonna open Gmail. Here's the email from [dbdocs.io](https://dbdocs.io/).
Let's copy this OTP, and paste it to the terminal.

![](../images/part38/3.png)

Then voilà, we've successfully logged in to the account. Now we can
generate the DB documentation page using the `dbdocs build` command.

It only needs 1 argument, which is the location of the DBML file:
`doc/db.dbml` in our case.

```shell
dbdocs build doc/db.dbml
```

![](../images/part38/4.png)

The page has been successfully generated. However, there are 2 issues:
first, the page is not yet protected by a password yet, so anyone
can access it. And second, its name is not very good: `first_project`.

That's because I forgot to change the example's project name.

So let's go back to the code and change the project name to 
`simple_bank`.

```
Project simple_bank {
  database_type: 'PostgreSQL'
  Note: '''
    # Simple Bank Database
  '''
}
```

Now if we run `dbdocs build` again, a new page will be created with
the correct name: `simple_bank`, exactly as we wanted it to be.

![](../images/part38/5.png)

We can access it via this [link](https://dbdocs.io/techschool.guru/simple_bank).

As you can see, there's no password protection yet at the moment.

![](../images/part38/6.png)

But the page looks really clean and well-organized! There's some
metadata and description about the project at the top. Then it
shows some summary, such as the number of tables and fields. And
at the bottom, we can see the list of the tables in the schema,
which we can expand to view their diagrams and relationships with
other tables.

We can see more details about each table by selecting them on the
left-hand side menu.

![](../images/part38/7.png)

On the right, it shows all the fields in this table, together with 
their types, settings, references, default values, and indexes. We
can also see the whole ER diagram of the database on this 
`Relationships` tab.

![](../images/part38/8.png)

Or we can copy this link to share it with other members of the team.

![](../images/part38/9.png)

But of course, we don't want to everyone on the internet to know our
internal DB design, right?

So let's learn how to set up a password to protect our documentation
page. It's actually pretty simple.

As you can [see](https://dbdocs.io/docs/password-protection), we just 
need to use the `dbdocs password` command.

![](../images/part38/10.png)

So let's copy it, and paste it to the terminal.

```shell
dbdocs password --set <password> --project <project name>
```

I'm gonna change this password value to "secret" and the project name
should be "simple_bank".

```shell
dbdocs password --set secret --project simple_bank
Password is set for 'simple_bank'
```

Then voilà, the password has been set for our project. Now, if we
visit [this page](https://dbdocs.io/techschool.guru/simple_bank) again,
it will say "protected project", and we have to enter the correct 
password to enter the page.

![](../images/part38/11.png)

Very cool and so easy to set up, right?

On the page, there's also a button to allow us to login with email
or GitHub.

![](../images/part38/12.png)

![](../images/part38/13.png)

It's the same as what we've done before in the terminal. Just enter
email here, click `Send`. Then open Gmail to get the OTP. Copy the 
OTP from this email, and paste it into this box.

Now we've successfully logged in.

![](../images/part38/14.png)

We can open this menu and select this to see all of our projects.

![](../images/part38/15.png)

As you can see, we're having 2 projects at the moment: `simple_bank`
and the `first_project`, which I created by mistake before.

![](../images/part38/16.png)

I want to delete the `first_project` to keep this page clean, so
I'm gonna run the `dbdocs remove` command in the terminal, and pass
in the name of that project.

```shell
dbdocs remove first_project
Removed successfully
```

OK, it's been removed successfully.

Now if we go back to the documentation list page, we only see 1 
project: `simple_bank`.

![](../images/part38/17.png)

So it works! Excellent!

Now you know how to generate a DB documentation page from DBML.
Next, let's learn how to generate the SQL codes directly in the
terminal as well. It will be very useful for us to use in the 
DB migration script.

If we go back to the [dbdiagram](https://dbdiagram.io/home) website, 
in the `dbx` menu, beside `dbdocs`, there's also a link to the `dbml`
[page](https://www.dbml.org/).

![](../images/part38/18.png)

In this page, let's open [command-line tool](https://www.dbml.org/cli/) 
(CLI), then scroll down a bit, there will be an instruction on how
to [install the tool](https://www.dbml.org/cli/#installation).

All we need to do is to run 

```shell
npm install -g @dbml/cli
```

in the terminal.

![](../images/part38/19.png)

OK, the `dbml` command line tool is successfully installed.

Here's how we can use it to generate SQL codes: just run `dbml2sql`,
then the database engine, which is Postgres in our case. Then `-o`, 
followed by the path of the output file, for which I'm gonna put
`doc/schema.sql` and finally, the path to the `dbml` file, which is
`doc/db.dbml`.

```shell
dbml2sql --postgres -o doc/schema.sql doc/db.dbml
Generated SQL dump file (PostgreSQL): schema.sql
```

That's it! The SQL dump file has been generated. We can check it out
in Visual Studio Code.

![](../images/part38/20.png)

![](../images/part38/21.png)

Here it is, the `schema.sql` file under the `doc` folder. So it
worked perfectly!

## Updating Makefile

Before we finish, I'm gonna update the Makefile to include 2 new 
commands. The first one is `db_docs`, which will be used to build the
database documentation page.

```makefile
db_docs:
	dbdocs build doc/db.dbml
```

And the second one is `db_schema`, which should be used to generate 
the Postgres SQL dump of the database schema.

```makefile
db_schema:
	dbml2sql --postgres -o doc/schema.sql doc/db.dbml
```

I'm gonna add `db_docs` and `db_schema` to the PHONY list as well.
Alright, now whenever we want to change the DB schema, then after 
updating the DBML file, we just need to run

```shell
make db_docs
```

to regenerate the documentation page, and

```shell
make db_schema
dbml2sql --postgres -o doc/schema.sql doc/db.dbml
Generated SQL dump file (PostgreSQL): schema.sql
```

to regenerate the SQL schema dump file.

Very easy and convenient, isn't it?

And by the way, in the `Makefile`, I notice that these are some 
duplications with the database URL.

We can improve this a bit by replacing them with a variable.
Let's call it `DB_URL`.

```makefile
migrateup:
	migrate -path db/migration -database "$(DB_URL)" -verbose up

migrateup1:
	migrate -path db/migration -database "$(DB_URL)" -verbose up 1

migratedown:
	migrate -path db/migration -database "$(DB_URL)" -verbose down

migratedown1:
	migrate -path db/migration -database "$(DB_URL)" -verbose down 1
```

Note that here I use a `$` symbol, and a pair of parentheses to get 
the value of the variable. Then now, we can define this variable at
the top of the `Makefile`.

```makefile
DB_URL=postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable
```

Just like that, and we're done!

Let's try to run

```shell
make migratedown
```

in the terminal.

Then 

```shell
make migrateup
```

![](../images/part38/22.png)

All successful! Perfect!

There's one more thing! When we use DBML CLI to generate the SQL 
codes, it will also generate a `dbml-error.log` file at the root
of the project.

![](../images/part38/23.png)

This file contains the error logs that will tell you if there's 
any problem with your DBML script or not. In our case, we don't 
have any errors, so the content of the file is empty. But in any
case, we never want to commit this log file to GitHub, right?

So let's create a `.gitignore` file at the root, and put `*.log`
inside the file to ignore all log files.

OK, now let's create a new branch called `ft/dbdocs`,

```shell
git checkout -b ft/dbdocs
```

add all the changes,

```shell
git add .
```

check it again,

```shell
git status
```

and commit it with a message saying "add dbdocs".

```shell
git commit -m "add dbdocs"
```

Finally, we can push it to GitHub

```shell
git push origin ft/dbdocs
```

and use this link `https://github.com/techschool/simplebank/pull/new/ft/docs`
to create a new pull request.

After the unit tests passed, we can safely merge it to `master` and 
delete the feature branch.

![](../images/part38/24.png)

![](../images/part38/25.png)

![](../images/part38/26.png)

And that wraps up this short lecture about generating DB documentation
page and SQL schema dump codes from the terminal.

I hope it was interesting and useful for you.

Thanks a lot for watching and see you in the next lecture!