# Next session — nn-friends
Updated 2026-09-15.

Pretendo's NEX friends fork rewired onto the OpenPak core: the friend graph
(requests, accepts, removes, blocks) is read and written in the account core
through `coregraph`; presence, notifications and Mii data stay in this
server's Postgres. Deployed 2026-09-10 (NEX UDP 22360/22361, gRPC 20061),
not console-verified — no Wii U or 3DS has signed in yet.

## Where things stand

- Last commit d3bc373 2026-09-10 ("secure: one offline mark, not two"), one
  commit past v0.2.0.
- Uncommitted in the working tree — the WU-1 code half
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
- Also untracked: `CHANGELOG.md`, `docs/`, `prds/` (docs effort, no code).

## Next steps

1. Commit the subscription work and tag v0.3.0; deploys are tag pushes
   (ghcr + podman-auto-update).
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
