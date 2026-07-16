// @vitest-environment jsdom

import { act, type ReactNode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { Gift } from '@/types';
import GiftsTable from './GiftsTable';

vi.mock('next/image', () => ({
  default: () => null,
}));

vi.mock('next/link', () => ({
  default: ({ href, children }: { href: string; children: ReactNode }) => (
    <a href={href}>{children}</a>
  ),
}));

vi.mock('./useGiftPhotoUrls', () => ({
  useGiftPhotoUrls: () => ({}),
}));

vi.mock('@/icons', () => ({
  CheckLineIcon: () => null,
  PencilIcon: () => null,
}));

const automaticGift: Gift = {
  id: 1,
  user_id: 10,
  event_id: 77,
  description: 'Automatic gift',
  review_status: 'approved',
  created_at: '2026-07-16T00:00:00Z',
};

const manualGift: Gift = {
  ...automaticGift,
  id: 2,
  description: 'Manual gift',
  manual_distribution: true,
};

function deferred(): { promise: Promise<void>; resolve: () => void } {
  let resolve!: () => void;
  const promise = new Promise<void>((settle) => {
    resolve = settle;
  });
  return { promise, resolve };
}

describe('GiftsTable random recipient actions', () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);
    (
      globalThis as typeof globalThis & {
        IS_REACT_ACT_ENVIRONMENT: boolean;
      }
    ).IS_REACT_ACT_ENVIRONMENT = true;
  });

  afterEach(async () => {
    await act(async () => root.unmount());
    container.remove();
  });

  it('shows the including-awarded action only for an approved unassigned manual gift', async () => {
    await act(async () => {
      root.render(
        <GiftsTable
          gifts={[automaticGift, manualGift]}
          onAssignRandomRecipient={vi.fn().mockResolvedValue(undefined)}
          onAssignRandomRecipientIncludingAwarded={vi.fn().mockResolvedValue(undefined)}
        />
      );
    });

    const strictActions = Array.from(container.querySelectorAll('button')).filter(
      (button) => button.textContent === 'Отдать рандомному участнику без награды'
    );
    const wideActions = Array.from(container.querySelectorAll('button')).filter(
      (button) => button.textContent === 'Отдать рандомному участнику'
    );

    expect(strictActions).toHaveLength(2);
    expect(wideActions).toHaveLength(1);
    expect(wideActions[0]?.getAttribute('title')).toBe('Отдать рандомному участнику');
  });

  it('disables both random actions for a row while either random request is pending', async () => {
    const pendingAssignment = deferred();
    const assignIncludingAwarded = vi.fn().mockReturnValue(pendingAssignment.promise);

    await act(async () => {
      root.render(
        <GiftsTable
          gifts={[manualGift]}
          onAssignRandomRecipient={vi.fn().mockResolvedValue(undefined)}
          onAssignRandomRecipientIncludingAwarded={assignIncludingAwarded}
        />
      );
    });

    const strictAction = Array.from(container.querySelectorAll('button')).find(
      (button) => button.textContent === 'Отдать рандомному участнику без награды'
    ) as HTMLButtonElement;
    const wideAction = Array.from(container.querySelectorAll('button')).find(
      (button) => button.textContent === 'Отдать рандомному участнику'
    ) as HTMLButtonElement;

    act(() => wideAction.click());
    await act(async () => Promise.resolve());

    expect(assignIncludingAwarded).toHaveBeenCalledWith(manualGift);
    expect(strictAction.disabled).toBe(true);
    expect(wideAction.disabled).toBe(true);

    pendingAssignment.resolve();
    await act(async () => pendingAssignment.promise);

    expect(strictAction.disabled).toBe(false);
    expect(wideAction.disabled).toBe(false);
  });
});
