# Changelog — nn-friends

> **Current Switch client (2026-10-13): Harbor**, at
> `~/REPOS/Openpak/emulators/harbor`. Harbor is an independent
> OpenPak project based on Eden. Citron, Ryujinx (desktop and Android), and
> the former OpenPak Eden checkouts have been removed. References to those
> forks below record historical evidence only: do not run their commands,
> reinstall them, or select them for current work. Use Harbor's README and
> current handoff for builds and testing; old results do not establish Harbor parity.

Generated from git history on 2026-09-15. `git log` stays the source
of truth; this file is the readable summary.

Note: the early history below is the upstream project (Pretendo friends);
OpenPak work starts at the port/fork commit.

## v0.9.2 — 2026-11-09

- World presence: the host PID is the gathering's owner (the game looks the gathering up by group id, which worked, then
  checks the host against it); the friend notification is still addressed by the friend's own PID (`SendPresenceTo` takes it).

## v0.9.1 — 2026-11-09

- The joinable world presence keeps the friend's own PID as the host PID (as a real host's does): the friend notification is
  addressed by it, so the 0.9.0 form (the world's owner PID) never reached the console. Each cross-platform presence
  notification is logged (ids and the line).

## v0.9.0 — 2026-11-09

- A friend who is in the shared Minecraft world (core session namespace `minecraft`, published by common/minecraft) shows to
  Wii U friends as "Playing Minecraft in the OpenPak world". A viewer who runs Minecraft: Wii U Edition gets a *joinable* presence
  for them, built from what a real game publishes when hosting (joinFlagMask 1, matchmakeType 2, joinGameId 0x101D9D00,
  joinGameMode 1, groupId = the world's gathering id, the viewer's own game key, the 20-byte application data), so the game's
  Join tab lists them and joins the world. Needs `MINECRAFT_WORLD_GATHERING` (the nex world gathering id); unset = plain line.
- Viewers are sent their world friends again when they start or leave Minecraft (joinable or not).

## v0.8.2 — 2026-11-09

- Log the join fields (title, join flags, matchmake type, join game id, mode, host PID, group id, application data) of any
  presence a console publishes while hosting a game: ids and flags only (card #162, to learn what a real game publishes).

## v0.8.1 — 2026-11-05

- Fix the core event cursor. A fresh cursor was set by asking the core for events after 2^63 and keeping the
  echoed value, so the stored cursor sat past every real event and no friend or profile event was delivered
  (the 2 s poll found nothing). The head is now found by paging the stream, and a cursor already past the head
  is reset to it at start. `profile_updated` refresh and live friend notifications depend on this.
- Orphan purge: on prod all nine cached PIDs have PNIDs and active links, so nothing was deleted (counts logged).

## v0.8.0 — 2026-11-05

- Data duplication audit phase 2 (card openpak/prds#131). The account core owns name and Mii; nn-friends keeps
  only protocol data (presence, comment, privacy flags, friend-request metadata).
  - `wiiu.mii`, `wiiu.principal_basic_info` and the `3ds.user_data` Mii columns are now a read-through cache of
    nn-account's GetUserData (which serves the core profile and Mii). `EnsureProfiles` upserts instead of
    `ON CONFLICT DO NOTHING`, so a rename or a new Mii reaches friends lists; rows are trusted for 5 minutes
    (`profilecache`). The Wii U birthday is no longer cached (no call reads it).
  - The core event `profile_updated` invalidates the account's PIDs and refreshes their cache rows.
  - A Mii uploaded by a 3DS (`UpdateMii`) is stored in the core (Mii service `PutMii`), then the local cache;
    it is no longer written first in nn-friends.
  - Startup migration deletes cache and settings rows whose PID has no PNID in nn-account (Pretendo-era orphans
    nobody could erase). Logs counts only; rows of linked or existing PNIDs are never touched.
  - Erasure (`erase/`) still covers every remaining table; account-api v0.3.0.

## v0.7.0 — 2026-11-04

- Account erasure (card openpak/prds#120). `DELETE /internal/accounts/{account_id}/data` with
  `Authorization: Bearer <ACCOUNT_ERASE_KEY>` on its own listener (`ERASE_ADDR`, default `:9099`, container
  network only; not served without the key). Erases, in one transaction, every row of every PID the account
  has (core links plus the PID nn-account resolves, shadow included): Wii U user data, network account info,
  principal info, Mii, 3DS user data and profile (comment, Mii, favourite game, region); friendships, blocks
  and friend requests on **both** sides (the other person's list loses the row too). Live sessions of those
  PIDs are ended first. Idempotent. Logs one line `account erased account=<uuid> rows=<n> service=nn-friends`.
- The core's `account_deleted` event runs the same erase as a backstop (by then the core links are gone, so it
  only finds what nn-account can still resolve).
- Authored content: nn-friends holds no posts or messages, only profile fields; all are deleted, nothing kept.
- account-api v0.1.1 (ListAccountLinks); the keyed sessions wrapper forwards GetPlayDays.

## v0.6.2 — 2026-11-03

- Logs one line per call to stderr: `grpc <method> -> <code> <ms>ms` for every gRPC unary call, and
  `nex <auth|secure> call <protocol>/<method> -> recv` for every NEX call received. Never request
  or response bodies, metadata (the API key) or call parameters.

## v0.6.1 — 2026-09-26

- Account gRPC calls carry `X-OpenPak-Caller: friends`, so nn-account does not count a console
  connecting to friends as a game reaching its server (playtime).

## v0.6.0 — 2026-09-26

- The core session carries the emulator's OS: at RegisterEx the OS nn-account recorded with
  the NEX token (`ResolveNexTokenClient.os`, nn-account v0.9.0) is kept on the connected user
  and sent as `RegisterSessionRequest.os`, so the website can mark titles tested per emulator
  and OS from playtime. "" for a console, an old token or an older nn-account. An OS change
  re-registers the session, like a client change. Vendored account and resolution protos
  refreshed (core v0.11.0).

## v0.5.2 — 2026-09-24

- deps: grpc 1.83.2 (Dependabot)

## v0.5.1 — 2026-09-23

- Auth server takes the secure account from init, not from the endpoint the secure goroutine
  may not have created yet (startup race). CI builds on `v*.*.*` tags only.

## v0.5.0 — 2026-09-23

- Bans (website/docs/ban-lookup.md): a login whose account nn-account refuses as banned
  ("Account is banned or deleted" from `GetNEXPassword`) is answered
  `RendezVous::AccountDisabled` (0x00030067), the NEX result for a disabled account, instead
  of InvalidPID. On the core's `account_banned`, `account_delete_started` or `account_deleted`
  event every live friends connection of that account (Wii U and 3DS alike) is cleaned up at
  once, which takes the player offline for their friends. No new environment variables.

## v0.4.0 — 2026-09-23

- Cemu and Azahar are told apart from real consoles: at RegisterEx the NEX token the person
  logged in with is looked up in nn-account (`Resolution.ResolveNexTokenClient`, nn-account
  v0.6.0), which recorded the client when it issued the token — the emulator's
  `X-OpenPak-Client` (`cemu`, `azahar`) or the console (`wiiu`, `3ds`). The published core
  session carries it, so friends elsewhere read "Playing … on Cemu". Unknown (old token,
  nn-account unreachable) stays `""`, shown as the platform.

## v0.3.0 — 2026-09-23

Cross-network presence:

- Wii U and 3DS presence is published into the account core: one live session per connected
  person (namespace `wiiu`/`3ds`, title from the presence game key when it is a game,
  endpoint `nex:<pid>`, 45 s lease renewed every 15 s, re-registered on a title change,
  expired on disconnect) — `crosspresence`. The client is `""`: a Cemu or Azahar identity is
  the same PNID and NEX login as the console's, so nothing here can tell them apart.
- Friends live on another platform per the core (a Switch, Ryujinx/Eden/Citron, the other of
  Wii U and 3DS) render online: Wii U friend list (UpdateAndGetAllInformation) and 3DS
  GetFriendPresence get a presence with no game key and "Playing <title> on <label>" /
  "Online on <label>" in the message; connected consoles are notified as that changes
  (`crossnotify`). Title names come from the website (`WEBSITE_INTERNAL_URL`,
  `WEBSITE_INTERNAL_KEY`). Whether either console shows the message for a friend with no game
  is unverified on hardware.
- WU-1: `coreevents` rides the core's `SubscribeAccountEvents` stream with the 2 s poll as
  fallback, same version cursor (committed in a49b88b).
- secure: one offline mark, not two [d3bc373]


## v0.2.0 — 2026-09-10

- One friend graph on every console: Wii U lists read the core; unknown PIDs get name and Mii
  from the adapter's GetUserData (Switch/phone users appear on Wii U/3DS); requests from
  elsewhere materialize as local metadata rows so accept-by-id works.
- A poller follows the core's friend_requested/accepted/removed events and notifies online
  consoles (NEX notifications on Wii U, FriendshipCompleted on 3DS).

## v0.1.0 — 2026-09-10

- Start without the 3DS Mii key [ff94f12]
- Go 1.26, the house version [5eeac26]
- example.env, an OpenPak README and an OpenPak build string [7690443]
- Vendor the account and nn-account contracts so a clean checkout builds [5c5eb8d]
- Release workflow like every other OpenPak service: tag → ghcr, amd64 and arm64 [fd14173]
- fix(M3): 3DS friend-list read projects core state (complete + one-sided outgoing) [bf37ccf]
- feat(M3): route canonical graph operations to the OpenPak account core [112b3f2]
- Merge pull request #51 from BDMCGaming/dev [eb85ca3]
- fix: update libraries to resolve x01-0807 [8cdd938]
- Merge pull request #47 from RusticMaple/dev [3be7e40]
- fix: adjust variable names back to how they were [7a6c1e4]
- chore: make error style more consistent [3d6fff0]
- fix: make friend requests accept properly [bfd8e9f]
- fix(3ds): Use RelationshipTypeIncomplete for new relationships [4ea3bbc]
- Merge pull request #46 from PretendoNetwork/feat/grpc-token [bcd8996]
- feat: update to use gRPC tokens [6ab58a2]
- Merge pull request #45 from PretendoNetwork/feat/pidhmac [4e6021b]
- feat(account-management): use new algorithm for pidHMAC [53d6d25]
- Merge pull request #43 from DaniElectra/grpc-get-user-data-3ds [c5221a5]
- fix(database/3ds): fix GetMii SQL query [ff10b7c]
- Merge pull request #42 from PretendoNetwork/feat/grpc-friends-data [2f04f8d]
- chore: go mod tidy [235b31a]
- fix(grpc): move logs to only occur when internal error occurs [084cd5e]
- feat(config): add checks for Mii decryption key config [eacd16b]
- fix(grpc): add missing log entries for get user friends data [38aec18]
- chore(grpc): split out 3DS & Wii U grpc methods [f2f00ec]
- fix(grpc): incorrect mii data length check + variable casing [a797ea5]
- feat(grpc): add decrypted mii data to 3DS response [ab0f152]
- fix(grpc): throw grpc errors [067363f]
- chore(grpc): remove old row scan [bfeadb6]
- fix(grpc): updated timestamps to be accurate, more cleanup [2ad590a]
- fix(database): switch to PID type for input and clean up redundant struct declarations [306399a]
- fix(grpc): fix mii data field assignment on getUserData3DS [a95ecba]
- feat(grpc): implement getUserData rpcs [dc02700]
- feat(grpc): add presence for 3DS [f34d3ea]
- chore: bump grpc version [87142fc]
- fix(grpc): switch to timestamps per updated proto [9a6edb4]
- fix(grpc): split out rpc requests [98c707d]
- feat(grpc): implement fetch user data for WIi U & 3DS [eb6bb06]
- Merge pull request #40 from Crisoxyz/fix-friend-requests [6c6d001]

- … 240 earlier commits omitted (see `git log`)
