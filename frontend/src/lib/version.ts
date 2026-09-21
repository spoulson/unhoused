/**
 * Formats a job version for display as "<version> tagged-at=<tagged-time-rfc3339>
 * image=<docker-image>:<tag>", e.g. "3 tagged-at=2026-08-12T15:30:00Z image=myrepo/app:1.2.3".
 * The tagged-at and image segments are each omitted (along with their separator) when empty — most
 * versions have no Nomad version tag.
 */
export function formatVersionLabel(version: number, taggedTime: string, dockerImage: string): string {
  let label = String(version)
  if (taggedTime) {
    label += ` tagged-at=${taggedTime}`
  }
  if (dockerImage) {
    label += ` image=${dockerImage}`
  }
  return label
}
