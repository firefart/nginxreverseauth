# nginxreverseauth

This small webserver can be used with the nginx [http_auth_request_module](https://nginx.org/en/docs/http/ngx_http_auth_request_module.html).

It will do a lookup of the domain in `dynamic_domains` and checks the ip of the requesting client matches the domain ip and returns a http 200 if both matches. This way you can use the module with dynamic ips that have dynamic dns entries. DNS results are currently cached for one hour.

Example nginx config to use with this:

```conf
location /private/ {
    auth_request /auth;
    ...
}

location = /auth {
  allow 127.0.0.0/8;
  allow 10.0.0.0/8;
  allow 172.16.0.0/12;
  allow 192.168.0.0/16;
  deny all;

  proxy_pass_request_body off;
  proxy_set_header Content-Length "";
  proxy_set_header X-Original-URI $request_uri;
  proxy_set_header X-Real-IP $remote_addr;
  proxy_pass http://localhost:8081;
}
```

To make this work you need to build the binary yourself or download it from the release section.

You need a config file containing all your dynamic dns records that you would like to allow. Just copy `config.sample.json` and edit it to your needs.

## Config Options

You can either set the options via command line parameters, environment variables or create a `.env` file containing the environment variable names. When you use `.env` the environment variables will be populated with the values.

| Commandline parameter | env variable/.env entry  | default | description |
|---|---|---|---|
| `-host` | `NGINX_HOST` | 127.0.0.1:8080 | IP and Port to bind to |
| `-debug` | `NGINX_DEBUG` | false | enable debug output |
| `-graceful-timeout` | `NGINX_GRACEFUL_TIMEOUT` | 2s | the duration for which the server waits on exit for existing connections to finish |
| `-timeout` | `NGINX_TIMEOUT` | 5s | dns and http timeout |
| `-config` | `NGINX_CONFIG` | config.json | config file to use |
