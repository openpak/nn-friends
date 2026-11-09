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

## Friends in the Minecraft world

A friend whose core session is `minecraft`/`world` (published by common/minecraft while they are in the shared world) is shown to a Wii U
viewer as "Playing Minecraft in the OpenPak world". When the viewer itself runs Minecraft: Wii U Edition (title ids 00050000101D9D00,
101D7500, 101DBE00) the presence is joinable, with the values a real game publishes while hosting (see common/minecraft
`docs/wiiu-join.md`): `joinFlagMask` 1, `matchmakeType` 2, `joinGameId` 0x101D9D00, `joinGameMode` 1, `groupId` the world's gathering id
(`MINECRAFT_WORLD_GATHERING`), host PID the gathering owner (`MINECRAFT_WORLD_PID`, default 2000000001), the viewer's own game key. The
notification is addressed by the friend's own PID. Viewers are sent their world friends again when they start or leave Minecraft.
