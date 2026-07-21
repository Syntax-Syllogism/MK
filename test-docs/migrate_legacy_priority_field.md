---
epic: QA
order: asap
project: MK
status: TODO
tags:
    - migration
---
# Migrate Legacy `priority` Field Into `order`

Before the dynamic-columns work, some of the oldest tickets in this repo used a
hand-rolled `priority: high|medium|low` convention instead of a real ordering field.
Whoever ported this one over typed the replacement value in by hand instead of picking
an actual number, so it's currently broken.

Go through the archive, find any ticket still carrying the old `priority:` key, and
replace it with a proper numeric `order:` value that reflects where it should actually
sit in its column.
