import type { Participant } from '@/types';

export type HasGiftFilter = 'all' | 'yes' | 'no';

export interface ParticipantFilterOptions {
  searchQuery?: string;
  hasGiftFilter?: HasGiftFilter;
}

function matchesSearch(participant: Participant, query: string): boolean {
  const normalized = query.trim().toLowerCase();
  if (!normalized) {
    return true;
  }

  return Boolean(
    participant.username?.toLowerCase().includes(normalized) ||
      participant.first_name?.toLowerCase().includes(normalized) ||
      participant.last_name?.toLowerCase().includes(normalized) ||
      String(participant.user_id).includes(normalized)
  );
}

function matchesHasGift(
  participant: Participant,
  filter: HasGiftFilter
): boolean {
  if (filter === 'all') {
    return true;
  }
  return filter === 'yes' ? participant.has_gift : !participant.has_gift;
}

/**
 * Клиентская фильтрация участников по поисковому запросу и наличию приза.
 * Гендер/тип велосипеда/статус финиша приходят уже отфильтрованными из API,
 * поэтому здесь не дублируются.
 */
export function filterParticipants(
  participants: Participant[],
  options: ParticipantFilterOptions = {}
): Participant[] {
  const { searchQuery = '', hasGiftFilter = 'all' } = options;
  return participants.filter(
    (participant) =>
      matchesSearch(participant, searchQuery) &&
      matchesHasGift(participant, hasGiftFilter)
  );
}
