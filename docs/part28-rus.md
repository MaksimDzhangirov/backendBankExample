# Как создать продакшен БД в AWS RDS

[Оригинал](https://www.youtube.com/watch?v=0EaG3T4Q5fQ)

Всем привет и добро пожаловать на мастер-класс по бэкенду. На этой лекции
мы узнаем, как создать базу данных для продакшен с помощью AWS RDS.

## AWS RDS

RDS - это сервис, позволяющий управлять реляционными базами данных в AWS.
Он поддерживает несколько видов реляционных баз данных и автоматически 
развертывается и управляется Amazon Web Service, поэтому нам не нужно особо 
заботиться о том, как поддерживать или масштабировать БД кластер. Его также
очень легко настроить. Давайте нажмем кнопку `Create database`, чтобы 
создать новую базу данных!

![](../images/part28/1.png)

Существует 2 способа создания БД: `Standard create` и `Easy create`. `Easy 
create` будет использовать рекомендуемую лучшими практиками конфигурацию 
для вашей БД, тогда как `Standard create` позволит вам настроить все 
параметры конфигурации самостоятельно. Для нашего примера я собираюсь 
использовать этот метод.

![](../images/part28/2.png)

Сейчас нам нужно выбрать тип движка БД. Как видите на рисунке, существует 
шесть различных типов движков. Я выберу `Postgres` для нашего простого 
банковского приложения. В качестве версии давайте выберем `12.6`, потому что 
мы используем `Postgres 12` для разработки.

Далее давайте выберем шаблон с настройками. Нам предлагают выбор из 
`Production`, `Dev/Test` или `Free Tier`. Для демонстрационных целей я 
буду использовать `Free Tier`.

Далее мы должны ввести названием для нашего инстанса БД. Назовем его
`simple-bank`. В качестве `Master username`, я буду использовать `root`,
как мы это делали для локальной разработки. Затем для пароля я установлю 
этот флажок (`Auto generate a password`), чтобы RDS автоматически 
сгенерировал для меня случайный пароль.

![](../images/part28/3.png)

Далее нужно выбрать `DB instance class`. Поскольку мы выбрали шаблон
`Free Tier`, доступен только один тип инстанса: `db.t2.micro`. Если вы 
выберете шаблон `Production` или `Dev/Test`, тогда будут доступны различные
другие более мощные инстансы на выбор.

Пока что для хранения нам будет доступно 20 ГБ бесплатного пространства на 
SSD, а также возможность включить автомасштабирование, которое позволит 
увеличить хранилище после превышения указанного предела. Нам не нужна
такая возможность для нашего демонстрационного примера, поэтому я отключу 
эту функцию.

![](../images/part28/4.png)

Для использования в продакшен у вас также будет возможность настроить 
несколько зон доступности или резервный инстанс в другой зоне, чтобы 
обеспечить избыточность данных и минимизировать задержку. Но поскольку мы 
используем `Free Tier`, эта опция недоступна. Итак, давайте перейдём к 
следующему разделу!

Здесь, в разделе `Connectivity` мы укажем каким образом можно получить доступ
к нашей БД. По умолчанию БД будет развернута внутри стандартного виртуального 
частного облака (или VPC). Мы подробнее расскажем о VPC в другой раз. А пока 
давайте просто воспользуемся этим значением по умолчанию. `Subnet group` для
БД — это способ указать подсети и диапазоны IP-адресов, которые инстанс 
БД может использовать в VPC. Оставим пока значения по умолчанию. Затем 
следует одна важная настройка. Вы хотите, чтобы ваша БД была общедоступной 
или нет? Если вы выберете `Yes`, все инстансы и устройства EC2 
за пределами VPC смогут подключаться к вашей базе данных. Но нам нужно будет 
настроить `VPC security group` для этого. Если вы выберете `No`, то RDS не 
будет назначать общедоступный IP-адрес базе данных, поэтому к ней могут 
подключаться только инстансы и устройства EC2 внутри одного и того же 
VPC. Для нашего примера, я хочу получить доступ к БД с моего локального 
компьютера, поэтому выберу `Yes`.

![](../images/part28/5.png)

Это гарантирует, что БД будет иметь общедоступный IP-адрес, но чтобы 
действительно иметь доступ к ней извне, мы должны настроить соответствующую
`VPC security group`. Уже доступна `VPC security group` по умолчанию, но
она не предоставляет доступ к порту БД откуда угодно, поэтому я создам 
новую. Назовем её `access-postgres-anywhere`. Затем выберем `Availability 
Zone`. Вы можете выбрать одну из этих зон на рисунке или использовать
значение по умолчанию `No preference`.

![](../images/part28/6.png)

Теперь перейдите в `Additional configurations`. При желании мы можем 
установить другой порт для БД или оставить всё как есть, то есть порт по 
умолчанию `5432`.

Для аутентификации при подключении к базе данных, мы будем использовать 
аутентификацию по паролю, так что здесь ничего менять не нужно.

![](../images/part28/7.png)

Далее у нас есть возможность создать базу данных по умолчанию. Если вы не 
укажете здесь название, Amazon RDS не создаст за вас базу данных. Я хочу, 
чтобы он создал для меня базу данных `simple_bank`, поэтому я впишу это 
название в этом поле.

![](../images/part28/8.png)

Не нужно что-то менять в `DB parameter group`. Вы также можете `Enable 
automatic backup` («Включить автоматическое резервное копирование») для 
своей БД или настроить некоторые параметры для мониторинга и логирования.

Установите этот флажок, если вы хотите включить автоматическое обновление до 
минорных версии (`Enable auto minor version upgrade`). И выберите интервал
(`Select window`) в течении которого будет проводиться сервисное 
обслуживание БД, если хотите.

![](../images/part28/9.png)

Если это продакшен БД, вы также можете включить защиту от удаления (`Enable 
deletion protection`). Это поможет защитить БД от случайного удаления.

В конце формы мы увидим примерную ежемесячную стоимость нашего инстанса БД.
Поскольку мы пользуемся тарифом `Free Tier`, нам предоставляют 750 часов 
бесплатного использования и 20 ГБ хранилища общего назначения. А также 20 
ГБ для хранилища резервных копий, создаваемых автоматически, и копий, 
которые создаются пользователем вручную. Если вы выходите за предел, будет
сниматься стандартная стоимость за использование сервиса. Более подробно
о стоимости услуг можно узнать по этой [ссылке](https://aws.amazon.com/rds/pricing/).

Хорошо, теперь давайте нажмём `Create database`.

![](../images/part28/10.png)

Как видите, сейчас создается база данных. Потребуется несколько 
минут прежде, чем мы сможем её использовать. В ожидании этого давайте 
нажмем эту кнопку, чтобы увидеть учетные данные для доступа к БД.

![](../images/part28/11.png)

Поскольку я ранее указал, чтобы RDS автоматически сгенерировала для меня
случайный пароль, здесь, во всплывающем окне, мы можем увидеть и скопировать
`Master password`.

![](../images/part28/12.png)

Теперь я открою TablePlus и создам новое соединение для доступа к нашей 
удаленной БД. Давайте назовём наше соединение `AWS Postgres`. У нас еще нет 
URL-адреса хоста, поэтому пока оставим его пустым. Имя пользователя должно 
быть `root` и давайте вставим пароль. Имя базы данных `simple_bank`. Вот и
всё что нужно сделать.

![](../images/part28/13.png)

Теперь нам просто нужно дождаться создания БД и получить URL-адрес её хоста.
Вернемся в консоль AWS. Закройте это всплывающее окно и обновите страницу.
База ещё не готова. Но здесь в таблице `Networking` мы видим, что наша БД 
находится в зоне `eu-west-1b`, она использует `VPC security group` и
`subnet group` по умолчанию. Давайте сейчас просмотрим `VPC security group`
`access-postgres-anywhere`, которую мы указали в процессе создания БД.

![](../images/part28/14.png)

Если мы перейдем по ссылке в эту группу, то увидим, что одно `Inbound
rule` типа PostgreSQL.

![](../images/part28/15.png)

Для которого протокол равен `TCP`, а порт `5432`. IP-адрес в строке `Source`
на самом деле - мой текущий. Я могу быстро убедиться в этом, 
выполнив поиск своего IP-адреса в браузере. Итак, по сути, это правило 
разрешает доступ к базе данных через порт `5432` только для моего IP-адреса. Но
он не статический, и я не хочу обновлять это правило каждый раз, когда 
IP-адрес меняется. Поэтому я изменю `Source` на `Anywhere` («Откуда угодно»).

![](../images/part28/16.png)

Это гарантирует, что любой IP-адрес сможет получить доступ к базе данных.
Конечно, если это продакшен БД, вы не захотите открывать к ней доступ кому 
угодно в Интернете. Итак, давайте сохраним правило и вернемся на страницу 
с базами данных RDS.

Как видно на рисунке, теперь база данных успешно создана.

![](../images/part28/17.png)

Давайте посмотрим на параметры подключения. На этот раз у нас есть конечная
точка для доступа к ней. Итак, давайте скопируем этот URL,

![](../images/part28/18.png)

и вставим его в поле `Host` нашего нового подключения в TablePlus. Затем 
нажмите `Test`. Как видно на рисунке все поля отмечаны зелёным цветом.
Это означает, что соединение успешно установлено.

![](../images/part28/19.png)

Итак, давайте нажмем `Connect`. И вуаля, теперь мы подключены к базе 
данных. Однако на данный момент она совершенно пуста. Это потому, что мы еще 
не выполнили миграцию БД для создания таблиц.

![](../images/part28/20.png)

## Запускаем миграции БД

Итак, давайте сделаем это сейчас. В репозитории нашего простого банковского
приложения давайте откроем Makefile. Там есть команда `migrate up`, которую
мы используем для локальной миграции БД.

```makefile
migrateup:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up
```

Все, что нам нужно сделать сейчас, это изменить этот URL
`postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable` на 
URL-адрес удаленной БД. Имя пользователя по-прежнему `root`, но пароль 
другой. Итак, давайте скопируем его из консоли AWS и вставим сюда.
`localhost` также следует изменить на удаленный хост, который AWS RDS 
предоставляет нам в консоли. Порт и название базы данных совпадают, 
поэтому их не нужно менять. Но мы должны удалить параметр `sslmode=disable`,
поскольку мы подключаемся к удаленной БД, то лучше использовать защищенное 
соединение. Итак, теперь команда должна выполниться без проблем.

```makefile
migrateup:
    migrate -path db/migration -database "postgresql://root:tupExr0Gp4In4Ww4WHKR@simple-bank.czutruo2pa5q.eu-west-1.rds.amazonaws.com:5432/simple_bank" -verbose up
```

Сохраним файл, затем откроем терминал и запустим `make migrate up`.

![](../images/part28/21.png)

Как видно из рисунка, все миграции успешно выполнены! Итак, давайте откроем 
TablePlus, чтобы просмотреть БД. Я обновлю соединение с базой. Как видите,
все таблицы успешно созданы.

![](../images/part28/22.png)

Потрясающе! Итак, теперь вы знаете, как настроить инстанс базы данных,
которым можно управлять из консоли, используя AWS RDS. Я надеюсь, что 
приобретенные из этой лекции знания будут вам полезны.

Большое спасибо за время, потраченное на чтение! Желаю вам получать 
удовольствие от обучения и до встречи на следующей лекции.