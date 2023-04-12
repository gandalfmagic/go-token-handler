# Test data

This directory contains the test data and configuration:

- `backend_services`: a little backend service used to test the proxy of `token-handler`, the service responds on the
  address `:9081`


- `frontend`: a simple html page used to access all the endpoints needed for the tests:
  - `Login` will call the `/login` endpoint of the `token-handler`
  - `Logout` will call the `/logout` endpoint of the `token-handler`
  - `Userinfo page` will call the `/userinfo` endpoint of the `token-handler`, that will get the data from the
    `userinfo` endpoint of the auth server
  - `Proxy page (local test)` will call the endpoint of the test proxy configured by the using the `proxy.yaml` file

You should use the following environment variables to execute `token-handler` in the test environment:

```bash
IS_PRODUCTION=false
LOG_LEVEL=info
OIDC_ISSUER=http://localhost:8080/realms/test01
OIDC_CLIENT_ID=api-gateway
OIDC_CLIENT_SECRET=Dv2SYqkbXEbiiyMJPXludEIvxLaV19KU
OIDC_REDIRECT_URL=http://localhost:9080/callback
OIDC_POST_LOGIN_REDIRECT_URL=http://localhost:8081
OIDC_POST_LOGOUT_REDIRECT_URL=http://localhost:8081
#LISTEN_ADDR=:9080
COOKIE_DOMAIN=localhost
COOKIE_NAME=session
SESSION_AUTH_SECRET=A_RANDOM_VALUE
SESSION_ENC_SECRET=A_RANDOM_VALUE
#SESSION_OLD_AUTH_SECRET=
#SESSION_OLD_ENC_SECRET=
#SESSION_DB_KEY=
#SESSION_OLD_DB_KEY=
PROXY_CONFIG=./test/proxy.yaml
DB_TYPE=sqlite
DB_NAME=./sessions.sqlite
```

> **Note**: you can use any value for `SESSION_AUTH_SECRET` and `SESSION_ENC_SECRET`, the previous values are only
> examples.
