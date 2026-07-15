import { describe, expect, it } from 'vitest';
import { buildParticipantListParams } from './participantListParams';

describe('buildParticipantListParams', () => {
  it('omits default filters while preserving pagination', () => {
    expect(
      buildParticipantListParams({
        gender: '',
        bikeType: '',
        isFinished: '',
        hasGift: 'all',
        criteriaId: '',
        q: '',
        sortKey: null,
        sortOrder: 'asc',
        page: 3,
        pageSize: 50,
      }),
    ).toEqual({
      gender: undefined,
      bike_type: undefined,
      is_finished: undefined,
      has_gift: undefined,
      criteria_id: undefined,
      q: undefined,
      sort: undefined,
      order: undefined,
      page: 3,
      page_size: 50,
    });
  });

  it('keeps all applied filters and the unpaginated export mode', () => {
    expect(
      buildParticipantListParams({
        gender: 'female',
        bikeType: 'gravel',
        isFinished: 'false',
        hasGift: 'no',
        criteriaId: '12',
        q: 'rider',
        sortKey: 'elapsed_time',
        sortOrder: 'desc',
        page: 1,
        pageSize: 'all',
      }),
    ).toEqual({
      gender: 'female',
      bike_type: 'gravel',
      is_finished: false,
      has_gift: false,
      criteria_id: 12,
      q: 'rider',
      sort: 'elapsed_time',
      order: 'desc',
      page: 1,
      page_size: 'all',
    });
  });

  it('drops an invalid criterion identifier', () => {
    expect(
      buildParticipantListParams({
        gender: 'male',
        bikeType: 'mtb',
        isFinished: 'true',
        hasGift: 'yes',
        criteriaId: '-1',
        q: 'alex',
        sortKey: null,
        sortOrder: 'asc',
        page: 1,
        pageSize: 100,
      }).criteria_id,
    ).toBeUndefined();
  });
});
