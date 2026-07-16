import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import type { ManualGift } from "@/types";
import MyGiftCard from "./MyGiftCard";

const automaticGift: ManualGift = {
  id: 2,
  event_id: 77,
  description: "Heart rate prize",
  review_status: "approved",
  manual_distribution: false,
  recipients: [
    { id: 10, display_name: "Ivan", username: "ivan", status: "active" },
    { id: 11, display_name: "Maria", status: "dnf" },
  ],
  created_at: "2026-07-16T00:00:00Z",
};

function renderCard(gift: ManualGift, showGiftRecipients: boolean): string {
  return renderToStaticMarkup(
    <MyGiftCard
      gift={gift}
      participants={[]}
      savingGiftID={null}
      showGiftRecipients={showGiftRecipients}
      onSaveRecipient={async () => undefined}
      onAssignRandomRecipient={async () => undefined}
    />
  );
}

describe("MyGiftCard automatic recipients", () => {
  it("hides automatic recipients when the event setting is disabled", () => {
    const markup = renderCard(automaticGift, false);

    expect(markup).not.toContain("Получатели:");
    expect(markup).not.toContain("Ivan");
    expect(markup).not.toContain("Maria");
  });

  it("shows every automatic recipient when the event setting is enabled", () => {
    const markup = renderCard(automaticGift, true);

    expect(markup).toContain("Получатели:");
    expect(markup).toContain("Ivan (@ivan)");
    expect(markup).toContain('href="https://t.me/ivan"');
    expect(markup).toContain("Maria");
    expect(markup).not.toContain('href="https://t.me/Maria"');
  });

  it("uses a singular label for one automatic recipient", () => {
    const markup = renderCard(
      { ...automaticGift, recipients: [automaticGift.recipients![0]] },
      true
    );

    expect(markup).toContain("Получатель:");
    expect(markup).not.toContain("Получатели:");
  });

  it("keeps a manually assigned recipient visible regardless of the event setting", () => {
    const markup = renderCard(
      {
        ...automaticGift,
        manual_distribution: true,
        recipients: undefined,
        recipient: { id: 12, display_name: "Alex", username: "alex", status: "active" },
      },
      false
    );

    expect(markup).toContain("Получатель:");
    expect(markup).toContain("Alex (@alex)");
    expect(markup).toContain('href="https://t.me/alex"');
  });
});
