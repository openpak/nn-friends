# Protocol: nn-friends

## Bans

An account banned in the OpenPak admin cannot sign in (nn-account refuses it; the console is
answered `RendezVous::AccountDisabled`), and its live connections are dropped the moment the
core's `account_banned` event arrives (`coreevents`). See
[`website/docs/ban-lookup.md`](../../../../../platforms/core/infra/website/docs/ban-lookup.md). Nothing to configure.

## Data ownership

The NEX friends service (authentication and secure servers, plus the gRPC that game servers
query for presence) for Wii U and 3DS:

- identity comes from `nn-account` over its `account.v2` gRPC and its Resolution service
  (PID <-> core account);
- the friend graph (friendships, requests, blocks) is read and written in the OpenPak account
  core through `coregraph`, so web and console agree;
- presence, notifications and Mii data stay local to this server's Postgres.
