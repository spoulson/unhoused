## Configuration

- A YAML configuration file is provided via CLI argument `-c <file>` for use by the backend.
- Configuration contains service settings:
  - HTTP public URL used for any generated links within the app.  Defaults to "http://localhost".
  - REST API listen port, defaults to 3001.
  - Refresh interval in seconds for job status page.
- Configuration contains 1 or more profiles describing a Nomad environment.  These define:
  - Profile name
  - Environment: staging or production
  - Region: us-west1, us-east4, europe-west1, or ause1
  - Nomad service URL (usually references port 4646)
  - Nomad API token (in plaintext)
  - Node hostname template: an optional string used to derive each port's node address, containing
    the literal placeholder `{node}`, substituted with the Nomad node name. Defaults to `{node}`
    (i.e. the bare node name) when unset.

