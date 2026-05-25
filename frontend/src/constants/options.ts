// Общие константы для выпадающих списков

export const GENDER_OPTIONS = [
  { value: 'all', label: 'Любой' },
  { value: 'male', label: 'Мужской' },
  { value: 'female', label: 'Женский' },
];

export const GENDER_OPTIONS_WITHOUT_ALL = [
  { value: 'male', label: 'Мужской' },
  { value: 'female', label: 'Женский' },
];

export const BIKE_TYPE_OPTIONS = [
  { value: 'all', label: 'Любой' },
  { value: 'gravel', label: 'Gravel' },
  { value: 'mtb', label: 'MTB' },
  { value: 'road', label: 'Шоссе' },
  { value: 'single_speed', label: 'Single Speed' },
  { value: 'tandem', label: 'Тандем' },
];

export const BIKE_TYPE_OPTIONS_WITHOUT_ALL = [
  { value: 'gravel', label: 'Gravel' },
  { value: 'mtb', label: 'MTB' },
  { value: 'road', label: 'Шоссе' },
  { value: 'single_speed', label: 'Single Speed' },
  { value: 'tandem', label: 'Тандем' },
];

export const GIFT_REVIEW_STATUS_FILTER_OPTIONS = [
  { value: 'all', label: 'Все' },
  { value: 'pending_review', label: 'На проверке' },
  { value: 'approved', label: 'Проверен' },
];

export const GIFT_REVIEW_STATUS_OPTIONS = [
  { value: 'pending_review', label: 'Новый / на проверке' },
  { value: 'approved', label: 'Проверен' },
];

export const CRITERIA_TYPE_OPTIONS = [
  { value: 'speed', label: 'Скорость' },
  { value: 'photo', label: 'Фото' },
  { value: 'beer', label: 'Пиво' },
  { value: 'custom', label: 'Кастомный' },
];
