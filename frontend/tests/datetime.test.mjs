import assert from "node:assert/strict";
import { test } from "node:test";
import {
  formatDateTime, getDateTimeLocal, parseDateTime, serializeDateTime, setBangkokTime,
} from "../src/lib/datetime.ts";

// Node 24's built-in TypeScript support runs these without another test runner.
for (const timeZone of ["UTC", "Asia/Bangkok", "America/Los_Angeles"]) {
  test(`assignment dates remain in Bangkok when the host uses ${timeZone}`, () => {
    const previous = process.env.TZ;
    process.env.TZ = timeZone;
    try {
      const midnight = "2026-10-25T17:00:00Z";
      assert.equal(getDateTimeLocal(new Date(midnight)), "2026-10-26T00:00");
      assert.match(formatDateTime(midnight), /Oct 26, 2026.*12:00 AM GMT\+7/);
      assert.equal(parseDateTime("October 26, 2026 12:00 AM").toISOString(), "2026-10-25T17:00:00.000Z");
      assert.equal(parseDateTime(formatDateTime(midnight)).toISOString(), "2026-10-25T17:00:00.000Z");

      const original = "2026-10-25T17:00:45.123456Z";
      assert.equal(serializeDateTime(new Date(original), original), original);
      assert.equal(serializeDateTime(new Date("2026-10-25T18:00Z"), original), "2026-10-25T18:00:00.000Z");

      // 02:30 is in the host's DST gap, but is a valid Bangkok time.
      const changed = setBangkokTime(new Date("2026-03-07T17:00:45.123Z"), 2, 30);
      assert.equal(changed.toISOString(), "2026-03-07T19:30:45.123Z");
    } finally {
      if (previous === undefined) delete process.env.TZ;
      else process.env.TZ = previous;
    }
  });
}
