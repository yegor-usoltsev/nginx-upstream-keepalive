# nginx-upstream-keepalive

This repository demonstrates the proper configuration for enabling HTTP keep-alive on upstream servers when using [NGINX](https://github.com/nginx/nginx) as a reverse proxy. Enabling HTTP keep-alive in this setup can significantly improve performance by:

- **Reducing CPU load** on upstream servers by minimizing the number of new connections required.
- **Improving request latency** by reusing connections, which also enhances the ability to handle high request volumes.

This repository was created to address questions raised in the [following Pull Request](https://github.com/antonputra/tutorials/pull/334).

## TL;DR

To configure NGINX optimally as a reverse proxy with HTTP keep-alive support, use the following configuration:

```nginx
server {
    location / {
        # Reference to "upstream" block with the name "backend" (see below)
        proxy_pass http://backend;
        # Use HTTP/1.1 instead of HTTP/1.0 for upstream connections
        proxy_http_version 1.1;
        # Remove any "Connection: close" header
        proxy_set_header Connection "";
    }
}

upstream backend {
    server ...;
    # Maintain up to 16 idle keep-alive connections from NGINX to upstream servers
    keepalive 16;
}
```

## Overview

This repository includes the following files:

- **`main.go`**: A simple HTTP server implemented in Go, serving as the NGINX upstream. It listens on port **8080** and logs requests, making it easy to check if HTTP keep-alive is active.
- **`nginx.conf`**: A minimal NGINX configuration file with 4 `server` blocks:
  - The 1st block (port **8081**) uses only the standard `proxy_pass`.
  - The 2nd block (port **8082**) adds `proxy_http_version 1.1`.
  - The 3rt block (port **8083**) adds `proxy_set_header Connection ""`.
  - The 4th block (port **8084**) includes an `upstream` block with `keepalive` enabled.
- **`docker-compose.yaml` & `Dockerfile`**: A Docker Compose setup to run the Go server with NGINX as a reverse proxy.

### Prerequisites

To run this example, you’ll need **Docker Compose** and **curl** as the client.

### Running

To start the Go server and NGINX proxy, run:

```shell
docker compose up -d --build
```

To observe logs from both NGINX and the Go server, use:

```shell
docker compose logs -f
```

When finished, you can stop the applications with:

```shell
docker compose down
```

## Results

### Step 0: Verifying Go Server Supports HTTP Keep-Alive

First, ensure that the Go server supports HTTP keep-alive by making 3 consecutive requests directly to it on port **8080** using `curl`. Check if the connection is reused:

```shell
curl -sv http://localhost:8080 http://localhost:8080 http://localhost:8080
```

In the `curl` output, you should see:

```
* Connection #0 to host localhost left intact
...
* Re-using existing connection with host localhost
```

This indicates that `curl` opened a connection for the first request and reused it for the next two. Additionally, in the Go server logs, you should see:

```
Received request from 192.168.107.1:55694 | Protocol: HTTP/1.1 | Will be closed: false
...
Received request from 192.168.107.1:55694 | Protocol: HTTP/1.1 | Will be closed: false
...
Received request from 192.168.107.1:55694 | Protocol: HTTP/1.1 | Will be closed: false
Request headers:
  User-Agent: curl/8.7.1
  Accept: */*
```

Each request uses the same port (`55694`), confirming that the **connection was reused**.

### Step 1: NGINX with Standard `proxy_pass`

Next, let's test NGINX with only the `proxy_pass` directive:

```nginx
server {
    listen 8081;
    location / {
        proxy_pass http://golang:8080;
    }
}
```

Run 3 requests to port **8081**:

```shell
curl -sv http://localhost:8081 http://localhost:8081 http://localhost:8081
```

```
Received request from 192.168.107.3:42110 | Protocol: HTTP/1.0 | Will be closed: true
...
Received request from 192.168.107.3:42122 | Protocol: HTTP/1.0 | Will be closed: true
...
Received request from 192.168.107.3:42136 | Protocol: HTTP/1.0 | Will be closed: true
Request headers:
  Connection: close
  User-Agent: curl/8.7.1
  Accept: */*
```

In the Go server logs, you will see that each request originates from different ports (`42110`, `42122`, `42136`), showing that **connections were not reused**. This happens because NGINX defaults to `HTTP/1.0` for upstream connections, which lacks connection reuse.

### Step 2: NGINX Upgraded to HTTP/1.1

Enable HTTP/1.1 by adding `proxy_http_version 1.1`:

```diff
 server {
-    listen 8081;
+    listen 8082;
     location / {
         proxy_pass http://golang:8080;
+        proxy_http_version 1.1;
     }
 }
```

Run 3 requests to port **8082**:

```shell
curl -sv http://localhost:8082 http://localhost:8082 http://localhost:8082
```

```
Received request from 192.168.107.3:60914 | Protocol: HTTP/1.1 | Will be closed: true
...
Received request from 192.168.107.3:60918 | Protocol: HTTP/1.1 | Will be closed: true
...
Received request from 192.168.107.3:60926 | Protocol: HTTP/1.1 | Will be closed: true
Request headers:
  User-Agent: curl/8.7.1
  Accept: */*
  Connection: close
```

The Go server logs show requests from different ports (`60914`, `60918`, `60926`), meaning **connections were not reused**. This happens because NGINX adds a `Connection: close` header by default, which instructs the upstream server to close the connection after each request.

### Step 3: NGINX Without `Connection: close` Header

Remove the `Connection: close` header by adding `proxy_set_header Connection "";`:

```diff
 server {
-    listen 8082;
+    listen 8083;
     location / {
         proxy_pass http://golang:8080;
         proxy_http_version 1.1;
+        proxy_set_header Connection "";
     }
 }
```

Run 3 requests to port **8083**:

```shell
curl -sv http://localhost:8083 http://localhost:8083 http://localhost:8083
```

```
Received request from 192.168.107.3:49260 | Protocol: HTTP/1.1 | Will be closed: false
...
Received request from 192.168.107.3:49270 | Protocol: HTTP/1.1 | Will be closed: false
...
Received request from 192.168.107.3:49274 | Protocol: HTTP/1.1 | Will be closed: false
Request headers:
  User-Agent: curl/8.7.1
  Accept: */*
```

Despite removing `Connection: close`, NGINX still **does not reuse connections**, closing them automatically after each request.

### Step 4: NGINX with `keepalive`

To enable connection reuse, define an `upstream` block with `keepalive` (see [NGINX docs: ngx_http_upstream_module](https://nginx.org/en/docs/http/ngx_http_upstream_module.html#keepalive)). This specifies the maximum number of idle keep-alive connections per worker process. Here is the final configuration:

```diff
 server {
-    listen 8083;
+    listen 8084;
     location / {
-        proxy_pass http://golang:8080;
+        proxy_pass http://backend;
         proxy_http_version 1.1;
         proxy_set_header Connection "";
     }
 }

+upstream backend {
+    server golang:8080;
+    keepalive 16;
+}
```

Run 3 requests to port **8084**:

```shell
curl -sv http://localhost:8084 http://localhost:8084 http://localhost:8084
```

```
Received request from 192.168.107.3:55980 | Protocol: HTTP/1.1 | Will be closed: false
...
Received request from 192.168.107.3:55980 | Protocol: HTTP/1.1 | Will be closed: false
...
Received request from 192.168.107.3:55980 | Protocol: HTTP/1.1 | Will be closed: false
Request headers:
  User-Agent: curl/8.7.1
  Accept: */*
```

Finally! In the Go server logs, all requests come from the same port (`55980`), confirming that the **connection was reused**.

## References

- [NGINX blog: 10 Tips for 10x Application Performance](https://www.f5.com/company/blog/nginx/10-tips-for-10x-application-performance#web-server-tuning) (Tip 9 – Tune Your Web Server for Performance)
- [NGINX blog: HTTP Keepalive Connections and Web Performance](https://www.f5.com/company/blog/nginx/http-keepalives-and-web-performance)
- [NGINX docs: ngx_http_upstream_module](https://nginx.org/en/docs/http/ngx_http_upstream_module.html#keepalive)

## Bonus: Comparing NGINX with Other Reverse Proxies

I also tested several popular open-source projects commonly used as reverse proxies, each with its default configuration:

- [Apache HTTP Server](https://github.com/apache/httpd) (port **9090**)
- [Caddy](https://github.com/caddyserver/caddy) (port **9091**)
- [Envoy](https://github.com/envoyproxy/envoy) (port **9092**)
- [HAProxy](https://github.com/haproxy/haproxy) (port **9093**)
- [Traefik](https://github.com/traefik/traefik) (port **9094**)

All configurations for these proxies can be found in the **[bonus](bonus)** directory.

Results: **All of these proxies use HTTP keep-alive by default**, unlike NGINX, which requires additional setup.

To run these tests yourself, start Docker Compose with the `bonus` profile:

```shell
docker compose --profile bonus up -d --build
```

To observe logs all applications, use:

```shell
docker compose --profile bonus logs -f
```

When finished, stop all applications with:

```shell
docker compose --profile bonus down
```

To send a sequence of HTTP requests to each proxy:

```bash
for PORT in 9090 9091 9092 9093 9094; do
  curl -sv http://localhost:$PORT http://localhost:$PORT http://localhost:$PORT
done
```
