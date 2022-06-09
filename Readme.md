# nginxreverseauth

This small webserver can be used with the nginx [http_auth_request_module](https://nginx.org/en/docs/http/ngx_http_auth_request_module.html).

It will do a lookup of the domain in `dynamic_domains` and checks the ip of the requesting client matches the domain ip and returns a http 200 if both matches. This way you can use the module with dynamic ips that have dynamic dns entries.

Example nginx config to use with this:

```conf
location /private/ {
    auth_request /auth;
    ...
}

location = /auth {
    proxy_pass_request_body off;
    proxy_set_header Content-Length "";
    proxy_set_header X-Original-URI $request_uri;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_pass http://localhost:8081;
}
```
