export type SubmissionRecord = { id: string; is_current?: boolean };

export function selectCurrentSubmission<T extends SubmissionRecord>(records: T[]): T | null {
  return records.find((record) => record.is_current === true) ?? null;
}

export function canEditSubmission(record: SubmissionRecord | null, existingRecords = 0): boolean {
  return record === null ? existingRecords === 0 : record.is_current === true;
}

// Student write results may reuse an ID even when the request was a POST.
export function mergeCurrentSubmission<T extends SubmissionRecord>(records: T[], result: T): T[] {
  return [
    { ...result, is_current: true },
    ...records.filter((record) => record.id !== result.id).map((record) => ({ ...record, is_current: false })),
  ];
}

export function summaryGradeValue(hasGrade: boolean | undefined, value: number): number | "" {
  return hasGrade === true ? value : "";
}
