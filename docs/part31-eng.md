# How to use kubectl & k9s to connect to a Kubernetes cluster on AWS EKS

[Original video](https://www.youtube.com/watch?v=hwMevai3_wQ)

Hello everyone and welcome back to the backend master class!

In the [previous lecture](part30-eng.md), we've learned how to set up a 
Kubernetes cluster using AWS EKS.

As you can see here, our simple bank cluster is already up and running.

![](../images/part31/1.png)

So today I'm gonna show you how to connect to this Kubernetes cluster 
using 2 command line tools: `kubectl` and `k9s`.

## Command line tool - `kubectl`

First, let's search for `kubectl` in the browser. Here it is, let's open
this [page](https://kubernetes.io/docs/tasks/tools/). So `kubectl` is
Kubernetes command-line tool, which allows you to run command against 
the Kubernetes clusters. You can use it to deploy applications, inspect
and manage cluster resources and view logs. I'm on a Mac, so I will 
follow this [link](https://kubernetes.io/docs/tasks/tools/install-kubectl-macos/) 
to know how to install the tool. It's pretty simple if you have Homebrew
installed on your machine. Just run: 

```shell
brew install kubectl
```

And that's it!

We can run this `kubectl` version client command to see if it's working
well or not.

```shell
kubectl version --client
```

If it prints out version of the client like this

![](../images/part31/2.png)

then it means the tool has been successfully installed.

Next step, we will need to verify `kubectl` configuration. Basically, 
`kubectl` will need to read some information from a config file in order
to know how to access the Kubernetes cluster. By default, the file is
located at this folder `~/.kube/config`. We can use this `kubectl 
cluster-info` command to check if the config is properly set up or not.
Here we see an error because it's trying to connect to a local cluster.

![](../images/part31/3.png)

But we don't have any cluster running in our local host. We want it to 
connect to the AWS EKS cluster that we set up in the previous lecture. 
So we have to tell `kubectl` how to look for that cluster. In order to
do that, here's the command we can use to fetch the AWS EKS cluster's 
information and store it in the config file:

```shell
aws eks update-kubeconfig --name
```

Then pass in the name of the cluster, which is `simple-bank` in our case.

```shell
aws eks update-kubeconfig --name simple-bank
```

Then the region where the cluster is located, which is `eu-west-1`.

```shell
aws eks update-kubeconfig --name simple-bank --region eu-west-1
```

And press `Enter`.

Oops, an error occurred: "AccessDeniedException". And the reason is: "user
github-ci is not authorized" to perform the `DescribeCluster` operation
on the `simple-bank` cluster. So what we need to do now is to grant EKS
permissions for this user.

Let's go to AWS IAM service, open the `Users` page and select the 
`github-ci` user. If you still remember, as we set up in previous 
lectures, this user only has permissions to access ECR and secret manager 
services.

![](../images/part31/4.png)

And these permissions are actually coming from the `deployment` user 
group that the `github-ci` is attached to. So to add a new permission,
let's open this user group, change to the `Permissions` tab, click `Add
permissions`, and select `Create Inline Policy`.

![](../images/part31/5.png)

In this form, we have to choose a service. Let's look for `EKS`. Here it is.
There are many access levels, and in the `Read` level, you can see the 
`DescribeCluster` permission.

![](../images/part31/6.png)

You can select only this permission if you want, but since I want the 
`github-ci` to be able to make changes to the cluster and deploy 
applications to it later, I will just allow it to perform all actions
by checking this box.

![](../images/part31/7.png)

Next, in the `Resources` section, you have the option to choose only some
specific resources you want this user group to manage, or you can simply
allow it to manage all resources as I'm doing here.

![](../images/part31/8.png)

OK, now let's click `Review policy`.

We must set a name for this policy. I'm gonna call it `EKSFullAccess`.
And click `Create policy`.

![](../images/part31/9.png)

And voilà, now the `deployment` user group will have full access to EKS 
clusters.

![](../images/part31/10.png)

If we go back to the `Users` page, and check the `github-ci` user's 
permissions, we can see that now it has `EKSFullAccess` permission.

![](../images/part31/11.png)

That's because this user belongs to the `deployment` group, so it will
be granted all permissions of the group. OK, cool!

Now let's go back to the terminal and run the `update-kubeconfig` command
again.

```shell
aws eks update-kubeconfig --name simple-bank --region eu-west-1
```

This time it is successful.

![](../images/part31/12.png)

A new context for the `simple-bank` cluster has been added to 
the `.kube/config` file.

Let's check the content of the `.kube` folder. There's a config file as
expected. Let's print out the content of this file using the `cat` 
command.

![](../images/part31/13.png)

![](../images/part31/14.png)

At the top of the file it has some certificate authority data, then the 
URL of the server, where we can access the cluster, followed by the name
of the cluster: `simple-bank`. Next, we have the context to access this
cluster. It includes the ARN of the cluster, and the user to access it,
followed by the name of the context, which is also the current context,
since we have only 1 context until now.

```
server: https://A20B4A292D1881BDA54A18F788FDAF38.yl4.eu-west-1.eks.amazonaws.com
name: arn:aws:eks:eu-west-1:095420225348:cluster/simple-bank
contexts: 
- context:
    cluster: arn:aws:eks:eu-west-1:095420225348:cluster/simple-bank
    user: arn:aws:eks:eu-west-1:095420225348:cluster/simple-bank
  name: arn:aws:eks:eu-west-1:095420225348:cluster/simple-bank
current-context: arn:aws:eks:eu-west-1:095420225348:cluster/simple-bank
```

And at the bottom of the file, there's some more detail information of 
the user, which `kubectl` will use to get the token to access the 
`simple-bank` cluster.

```
users:
- name: arn:aws:eks:eu-west-1:095420225348:cluster/simple-bank
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1alpha1
      args:
      - --region
      - eu-west-1
      - eks
      - get-token
      - --cluster-name
      - simple-bank
      command: aws
```

Now, in case you have multiple contexts for different clusters on your
machine, you can use this `kubectl config use-context` command to select
the context you want to connect to.

```shell
kubectl config use-context arn:aws:eks:eu-west-1:095420225348:cluster/simple-bank
```

Alright, now let's check if we can connect to our `simple-bank` cluster
yet or not. We can use the `kubectl cluster-info` for this purpose.

```shell
kubectl cluster-info
```

Oops, we've got an error: "Unauthorized, you must be logged in to the
server". OK, so if you face the same problem, then this might be the 
reason. Your IAM entity isn't authorized by the RBAC configuration of 
the EKS cluster. This happens when the cluster is created by an IAM user
or role that's different from the one you're using to authenticate and
connect to it. Remember that the user I'm using on my local machine is
`github-ci`. But initially, only the creator of the Amazon EKS cluster
has the master permissions to configure the cluster. If we want to extend
this permission to other users, we must add the `aws-auth ConfigMap` to
the configuration of the EKS cluster.

OK, so how can we fix this?

First, let's run this command in the terminal to check the current user
identity.

```shell
aws sts get-caller-identity
{
  "UserId": "AIDARMN35C5CMEQYNUFTE"
  "Account": "095420225348",
  "Arn": "arn:aws:iam:095420225348:user/github-ci"
}
```

As you can see, it's the `github-ci` user, not the `root` user we used to
create the cluster. That's why it doesn't have the permissions to access
the cluster. So if we run this `kubectl get pods` command,

```shell
kubectl get pods
error: You must be logged in to the server (Unathorized)
```

it will throw the same error as when we run the `cluster-info` command
before. In order to fix this, we must use `aws_access_key_id` and 
`aws_secret_access_key` of the cluster creator account. At the moment, we're 
still using the access key of the `github-ci` user as you can see in this 
aws credentials file.

```shell
cat ~./.aws/credentials
[default]
aws_access_key_id = AKIARMN35C5CG3LIRKG4
aws_secret_access_key = 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
```

Alright, now let's create a new access credential for the root user: 
`techschool`. I'm gonna open `My Security Credential` page in a new tab.

![](../images/part31/15.png)

Then select the `Access keys` section and click `Create New Access Key`.

![](../images/part31/16.png)

The key has been successfully created. We can click on this `Show Access 
Key` to show the access key. I'm gonna copy the access key ID.

![](../images/part31/17.png)

Then go back to the terminal, and edit the AWS credentials file using
`vim`.

```shell
vi ~./.aws/credentials
```

We can list multiple different access keys in this file. So here I will
put the old credentials under a profile named `github`. And the new 
access keys will go into the `default` profile. First, paste in 
`aws_access_key_id`, followed by `aws_secret_access_key`, I'm gonna copy its 
value from the AWS console, and paste it to the file. Alright, let's save 
this file.

```
[default]
aws_access_key_id = AKIARMN35C5CE3JSV3DE
aws_secret_access_key = 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

[github]
aws_access_key_id = AKIARMN35C5CG3LIRKG4
aws_secret_access_key = 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
```

Now I believe it will work. Let's try running the `kubectl get pods` 
command.

```shell
kubectl get pods
No resources found in default namespace.
```

This time, there's no authentication error as before. It just say:
"No resources found in default namespace." That's because we haven't
deployed anything to the cluster yet. Let's run the `kubectl 
cluster-info` command.

```shell
kubectl cluster-info
Kubernetes control plane is running at https://A20B4A292D1881BDA54A18F788FDAF38.yl4.eu-west-1.eks.amazonaws.com
CoreDNS is running at https://A20B4A292D1881BDA54A18F788FDAF38.yl4.eu-west-1.eks.amazonaws.com/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy

To futher debug and diagose cluster problems, use 'kubectl cluster-info dump'
```

Here you go! The info is successfully returned. It contains the address
of the control plane and CoreDNS of the cluster. So it worked! We can
now access the cluster using the credentials of the `root` user. Next,
I'm gonna show you how to give `github-ci` user access to this cluster
as well, because later we would want GitHub to automatically deploy
the app for us whenever we push new changes to the `master` branch.

But first, let's learn how to tell AWS CLU to use the GitHub 
credentials. It's pretty easy, we just need to 

```shell
export AWS_PROFILE=github
```

Note that the name of the profile must match the one we declare in the
AWS credentials file. Now if we run `kubectl cluster-info` again,
it will fail because of Unauthorized access as before.

If we want to get back to the default root user, just

```shell
export AWS_PROFILE=default
```

then this time, the `cluster-info` command will be successful again.

Alright, but how can we allow Github user to access the cluster?

Well, to do that, we need to add this user to a special ConfigMap
as shown in this example.

```
apiVersion: v1
kind: ConfigMap 
metadata: 
  name: aws-auth 
  namespace: kube-system
data:
  mapRoles: |
    - rolearn: <ARN of instance role (not instance profile)>
      username: system:node:{{EC2PrivateDNSName}}
      groups:
        - system:bootstrappers
        - system:nodes
```

I'm gonna add this file to our `simple-bank` project.

So let's open the project in Visual Studio Code. I will create a new
folder `eks` to store all files related to Kubernetes.

First, let's add a new file called `aws-auth.yaml`. Next, we have to 
add the `github-ci` user to the `mapUsers` section of this file. So
I'm gonna copy this example from the step 7 of this [tutorial](https://aws.amazon.com/premiumsupport/knowledge-center/amazon-eks-cluster-access/)
and paste it to our `aws-auth.yaml` file.

```yaml
apiVersion: v1
kind: ConfigMap 
metadata: 
  name: aws-auth 
  namespace: kube-system 
data:
  mapRoles: |
    - rolearn: arn:aws:iam:111222333:role/EKS-Worker-NodeInstanceRole-1I00GBC9U4U7B
      username: system:node:{{EC2PrivateDNSName}}
      groups:
        - system:bootstrappers
        - system:nodes
  mapUsers: | 
    - userarn: arn:aws:iam::111222333:user/designated_user
      username: designated_user
      groups:
        - system:masters
```

We don't need the `mapRoles` section, so let's delete it. Now, in the
`mapUsers` section, we have to put correct ARN of the `github-ci` user
we want to give access to the cluster. We can find its value in the
IAM console. Let's open the `Users` page, select `github-ci` and click
this button to copy its user ARN.

![](../images/part31/18.png)

Then paste it here.

```yaml
apiVersion: v1
kind: ConfigMap 
metadata: 
  name: aws-auth 
  namespace: kube-system 
data:
  mapUsers: | 
    - userarn: arnarn:aws:iam::095420225348:user/github-ci
      username: github-ci
      groups:
        - system:masters
```

The username should be `github-ci`. The groups should remain the same
as `system:masters`. Okay, now we need to apply this `ConfigMap` to
the RBAC configuration of the cluster.

Let's run this `kubectl apply -f aws-auth.yaml` command in the terminal,

```shell
kubectl apply -f eks/aws-auth.yaml
```

but we need to change the location of the file to `eks/aws-auth.yaml`.

There's a warning here, but it's OK, because this is the first time
we apply new configuration.

![](../images/part31/19.png)

As it clearly says here, the missing annotation will be patched 
automatically. So let's try switching to the Github profile, and run
the `kubectl cluster-info` command.

![](../images/part31/20.png)

Oh, we still got the same error: "Unauthorized".

So what happened? Let's go back to check the `ConfigMap`. OK, I see.
Looks like we've got a typo here in the user ARN value. The correct
value should not have a duplicate `arn` prefix. So let's fix it!

```yaml
apiVersion: v1
kind: ConfigMap 
metadata: 
  name: aws-auth 
  namespace: kube-system 
data:
  mapUsers: | 
    - userarn: arn:aws:iam::095420225348:user/github-ci
      username: github-ci
      groups:
        - system:masters
```

Then go back to the terminal and apply the change again.

```shell
kubectl apply -f eks/aws-auth.yaml
```

Oops, I forgot that we're still using the `github-ci` profile. We 
must change to the default `root` profile first.

```shell
export AWS_PROFILE=default
```

Then apply the change again.

```shell
kubectl apply -f eks/aws-auth.yaml
configmap/aws-auth configured
```

This time, it is successful.

Now let's switch back to the `github-ci` profile and run `kubectl 
cluster-info` command one more time.

![](../images/part31/21.png)

This time it worked! Excellent!

So now you know how to grant access to the cluster to a user who's
not the creator of the cluster. This will be useful when we want to 
set up continuous deployment process later.

Now before we finish, I'm gonna show you 1 more tool, which, I think,
is very cool and make it so much easier to interact with the 
Kubernetes cluster.

As you already known, the normal way to interact with the cluster
is using `kubectl` tool, for example, to get the services that are
running on the cluster, we run 

```shell
kubectl get service
```

or to get the list of running pods, we use `kubectl get pods` command.

For short and simple commands like that, it's ok to type, but it would
become very annoying when you have to run longer and more complex
commands, that requires you to copy around the name or id of the 
services or pods. To make it easier to interact with the cluster, we
can use a tool called `k9s`.

## Command line tool - `k9s`

It gives us a very nice user interface, and a set of convenient 
shortcuts to talk to the Kubernetes cluster. If you're on a mac, we can
install it with `Homebrew`. Let's run 

```shell
brew install k9s
```

in the terminal.

After it's successfully installed, we can run just 

```shell
k9s
```

to access the cluster. As you can see, a nice UI shows up, and it's
currently listing all pods of the cluster.

![](../images/part31/22.png)

There are 2 `core-dns` pods of the `kube-system`. Because of some 
reasons, their status is still `Pending`, but we will deal with it later, 
in the next lecture.

For now, I'm gonna show you some useful shortcut commands to navigate
the cluster resources.

To switch the namespace, just type a colon, then `ns`, `Enter`. It will
show you all available namespaces of the cluster.

![](../images/part31/23.png)

You can use the arrows to select the namespace you want to access, and
simply press `Enter` to go into that namespace.

And to get out to the previous screen, just press `Escape`.

Similarly, to list all services, we type a colon, followed by `service`,
then `Enter`.

![](../images/part31/24.png)

To list all pods, just type a colon, then `pods` and `Enter`.

![](../images/part31/25.png)

If you have `cronjobs` running on the cluster, you can list them by 
typing a colon, followed by `cj`, `Enter`.

![](../images/part31/26.png)

Or to list all available nodes, just type a colon, then `nodes`, 
`Enter`.

![](../images/part31/27.png)

For now there's no node in the cluster. Perhaps that's why the 
core-dns services cannot start, and stay in pending state.

Also, there are several more commands that are listed here on the UI.
For example, to delete a resource, press `Ctrl + d`. Or to decsribe a
resource, just press `d`.

![](../images/part31/28.png)

You will see all the information of your chosen resource. And you can 
always use the `Escape` key to go back to the previous screen.

Now let's take a look at the `СonfigMap` that we're updated before.

![](../images/part31/29.png)

It's the `aws-auth` config map in the `kube-system` namespace. If we
press `d` to describe it, we will see the `github-ci` user in the `mapUsers` 
section.

![](../images/part31/30.png)

And it belongs to the `system:masters` group, exactly as we wanted.

Now press `Escape` to go back to the previous screen.

And finally use the `quit` command to exit from `k9s`.

And that's it for today's lecture. I hope you find it useful.

Thanks a lot for watching, happy learning, and see you in the next
lecture!