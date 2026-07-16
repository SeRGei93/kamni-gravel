// @vitest-environment jsdom

import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MiniappApiError } from "@/api/miniapp";
import type { ManualGift } from "@/types";
import MyGiftRecipientSelect from "./MyGiftRecipientSelect";

const gift: ManualGift = {
  id: 15,
  event_id: 77,
  description: "Manual gift",
  review_status: "approved",
  manual_distribution: true,
  created_at: "2026-07-16T00:00:00Z",
};

function randomActionButton(container: HTMLElement, label: string): HTMLButtonElement {
  const button = Array.from(container.querySelectorAll("button")).find(
    (candidate) => candidate.textContent === label
  );
  if (!(button instanceof HTMLButtonElement)) {
    throw new Error(`Button not found: ${label}`);
  }
  return button;
}

describe("MyGiftRecipientSelect random actions", () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
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
  });

  it("shows both actions and disables recipient selection with both actions while saving", async () => {
    await act(async () => {
      root.render(
        <MyGiftRecipientSelect
          gift={gift}
          participants={[]}
          isSaving
          onSave={async () => undefined}
          onAssignRandom={async () => undefined}
          onAssignRandomIncludingAwarded={async () => undefined}
        />
      );
    });

    expect(randomActionButton(container, "Отдать рандомному участнику без награды").disabled).toBe(true);
    expect(randomActionButton(container, "Отдать рандомному участнику").disabled).toBe(true);
    expect(
      Array.from(container.querySelectorAll("button")).find(
        (button) => button.textContent === "Выберите получателя"
      )
    ).toHaveProperty("disabled", true);
  });

  it("uses action-specific conflict copy for the including-awarded action", async () => {
    const assignIncludingAwarded = vi
      .fn()
      .mockRejectedValue(new MiniappApiError(409, "Conflict"));
    const consoleWarn = vi.spyOn(console, "warn").mockImplementation(() => undefined);

    await act(async () => {
      root.render(
        <MyGiftRecipientSelect
          gift={gift}
          participants={[]}
          isSaving={false}
          onSave={async () => undefined}
          onAssignRandom={async () => undefined}
          onAssignRandomIncludingAwarded={assignIncludingAwarded}
        />
      );
    });

    act(() => randomActionButton(container, "Отдать рандомному участнику").click());
    await act(async () => Promise.resolve());

    expect(assignIncludingAwarded).toHaveBeenCalledWith(15);
    expect(container.textContent).toContain("не находится в ручном режиме");
    expect(container.textContent).not.toContain("участников без награды не осталось");
    consoleWarn.mockRestore();
  });
});
