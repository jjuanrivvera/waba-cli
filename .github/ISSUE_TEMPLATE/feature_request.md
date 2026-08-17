---
name: Feature request
about: Suggest a command or capability
labels: enhancement
---

**What you are trying to do**

**Why the current commands do not cover it**

If it is an API operation without a dedicated command, note that every documented operation
is already reachable:

```sh
atlassian op search <keyword>
atlassian op call <operationId> --param k=v
```

A dedicated command is worth adding when the raw operation is awkward — a required version
number, an id that has to be looked up first, a body that needs building by hand.
