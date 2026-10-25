import * as chrono from "chrono-node";

export const ASSIGNMENT_TIME_ZONE = "Asia/Bangkok";
const BANGKOK_OFFSET_MINUTES = 420;

export const parseDateTime = (str: Date | string) => {
  if (str instanceof Date) return str;
  return chrono.parseDate(str, { instant: new Date(), timezone: BANGKOK_OFFSET_MINUTES });
};

export const formatDateTime = (datetime: Date | string) => {
  return new Date(datetime).toLocaleString("en-US", {
    timeZone: ASSIGNMENT_TIME_ZONE,
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "numeric",
    hour12: true,
    timeZoneName: "short",
  });
};

export const getDateTimeLocal = (timestamp?: Date): string => {
  const d = timestamp ? new Date(timestamp) : new Date();
  if (isNaN(d.getTime())) return "";
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone: ASSIGNMENT_TIME_ZONE,
    year: "numeric", month: "2-digit", day: "2-digit",
    hour: "2-digit", minute: "2-digit", hourCycle: "h23",
  }).formatToParts(d);
  const part = (type: Intl.DateTimeFormatPartTypes) => parts.find((p) => p.type === type)?.value;
  return `${part("year")}-${part("month")}-${part("day")}T${part("hour")}:${part("minute")}`;
};

// Keep API precision on untouched edits: JavaScript Dates hold milliseconds,
// while PostgreSQL timestamps can contain additional fractional digits.
export const serializeDateTime = (date: Date, original?: string): string => {
  if (original && date.getTime() === new Date(original).getTime()) return original;
  return date.toISOString();
};

// Use UTC arithmetic for Bangkok's fixed offset. Local Date/TZDate setters can
// normalize a selected hour through the browser's unrelated DST transition.
export const setBangkokTime = (date: Date, hour: number, minute: number,
  second?: number, millisecond?: number): Date => {
  const local = new Date(date.getTime() + BANGKOK_OFFSET_MINUTES * 60000);
  local.setUTCHours(hour, minute, second ?? local.getUTCSeconds(), millisecond ?? local.getUTCMilliseconds());
  return new Date(local.getTime() - BANGKOK_OFFSET_MINUTES * 60000);
};
