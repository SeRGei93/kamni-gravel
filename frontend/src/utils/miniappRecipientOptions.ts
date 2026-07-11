import type { MiniappParticipantOption } from "@/types";
import { matchesSearchQuery } from "@/utils/search";

export function filterMiniappRecipientOptions(
  participants: MiniappParticipantOption[],
  searchQuery: string,
): MiniappParticipantOption[] {
  return participants
    .filter((participant) =>
      matchesSearchQuery(searchQuery, [
        participant.display_name,
        participant.username,
        participant.id,
      ]),
    )
    .sort((left, right) => Number(left.has_prize) - Number(right.has_prize));
}
