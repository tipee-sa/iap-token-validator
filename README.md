# iap-token-validator
Validate IAP-issued JWT, to be used with nginx auth_request.

```
Usage of ./iap-token-validator:
  -audience string
        the JWT audience
  -listen string
        listen address (default ":8080")
  -skew int
        the acceptable skew in seconds
  -verbose
        enable verbose logging
```

## Why?

Because https://github.com/imkira/gcp-iap-auth is not longer maintained, uses old libraries with known security flaws, doesn't handle update to the key set properly, doesn't allow for clock skew tolerence and seems a bit complicated for the dead simple task it has to do (in our case).

## Usage (docker)

```bash
#!/bin/sh

TAG=1.0

docker pull ghcr.io/gammadia/iap-token-validator:$TAG
docker rm -f iap-token-validator
docker run -d --name iap-token-validator --restart=unless-stopped \
  -p 127.0.0.1:8403:80 \
  ghcr.io/gammadia/iap-token-validator:$TAG \
    -audience /projects/.../global/backendServices/... \
    -skew 15 \
    -listen :80
```

## Nginx configuration
```
server {
    listen 443 ssl http2;
    ...
    auth_request /iap-token-validator;

    location = /iap-token-validator {
        internal;
        proxy_pass                  http://127.0.0.1:8403/auth;
        proxy_pass_request_body     off;
        proxy_pass_request_headers  off;
        proxy_set_header            X-Goog-IAP-JWT-Assertion $http_x_goog_iap_jwt_assertion;
    }
}

```
