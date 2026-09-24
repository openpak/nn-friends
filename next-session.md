# Next session — nn-friends
Updated 2026-09-24.

Pretendo's NEX friends fork rewired onto the OpenPak core: the friend graph
(requests, accepts, removes, blocks) is read and written in the account core
through `coregraph`; presence, notifications and Mii data stay in this
server's Postgres. Deployed 2026-09-10 (NEX UDP 22360/22361, gRPC 20061),
not console-verified — no Wii U or 3DS has signed in yet.

Current status 2026-09-24: latest tag v0.5.1 (1a517eb), `dev` clean. Since
the last doc: WU-1 subscription committed (a49b88b) and shipped in v0.3.0
with cross-network presence; v0.4.0 names Cemu/Azahar from the NEX token;
v0.5.0 bans (AccountDisabled, sessions dropped on `account_banned`); v0.5.1
auth-server startup race fix; CI builds on `v*.*.*` tags only.

## Where things stand

- v0.3.0 (2026-09-23): cross-network presence — Wii U/3DS sessions published
  to the core (`crosspresence`), friends on other platforms rendered online
  (`crossnotify`, titles from `WEBSITE_INTERNAL_URL`/`_KEY`).
- v0.4.0: presence client (`cemu`/`azahar`/console) from nn-account's
  `ResolveNexTokenClient`. v0.5.0: banned login -> `AccountDisabled`, live
  sessions dropped on ban/delete events. v0.5.1: auth-server init race.
- Committed (a49b88b), released in v0.3.0 — the WU-1 code half
  (../prds/platform-wiiu-prd.md §5, dated 2026-09-12): `coreevents/poller.go`
  rides the core's `SubscribeAccountEvents` with the poll as fallback and the
  same version cursor either way; `coregraph.SubscribeEvents`; the vendored
  `internal/accountpb/events.proto` (+ regenerated pb) adds
  `SubscribeAccountEvents`, `EmitEvent`, and the closed event set now includes
  `friend_requested|friend_accepted|friend_removed|message`.
- Notification paths: Wii U pushes request/accepted/removed over NEX; 3DS
  pushes `FriendshipCompleted` only and learns the rest at list sync;
  `message` events are dropped here by design — the chat surfaces deliver
  them (universal-social-prd §4b.3), logged honestly.
- A fresh cursor starts at the stream head so history is never replayed;
  `friend_removed` also deletes local `wiiu.friend_requests` rows so stale
  metadata cannot resurrect a core-ended friendship.
- Legacy cores (no subscription RPC) stay on the 2 s poll for 30 s stretches.

## Next steps

1. ~~Commit the subscription work and tag v0.3.0~~ — done (a49b88b, v0.3.0);
   deploys are tag pushes (ghcr + podman-auto-update).
2. Exercise the failover against a live core: stream break -> drain ->
   resubscribe from the cursor; restart continuity.
3. Console verification: first sign-in (nn-account's frontier), then the
   C3–C7 parity pass on hardware (WU-0 / DS3-2).
4. 3DS Mii key (DS3-3) still missing — the server starts without it
   (ff94f12); consoles see a default Mii until it exists.

## Pointers

- README.md (env table), example.env
- coreevents/poller.go, coregraph/coregraph.go, internal/accountpb/events.proto
- ../prds/universal-social-prd.md (US-2), ../prds/platform-wiiu-prd.md (WU-1),
  ../prds/platform-3ds-prd.md
- ../nn-account/docs/client-testing.md

## Scratch (research and throwaway work)

Decompiles, Ghidra projects, dumps, exefs/romfs extracts, packet captures,
strace and emulator logs, probe harnesses: put them in
`~/REPOS/Openpak/scratch/<topic>`. That folder is a local mount of the media pool,
outside every repository, so nothing in it is committed. Never use `/tmp` (a
shared 15 GB RAM disk) or elsewhere on `/home` for this. Keys and signing
material never go there. Rule: `docs/playbooks/conventions.md` in the workspace.
