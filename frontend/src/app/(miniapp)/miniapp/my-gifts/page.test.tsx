// @vitest-environment jsdom

import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ManualGiftListResponse } from "@/types";
import MiniappMyGiftsPage from "./page";

const mocks = vi.hoisted(() => ({
  getMyGifts: vi.fn(),
  getParticipants: vi.fn(),
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
}));

vi.mock("@/api/miniapp", () => ({
  miniappApi: {
    getMyGifts: mocks.getMyGifts,
    getParticipants: mocks.getParticipants,
    updateMyGiftRecipient: vi.fn(),
    assignRandomMyGiftRecipient: vi.fn(),
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

vi.mock("@/components/miniapp/MyGiftCard", () => ({ default: () => null }));
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
    mocks.setSnapshot.mockReset();
    mocks.updateSnapshot.mockReset();
    mocks.myGiftsState.snapshot = null;
    mocks.myGiftsState.setSnapshot = mocks.setSnapshot;
    mocks.myGiftsState.updateSnapshot = mocks.updateSnapshot;
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
});
