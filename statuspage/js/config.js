checkup.config = {
  // How much history to show on the status page. Long durations and
  // frequent checks make for slow loading, so be conservative.
  // This value is in NANOSECONDS to mirror Go's time package.
  "timeframe": 3600000000000,

  // How often, in seconds, to pull new checks and update the page.
  "refresh_interval": 60,

  // Configure read-only access to stored checks. This configuration
  // depends on your storage provider. Any credentials and other values
  // here will be visible to everyone, so use keys with ONLY read access!
  "storage": {
    "url": "/check_files"
  },

  // The text to display along the top bar depending on overall status.
  "status_text": {
    "healthy": "Situation Normal",
    "degraded": "Degraded Service",
    "down": "Service Disruption"
  }
};
