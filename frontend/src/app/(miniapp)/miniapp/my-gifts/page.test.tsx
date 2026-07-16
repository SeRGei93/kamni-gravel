// @vitest-environment jsdom

import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ManualGiftListResponse } from "@/types";
import MiniappMyGiftsPage from "./page";

const mocks = vi.hoisted(() => ({
  getMyGifts: vi.fn(),
  getParticipants: vi.fn(),
  updateMyGiftRecipient: vi.fn(),
  assignRandomMyGiftRecipient: vi.fn(),
  assignRandomMyGiftRecipientIncludingAwarded: vi.fn(),
  setSnapshot: vi.fn(),
  updateSnapshot: vi.fn(),
  sessionState: {
    session: { event: { show_gift_recipients: true } },
    isLoading: false,
    error: null,
  },
  myGiftsState: {
    snapshot: null,
    setSnapshot: vi.fn(),
    updateSnapshot: vi.fn(),
    scrollYRef: { current: 0 },
  },
  myGiftCardPropsByGiftID: {} as Record<number, Record<string, unknown>>,
}));

vi.mock("@/api/miniapp", () => ({
  miniappApi: {
    getMyGifts: mocks.getMyGifts,
    getParticipants: mocks.getParticipants,
    updateMyGiftRecipient: mocks.updateMyGiftRecipient,
    assignRandomMyGiftRecipient: mocks.assignRandomMyGiftRecipient,
    assignRandomMyGiftRecipientIncludingAwarded:
      mocks.assignRandomMyGiftRecipientIncludingAwarded,
  },
  MiniappApiError: class MiniappApiError extends Error {
    status = 0;
  },
}));

vi.mock("@/components/miniapp/MiniappSessionContext", () => ({
  useMiniappSession: () => mocks.sessionState,
}));

vi.mock("@/components/miniapp/MiniappMyGiftsContext", () => ({
  useMiniappMyGifts: () => mocks.myGiftsState,
}));

vi.mock("@/components/miniapp/MyGiftCard", () => ({
  default: (props: Record<string, unknown>) => {
    const gift = props.gift as { id: number };
    mocks.myGiftCardPropsByGiftID[gift.id] = props;
    return null;
  },
}));
vi.mock("@/components/miniapp/MiniappSpinner", () => ({ default: () => null }));
vi.mock("@/hooks/useIsomorphicLayoutEffect", () => ({
  useIsomorphicLayoutEffect: () => undefined,
}));

const RESPONSE: ManualGiftListResponse = {
  gifts: [],
  participants: [
    { id: 12, display_name: "Alex", status: "active", has_prize: false },
  ],
};

describe("MiniappMyGiftsPage", () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    mocks.getMyGifts.mockReset().mockResolvedValue(RESPONSE);
    mocks.getParticipants.mockReset();
    mocks.updateMyGiftRecipient.mockReset();
    mocks.assignRandomMyGiftRecipient.mockReset();
    mocks.assignRandomMyGiftRecipientIncludingAwarded.mockReset();
    mocks.setSnapshot.mockReset();
    mocks.updateSnapshot.mockReset();
    mocks.myGiftsState.snapshot = null;
    mocks.myGiftsState.setSnapshot = mocks.setSnapshot;
    mocks.myGiftsState.updateSnapshot = mocks.updateSnapshot;
    mocks.myGiftCardPropsByGiftID = {};
    container = document.createElement("div");
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
    vi.restoreAllMocks();
  });

  it("loads participant options from my-gifts without a second request", async () => {
    await act(async () => {
      root.render(<MiniappMyGiftsPage />);
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.getMyGifts).toHaveBeenCalledTimes(1);
    expect(mocks.getParticipants).not.toHaveBeenCalled();
    expect(mocks.setSnapshot).toHaveBeenCalledWith({
      gifts: RESPONSE.gifts,
      participants: RESPONSE.participants,
    });
  });

  it("wires the including-awarded action through the card and refreshes My Prizes after success", async () => {
    mocks.myGiftsState.snapshot = {
      gifts: [
        {
          id: 15,
          event_id: 77,
          description: "Manual gift",
          review_status: "approved",
          manual_distribution: true,
          created_at: "2026-07-16T00:00:00Z",
        },
      ],
      participants: RESPONSE.participants ?? [],
    };
    mocks.assignRandomMyGiftRecipientIncludingAwarded.mockResolvedValue(undefined);

    await act(async () => {
      root.render(<MiniappMyGiftsPage />);
      await Promise.resolve();
    });

    const callback = mocks.myGiftCardPropsByGiftID[15]?.onAssignRandomRecipientIncludingAwarded;
    if (typeof callback !== "function") {
      throw new Error("Including-awarded callback was not passed to MyGiftCard");
    }
    mocks.getMyGifts.mockClear();

    await act(async () => {
      await callback(15);
    });

    expect(mocks.assignRandomMyGiftRecipientIncludingAwarded).toHaveBeenCalledWith(15);
    expect(mocks.getMyGifts).toHaveBeenCalledTimes(1);
  });

  it("does not refresh or update the snapshot when an including-awarded request completes after unmount", async () => {
    let resolveAssignment!: () => void;
    const pendingAssignment = new Promise<void>((resolve) => {
      resolveAssignment = resolve;
    });
    mocks.assignRandomMyGiftRecipientIncludingAwarded.mockImplementationOnce(
      () => pendingAssignment
    );
    mocks.myGiftsState.snapshot = {
      gifts: [
        {
          id: 15,
          event_id: 77,
          description: "Manual gift",
          review_status: "approved",
          manual_distribution: true,
          created_at: "2026-07-16T00:00:00Z",
        },
      ],
      participants: RESPONSE.participants ?? [],
    };

    await act(async () => {
      root.render(<MiniappMyGiftsPage />);
      await Promise.resolve();
    });
    const callback = mocks.myGiftCardPropsByGiftID[15]?.onAssignRandomRecipientIncludingAwarded;
    if (typeof callback !== "function") {
      throw new Error("Including-awarded callback was not passed to MyGiftCard");
    }
    const pendingCallback = callback(15);
    await act(async () => Promise.resolve());

    await act(async () => root.unmount());
    mocks.getMyGifts.mockClear();
    mocks.updateSnapshot.mockClear();
    resolveAssignment();
    await pendingCallback;

    expect(mocks.getMyGifts).not.toHaveBeenCalled();
    expect(mocks.updateSnapshot).not.toHaveBeenCalled();
  });

  it("serializes mutations across gifts so the completed action always refreshes My Prizes", async () => {
    let resolveAssignment!: () => void;
    const pendingAssignment = new Promise<void>((resolve) => {
      resolveAssignment = resolve;
    });
    mocks.assignRandomMyGiftRecipientIncludingAwarded.mockImplementationOnce(
      () => pendingAssignment
    );
    mocks.myGiftsState.snapshot = {
      gifts: [
        {
          id: 15,
          event_id: 77,
          description: "First manual gift",
          review_status: "approved",
          manual_distribution: true,
          created_at: "2026-07-16T00:00:00Z",
        },
        {
          id: 16,
          event_id: 77,
          description: "Second manual gift",
          review_status: "approved",
          manual_distribution: true,
          created_at: "2026-07-16T00:00:00Z",
        },
      ],
      participants: RESPONSE.participants ?? [],
    };

    await act(async () => {
      root.render(<MiniappMyGiftsPage />);
      await Promise.resolve();
    });

    const firstCallback = mocks.myGiftCardPropsByGiftID[15]?.onAssignRandomRecipientIncludingAwarded;
    const secondCallback = mocks.myGiftCardPropsByGiftID[16]?.onAssignRandomRecipient;
    if (typeof firstCallback !== "function" || typeof secondCallback !== "function") {
      throw new Error("Random-recipient callbacks were not passed to both MyGiftCard instances");
    }

    let pendingCallback!: Promise<void>;
    act(() => {
      pendingCallback = firstCallback(15);
    });
    await act(async () => {
      await Promise.resolve();
    });

    expect(mocks.myGiftCardPropsByGiftID[15]?.isSaving).toBe(true);
    expect(mocks.myGiftCardPropsByGiftID[16]?.isSaving).toBe(true);

    await act(async () => {
      await secondCallback(16);
    });
    expect(mocks.assignRandomMyGiftRecipient).not.toHaveBeenCalled();

    mocks.getMyGifts.mockClear();
    resolveAssignment();
    await act(async () => {
      await pendingCallback;
    });

    expect(mocks.assignRandomMyGiftRecipientIncludingAwarded).toHaveBeenCalledWith(15);
    expect(mocks.getMyGifts).toHaveBeenCalledTimes(1);
  });
});
