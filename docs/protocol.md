# Protocol: nn-friends

## Bans

An account banned in the OpenPak admin cannot sign in (nn-account refuses it; the console is
answered `RendezVous::AccountDisabled`), and its live connections are dropped the moment the
core's `account_banned` event arrives (`coreevents`). See
[`website/docs/ban-lookup.md`](../../../../../platforms/core/infra/website/docs/ban-lookup.md). Nothing to configure.
