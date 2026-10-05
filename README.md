# nginx-upstream-keepalive

A runnable comparison of four NGINX 1.27.2 reverse-proxy configurations against a logging Go server. It shows how to reuse HTTP connections from NGINX to an upstream: use HTTP/1.1, clear the default `Connection: close` header, and configure an upstream keepalive cache.

The Go server logs the protocol and peer address for each request, so you can see whether NGINX reuses an upstream connection.

## Contents

- [Quick start](#quick-start)
- [Compare the NGINX configurations](#compare-the-nginx-configurations)
- [Configuration](#configuration)
- [WebSocket requests](#websocket-requests)
- [Other proxy examples](#other-proxy-examples)
- [NGINX version note](#nginx-version-note)

## Quick start

Prerequisites: Docker Compose and curl.

Start the Go server and NGINX:

```sh
docker compose up -d --build
```

Send three requests through the keepalive-enabled NGINX server:

```sh
curl -sv http://localhost:8084/ http://localhost:8084/ http://localhost:8084/
```

Inspect the Go server log:

```sh
docker compose logs -f golang
```

For port 8084, the Go log should show HTTP/1.1 requests with the same peer address and port. To confirm that the Go server itself supports keep-alive, send the same three requests to port 8080 and check that curl reports `Re-using existing connection`. Stop the services with `docker compose down`.

## Compare the NGINX configurations

Run the same three-request sequence against each port:

```sh
for PORT in 8081 8082 8083 8084; do
  echo "--- port $PORT ---"
  curl -sv "http://localhost:$PORT/" "http://localhost:$PORT/" "http://localhost:$PORT/"
done
```

The Go server log reports the upstream protocol, whether the request asked to close, and the peer address:

| Port | Configuration | Expected upstream behavior |
| --- | --- | --- |
| 8081 | Default `proxy_pass` | HTTP/1.0; the connection closes after each request. |
| 8082 | Adds `proxy_http_version 1.1` | HTTP/1.1, but `Connection: close` still prevents reuse. |
| 8083 | Clears `Connection` unless the client requests an upgrade | HTTP/1.1; connections close after each request because no upstream keepalive cache is configured. |
| 8084 | Adds an upstream block with `keepalive 2` | Repeated requests reuse an idle upstream connection. |

The client-to-NGINX connection and the NGINX-to-upstream connection are separate. The Go server's peer address shows reuse on the upstream side.

## Configuration

This is the keepalive-enabled configuration used on port 8084:

```nginx
map $http_upgrade $connection_upgrade {
    default upgrade;
    "" "";
}

upstream backend {
    server golang:8080;
    keepalive 2;
}

server {
    listen 8084;

    location / {
        proxy_pass http://backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
    }
}
```

The `keepalive` value is the maximum number of idle upstream connections cached per NGINX worker; it does not limit the total number of connections a worker can open. NGINX recommends a value of about twice the number of servers in the `upstream` block: large enough to keep connections to every server, small enough that upstream servers can still accept new connections. The value `2` fits this single-server demonstration.

## WebSocket requests

The map forwards `Upgrade` and sets `Connection: upgrade` when the client requests a protocol upgrade. Otherwise, the empty value removes the `Connection` header so regular HTTP upstream connections can be reused. WebSocket tunneling also requires an upstream application that supports WebSockets; the included Go handler is a plain HTTP server.

See the [NGINX WebSocket proxying guide](https://nginx.org/en/docs/http/websocket.html).

## Other proxy examples

The optional Compose `bonus` profile compares the same Go upstream with Apache, Caddy, Envoy, HAProxy, and Traefik. Their configurations are in [bonus/](bonus/).

Start the examples:

```sh
docker compose --profile bonus up -d --build
```

Send three requests to each proxy:

```sh
for PORT in 9090 9091 9092 9093 9094; do
  echo "--- port $PORT ---"
  curl -sv "http://localhost:$PORT/" "http://localhost:$PORT/" "http://localhost:$PORT/"
done
```

Stop them with `docker compose --profile bonus down`.

| Proxy | Port |
| --- | ---: |
| Apache | 9090 |
| Caddy | 9091 |
| Envoy | 9092 |
| HAProxy | 9093 |
| Traefik | 9094 |

## NGINX version note

The Compose file pins `nginx:1.27.2-alpine`; the port comparison above reflects that version. NGINX 1.29.7 changed the default proxy HTTP version to 1.1 and enabled the upstream keepalive cache by default, so the first stages behave differently on newer versions. See the [proxy module](https://nginx.org/en/docs/http/ngx_http_proxy_module.html) and [upstream module](https://nginx.org/en/docs/http/ngx_http_upstream_module.html) documentation.

## References

- [NGINX upstream module](https://nginx.org/en/docs/http/ngx_http_upstream_module.html#keepalive)
- [NGINX blog: Avoiding the Top 10 NGINX Configuration Mistakes](https://www.f5.com/company/blog/nginx/avoiding-top-10-nginx-configuration-mistakes#no-keepalives) (mistake 3: not enabling keepalive connections to upstream servers)
- [NGINX blog: HTTP Keepalive Connections and Web Performance](https://www.f5.com/company/blog/nginx/http-keepalives-and-web-performance)
- [NGINX blog: 10 Tips for 10x Application Performance](https://www.f5.com/company/blog/nginx/10-tips-for-10x-application-performance#web-server-tuning) (tip 9)
- [Discussion that prompted this example](https://github.com/antonputra/tutorials/pull/334)

## Contributing

Pull requests are welcome. For major changes, please [open an issue](https://github.com/yegor-usoltsev/nginx-upstream-keepalive/issues/new) first.

## License

[MIT](LICENSE)
