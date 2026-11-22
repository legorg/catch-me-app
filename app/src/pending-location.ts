export type LocationReport = { latitude: number; longitude: number; accuracyMeters: number };

// The latest location not reported yet. The next state poll reports it and gets
// the state in the same response, so a player sends one request per poll.
// Shared by the foreground watcher and the background task, which run in the
// same JavaScript runtime.
let pendingLocation: LocationReport | null = null;

/** Leaves the location for the next poll to report, replacing an older unreported one. */
export function offerLocation(location: LocationReport) {
  pendingLocation = location;
}

/** The location to report with this poll, if any; it is reported only once. */
export function takeLocation(): LocationReport | null {
  const location = pendingLocation;
  pendingLocation = null;
  return location;
}

/** Forgets an unreported location, once sharing stops. */
export function dropLocation() {
  pendingLocation = null;
}
