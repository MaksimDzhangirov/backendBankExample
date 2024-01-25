# Grant AWS EKS cluster access to Postgres and Redis using security group

[Original video](https://www.youtube.com/watch?v=Pd7aeh014nU)

Hello everyone, welcome to the backend master class! In the previous 
lecture, we've prepared the AWS infrastructures including Postgres, Redis,
and the EKS cluster. So in this lecture, let's learn how to deploy our gRPC
and HTTP web server to this AWS infra.

## Deploy gRPC and HTTP web server to AWS

As you know, we've using GitHub actions to perform the deployment 
automatically, so I've prepared the `github-ci` user with access to the 
Secrets manager, the ECR, and our EKS cluster.

[](../images/part72/1.png)

[](../images/part72/2.png)

You can watch the videos in section 3 of the course if you don't know 
how to do that.

And since those videos are recorded a long time ago, I don't keep the 
same AWS account since then, that's why I have to recreate a completely 
new one, and thus, now I'll have to copy this new `github-ci` user's ARN,
to replace the old one in the `aws-auth.yaml` file.

```yaml
apiVersion: v1 
kind: ConfigMap 
metadata: 
  name: aws-auth 
  namespace: kube-system 
data: 
  mapUsers: | 
    - userarn: arn:aws:iam::760486049168:user/github-ci
      username: github-ci
      groups:
        - system:masters
```

Also note that, if you want to interact with the new EKS cluster using 
your terminal, you'll have to set up an access key for your root account
on AWS, because only the account that creates the EKS cluster can access
it at first.

[](../images/part72/3.png)

As you can see here, I have put my root account's access key

[](../images/part72/4.png)

in the AWS credentials file as "default" profile, and the credentials of
the `github-ci` profile underneath.

Now let's run this

```shell
kubectl apply -f eks/aws-auth.yaml
```

to deploy the `aws-auth.yaml` file to EKS.

[](../images/part72/5.png)

That would give the `github-ci` user permissions to access the EKS cluster
for automatic deployment.

Next, I also have to change the path to the new ECR repository, where 
the docker images of simple-bank will be pushed to.

[](../images/part72/6.png)

Let's copy this URI of the repo, and replace the old one in the 
`deployment.yaml` file.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: simple-bank-api-deployment
  labels:
    app: simple-bank-api
spec:
  replicas: 2
  selector:
    matchLabels:
      app: simple-bank-api
  template:
    metadata:
      labels:
        app: simple-bank-api
    spec:
      containers:
      - name: simple-bank-api
        image: 760486049168.dkr.ecr.eu-west-1.amazonaws.com/simplebank:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 8080
          name: http-server
        - containerPort: 9090
          name: grpc-server
```

Alright, now let's see how to deploy our web server to the cluster.

To remind, our web server will serve both gRPC and HTTP requests from 
client. And right now, they are set up to run on different ports. So
first, in the Dockerfile let's expose both ports: 8080 for the HTTP server,
and 9090 for the gRPC server.

```
# Builds stage
FROM golang:1.21-alpine3.18 AS builder
WORKDIR /app
COPY . .
RUN go build -o main main.go

# Run stage
FROM alpine3.16
WORKDIR /app
COPY --from=builder /app/main .
COPY app.env .
COPY start.sh .
COPY wait-for.sh .
COPY db/migration ./migration

EXPOSE 8080 9090
CMD ["/app/main"]
ENTRYPOINT ["/app/start.sh"]
```

This should be reflected in the `deployment.yaml` file as well. So here,
I'm gonna set a name for the first port `8080` as "http-server". Then, 
let's duplicate this, and change the port number to `9090` and its name 
to "grpc-server".

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: simple-bank-api-deployment
  labels:
    app: simple-bank-api
spec:
  replicas: 2
  selector:
    matchLabels:
      app: simple-bank-api
  template:
    metadata:
      labels:
        app: simple-bank-api
    spec:
      containers:
      - name: simple-bank-api
        image: 760486049168.dkr.ecr.eu-west-1.amazonaws.com/simplebank:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 8080
          name: http-server
        - containerPort: 9090
          name: grpc-server
```

OK, now we'll have to update the `service.yaml` file to route HTTP and 
gRPC traffics to different server ports. For the HTTP traffics that go
to port `80`, they will be routed to the target port `8080`. Of, if we use 
the name defined in the deployment, it should be "http-server".

```yaml
apiVersion: v1
kind: Service
metadata:
  name: simple-bank-api-service
spec:
  selector:
    app: simple-bank-api
  ports:
    - protocol: TCP
      port: 80
      targetPort: http-server

  type: ClusterIP
```

Before we declare the gRPC port, we should give this one a name. Let's 
call it "http-service" since it is a service port that will receive HTTP
traffics.

```yaml
...
- protocol: TCP
    port: 80
    targetPort: http-server
    name: http-service
```

Now, let's duplicate the port definition, then change the port number to 
another value, such as `90`. The target port will be "grpc-server", as
defined in the deployment. And the name of this service port should be
"grpc-service", as it will receive gRPC traffics. 

```yaml
...
- protocol: TCP
    port: 90
    targetPort: grpc-server
    name: grpc-service
```

Next, we have to update the Ingress, so that it can route both HTTP and
gRPC requests to the correct service port. In order to do that, we'll 
actually need to deploy 2 different Ingresses. 

So, I'm gonna create a new file called `ingress-http.yaml`. And copy 
this setting over to the new file.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: simple-bank-ingress
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt
spec:
  ingressClassName: nginx
  rules:
    - host: "api.simple-bank.org"
      http:
        paths:
          - pathType: Prefix
            path: "/"
            backend:
              service:
                name: simple-bank-api-service
                port:
                  number: 80
  tls:
    - hosts:
        - api.simple-bank.org
      secretName: simple-bank-api-cert
```

Let's rename it to "simple-bank-ingress-http". Now, for the host. The old
`simple-bank.org` domain has expired, and I no longer own it. So I have
bought a completely new domain: "simplebanktest.com" in the AWS Route53
service.

AWS also created a hosted zone for it here.

![](../images/part72/7.png)

Later, we'll use this page

![](../images/part72/8.png)

to set up an A record that route requests to our Ingress load balancer.

For now, let's just copy the domain name and paste it to the yaml file,
to replace the old domain.

```yaml
spec:
  ingressClassName: nginx
  rules:
    - host: "api.simplebanktest.com"
      http:
        paths:
          - pathType: Prefix
            path: "/"
            backend:
              service:
                name: simple-bank-api-service
                port:
                  number: 80
  tls:
    - hosts:
        - api.simplebanktest.com
      secretName: simple-bank-api-cert
```

Everything else can be kept the same. The port `80` is the port of the 
service that we're using to receive HTTP request.

Next, let's create a new "ingress-grpc.yaml" file for the gRPC Ingress.
I'm gonna copy and paste the content of the HTTP Ingress to this new
file.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: simple-bank-ingress-http
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt
spec:
  ingressClassName: nginx
  rules:
    - host: "api.simplebanktest.com"
      http:
        paths:
          - pathType: Prefix
            path: "/"
            backend:
              service:
                name: simple-bank-api-service
                port:
                  number: 80
  tls:
    - hosts:
        - api.simplebanktest.com
      secretName: simple-bank-api-cert
```

Then rename it to "simple-bank-ingress-grpc". And we're gonna use a 
different sub-domain for gRPC. The HTTP requests are routed to 
`api.simplebanktest.com`. So let's send the gRPC requests to 
`gapi.simplebanktest.com`. Letter "g" here stands for gRPC. We should 
also change the TLS secret name to `simple-bank-gapi-cert`. And the
port number of the gRPC service should be `90`, as we defined before.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: simple-bank-ingress-grpc
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt
spec:
  ingressClassName: nginx
  rules:
    - host: "gapi.simplebanktest.com"
      http:
        paths:
          - pathType: Prefix
            path: "/"
            backend:
              service:
                name: simple-bank-api-service
                port:
                  number: 90
  tls:
    - hosts:
        - gapi.simplebanktest.com
      secretName: simple-bank-gapi-cert
```

OK, now we have to add some important annotations to this gRPC Ingress. 
You can find them in the gRPC example of the [Ingress Nginx documentation
page](https://kubernetes.github.io/ingress-nginx/examples/grpc/#step-3-create-the-kubernetes-ingress-resource-for-the-grpc-app).
There is 1 annotation for the SSL redirect, which should be set to "true"
and another annotation for the backend protocol, which should be "GRPC".

```yaml
metadata:
  name: simple-bank-ingress-grpc
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/backend-protocol: "GRPC"
```

And that's basically it. We don't have to touch anything else in this file.
I'm gonna rename the original `ingress.yaml` file to `ingress-nginx.yaml`,
since it's only used to deploy the "nginx" Ingress class. There's one more
change we must do in the `issuer.yaml` file. It's how we declare the 
Ingress class for the HTTP resolver. To remind, this is used to 
automatically generate LetsEncrypt TSL certificates. It seems the API has
changed since the last time, so now we have to write: 
"ingressClassName: nginx" like this.

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt
spec:
  acme:
    email: techschool.guru@gmail.com
    server: https://acme-v02.api.letsencrypt.org/directory
    privateKeySecretRef:
      # Secret resource that will be used to store the account's private key.
      name: letsencrypt-account-private-key
    # Add a single challenge solver, HTTP01 using nginx
    solvers:
      - http01:
          ingress:
            ingressClassName: nginx
```

Alright, now let's head over to the `deploy` GitHub workflow. We can keep
most content of this file untouched. But one thing we must change is: the
command to fetch the config of the cluster. The name of the new EKS
cluster should be "simple-bank-eks".

```yaml
 - name: Update kube config
   run: aws eks update-kubeconfig --name simple-bank-eks --region eu-west-1
```

By the way, in the past, several students have told me that sometimes
the `deployment.yaml` is not getting deployed when there's nothing change
in its content. In that case, we can use the `kubectl rollout restart`
command to force the deployment.

```yaml
- name: Deploy image to Amazon EKS
    run: |
      kubectl apply -f eks/aws-auth.yaml
      kubectl rollout restart -f eks/deployment.yaml
``` 
    
So let's give it a try here. Finally, I'm gonna change the branch to 
"master" instead of "release", so that the deploy flow can be run every 
time we push to "master".

```yaml
on:
  push:
    branches: [ master ]
```

Then, in the terminal, let's add all changes, 

```shell
git add .
```

create a Git commit with message "deploy grpc and http to eks"

```shell
git commit -m "deploy grpc and http to eks"
```

and push the change to GitHub.

```shell
git push origin master
```

Now, if we open [simple bank repository](https://github.com/techschool/simplebank),
we'll see that the `deploy` workflow is now running, and the docker image
is being built.

![](../images/part72/9.png)

This will take a while to complete, so while waiting for it, I just 
remember that we still need to install some tools for the EKS cluster.

First, the [Ingress Nginx controller](https://github.com/kubernetes/ingress-nginx?tab=readme-ov-file#ingress-nginx-controller).
Let's follow [this link](https://kubernetes.github.io/ingress-nginx/deploy/) 
on the documentation page, and select the [AWS provider](https://kubernetes.github.io/ingress-nginx/deploy/#aws).
Here you'll find the "kubectl apply" command to deploy the package. Let's 
copy and run it in the terminal.

```shell
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/aws/deploy.yaml
```

![](../images/part72/10.png)

The second package we must deploy is the [cert-manager](https://cert-manager.io/docs/).
In its documentation page, the installation section, let's choose [kubectl apply](https://cert-manager.io/docs/installation/kubectl/).
Then copy this command,

```shell
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.1/cert-manager.yaml
```

and run it in the terminal.

![](../images/part72/11.png)

OK, now check the `k9s` console,

![](../images/part72/12.png)

we should see the `cert-manager` and `ingress-nginx-controller` pods.

Let's check the GitHub Action workflow to see how it goes.

![](../images/part72/13.png)

OK, so we've got an error when deploying the image to EKS. It says 
"simple-bank-api-deployment" not found. So it seems that when we use the
"kubectl rollout restart" command, it expects that the deployment already
existed. But since we're deploying the app for the first time, there's 
no existing deployment. Let's change this back to "kubectl apply"

```yaml
kubectl apply -f eks/deployment.yaml
```

and try again. By the way, for future reference, I'm gonna create an 
`install.sh` file inside the `eks` folder. And add the 2 commands we used
to deploy `cert-manager` and `ingress-nginx` controller to that file.

```
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/aws/deploy.yaml
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.1/cert-manager.yaml
```

OK, now let's add all changes, create a new commit, and push ot to GitHub
again.

```shell
git add .
git commit -m "use kubectl apply"
git push origin master
```

Let's wait a bit for the `deploy` workflow to run. It still failed, but
this time we've got a different error: "ingress.yaml" file doesn't exist. 

![](../images/part72/14.png)

Indeed, we've splitted that file into 3 different `yaml` files, but I
forgot to update this. So let's go back to the `deploy.yaml` file, and
change this file name to `ingress-nginx.yaml`. Then make 2 more clone of
this line, and rename the first file to `ingress-http.yaml` and the 
second file to `ingress-grpc.yaml`.

```yaml
kubectl apply -f eks/ingress-nginx.yaml
kubectl apply -f eks/ingress-http.yaml
kubectl apply -f eks/ingress-grpc.yaml
```

I hope this time it will work. Let's add everything, create a new commit:
"update ingress yaml file names" and push it to GitHub one more time.

```shell
git add .
git commit -m "update ingress yaml file names"
git push origin master
```

The workflow will take a bit of time to run.

![](../images/part72/15.png)

And finally, it has completed successfully. 

Let's check the service in `k9s` console.

![](../images/part72/16.png)

The `simple-bank-api-service` is up and running here. I'm gonna view its
logs to see if there are any errors.

![](../images/part72/17.png)

There are no error logs here, so it means the server has successfully 
connected to both DB and Redis, and it is working properly. Awesome! Now
let's check the list of Ingresses.

![](../images/part72/18.png)

There are 4, but we only know 2 of them, which is the Ingress for serving
HTTP and GRPC requests. The remaining 2 Ingresses are automatically
created by cert-manager for ACME challenge resolver, so that LetsEncrypt
can verify that we really own the domain, and issue the TLS certificate
for the domain. You can notice that all these 4 Ingresses share the same
address,

![](../images/part72/19.png)

so this is the address that we should put in the A-record of the domain.

Only after we've done that, the call from LetsEncrypt will be able to 
reach our EKS cluster and then the TLS certificate can be approved and 
available to use.

So let's open the `Hosted Zones` page in the AWS Route 53 service, and 
click "Create record".

![](../images/part72/20.png)

![](../images/part72/21.png)

First record, let's set it to the HTTP subdomain: "api.simplebanktest.com", 
enable the "Alias" switch, and choose the "Network Load Balancer" endpoint.
Select the region of our EKS cluster, which is "eu-west-1" (Ireland). Then
in the last box, paste in the address of our Ingress.

![](../images/part72/22.png)

Similarly, let's add another record for the gRPC subdomain:
"gapi.simplebanktest.com". Switch on the "Alias", choose 
"Network Load Balancer", region "eu-west-1", paste in the address of the
Ingress, and finally click "Create record".

![](../images/part72/23.png)

OK, so now the 2 A-records have been created.

![](../images/part72/24.png)

But keep in mind that it might take a while for LetsEncrypt to pick it up, 
so the certificates won't be available right away. As you can see here,
2 certificates are created, but neither is ready.

![](../images/part72/25.png)

So, just wait patiently until they are ready. One thing we can check is,
we can run `nslookup` the subdomains to make sure they are reachable.

```shell
nslookup api.simplebanktest.com
nslookup gapi.simplebanktest.com
```

If the response contains the IP addresses like this, then it's all good.

![](../images/part72/26.png)

We can check further by describing the ACME resolver Ingress.

![](../images/part72/27.png)

Here, we'll be able to see the challenge file it is serving to 
LetsEncrypt.

Let's copy this path, and try to `curl` it from the terminal.

```shell
curl http://api.simplebanktest.com/.well-known/acme-challenge/ftRObCSbiBXkWapr6FnreU5VK895TpIoLj5214FtkbY
```

If it returns a string like this,

![](../images/part72/28.png)

then it is working well, and we can be confident that the certificate will
be ready soon.

Let's try to `curl` the challenge file of the `gapi` subdomain to see how
it goes.

```shell
curl http://gapi.simplebanktest.com/.well-known/acme-challenge/LNDeFKdU1UqfL9F09D96I9Iwtv2o5MaiHKk0KEcf9Wo
```

![](../images/part72/29.png)

OK, so for this one, we've got a 503 error: "service unavailable".

Even if you get something like this, don't panic, and just wait patiently
for a few minutes. From my experience, sometimes it can take 10 minutes 
or more to complete the domain verification.

OK, so after about 7 minutes, the ACME Ingress for the subdomain 
"api.simplebanktest.com" has disappeared. Only the one for the `gapi`
subdomain remains.

![](../images/part72/30.png)

Now, if we check the certificates, we will see that the 
`simple-bank-api-cert` is now ready.

![](../images/part72/31.png)

The certificate is up to date and has not expired. How about the `gapi`
certificate? Let's try to `curl` its ACME challenge file again.

![](../images/part72/32.png)

This time, the request succeeded. So I'm sure it will be available soon.
There's nothing to worry about.

In the mean time, let's test the HTTP server since its domain and TLS
certificate is ready.

In Postman, let's try to send a "Create user" request. Make sure you use
the HTTPS protocol here, follow by the `root_url` of the server.

![](../images/part72/33.png)

It is a variable, so we can easily modify it wherever we want. Let's change
it to our subdomain "api.simplebanktest.com".

![](../images/part72/34.png)

Then go back to the Create User request, and send it.

![](../images/part72/35.png)

Yee! The request is successful. User `techschool` has been successfully 
created. And in about 10 seconds, we should receive a welcome email from
the Simple Bank server.

![](../images/part72/36.png)

So the HTTP server is working well as expected.

Now let's check the `k9s` console again.

![](../images/part72/37.png)

This time, the ACME Ingress of the `gapi` domain is gone as well. So we
know that its certificate is now ready.

Let's go back to Postman and test the gRPC server. First, in the Simple
Bank gRPC collection, let's change the value of the `root_uri` variable
to "gapi.simplebanktest.com".

![](../images/part72/38.png)

Then, I'm gonna call the Login User RPC, for the `techschool` user we've
just created.

Note that we have to make sure it's pointing to the right domain, and that
the TLS is enabled using this Lock icon.

![](../images/part72/39.png)

We can also refresh the server reflection so see the list of available
methods, and choose LoginUser RPC in the list, then invoke the RPC.

![](../images/part72/40.png)

![](../images/part72/41.png)

It's successful. Awesome.

So now we can confirm that both the HTTP and gRPC service are working 
well.

We've successfully deployed our simple bank server to the AWS EKS cluster.
I hope you find the video interesting and useful. Thanks a lot for 
watching, happy learning, and see you in the next lecture!