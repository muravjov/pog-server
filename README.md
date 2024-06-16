# Proxy over gRPC

pog-server is a HTTP proxy which uses gRPC for sending bytes:
    User <= (HTTP proxying) => pog client <= (gRPC) => pog server <= (HTTP proxying) => destination server

# Applications of Proxy over gRPC (PoG)

* terraform: if https://registry.terraform.io/ is `not currently available in your region`, one can use it with a proxy like:
```bash
$ export HTTPS_PROXY=http://localhost:18080
$ terraform init
...
```

* If a site is not available from your IP address, use the proxy, e.g. with Google Chrome:
```bash
$ /Applications/Chromium.app/Contents/MacOS/Chromium --proxy-server="http://localhost:18080"
```

# Simple example of use

The pog server:
```bash
$ go run ./grpcproxy/server
2024/06/15 13:28:02 proxy-over-grpc server, version: dev
2024/06/15 13:28:02 starting on port 8080
2024/06/15 13:28:02 PID: 43396
2024/06/15 13:28:02 waiting for termination signal...
pog: ifconfig.me:443 anonymous HTTPS [::1]:51225 [2024-06-15T13:28:12+03:00] OK
```

The pog client:
```bash
$ SERVER_ADDR=localhost:8080 INSECURE=1 go run ./grpcproxy/client
2024/06/15 13:28:07 proxy-over-grpc client listening address :18080
2024/06/15 13:28:07 PID: 43580
2024/06/15 13:28:07 waiting for termination signal...
pog: ifconfig.me:443 anonymous HTTPS 127.0.0.1:51226 [2024-06-15T13:28:12+03:00] 200
```

A user of the HTTP proxy:
```bash
$ curl -i --proxy http://localhost:18080 https://ifconfig.me
HTTP/1.1 200 OK
Date: Sat, 15 Jun 2024 10:28:12 GMT
Transfer-Encoding: chunked

HTTP/2 200
date: Sat, 15 Jun 2024 10:28:12 GMT
content-type: text/plain
content-length: 14
access-control-allow-origin: *
via: 1.1 google
alt-svc: h3=":443"; ma=2592000,h3-29=":443"; ma=2592000

1.136.246.102
```

Here the user reaches the destination URL ifconfig.me via the HTTP proxy at https://localhost:18080 .

# How to build

[Go](https://go.dev/) programming language version > 1.21 is required.

```bash
go build -o server ./grpcproxy/server
go build -o client ./grpcproxy/client
go build -o client ./grpcproxy/genauthitem
```

# How to build Docker image

```bash
$ docker build --build-arg GRPCPROXY_COMMIT=$(git rev-parse --short HEAD) \
  -f grpcproxy/server/Dockerfile -t pog-server:dev .
```

# How to deploy to Google Cloud Run using Terraform[^terraform-vs-console]

[^terraform-vs-console]: One can deploy a Cloud Run service with the [console](https://console.cloud.google.com/), too.

The server part can be deployed into GCP Cloud Run, see `terraform/pog-server.tf` as an example. First, let's gate our server service with auth with login and password:

```bash
$ docker run --rm --entrypoint /genauthitem catbo.net/pog-server:dev --name pog-client --password password --timeToLive 100000h
{"name":"pog-client","hash":"$2a$10$oiV27ssmxy3ihPYA4w.rIOAH2eUQOnwCXoHL4PKXSZz2goKvL.Nwq","exp_date":"2035-11-12T05:22:05Z"}
```
This JSON value we assign to the env variable POG_AUTH_ITEM1, see `terraform/pog-server.tf`

Having a GCP project `PROJECT`, do:

```bash
$ cd terraform
$ terraform version
$ terraform init
$ terraform plan
$ terraform apply

# our deployed service
$ gcloud run services list
   SERVICE     REGION    URL                                         LAST DEPLOYED BY       LAST DEPLOYED AT
✔  pog-server  us-east4  https://pog-server-XXXXXXXXXX-uk.a.run.app  name@example.com  2024-04-29T21:35:50.461542Z
```

Optionally, let's gate user requests to our client service, login=user and password=password:
```bash
$ docker run --rm --entrypoint /genauthitem catbo.net/pog-server:dev --name user --password password --timeToLive 100000h
{"name":"user","hash":"$2a$10$uIGiYtUKu5OJtCfRO95yIOoE.udSSjMoTc1CqCXe3iRHpS4DA.b9m","exp_date":"2035-11-12T05:22:55Z"}
```

The client part:
```bash
$ cat .env.pog_client_scanly
METRIC_NAMESPACE=pog_client

SERVER_ADDR=pog-server-XXXXXXXXXX-uk.a.run.app:443

CLIENT_LISTEN=:18080
CLIENT_POG_AUTH=pog-client:password

# auth for users
CLIENT_AUTH_USER1={"name":"user","hash":"$2a$10$uIGiYtUKu5OJtCfRO95yIOoE.udSSjMoTc1CqCXe3iRHpS4DA.b9m","exp_date":"2035-11-12T05:22:55Z"}

MUX_SERVER_METRICS=1

$ docker run --rm -it \
       --env-file .env.pog_client_scanly \
       --name my-proxy-client -p 18080:18080 --entrypoint /client pog-server:dev
pog: ifconfig.me:443 ilya HTTPS 172.17.0.1:60748 [2024-06-15T12:53:42Z] 200
```

Finally, the user request:
```bash
$ curl -is --proxy http://user:password@localhost:18080 https://ifconfig.me | tee >(head -n1) >(tail -n1) >/dev/null
HTTP/1.1 200 OK
2600:1900:2000:ec::1:100
```

# Optional tweaks to GCP service config (`pog-server.tf`)

For a single home usage, the Cloud Run costs might be not affordable (around 15$/m), so one might try those tweaks:

```diff
$ git diff 89e3493376..HEAD terraform/pog-server.tf
diff --git a/terraform/pog-server.tf b/terraform/pog-server.tf
index 4194f47..d7bf4e0 100644
--- a/terraform/pog-server.tf
+++ b/terraform/pog-server.tf
@@ -17,9 +17,14 @@ EOVALUE
 
       resources {
         limits = {
-          cpu    = "1000m"
-          memory = "1Gi"
+          # cannot make it smaller like 0.5 = 500m because in that case
+          # one must set max_instance_request_concurrency = 1 => no parallel request processing (we have now it equals to 80)
+          cpu = "1000m"
+          # minimum is 128 MiB for first generation,
+          # https://cloud.google.com/run/docs/configuring/services/memory-limits#memory-minimum
+          memory = "128Mi"
         }
+        cpu_idle = true
       }
 
       ports {
@@ -33,6 +38,8 @@ EOVALUE
       max_instance_count = 10
 
     }
+
+    session_affinity = true
   }
 }
```

# Options

All the options are represented as environment variables.

The server part options:
| Variable                           | Description                                   |
|------------------------------------|-----------------------------------------------|
| PORT                     | Port to listen to. Default: `8080`|
| POG_AUTH_*               | Enables authorization for PoG clients. Use `genauthitem` to generate JSON values |
| GRPC_AND_HTTP_MUX        | Listen to both gRPC and HTTP requests (/metrics). Default: `1` (enabled) |

The client part options:
| Variable                 | Description                                   |
|--------------------------|-----------------------------------------------|
| SERVER_ADDR              | PoG server address (host:port). **Required**. Example: `localhost:8080` |
| INSECURE                 | Skip SSL validation. Default: `` (false)      |
| CLIENT_LISTEN            | Client address to listen to ([host]:port). Default: `:18080` |
| CLIENT_POG_AUTH          | Auth string to connect to PoG server, in the form `user:password` |
| CLIENT_AUTH_*            | Enables authorization for proxy users. Use `genauthitem` to generate JSON values |
| MUX_SERVER_METRICS       | Serve both server and client `Prometheus` metrics from `/metrics`, iff there is any connection to the server. Default: `` (false) |

The common options:
| Variable                 | Description                                   |
|--------------------------|-----------------------------------------------|
| DISABLE_ACCESS_LOGGING   | Disables request logging in the form `pog: ifconfig.me:443 ilya HTTPS 172.17.0.1:60748 [2024-06-15T12:53:42Z] 200` |
| METRIC_NAMESPACE         | Prepends `Prometheus` metrics with a prefix (useful to avoid confusion between server and client metrics in case of `MUX_SERVER_METRICS`) |
| GRPC_BUILTIN_METRICS     | Populates `/metrics` with the builtin gRPC metrics. Default: `1` (enabled) |

# Metrics and operations

Both PoG server and client provides Prometheus metrics at `/metrics`. An example:
```bash
$ curl -is http://localhost:18080/metrics | grep -E '(rpcs|tunnelling|mux|expiry)'
# HELP grpc_io_client_completed_rpcs Number of completed RPCs by method and status.
# TYPE grpc_io_client_completed_rpcs counter
grpc_io_client_completed_rpcs{grpc_client_method="HTTPProxy/Run",grpc_client_status="CANCELLED"} 25
grpc_io_client_completed_rpcs{grpc_client_method="HTTPProxy/Run",grpc_client_status="INTERNAL"} 3
grpc_io_client_completed_rpcs{grpc_client_method="HTTPProxy/Run",grpc_client_status="OK"} 2
# HELP grpc_io_client_started_rpcs Number of opened client RPCs, by method.
# TYPE grpc_io_client_started_rpcs counter
grpc_io_client_started_rpcs{grpc_client_method="HTTPProxy/Run"} 58
# HELP pog_client_auth_item_earliest_expiry Returns earliest auth item expiry in unixtime
# TYPE pog_client_auth_item_earliest_expiry gauge
pog_client_auth_item_earliest_expiry{name="CLIENT_AUTH_"} 1.724295809e+09
# HELP pog_client_server_client_metrics_mux_errors_total Number of errors while getting pog server's /metrics
# TYPE pog_client_server_client_metrics_mux_errors_total counter
pog_client_server_client_metrics_mux_errors_total{name="ok"} 1
# HELP pog_client_tunnelling_connections_total Number of connections tunneling through the proxy.
# TYPE pog_client_tunnelling_connections_total gauge
pog_client_tunnelling_connections_total 28
# HELP auth_item_earliest_expiry Returns earliest auth item expiry in unixtime
# TYPE auth_item_earliest_expiry gauge
auth_item_earliest_expiry{name="POG_AUTH_"} 1.724185253e+09
# HELP grpc_io_server_completed_rpcs Number of completed RPCs by method and status.
# TYPE grpc_io_server_completed_rpcs counter
grpc_io_server_completed_rpcs{grpc_server_method="HTTPProxy/Run",grpc_server_status="OK"} 9
grpc_io_server_completed_rpcs{grpc_server_method="HTTPProxy/Run",grpc_server_status="UNAVAILABLE"} 20
# HELP grpc_io_server_started_rpcs Number of opened server RPCs, by method.
# TYPE grpc_io_server_started_rpcs counter
grpc_io_server_started_rpcs{grpc_server_method="HTTPProxy/Run"} 57
# HELP tunnelling_connections_total Number of connections tunneling through the proxy.
# TYPE tunnelling_connections_total gauge
tunnelling_connections_total 28
```

What does it mean:
* `rpcs`: stats how much requests were processed and their success
* `tunnelling_connections_total`: a gauge featuring how many connections are being proccessed now; if the value is growing over time then there is a memory leak
* `auth_item_earliest_expiry`: time when a user account is to expire (both at server and client side)
