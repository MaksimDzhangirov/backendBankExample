# Как использовать kubectl и k9s для подключения к Kubernetes кластеру на AWS EKS

[Оригинал](https://www.youtube.com/watch?v=hwMevai3_wQ)

Всем привет и рад снова видеть Вас на мастер-классе по бэкенду! 

На [предыдущей лекции](part30-rus.md) мы узнали, как настроить Kubernetes 
кластер с помощью AWS EKS.

Как видите кластер для нашего простого банковского приложения уже запущен
и работает.

![](../images/part31/1.png)

Итак, сегодня я покажу Вам как подключиться к этому Kubernetes кластеру, 
используя два инструмента командной строки: `kubectl` и `k9s`.

## Инструмент командной строки - `kubectl`

Во-первых, давайте поищем по ключевому слову `kubectl` через поисковик в 
браузере. В результате мы откроем эту [страницу](https://kubernetes.io/docs/tasks/tools/).
Итак, `kubectl` — это инструмент командной строки для Kubernetes, который 
позволяет запускать команды для Kubernetes кластеров. Вы можете использовать 
его для развертывания приложений, проверки ресурсов кластера и управления 
ими, а также для просмотра логов. У меня Mac OS, поэтому я перейду по этой 
[ссылке](https://kubernetes.io/docs/tasks/tools/install-kubectl-macos/),
чтобы узнать, как установить этот инструмент. Это довольно просто, если на 
вашем компьютере установлен Homebrew. Просто выполните в терминале:

```shell
brew install kubectl
```

И этого будет достаточно!

Мы можем запустить эту команду для просмотра версии клиента `kubectl`,
чтобы проверить, успешно ли она была установлена или нет.

```shell
kubectl version --client
```

Если она выведет версию клиента примерно так как показано на рисунке

![](../images/part31/2.png)

то это означает, что инструмент был успешно установлен.

На следующем шаге нам нужно будет проверить конфигурацию `kubectl`. По сути, 
`kubectl` нужно будет прочитать некоторую информацию из файла настроек,
чтобы узнать, как получить доступ к Kubernetes кластеру. По умолчанию файл 
находится в этой папке `~/.kube/config`. Мы можем использовать эту команду 
`kubectl cluster-info`, чтобы проверить, правильно ли настроена конфигурация.
Здесь на рисунке мы получили ошибку, потому что `kubectl` пытается 
подключиться к локальному кластеру.

![](../images/part31/3.png)

Но у нас нет кластера, работающего на нашем локальном хосте. Мы хотим, чтобы 
он подключался к AWS EKS кластеру, который мы настроили в предыдущей лекции.
Поэтому мы должны указать `kubectl`, как искать этот кластер. Ниже показана 
команда, которую мы можем использовать для получения информации о AWS EKS 
кластере и сохранения её в файле с настройками:

```shell
aws eks update-kubeconfig --name
```

Затем введите название кластера, в нашем случае это `simple-bank`.

```shell
aws eks update-kubeconfig --name simple-bank
```

Затем регион, в котором расположен кластер, то есть `eu-west-1`.

```shell
aws eks update-kubeconfig --name simple-bank --region eu-west-1
```

И нажмите `Enter`.

К сожалению, произошла ошибка: "AccessDeniedException" («Доступ запрещён»).
И причина в том, что "user github-ci is not authorized" («Пользователь 
github-ci не авторизован») для выполнения операции `DescribeCluster` в 
кластере `simple-bank`. Итак, что нам нужно сделать сейчас, это предоставить 
EKS права доступа для этого пользователя.

Перейдём в AWS IAM сервис, откроем страницу `Users` и выберем пользователя
`github-ci`. Если вы еще помните, как мы осуществляли настройку в 
предыдущих лекциях, этот пользователь имеет права доступа только к ECR 
сервисам и менеджеру конфиденциальных данных.

![](../images/part31/4.png)

И эти права доступа на самом деле прописаны для группы пользователей
`deployment`, к которой относится `github-ci`. Итак, чтобы добавить новое
право доступа, давайте откроем эту группу пользователей, перейдем на вкладку
`Permissions`, нажмём `Add permissions` и выберем `Create Inline Policy`.

![](../images/part31/5.png)

В этой форме мы должны выбрать сервис. Давайте поищем `EKS`. Мы легко его
найдём. Существует множество уровней доступа, и на уровне `Read`, вы увидите 
право доступа `DescribeCluster`.

![](../images/part31/6.png)

Вы можете выбрать только это право доступа, если хотите, но поскольку я 
хочу, чтобы `github-ci` мог вносить изменения в кластер и развертывать на 
нём приложения позже, я просто позволю ему выполнять все действия, установив 
этот флажок как показано на рисунке.

![](../images/part31/7.png)

Затем в разделе `Resources` у вас есть возможность выбрать только некоторые 
конкретные ресурсы, которыми эта группа пользователей должна управлять,
или можно просто разрешить ей управлять всеми ресурсами, как я делаю 
здесь.

![](../images/part31/8.png)

Хорошо, теперь давайте нажмем `Review policy`.

Мы должны задать название для этого набора прав. Я назову их `EKSFullAccess`.
И нажму `Create policy`.

![](../images/part31/9.png)

И вуаля, теперь группа пользователей `deployment` будет иметь полный доступ 
к EKS кластерам.

![](../images/part31/10.png)

Если мы вернемся на страницу `Users` и просмотрим права доступа
пользователя `github-ci`, то увидим, что теперь у него есть право
`EKSFullAccess`.

![](../images/part31/11.png)

Это связано с тем, что этот пользователь принадлежит к группе `deployment`,
поэтому ему будут предоставлены все права доступа группы. Здорово!

Теперь вернемся в терминал и снова запустим команду `update-kubeconfig`.

```shell
aws eks update-kubeconfig --name simple-bank --region eu-west-1
```

На этот раз команда успешно выполнена.

![](../images/part31/12.png)

Добавлены новые данные для кластера `simple-bank` в файл `.kube/config`.

Давайте просмотрим содержимое папки `.kube`. Там находится файл с настройками,
как и ожидалось. Давайте выведем на экран информацию внутри этого файла с
помощью команды `cat`.

![](../images/part31/13.png)

![](../images/part31/14.png)

В верхней части файла находятся некоторые данные центра сертификации,
затем URL-адрес сервера, с которого мы можем получить доступ к кластеру, а 
затем следует название кластера: `simple-bank`. Далее находятся контекст
для доступа к этому кластеру. Он включает в себя ARN кластера и пользователя,
затем название контекста, который также является текущим контекстом, пока 
что нам доступен только один контекст.

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

И в конце файла находится более подробная информация о пользователе,
которую `kubectl` будет использовать, чтобы получить токен для доступа к 
кластеру `simple-bank`.

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

Теперь в случае если у вас существует несколько контекстов для разных 
кластеров на вашем компьютере, вы можете использовать эту команду 
`kubectl config use-context`, чтобы выбрать контекст, к которому вы хотите
подключиться.

```shell
kubectl config use-context arn:aws:eks:eu-west-1:095420225348:cluster/simple-bank
```

Хорошо, теперь давайте проверим, можем ли мы подключиться к нашему кластеру
`simple-bank` или нет. Для этой цели мы можем использовать команду
`kubectl cluster-info`.

```shell
kubectl cluster-info
```

Мы получили ошибку: "Unauthorized, you must be logged in to the server" 
(«Пользователь не авторизирован, вы должны войти на сервер»). Итак, если вы 
столкнулись с этой же проблемой, то причина может быть в следующем. У вашей
IAM учётной записи нет доступа к EKS кластеру для текущих RBAC настроек.
Это происходит, когда кластер создается IAM пользователем или ролью, 
отличной от той, которую вы используете для аутентификации и подключения 
к нему. Вспомните, что я подключаюсь на своей локальной машине под 
пользователем `github-ci`. Но изначально только создатель Amazon EKS кластера 
имеет права администратора для настройки кластера. Если мы хотим, чтобы
эти права были доступы другим пользователям, мы должны добавить `aws-auth 
ConfigMap` в настройки EKS кластера.

Итак, как мы можем это исправить?

Во-первых, давайте запустим эту команду в терминале, чтобы просмотреть 
кто является текущим пользователем.

```shell
aws sts get-caller-identity
{
  "UserId": "AIDARMN35C5CMEQYNUFTE"
  "Account": "095420225348",
  "Arn": "arn:aws:iam:095420225348:user/github-ci"
}
```

Как видите, это пользователь `github-ci`, а не пользователь `root`,
которого мы использовали для создания кластера. Вот почему у него нет прав 
доступа к кластеру. Итак, если мы запустим эту команду `kubectl get pods`,

```shell
kubectl get pods
error: You must be logged in to the server (Unathorized)
```

то она выдаст ту же ошибку, что и при запуске команды `cluster-info` ранее.
Чтобы исправить это, мы должны использовать `aws_access_key_id` и 
`aws_secret_access_key` создателя кластера. На данный момент мы все ещё 
используем ключ доступа пользователя `github-ci`, как видно в этом файле 
учетных данных AWS.

```shell
cat ~./.aws/credentials
[default]
aws_access_key_id = AKIARMN35C5CG3LIRKG4
aws_secret_access_key = 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
```

Хорошо, теперь давайте создадим новые учетные данные для пользователя `root`:
`techschool`. Я открою в браузере страницу `My Security Credential` в новой
вкладке.

![](../images/part31/15.png)

Затем выберите раздел `Access keys` и нажмите `Create New Access Key`.

![](../images/part31/16.png)

Ключ был успешно создан. Мы можем нажать на эту ссылку `Show Access
Key`, чтобы увидеть ключ доступа. Я скопирую идентификатор ключа доступа.

![](../images/part31/17.png)

Затем вернусь в терминал и отредактирую файл учетных данных AWS, используя
`vim`.

```shell
vi ~./.aws/credentials
```

Мы можем перечислить несколько разных ключей доступа в этом файле. Здесь 
данные, уже хранящиеся в файле, я добавлю в профиль с названием `github`. А
новые ключи доступа войдут в профиль по умолчанию `default`. Сначала вставьте
`aws_access_key_id`, а затем `aws_secret_access_key`, я скопирую его 
значение из AWS консоли и вставлю в файл. Хорошо, давайте сохраним этот 
файл.

```
[default]
aws_access_key_id = AKIARMN35C5CE3JSV3DE
aws_secret_access_key = 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

[github]
aws_access_key_id = AKIARMN35C5CG3LIRKG4
aws_secret_access_key = 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
```

Теперь всё должно сработать как надо. Давайте попробуем выполнить команду 
`kubectl get pods`.

```shell
kubectl get pods
No resources found in default namespace.
```

На этот раз ошибки аутентификации не возникло. `kubectl` просто сообщает:
"No resources found in default namespace" («Ресурсы не найдены в 
пространстве имен по умолчанию»). Это связано с тем, что мы ещё ничего не 
развернули в кластере. Давайте выполним команду `kubectl cluster-info`.

```shell
kubectl cluster-info
Kubernetes control plane is running at https://A20B4A292D1881BDA54A18F788FDAF38.yl4.eu-west-1.eks.amazonaws.com
CoreDNS is running at https://A20B4A292D1881BDA54A18F788FDAF38.yl4.eu-west-1.eks.amazonaws.com/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy

To futher debug and diagose cluster problems, use 'kubectl cluster-info dump'
```

Как мы и ожидали! Информация о кластере успешно получена. Она содержит
адрес плоскости управления и CoreDNS кластера. Так что всё работает как надо!
Теперь мы можем получить доступ к кластеру, используя учетные данные 
пользователя `root`. Затем я покажу вам, как предоставить пользователю
`github-ci` доступ к этому кластеру, потому что позднее мы захотим, чтобы
GitHub автоматически развертывал приложение за нас всякий раз, когда мы 
отправляем новые изменения в ветку `master`.

Но сначала давайте узнаем, как указать AWS CLU использовать учетные 
данные GitHub. Это довольно просто, достаточно экспортировать

```shell
export AWS_PROFILE=github
```

Обратите внимание, что название профиля должно совпадать с названием, которое
мы объявляем в файле учетных данных AWS. Теперь, если мы снова выполним
`kubectl cluster-info`, то получим ошибку из-за несанкционированного 
доступа, как и раньше. 

Если мы хотим вернуться к пользователю `root`, то достаточно экспортировать

```shell
export AWS_PROFILE=default
```

то на этот раз команда `cluster-info` снова будет успешно выполнена.

Хорошо, но как мы можем разрешить пользователю GitHub доступ к кластеру?

Что ж, для этого нам нужно добавить этого пользователя в специальную
ConfigMap, как показано в этом примере.

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

Я добавлю файл с этими настройками в наш проект `simple-bank`.

Итак, давайте откроем проект в Visual Studio Code. Я создам новую папку `eks` 
для хранения всех файлов, связанных с Kubernetes.

Во-первых, давайте добавим новый файл под названием `aws-auth.yaml`. Затем 
мы должны пользователя `github-ci` в раздел `mapUsers` этого файла. Поэтому 
я скопирую этот пример из шага 7 этого [руководства](https://aws.amazon.com/premiumsupport/knowledge-center/amazon-eks-cluster-access/)
и вставлю его в наш `aws-auth.yaml` файл.

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

Раздел `mapRoles` нам не нужен, поэтому удалим его. Теперь в разделе
`mapUsers` мы должны указать правильный ARN пользователя `github-ci`, 
которому мы хотим предоставить доступ к кластеру. Мы можем найти это 
значение в IAM консоли. Давайте откроем страницу `Users`, выберем `github-ci`
и нажмем эту кнопку, чтобы скопировать ARN пользователя.

![](../images/part31/18.png)

Затем вставьте его сюда.

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

Имя пользователя должно быть `github-ci`. Группа должна остаться такой 
же `system:masters`. Хорошо, теперь нам нужно добавить эту `ConfigMap` 
к RBAC настройкам кластера.

Давайте запустим эту команду `kubectl apply -f aws-auth.yaml` в терминале,

```shell
kubectl apply -f eks/aws-auth.yaml
```

но нам нужно изменить путь к файлу на `eks/aws-auth.yaml`.

Мы получим предупреждение, но это нормально, потому что мы впервые применяем 
новые настройки.

![](../images/part31/19.png)

Как сказано в предупреждении, отсутствующая аннотация будет исправлена 
автоматически. Итак, давайте попробуем переключиться на GitHub профиль и 
запустить команду `kubectl cluster-info`.

![](../images/part31/20.png)

К сожалению, мы всё ещё получаем ту же ошибку: "Unauthorized" («Пользователь
не авторизован»).

Почему так происходит? Вернёмся и внимательно посмотрим на `ConfigMap`. О, я
понял в чём дело. Похоже, здесь опечатка в значении ARN пользователя. 
Правильное значение не должно содержать повторяющегося префикса `arn`. Так что 
давайте её исправим!

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

Затем вернитесь в терминал и попробуйте снова добавить изменения. 

```shell
kubectl apply -f eks/aws-auth.yaml
```

Ой, я забыл, что мы всё ещё используем профиль `github-ci`. Сначала мы должны
переключиться на профиль по умолчанию `root`.

```shell
export AWS_PROFILE=default
```

Затем снова добавьте изменения.

```shell
kubectl apply -f eks/aws-auth.yaml
configmap/aws-auth configured
```

На этот раз команда выполнилась успешно.

Теперь давайте обратно переключимся на профиль `github-ci` и запустим 
команду `kubectl cluster-info` ещё раз. 

![](../images/part31/21.png)

На этот раз она выполнилась без ошибок! Превосходно!

Итак, теперь вы знаете, как предоставить доступ к кластеру пользователю, 
который не является создателем кластера. Это нам пригодиться, когда мы 
позднее захотим настроить процесс непрерывного развертывания.

Теперь прежде чем мы закончим, я покажу вам ещё один инструмент, который,
как мне кажется, очень классный и значительно упрощает взаимодействие с 
Kubernetes кластером.

Как вы уже знаете, обычным способом взаимодействия с кластером является 
использование инструмента `kubectl`, например, чтобы получить сервисы, 
работающие в кластере, мы выполняем

```shell
kubectl get service
```

или чтобы получить список запущенных подов, мы используем команду `kubectl 
get pods`.

Такие короткие и простые команды можно набирать вручную, но вас будут 
раздражать набор более длинных и сложных команд, требующих копирования имени 
или идентификатора сервисов или подов. Чтобы упростить взаимодействие с 
кластером, мы можем использовать инструмент под названием `k9s`.

## Инструмент командной строки - `k9s`

Он предоставляет нам очень красивый пользовательский интерфейс и набор удобных 
в использовании упрощенных команд для взаимодействия с Kubernetes кластером.
Если у вас Mac OS, мы можем установить его с помощью Homebrew. Давайте
запустим

```shell
brew install k9s
```

в терминале.

После успешной установки, достаточно просто ввести

```shell
k9s
```

чтобы получить доступ к кластеру. Как видите появляется красивый 
пользовательский интерфейс, и сейчас в нём перечислены все поды кластера.

![](../images/part31/22.png)

Работают два `core-dns` пода в `kube-system`. По каким-то причинам их статус
до сих пор `Pending` («В режиме ожидания»), но об этом мы поговорим позже, 
на следующей лекции.

А пока я покажу вам несколько полезных упрощенных команд для навигации по 
ресурсам кластера.

Чтобы переключить пространство имен, просто введите двоеточие, затем `ns`, 
`Enter`. `k9s` отобразит вам все доступные пространства имен кластера.

![](../images/part31/23.png)

Вы можете использовать стрелки, чтобы выбрать пространство имен, к которому 
хотите получить доступ, и просто нажать клавишу `Enter`, чтобы перейти в это 
пространство имен.

А чтобы вернуться, достаточно нажать `Escape`.

Точно так же, чтобы вывести список всех сервисов, мы вводим двоеточие, за 
которым следует слово `service`, затем нажимаем `Enter`.

![](../images/part31/24.png)

Чтобы вывести список всех подов, достаточно ввести двоеточие, `pods` и 
`Enter`.

![](../images/part31/25.png)

Если у вас есть `cronjobs`, работающие в кластере, вы можете вывести их 
список, введя двоеточие, `cj`, `Enter`.

![](../images/part31/26.png)

А чтобы вывести список всех доступных узлов, воспользуйтесь командой 
двоеточие, затем `nodes`, `Enter`.

![](../images/part31/27.png)

На данный момент в кластере нет узлов. Возможно, поэтому службы core-dns не 
могут запуститься и остаются в режиме ожидания.

Кроме того, существует ещё несколько команд, которые перечислены здесь в 
пользовательском интерфейсе. Например, чтобы удалить ресурс, нажмите 
`Ctrl + d`, а чтобы просмотреть информацию о ресурсе, достаточно нажать `d`.

![](../images/part31/28.png)

Вы увидите всю информацию о выбранном вами ресурсе. И вы всегда можете 
использовать клавишу «Escape», чтобы вернуться назад.

Теперь давайте посмотрим на `ConfigMap`, которую мы обновили ранее.

![](../images/part31/29.png)

Это `aws-auth` `ConfigMap` в пространстве имен `kube-system`. Если мы нажмем 
`d`, чтобы просмотреть её, то увидим пользователя `github-ci` в разделе
`mapUsers`. 

![](../images/part31/30.png)

И он относится к группе `system:masters`, как мы и хотели.

Теперь нажмите «Escape», чтобы вернуться назад.

И, наконец, воспользуйтесь командой `quit` для выхода из `k9s`.

На этом закончим сегодняшнюю лекцию.  Я надеюсь, что приобретенные из этой 
лекции знания будут вам полезны.

Большое спасибо за время, потраченное на чтение, желаю вам получать
удовольствие от обучения и до встречи на следующей лекции!