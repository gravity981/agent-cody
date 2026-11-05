# Cody – Requirements

All requirements of the project, one section per topic. Conventions (IDs, priorities, withdrawn requirements) are in [README.md](README.md). Topics 1 and 2 come from the project blueprint and apply to every project; adapt them where the project needs it.

## 1. Requirement traceability (TRC)

The requirement report is built in the project's own technology during stage 1. It links every requirement to its evidence (see [verification.md](verification.md)).

| ID | Requirement | Prio |
|---|---|---|
| TRC-01 | The requirement report reads every requirement row of `spec/requirements.md`, including withdrawn ones; duplicate IDs and unknown priorities are errors. | Must |
| TRC-02 | It reads the "not yet implemented" list in `spec/verification.md`, including ranges (`ABC-01 – ABC-05`); unknown or withdrawn IDs on the list are errors. | Must |
| TRC-03 | It reads the results of every automated verification the project uses, with the requirement IDs each result is tagged with (tag convention in `spec/verification.md`); malformed, unknown or withdrawn tags are errors. | Must |
| TRC-04 | It reads `spec/verification-log.md`; for each requirement only its latest manual record counts. | Must |
| TRC-05 | Errors are: an implemented requirement (not on the "not yet implemented" list) without passing evidence, and any failing evidence of a requirement. | Must |
| TRC-06 | Warnings are: a requirement with passing evidence that is still on the "not yet implemented" list, and a requirement planned in no stage or in several stages of `spec/implementation-plan.md`. | Should |
| TRC-07 | It writes `reports/requirements.md` (totals, totals per stage, errors, warnings, and requirement → evidence → result) and exits non-zero on any error. | Must |

## 2. CI, versions and releases (REL)

Details in [delivery.md](delivery.md). The GitHub workflows come with the blueprint; the project provides the entry points in `ci/`.

| ID | Requirement | Prio |
|---|---|---|
| REL-01 | On every pull request and every push to `main`, CI runs `ci/check` in the CI environment (delivery.md 2); it includes all automated verification and the requirement report, and fails if any of them fails. | Must |
| REL-02 | Every build carries a version according to the version scheme (delivery.md 1) and its commit; how a built artifact reveals both is defined in architecture.md. | Must |
| REL-03 | A release candidate is built from `main` by a manually started workflow, only for a commit whose CI passed, after the full check passed again; it is tagged `v<version>-rc.<M>` and published as a GitHub pre-release with the release files. | Must |
| REL-04 | Only the latest candidate of a version can be promoted; promotion does not rebuild, tags `v<version>` on the candidate's commit, and publishes a GitHub release with exactly the candidate's release files. | Must |
| REL-05 | Tags `v*` can't be moved or deleted (repository tag ruleset); only the release workflows create them, and they never reuse an existing tag. | Must |
| REL-06 | Releases are delivered as defined in delivery.md 4 (deployment, distribution, or download from the GitHub release only). | Must |

<!-- TODO(blueprint): project topics from here on, numbered 3, 4, …

## 3. <Topic> (ABC)

Short context for the topic. Design details go to architecture.md.

| ID | Requirement | Prio |
|---|---|---|
| ABC-01 | … | Must |
-->
