# Notes: nn-friends

## Design

The NEX friends service (authentication and secure servers, plus the gRPC that game servers
query for presence) for Wii U and 3DS, forked from Pretendo's `friends` (AGPL-3.0) and rewired
onto the OpenPak stack:

- identity comes from [`nn-account`](../../nn-account), the Wii U/3DS adapter, over its
  `account.v2` gRPC and its Resolution service (PID <-> core account);
- the friend graph (friendships, requests, blocks) is read and written in the OpenPak
  [account core](../../../../../platforms/core/infra/account) through `coregraph`, so web and console agree;
- presence, notifications and Mii data stay local to this server's Postgres.
