---
---
# Spike: Notification Service Options

Copied this ticket template from another repo and never got around to filling in the
metadata block up top — it's currently empty. Leaving the body as-is for now.

We need to decide between rolling our own notification fan-out vs. bolting on a
third-party provider. At minimum, compare cost at our current volume, delivery
latency, and how much custom retry logic we'd have to write ourselves either way.

Loop in the infra team before committing to a direction.
