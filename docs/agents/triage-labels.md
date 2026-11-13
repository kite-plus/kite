# Triage Labels

The skills speak in terms of five canonical triage roles. This file maps those roles to the actual label strings used in this repo's issue tracker.

| Label in mattpocock/skills | Label in our tracker | Meaning                                  |
| -------------------------- | -------------------- | ---------------------------------------- |
| `needs-triage`             | `needs-triage`       | Maintainer needs to evaluate this issue  |
| `needs-info`               | `needs-info`         | Waiting on reporter for more information |
| `ready-for-agent`          | `ready-for-agent`    | Fully specified, ready for an AFK agent  |
| `ready-for-human`          | `ready-for-human`    | Requires human implementation            |
| `wontfix`                  | `wontfix`            | Will not be actioned                     |

When a skill mentions a role (e.g. "apply the AFK-ready triage label"), use the corresponding label string from this table.

Edit the right-hand column to match whatever vocabulary you actually use.

## Wayfinding labels

`/wayfinder` uses a second, separate vocabulary, created alongside the triage labels:

| Label                | Used on                                    |
| -------------------- | ------------------------------------------ |
| `wayfinder:map`      | the map issue itself                       |
| `wayfinder:research` | a child ticket that answers a question     |
| `wayfinder:prototype`| a child ticket that builds a throwaway     |
| `wayfinder:grilling` | a child ticket that stress-tests a plan    |
| `wayfinder:task`     | a child ticket that is plain implementation |

These are the only wayfinder labels; a child ticket carries exactly one `wayfinder:<type>`.
