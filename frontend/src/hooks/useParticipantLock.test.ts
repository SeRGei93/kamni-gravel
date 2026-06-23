import { describe, expect, it } from 'vitest';
import {
  isStatusLockedByOther,
  ownerNameFromStatus,
  releaseOnEnd,
} from './participantLock.helpers';
import type { LockStatus } from '@/types';

const free: LockStatus = { participant_id: 1, locked: false, is_mine: false };
const mine: LockStatus = {
  participant_id: 1,
  locked: true,
  is_mine: true,
  locked_by_username: 'alice',
};
const other: LockStatus = {
  participant_id: 1,
  locked: true,
  is_mine: false,
  locked_by_username: 'bob',
};

describe('isStatusLockedByOther', () => {
  it('is false when no lock, free, or owned by us', () => {
    expect(isStatusLockedByOther(null)).toBe(false);
    expect(isStatusLockedByOther(free)).toBe(false);
    expect(isStatusLockedByOther(mine)).toBe(false);
  });

  it('is true only when another admin holds the lock', () => {
    expect(isStatusLockedByOther(other)).toBe(true);
  });
});

describe('ownerNameFromStatus', () => {
  it('returns the owner name only for a foreign lock', () => {
    expect(ownerNameFromStatus(other)).toBe('bob');
    expect(ownerNameFromStatus(mine)).toBeUndefined();
    expect(ownerNameFromStatus(free)).toBeUndefined();
    expect(ownerNameFromStatus(null)).toBeUndefined();
  });
});

describe('releaseOnEnd (shared-lock ref count)', () => {
  it('keeps the lock while other sections are still editing', () => {
    // Две секции открыты, одна закрылась — лок ещё держим.
    expect(releaseOnEnd(2)).toEqual({ activeEditsAfter: 1, release: false });
  });

  it('releases the lock when the last section closes', () => {
    expect(releaseOnEnd(1)).toEqual({ activeEditsAfter: 0, release: true });
  });

  it('never goes negative and does not release when nothing was held', () => {
    expect(releaseOnEnd(0)).toEqual({ activeEditsAfter: 0, release: false });
  });
});
