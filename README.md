# Wii U/3DS Friends Server

OpenPak's Wii U/3DS Friends Server.

| | |
| --- | --- |
| Network | Wii U, 3DS |
| Role | NEX friends service: authentication and secure servers, plus the gRPC game servers query for presence |
| Depends on | [`nn-account`](https://github.com/openpak/nn-account), [`account`](https://github.com/openpak/account), PostgreSQL |
| Image | `ghcr.io/openpak/nn-friends` |
| Based on | [Pretendo `friends`](https://github.com/PretendoNetwork/friends) (AGPL-3.0) |
| Licence | AGPL-3.0-only |

## What it serves

| Endpoint | What |
| --- | --- |
| NEX authentication + secure servers | The NEX friends service for Wii U and 3DS |
| gRPC | Presence queries from game servers |

## Run

```sh
cp example.env .env   # fill in required values
go run .
```

## Docs

- [Protocol](docs/protocol.md)
- [Evidence](docs/evidence.md)
- [Deploy](docs/deploy.md)
- [Notes](docs/notes.md)
- [Docs index](docs/README.md)
- [Deployment lifecycle](docs/deployment-lifecycle.md)

## Licence

AGPL-3.0-only. See [LICENSE](LICENSE). Derived from [PretendoNetwork/friends](https://github.com/PretendoNetwork/friends) (AGPL-3.0).
