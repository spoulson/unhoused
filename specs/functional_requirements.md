# Functional Requirements

High level functional requirements for Unhoused.

## Home Page

- Home page starts with a profile selection, which are defined in configuration.
- User clicks one of the available profiles.  Link opens the Profile Page.

## Table UX: Search, Filter, Sort, and Pagination

The Profile Page's Jobs table and the Job Status Page's Allocations table share one interaction pattern for
searching, filtering, sorting, and paginating. Each page's section below lists which of these apply to it,
its specific fields/columns, and its defaults — this section describes the shared mechanics once.

- **Search**: a single text box above the table filters rows to those where one or more designated fields
  contain the search text, case-insensitive, substring match (not a full-word or prefix match).
- **Filter dropdowns** (Job Status Page only): each dropdown narrows the table to rows with an exact value in
  one field. Multiple active filters combine with AND. A dropdown's options are either derived from the
  table's actual data (e.g. only task groups/versions/nodes currently present) or a fixed set of valid
  values (e.g. status enums), and every dropdown's option list is itself sorted — see the Job Status Page
  section for which fields sort ascending vs. descending.
- **Sortable columns**: clicking a column name sorts the table by that column. An up/down arrow next to the
  column name indicates the active sort direction. Clicking the same column again cycles it through
  ascending → descending → unsorted (arrow removed), repeating; clicking a different column always starts
  that column at ascending. Each table has its own default sort column/direction (see per-page defaults
  below), applied whenever nothing else has been explicitly chosen — so on first load the default column
  already shows its arrow. That default can itself be cycled: clicking its column steps it from its default
  direction onward through the same ascending → descending → unsorted cycle as any other column.
- **Pagination**: below the table, Previous/Next buttons (each disabled at the first/last page respectively)
  step through pages; a "Showing X–Y of Z <items>" summary sits above the table; a "Per page" dropdown
  (25/50/100/200, default 50) controls page size.
- **Interactions reset paging**: changing the search text, any filter, the sort column/direction, or the
  page size returns the view to page 1, so the user isn't left looking at a now out-of-range page.
- **Clear filters** (Job Status Page only): a button appears next to the filter dropdowns whenever the
  search text, any filter, or the sort deviates from its default, and resets all of them back to default in
  one click.
- **URL persistence**: search text, filter selections, sort state, and pagination state are all reflected in
  the URL's query string, so reloading the page or opening a bookmarked/shared link restores the exact same
  view. Default values (no search, no filter, default sort, page 1, default page size) are represented by
  the *absence* of their query parameter rather than an explicit value, so the URL stays clean when nothing
  has been changed from the default.

## Profile Page

- Profile page lists the available Nomad jobs.
- User clicks a job to view the job status page.
- Page title is "Profile: `<profile name>`", shown both as the browser tab title and as the page's H1
  heading.
- Search matches job name.
- Sortable columns: Job, Submitted. Default sort is Submitted, descending.
- URL query params: `q` (search), `sort`/`dir` (sort column/direction), `page`/`pageSize` (pagination).

## Job Status Page

- Page title is "Job: `<job id>`", shown both as the browser tab title and as the page's H1 heading
  (alongside the running/stopped/etc. indicator described next).
- Show indicator whether job status is currently running, stopped, etc.
- List the counts of allocations by version, then by status.
  - Status refers to running, stopped, etc.
  - Also shows last modified time of newest allocation in the group.
  - The version list uses the same pagination control described above (Previous/Next, adjustable page
    size, "Showing X–Y of Z versions" summary), but with its own page size options (5/10/25/50) and
    defaults to 5 per page.
  - Clicking a status count (e.g. "✓ Running 2") sets the Allocations table's version and status filters to
    that version and status, replacing whatever filters were previously set.
- Below the status groups is the full list of allocations in tabular layout.  This includes
fields:
  - Allocation ID
  - Node name
  - Node IP
  - Current status and desired status
  - Task group name
  - Version number
  - Last Modified
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
- Search matches Allocation ID, Node name, or Node IP.
- Filter dropdowns: task group, version, and node (options drawn from the job's actual allocations), plus
  status and desired (options are Nomad's fixed enums for those fields). Version's options are sorted
  numerically descending (newest first); every other dropdown's options are sorted ascending.
- Sortable columns: Allocation, Node, Status, Desired, Task Group, Version, Last Modified (Ports is not
  sortable). Default sort is Last Modified, ascending (most recently modified allocations first).
- URL query params: `q` (search), `taskGroup`/`version`/`node`/`status`/`desired` (filters), `sort`/`dir`
  (sort column/direction), `page`/`pageSize` (allocation table pagination), `versionPage`/`versionPageSize`
  (version list pagination).
- Page updates periodically based on configuration.
  - Default every 5 seconds.
