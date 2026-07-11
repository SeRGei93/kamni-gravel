import { describe, expect, it } from "vitest";
import type { MiniappParticipantOption } from "@/types";
import { filterMiniappRecipientOptions } from "./miniappRecipientOptions";

const participants: MiniappParticipantOption[] = [
  { id: 1, display_name: "Alex", username: "alex", status: "active", has_prize: true },
  { id: 2, display_name: "Bella", username: "bella", status: "active", has_prize: false },
  { id: 3, display_name: "Alexey", username: "alexey", status: "dnf", has_prize: false },
];

describe("filterMiniappRecipientOptions", () => {
  it("puts participants without prizes before awarded participants", () => {
    expect(filterMiniappRecipientOptions(participants, "").map(({ id }) => id)).toEqual([2, 3, 1]);
  });

  it("keeps participants without prizes first in search results", () => {
    expect(filterMiniappRecipientOptions(participants, "alex").map(({ id }) => id)).toEqual([3, 1]);
  });
});
