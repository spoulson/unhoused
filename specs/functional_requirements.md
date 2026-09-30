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
- Each job's Status column shows the same running/pending/stopped/dead indicator as the Job Status Page
  header. A separate Deployment column shows an indicator for the job's most recent deployment: still in
  progress ("Deploying for `<duration>`"), finished ("Deployed `<duration>` ago"), or failed/was cancelled
  ("Failed `<duration>` ago"). The duration is omitted (bare "Deploying"/"Deployed"/"Failed") when it isn't
  known. When it is known, hovering the indicator shows a tooltip with the same text plus the absolute
  local timestamp it's measured from in parentheses, e.g. "Deploying for 2m 10s (9/30/2026 10:54:04am)".
  No indicator at all for a job with no deployment (batch/system jobs, or service jobs without an `update`
  block).
- Sortable columns: Job, Status, Deployment, Submitted. Default sort is Submitted, descending.
- URL query params: `q` (search), `sort`/`dir` (sort column/direction), `page`/`pageSize` (pagination).
- Page updates periodically based on configuration (same refresh interval as the Job Status Page), so a
  job's deployment indicator (including its elapsed-time text) stays current without a manual reload.

## Job Status Page

- Page title is "Job: `<job id>`", shown both as the browser tab title and as the page's H1 heading
  (alongside the running/stopped/etc. indicator described next).
- Show indicator whether job status is currently running, stopped, etc., plus the same deployment
  indicator as the Profile Page's Deployment column shown next to it.
- List the counts of allocations by version, then by status.
  - Status refers to running, stopped, etc.
  - Also shows last modified time of newest allocation in the group.
  - The version list uses the same pagination control described above (Previous/Next, adjustable page
    size, "Showing X–Y of Z versions" summary), but with its own page size options (5/10/25/50) and
    defaults to 5 per page.
  - Each version is labeled "Version" followed by its version number, its Nomad version-tag time (if the
    version has been tagged via `nomad job tag apply`), and its Docker image tag (if that version has a
    docker-driver task), in the format `Version <version-number> tagged-at=<tagged-time-rfc3339>
    image=<docker-image>:<tag>`. Either or both of the ` tagged-at=<tagged-time-rfc3339>` and
    ` image=<docker-image>:<tag>` segments are omitted when that data isn't available for the version,
    falling back to just `Version <version-number>` when neither is. The version number, the `tagged-at=`/
    `image=` keys, the tagged time, and the image are each syntax-highlighted in a distinct color to keep
    the value readable at a glance (see below wherever this format is reused, except the Version filter
    dropdown, whose native options can only render plain text).
  - Clicking a status count (e.g. "✓ Running 2") sets the Allocations table's version and status filters to
    that version and status, replacing whatever filters were previously set.
- Below the status groups is the full list of allocations in tabular layout.  This includes
fields:
  - Allocation ID
  - Node name
  - Node IP
  - Current status and desired status
  - Task group name
  - Version number, followed by its tagged time and Docker image tag the same way as the version list above
    (see above), but without the leading "Version" word (redundant with the column header):
    `<version-number> tagged-at=<tagged-time-rfc3339> image=<docker-image>:<tag>`
  - Last Modified
  - For each network port defined, list its address as `<ip>:<port>`.
    - Also list the node's address as `<host>:<port>`.
      - `host` is derived from the profile's configured node hostname template (see
        [specs/configuration.md](configuration.md)), substituting the literal placeholder `{node}`
        with the Nomad node name.
- Search matches Allocation ID, Node name, or Node IP.
- Filter dropdowns: task group, version, and node (options drawn from the job's actual allocations), plus
  status and desired (options are Nomad's fixed enums for those fields). Version's options are sorted
  numerically descending (newest first) and labeled the same way as the Allocations table's Version column
  above (no leading "Version" word — redundant with the dropdown's own "Version" label); every other
  dropdown's options are sorted ascending.
- Sortable columns: Allocation, Node, Status, Desired, Task Group, Version, Last Modified (Ports is not
  sortable). Default sort is Last Modified, ascending (most recently modified allocations first).
- URL query params: `q` (search), `taskGroup`/`version`/`node`/`status`/`desired` (filters), `sort`/`dir`
  (sort column/direction), `page`/`pageSize` (allocation table pagination), `versionPage`/`versionPageSize`
  (version list pagination).
- Page updates periodically based on configuration.
  - Default every 5 seconds.
- A header toggle lets the user opt in to browser (OS-level) notifications for this job, off by default so
  no permission prompt appears unasked. Once enabled, a notification fires whenever the job's deployment
  status changes — including into "deployed", a rollout finishing being exactly what's worth notifying
  about — regardless of whether the tab is focused. The choice is remembered per-browser and scoped to
  whichever job page is currently open, not a background watch across all jobs. The notification title is
  "Deployment `<job name>`"; the body is "`<emoji>` `<job name>` `<deployment status>`", where the emoji
  indicates the new status: ✅ deployed, 🔄 deploying, ❌ failed.
