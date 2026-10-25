import assert from "node:assert/strict";
import { test } from "node:test";
import {
  canEditSubmission, mergeCurrentSubmission, selectCurrentSubmission, summaryGradeValue,
} from "../src/lib/submission-record.ts";

test("default selection uses the current marker even when history has a file", () => {
  const current = { id: "current", is_current: true, attachment_id: null };
  const history = { id: "history", is_current: false, attachment_id: "old-file" };
  assert.equal(selectCurrentSubmission([history, current]), current);
  assert.equal(selectCurrentSubmission([current, history]), current);
  assert.equal(selectCurrentSubmission([]), null);
  assert.equal(selectCurrentSubmission([history]), null);
});

test("history and unmarked legacy responses cannot enable student edits", () => {
  assert.equal(canEditSubmission(null), true);
  assert.equal(canEditSubmission(null, 2), false);
  assert.equal(canEditSubmission({ id: "current", is_current: true }), true);
  assert.equal(canEditSubmission({ id: "history", is_current: false }), false);
  assert.equal(canEditSubmission({ id: "legacy" }), false);
});

test("POST replacements keep one current ID, history, grades and visible feedback", () => {
  const history = { id: "history", is_current: false, point: 9 };
  const previous = { id: "current", is_current: true, point: 7, comments: ["feedback"] };
  const replacement = { ...previous, attachment_id: "replacement" };
  const merged = mergeCurrentSubmission([history, previous], replacement);
  assert.equal(merged.length, 2);
  assert.deepEqual(merged[0], replacement);
  assert.deepEqual(merged[1], history);
  assert.deepEqual(mergeCurrentSubmission(merged, replacement), merged);
  assert.equal(previous.attachment_id, undefined);
  assert.equal(mergeCurrentSubmission([], replacement).length, 1);
});

test("summary/export preserves graded zero and leaves absent grades blank", () => {
  assert.equal(summaryGradeValue(true, 0), 0);
  assert.equal(summaryGradeValue(true, 7), 7);
  assert.equal(summaryGradeValue(false, 0), "");
  assert.equal(summaryGradeValue(false, 9), "");
  assert.equal(summaryGradeValue(undefined, 0), "");
});
