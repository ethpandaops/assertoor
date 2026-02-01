/**
 * Format a duration in seconds to a human-readable string
 */
export function formatDuration(seconds: number): string {
  if (seconds < 0) return '-';

  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = Math.floor(seconds % 60);

  if (hours > 0) {
    return `${hours}h ${minutes}m ${secs}s`;
  }
  if (minutes > 0) {
    return `${minutes}m ${secs}s`;
  }
  return `${secs}s`;
}

/**
 * Parse a timestamp that can be either a Unix timestamp (number/seconds or milliseconds)
 * or an ISO date string
 */
function parseTimestamp(timestamp: number | string): Date {
  if (typeof timestamp === 'string') {
    return new Date(timestamp);
  }
  // Handle both seconds and milliseconds
  // If timestamp > 1e12, it's likely in milliseconds
  if (timestamp > 1e12) {
    return new Date(timestamp);
  }
  return new Date(timestamp * 1000);
}

/**
 * Format a Unix timestamp to a relative time string
 */
export function formatRelativeTime(timestamp: number | string): string {
  const date = parseTimestamp(timestamp);
  const now = Date.now();
  const diff = Math.floor((now - date.getTime()) / 1000);

  if (diff < 0) {
    return 'in the future';
  }

  if (diff < 60) {
    return 'just now';
  }

  if (diff < 3600) {
    const minutes = Math.floor(diff / 60);
    return `${minutes} minute${minutes !== 1 ? 's' : ''} ago`;
  }

  if (diff < 86400) {
    const hours = Math.floor(diff / 3600);
    return `${hours} hour${hours !== 1 ? 's' : ''} ago`;
  }

  if (diff < 604800) {
    const days = Math.floor(diff / 86400);
    return `${days} day${days !== 1 ? 's' : ''} ago`;
  }

  // For older dates, return the formatted date
  return date.toLocaleDateString();
}

/**
 * Format a timestamp to a full datetime string
 */
export function formatDateTime(timestamp: number | string): string {
  if (!timestamp) return '-';
  const date = parseTimestamp(timestamp);
  return date.toLocaleString();
}

/**
 * Format a timestamp to a time string
 */
export function formatTime(timestamp: number | string): string {
  if (!timestamp) return '-';
  const date = parseTimestamp(timestamp);
  return date.toLocaleTimeString();
}

/**
 * Format milliseconds duration to a human-readable string
 */
export function formatDurationMs(ms: number): string {
  return formatDuration(ms / 1000);
}
