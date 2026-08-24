# Functional Requirements

High level functional requirements for Unhoused.

## Home Page

- Home page starts with a profile selection, which are defined in configuration.
- User clicks one of the available profiles.  Link opens the Profile Page.

## Profile Page

- Profile page lists the available Nomad jobs sorted newest to oldest.
- User clicks a job to view the job status page.
- Text search box filters the job list to names containing the search text (case-insensitive substring
  match).
- Job and Submitted columns are sortable: an up/down arrow next to the column name indicates the active
  sort. Clicking a column name toggles it through ascending → descending → unsorted (arrow removed),
  repeating; clicking a different column starts it at ascending. Default is Submitted, descending.
- Search text and sort state are reflected in the URL query string (`q`, `sort`, `dir`), so reloading the
  page or opening a bookmarked/shared link restores the same search and sort. The default sort (Submitted,
  descending) is the absence of `sort`/`dir` rather than an explicit value, keeping the URL clean when
  nothing's been changed.

# Job Status Page

- Show indicator whether job status is currently running, stopped, etc.
- List the counts of allocations by version, then by status.
  - Status refers to running, stopped, etc.
  - Also shows uptime of newest allocation in the group.
- Below the status groups is the full list of allocations in tabular layout.  This includes
fields:
  - Allocation ID
  - Node name
  - Node IP
  - Current status and desired status
  - Task group name
  - Version number
  - Uptime
  - For each network port defined, link to its IP:port as URL like: `http://<ip>:<port>`.
    - And if the port is labeled `http`, also include a link to the node like: `http://<host>:<port>`.
      - `host` is a hostname value derived from the Nomad node name with special rules based on the Nomad environment and region:
        - For environment "staging", `host` follows format: `http://<node_name>.node.<region>.staging.mailforce:<port>`.
        - For environment "production", `host` follows format: `http://<node_name>.c.mailforce-production-<short_region>.internal:<port>`.
      - `short_region` is derived from `region` as:
        - `us-east4` -> `use4`
        - `us-west1` -> `usw1`
        - `europe-west1` -> `euw1`
        - `ause1` -> `ause1`
- Allow selectable filters on the table data:
  - On fields "task group", "version", and "node" from existing values.
  - On fields "status" and "desired" from possible valid values.
- Page updates periodically based on configuration.
  - Default every 5 seconds.

