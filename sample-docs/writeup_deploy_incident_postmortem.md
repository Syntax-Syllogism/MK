---
epic: QA
order: -1.75
project: MK
status: UNDER REVIEW
tags:
    - incident
    - postmortem
---
# Write Up the Tuesday Deploy Incident Postmortem

Draft below, using our usual three-section format. Please review and leave comments
before Friday's retro.

---

**What happened:** the 2:15pm deploy shipped a config change that pointed staging's
board at the prod `docs/` directory for about six minutes before someone noticed the
task counts looked wrong and rolled it back.

---

**Impact:** no data was lost — it was a read-only mount — but two engineers spent time
debugging phantom tickets that were actually just prod tasks bleeding into their local
view.

---

**Follow-ups:** add an environment-name assertion to the deploy script so this class of
mistake fails loudly instead of silently pointing at the wrong directory.