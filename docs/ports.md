# Ports — nn-friends (trimmed)

> Trimmed copy for this repository; the canonical document lives at
> `Openpak/ports.md` and governs. Last synchronised 2026-09-15.

One block per concern, nothing below 20000, nothing at or above 27000 (Photon-Nextendo and
Steam live there). Every service reads its listeners from `<SVC>_HTTP_ADDR`, `<SVC>_GRPC_ADDR`,
`<SVC>_METRICS_ADDR`; the values below are the defaults and the local-run convention. Each
service owns a block of ten: +0 HTTP, +1 gRPC, +2 metrics/pprof, +3..+9 spare.

## 20060 block

| Block | Service | HTTP | gRPC | metrics |
| --- | --- | --- | --- | --- |
| 20060 | `nn-friends` (Wii U/3DS friends, gRPC side) | — | 20061 | 20062 |

## Game transport (UDP)

| Port(s) | Service |
| --- | --- |
| 22360 / 22361 | `nn-friends` — Wii U/3DS friends NEX authentication / secure (PRUDP over UDP, deployed 2026-09-10) |

Other services' rows live in the canonical ports.md.
