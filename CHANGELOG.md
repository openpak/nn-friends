# Changelog — nn-friends

Generated from git history on 2026-09-15. `git log` stays the source
of truth; this file is the readable summary.

Note: the early history below is the upstream project (Pretendo friends);
OpenPak work starts at the port/fork commit.

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
- secure: one offline mark, not two [d3bc373]


## v0.1.0 — 2026-09-10



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
