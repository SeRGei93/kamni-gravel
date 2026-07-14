import type { Gift } from "@/types";
import { hasSearchQuery, matchesSearchQuery } from "@/utils/search";

function getGiftAuthorName(gift: Gift): string {
  return [gift.first_name, gift.last_name].filter(Boolean).join(" ");
}

export function filterMiniappGiftsBySearch(gifts: Gift[], searchQuery: string): Gift[] {
  return gifts.filter((gift) =>
    matchesSearchQuery(searchQuery, [
      gift.description,
      gift.username,
      gift.first_name,
      gift.last_name,
      getGiftAuthorName(gift),
    ])
  );
}

export function shouldShowGiftPlaceGaps(searchQuery: string): boolean {
  return !hasSearchQuery(searchQuery);
}
