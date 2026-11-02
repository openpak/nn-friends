# Deployment lifecycle

A remote identity adapter is a network dependency. Preserve startup ordering with
`After=openpak-nn-account.service` and request its startup with
`Wants=openpak-nn-account.service`. Do not use `Requires=` for that relationship:
restarting the adapter stops friends, but starting the adapter again does not
start its stopped dependents. `Restart=always` does not undo an explicit systemd
stop propagated by a dependency.

The same rule applies to BOSS and Miiverse dependencies on account/friends, and
the Miiverse UI dependency on its API. Local database/network prerequisites may
retain their existing Requires/After relationships. gRPC clients can reconnect;
keeping the friends process running also preserves its live NEX connections.
