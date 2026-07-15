// @vitest-environment jsdom

import { act, type ReactNode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { GiftListResponse } from '@/types';
import GiftsPage from './page';

const mocks = vi.hoisted(() => ({
  query: '',
  routerReplace: vi.fn(),
  getActiveEvent: vi.fn(),
  listGifts: vi.fn(),
  getGifts: vi.fn(),
  getManualGifts: vi.fn(),
  updateGift: vi.fn(),
  createGift: vi.fn(),
  assignRandomRecipient: vi.fn(),
  getParticipants: vi.fn(),
  getPrizeDistribution: vi.fn(),
  downloadGiftCsv: vi.fn(),
}));

vi.mock('next/navigation', () => ({
  usePathname: () => '/gifts',
  useRouter: () => ({ replace: mocks.routerReplace }),
  useSearchParams: () => new URLSearchParams(mocks.query),
}));

vi.mock('@/api/events', () => ({
  eventsApi: { getActive: mocks.getActiveEvent },
}));

vi.mock('@/api/gifts', () => ({
  giftsApi: {
    listByEvent: mocks.listGifts,
    getByEvent: mocks.getGifts,
    getManualByEvent: mocks.getManualGifts,
    update: mocks.updateGift,
    create: mocks.createGift,
    assignRandomRecipient: mocks.assignRandomRecipient,
  },
}));

vi.mock('@/api/participants', () => ({
  participantsApi: { getByEvent: mocks.getParticipants },
}));

vi.mock('@/api/prizeDistribution', () => ({
  prizeDistributionApi: {
    getPrizeDistribution: mocks.getPrizeDistribution,
  },
}));

vi.mock('@/utils/events', () => ({
  extractActiveEvent: () => ({ id: 77 }),
}));

vi.mock('@/utils/giftCsv', () => ({
  downloadGiftCsv: mocks.downloadGiftCsv,
  isCurrentGiftExportRequest: (
    requestVersion: number,
    latestRequestVersion: number
  ) => requestVersion === latestRequestVersion,
  shouldSettleGiftExportRequest: (
    requestVersion: number,
    latestRequestVersion: number,
    isMounted: boolean
  ) => isMounted && requestVersion === latestRequestVersion,
}));

vi.mock('@/components/gifts/GiftsTable', () => ({ default: () => null }));
vi.mock('@/components/tables/PaginationControls', () => ({ default: () => null }));
vi.mock('@/components/gifts/GiftOwnerFilter', () => ({ default: () => null }));
vi.mock('@/components/form/input/InputField', () => ({ default: () => null }));
vi.mock('@/components/form/Select', () => ({ default: () => null }));
vi.mock('@/components/form/Label', () => ({ default: () => null }));
vi.mock('@/components/form/input/TextArea', () => ({ default: () => null }));
vi.mock('@/icons', () => ({
  CheckLineIcon: () => null,
  CloseLineIcon: () => null,
  DownloadIcon: () => null,
  PlusIcon: () => null,
}));
vi.mock('@/components/ui/button/Button', () => ({
  default: ({
    children,
    disabled,
    onClick,
    type = 'button',
  }: {
    children?: ReactNode;
    disabled?: boolean;
    onClick?: () => void;
    type?: 'button' | 'submit' | 'reset';
  }) => (
    <button type={type} disabled={disabled} onClick={onClick}>
      {children}
    </button>
  ),
}));

const EMPTY_GIFT_RESPONSE: GiftListResponse = {
  gifts: [],
  total: 0,
  status_counts: {},
};

function deferred<Value>() {
  let resolve!: (value: Value) => void;
  const promise = new Promise<Value>((settle) => {
    resolve = settle;
  });
  return { promise, resolve };
}

async function flushAsyncWork(): Promise<void> {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
}

function getExportButton(container: HTMLElement): HTMLButtonElement {
  const button = Array.from(container.querySelectorAll('button')).find((candidate) =>
    candidate.textContent?.includes('Экспорт')
  );

  if (!(button instanceof HTMLButtonElement)) {
    throw new Error('Export button was not rendered');
  }
  return button;
}

describe('GiftsPage export lifecycle', () => {
  let container: HTMLDivElement;
  let root: Root | null;

  beforeEach(() => {
    mocks.query = '';
    for (const mock of [
      mocks.routerReplace,
      mocks.getActiveEvent,
      mocks.listGifts,
      mocks.getGifts,
      mocks.getManualGifts,
      mocks.updateGift,
      mocks.createGift,
      mocks.assignRandomRecipient,
      mocks.getParticipants,
      mocks.getPrizeDistribution,
      mocks.downloadGiftCsv,
    ]) {
      mock.mockReset();
    }

    mocks.getActiveEvent.mockResolvedValue({});
    mocks.listGifts.mockResolvedValue(EMPTY_GIFT_RESPONSE);
    mocks.getGifts.mockResolvedValue(EMPTY_GIFT_RESPONSE);
    mocks.getManualGifts.mockResolvedValue({ gifts: [] });
    mocks.getParticipants.mockResolvedValue({ participants: [], total: 0 });
    mocks.getPrizeDistribution.mockResolvedValue({ distribution: [], total: 0 });

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
    if (root) {
      await act(async () => root?.unmount());
    }
    container.remove();
  });

  it('does not download a stale export after the distribution filter changes', async () => {
    const exportResponse = deferred<GiftListResponse>();
    mocks.getGifts
      .mockImplementationOnce(() => exportResponse.promise)
      .mockResolvedValue(EMPTY_GIFT_RESPONSE);

    await act(async () => root?.render(<GiftsPage />));
    await flushAsyncWork();

    const exportButton = getExportButton(container);
    expect(exportButton.disabled).toBe(false);

    act(() => exportButton.click());
    expect(getExportButton(container).textContent).toContain('Экспорт…');

    mocks.query = 'distribution=manual';
    await act(async () => root?.render(<GiftsPage />));
    await flushAsyncWork();

    exportResponse.resolve(EMPTY_GIFT_RESPONSE);
    await flushAsyncWork();

    expect(mocks.downloadGiftCsv).not.toHaveBeenCalled();
    expect(getExportButton(container).disabled).toBe(false);
    expect(getExportButton(container).textContent).toContain('Экспорт в CSV');
  });

  it('does not download or update export state after unmount', async () => {
    const exportResponse = deferred<GiftListResponse>();
    mocks.getGifts.mockImplementationOnce(() => exportResponse.promise);

    await act(async () => root?.render(<GiftsPage />));
    await flushAsyncWork();

    act(() => getExportButton(container).click());
    await act(async () => root?.unmount());
    root = null;

    exportResponse.resolve(EMPTY_GIFT_RESPONSE);
    await flushAsyncWork();

    expect(mocks.downloadGiftCsv).not.toHaveBeenCalled();
  });

  it('blocks a repeated export click while the current request is pending', async () => {
    const exportResponse = deferred<GiftListResponse>();
    mocks.getGifts.mockImplementationOnce(() => exportResponse.promise);

    await act(async () => root?.render(<GiftsPage />));
    await flushAsyncWork();

    act(() => getExportButton(container).click());
    expect(getExportButton(container).disabled).toBe(true);

    act(() => getExportButton(container).click());
    expect(mocks.getGifts).toHaveBeenCalledTimes(1);

    exportResponse.resolve(EMPTY_GIFT_RESPONSE);
    await flushAsyncWork();

    expect(mocks.downloadGiftCsv).toHaveBeenCalledTimes(1);
    expect(getExportButton(container).disabled).toBe(false);
  });

  it('shows an export error and allows a successful retry', async () => {
    mocks.getGifts.mockRejectedValueOnce(new Error('export failed'));

    await act(async () => root?.render(<GiftsPage />));
    await flushAsyncWork();

    act(() => getExportButton(container).click());
    await flushAsyncWork();

    expect(container.textContent).toContain('Не удалось выгрузить список призов');
    expect(mocks.downloadGiftCsv).not.toHaveBeenCalled();
    expect(getExportButton(container).disabled).toBe(false);

    mocks.getGifts.mockResolvedValueOnce(EMPTY_GIFT_RESPONSE);
    act(() => getExportButton(container).click());
    await flushAsyncWork();

    expect(mocks.downloadGiftCsv).toHaveBeenCalledTimes(1);
    expect(container.textContent).not.toContain('Не удалось выгрузить список призов');
    expect(getExportButton(container).disabled).toBe(false);
  });
});
