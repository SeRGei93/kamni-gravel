import type { Participant } from '@/types';
import { matchesSearchQuery } from './search';

export type HasGiftFilter = 'all' | 'yes' | 'no';

export interface ParticipantFilterOptions {
  searchQuery?: string;
  hasGiftFilter?: HasGiftFilter;
}

function matchesSearch(participant: Participant, query: string): boolean {
  return matchesSearchQuery(query, [
    participant.username,
    participant.first_name,
    participant.last_name,
    participant.user_id,
  ]);
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
