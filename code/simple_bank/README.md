[Original repository](https://github.com/techschool/simplebank)

## Simple bank service

The service that we’re going to build is a simple bank. It will provide APIs 
for the frontend to do following things:

1. Create and manage bank accounts, which are composed of owner’s name, 
   balance, and currency.
2. Record all balance changes to each of the account. So every time some 
   money is added to or subtracted from the account, an account entry record 
   will be created.
3. Perform a money transfer between 2 accounts. This should happen within 
   a transaction, so that either both accounts’ balance are updated 
   successfully or none of them are.

## Setup local development

### Install tools

* [Docker desktop](https://www.docker.com/products/docker-desktop)

* [TablePlus](https://tableplus.com/)

* [Golang](https://golang.org/)

* [Homebrew](https://brew.sh/)

* [Migrate](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate)

```bash
brew install golang-migrate
```

* [DB Docs](https://dbdocs.io/docs)

```
npm install -g dbdocs
dbdocs login
```

* [DBML CLI](https://www.dbml.org/cli/#installation)

```shell
npm install -g @dbml/cli
dbml2sql --version
```

* [Sqlc](https://github.com/kyleconroy/sqlc#installation)

```shell
brew install sqlc
```

* [Gomock](https://github.com/golang/mock)

```shell
go install github.com/golang/mock/mockgen@v1.6.0
```

### Setup infrastructure

* Create the bank-network

```shell
make network
```

* Start postgres container:

```shell
make postgres
```

* Create simple_bank database:

```shell
make createdb
```

* Run db migration up all versions:

```shell
make migrateup
```

* Run db migration up 1 version:

```shell
make migrateup1
```

* Run db migration down all versions:

```shell
make migratedown
```

* Run db migration down 1 version:

```shell
make migratedown1
```

### Documentation

* Generate DB documentation:

```shell
make db_docs
```

* Access the DB documentation at [this address](https://dbdocs.io/techschool.guru/simple_bank). Password: `secret`

### How to generate code

* Generate schema SQL file with DBML:

```shell
make db_schema
```

* Generate SQL CRUD with sqlc:

```shell
make sqlc
```

* Generate DB mock with gomock:

```shell
make mock
```

* Create a new db migration:

```shell
make new_migration name=<migration_name>
```

### How to run

* Run server:

```shell
make server
```

* Run test:

```shell
make test
```

### Deploy to kubernetes cluster

* [Install nginx ingress controller:](https://kubernetes.github.io/ingress-nginx/deploy/#aws)

```shell
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v0.48.1/deploy/static/provider/aws/deploy.yaml
```

* [Install cert-manager:](https://cert-manager.io/docs/installation/kubernetes/)

```shell
kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.4.0/cert-manager.yaml
```
