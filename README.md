# OpenPak nn-friends — Wii U/3DS friends server

The NEX friends service (authentication and secure servers, plus the gRPC that game servers
query for presence) for Wii U and 3DS, forked from Pretendo's `friends` (AGPL-3.0) and rewired
onto the OpenPak stack:

- identity comes from [`nn-account`](../nn-account), the Wii U/3DS adapter, over its
  `account.v2` gRPC and its Resolution service (PID <-> core account);
- the friend graph (friendships, requests, blocks) is read and written in the OpenPak
  [account core](../account) through `coregraph`, so web and console agree;
- presence, notifications and Mii data stay local to this server's Postgres.

**Status:** M3 core integration landed; passes end to end against a live core and adapter. No
console or emulator has been run against it yet (`../nn-account/docs/client-testing.md`).

## Run

```sh
cp example.env .env   # fill in required values
go run .
```

Requires PostgreSQL, the account core and nn-account running. A container image is
published on tag as `ghcr.io/openpak/nn-friends` (`.github/workflows`); build locally with
`podman build -t nn-friends .`.

## Configuration
All configuration options are handled via environment variables

`.env` files are supported; `example.env` lists every option.

| Name                                           | Description                                                                                                            | Required                            |
| ---------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- | ----------------------------------- |
| `PN_FRIENDS_CONFIG_POSTGRES_URI`               | Fully qualified URI to your Postgres server (Example `postgres://username:password@localhost/friends?sslmode=disable`) | Yes                                 |
| `PN_FRIENDS_CONFIG_POSTGRES_MAX_CONNECTIONS`   | Postgres server max connections                                                                                        | Yes                                 |
| `PN_FRIENDS_CONFIG_GRPC_API_KEY`               | API key for your GRPC server                                                                                           | No (Assumed to be an open gRPC API) |
| `PN_FRIENDS_CONFIG_GRPC_SERVER_PORT`           | Port for the GRPC server                                                                                               | Yes                                 |
| `PN_FRIENDS_CONFIG_AUTHENTICATION_SERVER_PORT` | Port for the authentication server                                                                                     | Yes                                 |
| `PN_FRIENDS_CONFIG_SECURE_SERVER_HOST`         | Host name for the secure server (should point to the same address as the authentication server)                        | Yes                                 |
| `PN_FRIENDS_CONFIG_SECURE_SERVER_PORT`         | Port for the secure server                                                                                             | Yes                                 |
| `PN_FRIENDS_CONFIG_ACCOUNT_GRPC_HOST`          | Host name for your account server gRPC service                                                                         | Yes                                 |
| `PN_FRIENDS_CONFIG_ACCOUNT_GRPC_PORT`          | Port for your account server gRPC service                                                                              | Yes                                 |
| `PN_FRIENDS_CONFIG_ACCOUNT_GRPC_API_KEY`       | API key for your account server gRPC service                                                                           | No (Assumed to be an open gRPC API) |
| `PN_FRIENDS_CONFIG_HEALTH_CHECK_PORT`          | Port for the basic UDP health check server                                                                             | No                                  |
| `PN_FRIENDS_CONFIG_ENABLE_BELLA`               | Enables a debug user named "Bella" which is always assigned as your friend                                             | No                                  |
| `PN_FRIENDS_CONFIG_MII_DECRYPT_KEY`            | AES key used to decrypt 3DS Mii data (as a hex string)                                                                 | Yes                                 |
| `PN_FRIENDS_CONFIG_PID_HMAC_KEY`               | AES key used for the `pidHMAC` field in accounts                                                                       | Yes                                 |
| `PN_FRIENDS_CONFIG_OPEN_PAK_CORE_HOST`         | Host name of the OpenPak account core internal gRPC                                                                    | Yes                                 |
| `PN_FRIENDS_CONFIG_OPEN_PAK_CORE_PORT`         | Port of the OpenPak account core internal gRPC                                                                         | Yes                                 |
| `PN_FRIENDS_CONFIG_OPEN_PAK_CORE_KEY`          | Shared key the core expects from adapters                                                                              | Yes                                 |

## License

AGPL-3.0-only. Derived from PretendoNetwork/friends (AGPL-3.0).
