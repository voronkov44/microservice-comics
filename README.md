# Kubernetes (Minikube) - поднятие dev-кластера для ComicsHub
Этот документ описывает, как я поднимал Kubernetes-кластер для проекта **ComicsHub** на **macOS** через **minikube + driver=docker** и деплоил сервисы через **Kustomize**.

```bash
# 1) старт 3-ноды
minikube start -p comics --driver=docker --nodes=3 --cpus=2 --memory=6144
kubectl config use-context comics

# 2) аддоны
minikube -p comics addons enable metrics-server
minikube -p comics addons enable csi-hostpath-driver

# 3) собрать образы внутрь minikube
minikube -p comics image build --all -t api:latest       -f Dockerfile.api       search-services
minikube -p comics image build --all -t words:latest     -f Dockerfile.words     search-services
minikube -p comics image build --all -t update:latest    -f Dockerfile.update    search-services
minikube -p comics image build --all -t auth:latest      -f Dockerfile.auth      search-services
minikube -p comics image build --all -t search:latest    -f Dockerfile.search    search-services
minikube -p comics image build --all -t favorites:latest -f Dockerfile.favorites search-services
minikube -p comics image build --all -t comicsbot:latest -f Dockerfile.bot       bot/comicsbot

# NOTE: Release-режим (GHCR)
# В будущем добавлю сборку/публикацию образов в GHCR и переключу k8s/overlays/minikube на ghcr.io/...:<tag>.
# Тогда шаг 3 можно будет пропустить - кластер сам скачает готовые образы из registry.


# 4) деплой
kubectl kustomize k8s/overlays/minikube | head
kubectl apply -k k8s/overlays/minikube

# 5) проверить поды
kubectl -n comics get pods -o wide
```

Доступ к API на macOS:
```bash
# вариант 1 - tunnel
minikube -p comics service -n comics api --url

# вариант 2 - port-forward
kubectl -n comics port-forward svc/api 8080:8080
kubectl -n comics port-forward svc/api 8081:8081
```

# Теперь более подробнее про каждый шаг

## 0) Предпосылки и заметки

### Почему на Docker Desktop можно упереться в лимиты
Я пробовал стартануть так:
```bash
minikube start -p comics --driver=docker --nodes=3 --cpus=4 --memory=8192
```

Но Docker Desktop ругнулся (памяти меньше, чем я запросил):
```text
Exiting due to MK_USAGE: Docker Desktop has only 7837MB memory but you specified 8192MB
```

Решение: либо увеличить лимиты в Docker Desktop, либо просто уменьшить параметры
```bash
minikube start -p comics --driver=docker --nodes=3 --cpus=2 --memory=6144
```

### Про версию kubectl
Minikube может предупредить:
```text
kubectl is version X, which may have incompatibilities with Kubernetes Y
Want kubectl vY? Try 'minikube kubectl -- get pods -A'
```

В большинстве dev-сценариев это не критично, но если что-то странное - можно использовать:
```bash
minikube -p comics kubectl -- get pods -A
```

## 1) Создание кластера (3 ноды)

```bash
minikube start -p comics --driver=docker --nodes=3 --cpus=2 --memory=6144
kubectl config use-context comics
```

Проверки:
```bash
kubectl get nodes -o wide
kubectl get pods -A
```

## 2) Addons ( metrics / csi-hostpath)

Я включал:
```bash
minikube -p comics addons enable metrics-server
minikube -p comics addons enable csi-hostpath-driver
```

Проверить, что всё поднялось:
```bash
kubectl get pods -A
kubectl top nodes   # работает только если metrics-server включен
kubectl top pods -n comics
```

Посмотреть список аддонов:
```bash
minikube -p comics addons list
```

## 3) Как Kubernetes увидит Docker images

Вариант 1 (A) - я использовал в этом репозитории. Он самый простой и предсказуемый для Minikube.  
Вариант 2 (B) - то же самое, но через обычный `docker build`: мы переключаем окружение Docker так, чтобы сборка шла в daemon, который использует Minikube (и тогда kubelet видит образы без загрузки в registry).  
Вариант 3 (C) - пригодится позже, когда появятся релизные образы в registry (GHCR) или когда образы уже собраны локально - тогда можно загрузить их в Minikube и пропустить сборку.
### Вариант A - `minikube image build`
Собираем образы напрямую “внутрь” minikube:

```bash
minikube -p comics image build --all -t api:latest       -f Dockerfile.api       search-services
minikube -p comics image build --all -t words:latest     -f Dockerfile.words     search-services
minikube -p comics image build --all -t update:latest    -f Dockerfile.update    search-services
minikube -p comics image build --all -t auth:latest      -f Dockerfile.auth      search-services
minikube -p comics image build --all -t search:latest    -f Dockerfile.search    search-services
minikube -p comics image build --all -t favorites:latest -f Dockerfile.favorites search-services
minikube -p comics image build --all -t comicsbot:latest -f Dockerfile.bot       bot/comicsbot
```

Проверить, что образы есть в minikube:
```bash
minikube -p comics image ls | head
```

### Вариант B - `eval $(minikube docker-env)` + обычный `docker build`
Подходит, если хочется собирать стандартным docker, но в docker-демон minikube:

```bash
eval $(minikube -p comics docker-env)

docker build -t api:latest       -f search-services/Dockerfile.api       search-services
docker build -t words:latest     -f search-services/Dockerfile.words     search-services
docker build -t update:latest    -f search-services/Dockerfile.update    search-services
docker build -t auth:latest      -f search-services/Dockerfile.auth      search-services
docker build -t search:latest    -f search-services/Dockerfile.search    search-services
docker build -t favorites:latest -f search-services/Dockerfile.favorites search-services
docker build -t comicsbot:latest -f bot/comicsbot/Dockerfile.bot         bot/comicsbot

docker images | head  # покажет images внутри окружения minikube
```

Вернуть окружение обратно:
```bash
eval $(minikube -p comics docker-env -u)
```

### Вариант C - `minikube image load`
Удобно, если образы уже собраны локально или скачаны из registry:
```bash
minikube -p comics image load api:latest
```

## 4) ConfigMaps для `config.yaml` (как “volume mount” из compose)

В compose я делал:
```yaml
volumes:
  - ./search-services/api/config.yaml:/config.yaml
```

В Kubernetes это стало:
- `ConfigMap` (файл `config.yaml`)
- `volumeMount` в контейнер на `/config.yaml`

Я вынес `configs/*/config.yaml` в `k8s/base/configs/...` и сделал генерацию через kustomize:

```yaml
configMapGenerator:
  - name: api-config
    files:
      - config.yaml=configs/api/config.yaml
  - name: words-config
    files:
      - config.yaml=configs/words/config.yaml
  - name: update-config
    files:
      - config.yaml=configs/update/config.yaml
  - name: auth-config
    files:
      - config.yaml=configs/auth/config.yaml
  - name: search-config
    files:
      - config.yaml=configs/search/config.yaml
  - name: favorites-config
    files:
      - config.yaml=configs/favorites/config.yaml
```


## 5) Secrets (kustomize secretGenerator)

Я использую `secrets.env` в overlay и генерю секрет через kustomize.

Важно: следить за **namespace**. У меня один раз секрет уехал в `default`, поэтому пришлось удалить:
```bash
kubectl delete secret app-secrets -n default --ignore-not-found
```

Проверки:
```bash
kubectl get secret -n comics app-secrets
kubectl describe secret -n comics app-secrets
```

## 6) Postgres как StatefulSet + PVC

Postgres поднимается как `StatefulSet` и использует `PVC` (persisted storage).
Для dev-среды minikube этого достаточно (особенно с `csi-hostpath-driver`).

---

## 7) NATS как Deployment + Service

NATS - обычный `Deployment` + `Service` внутри кластера.


## 8) Сервисы как Deployment + Service

Соответствие моему compose:

- `api` -> Deployment (можно `replicas: 2`) + Service (ClusterIP) + NodePort/Ingress наружу
- `words` -> Deployment + Service
- `update` -> Deployment + Service
- `auth` -> Deployment + Service
- `search` -> Deployment + Service
- `favorites` -> Deployment + Service
- `bot` -> Deployment (обычно `replicas: 1`), Service не нужен (нет входящего трафика)


## 9) Overlay для Minikube (NodePort)

Чтобы открыть API наружу в dev, я патчу сервис `api` на `NodePort`.
Пример `api-nodeport.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: api
spec:
  type: NodePort
  ports:
    - name: http
      port: 8080
      targetPort: 8080
      nodePort: 30080
    - name: internal
      port: 8081
      targetPort: 8081
      nodePort: 30081
```


## 10) Деплой одной командой

Перед применением (быстрая проверка рендера):
```bash
kubectl kustomize k8s/overlays/minikube | head
```

Применить:
```bash
kubectl apply -k k8s/overlays/minikube
```

Проверить поды:
```bash
kubectl -n comics get pods -o wide
kubectl -n comics get svc
kubectl -n comics get events --sort-by=.metadata.creationTimestamp | tail -n 40
```

Перезапустить деплои (когда поменялись конфиги/секреты/образы):
```bash
kubectl -n comics rollout restart deploy api auth search update favorites bot
kubectl -n comics rollout status deploy/api
```

---

## 11) Доступ к API на macOS (важный момент)

На **macOS + minikube driver=docker** кластер живёт внутри Docker Desktop VM, поэтому NodePort часто **не доступен напрямую** как в VM-драйверах.
В итоге: используем **tunnel** или **port-forward**.

### Вариант 1 - tunnel (удобный вариант)
```bash
minikube -p comics service -n comics api --url
```

Он выдаст URL. Туннель нужно держать открытым.
Дёргаем:
- `http://127.0.0.1:<port>/api/ping`

### Вариант 2 - port-forward
```bash
kubectl -n comics port-forward svc/api 8080:8080
kubectl -n comics port-forward svc/api 8081:8081
```

Дёргаем:
- `http://127.0.0.1:8080/api/ping`


## 12) Команды для проверки кластера (шпаргалка)

```
### Общие
kubectl config current-context             # какой кластер/контекст сейчас активен
kubectl get nodes -o wide                  # список нод + IP/версия/роль (проверить что все ready)
kubectl get pods -A                        # все поды во всех namespaces (видно coredns/metrics-server)

kubectl -n comics get all                  # общий обзор ресурсов в namespace comics (pods/svc/deploy/rs)
kubectl -n comics get pods -o wide         # поды в comics полная информация
kubectl -n comics get svc                  # сервисы (ClusterIP/NodePort), порты и тип сервиса
kubectl -n comics get cm                   # configmap (обычно config.yaml для сервисов)
kubectl -n comics get secret               # секреты (jwt/db creds и т.д.)

### Диагностика проблем
kubectl -n comics describe pod <pod>       # подробности по pod: события, причины Pending/CrashLoopBackOff, env/volumes
kubectl -n comics logs <pod> --since=10m   # логи конкретного pod за последние 10 минут
kubectl -n comics logs deploy/<name> --since=10m  # логи deployment (kubectl выберет один pod из реплик)
kubectl -n comics get events --sort-by=.metadata.creationTimestamp | tail -n 60  # последние события (часто там вся правда)

### Проверка сети и DNS (внутри кластера)
kubectl -n comics exec -it deploy/api -- sh    # зайти в контейнер api (проверить DNS/сеть изнутри)
# nslookup search                               # проверка, резолвится ли service-name "search"
# wget -qO- http://search:8080/ping             # проверить доступность search по service DNS/порту
# wget -qO- http://words:8080/ping              # аналогично для words (если у него есть ping по HTTP)

### Полезное про ресурсные метрики (если включён metrics-server)
kubectl top nodes                          # cpu/ram по нодам (нагрузка на кластер)
kubectl top pods -n comics                 # cpu/ram по подам в namespace comics
```

## 13) Очистка

Удалить деплой приложения:
```bash
kubectl delete -k k8s/overlays/minikube
```

Удалить кластер minikube:
```bash
minikube delete -p comics
```

