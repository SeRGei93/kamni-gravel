// Типы для API

export type BikeType = 'gravel' | 'mtb' | 'road' | 'single_speed' | 'tandem';
export type Gender = 'male' | 'female';
export type GenderFilter = 'all' | 'male' | 'female';
export type BikeTypeFilter = 'all' | 'gravel' | 'mtb' | 'road' | 'single_speed' | 'tandem';
export type FileType = 'photo' | 'document';
export type CriteriaType = 'speed' | 'photo' | 'beer' | 'random' | 'custom';
export type GiftReviewStatus = 'pending_review' | 'approved';
// Статус участия в зачёте: active — обычный участник; dnf — сошёл с дистанции
// (исключён из зачёта и призов по местам, но участвует в призах по критериям);
// disqualified — дисквалификация (исключён из любого распределения призов).
export type ParticipantStatus = 'active' | 'dnf' | 'disqualified';

export const PARTICIPANT_STATUS_LABELS: Record<ParticipantStatus, string> = {
  active: 'Участвует',
  dnf: 'Сошёл с дистанции',
  disqualified: 'Дисквалификация',
};

export interface User {
  id: number;
  username: string;
  role: string;
}

export interface AdminUser {
  id: number;
  username: string;
  role: string;
  created_at: string;
  last_login?: string | null;
}

export interface AdminUserListResponse {
  admins: AdminUser[];
  total: number;
}

export interface CreateAdminRequest {
  username: string;
  password: string;
}

export interface ChangePasswordRequest {
  current_password: string;
  new_password: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  user: User;
}

export interface Participant {
  id: number;
  user_id: number;
  username: string;
  first_name: string;
  last_name: string;
  event_id: number;
  bike_type: BikeType;
  gender: Gender;
  status: ParticipantStatus;
  result_link?: string;
  is_finished: boolean;
  elapsed_time?: string; // формат ЧЧ:ММ:СС
  moving_time?: string; // формат ЧЧ:ММ:СС
  elapsed_time_sec?: number;
  moving_time_sec?: number;
  prev_elapsed_time?: string; // время прошлого года (ручное > вычисленное), ЧЧ:ММ:СС
  prev_elapsed_time_sec?: number;
  prev_elapsed_time_manual_sec?: number; // ручное время прошлого года (для формы правки)
  prev_elapsed_delta?: string; // прошлый год − общее время, ±ЧЧ:ММ:СС (плюс = быстрее)
  prev_elapsed_delta_sec?: number;
  notes?: string;
  registered_at: string;
  finished_at?: string;
  place?: number; // место в зачёте (0 если нет) - устаревшее, используйте place_absolute
  place_absolute?: number; // место в абсолютном зачёте
  place_by_gender?: number; // место в зачёте по гендеру
  place_by_gender_bike?: number; // место в зачёте по гендеру+тип велосипеда
  has_gift: boolean;
  prizes_count: number;
  matched_gifts?: Gift[]; // все подобранные призы
  matched_gift_assignments?: PrizeGiftAssignment[];

  // Метрики заезда из текущего результата (опциональны).
  // ВНИМАНИЕ: finished_at выше — дата отправки результата; время финиша
  // заезда приходит как ride_finished_at (см. backend ParticipantDTO).
  started_at?: string; // ISO 8601 — время старта заезда
  ride_finished_at?: string; // ISO 8601 — время финиша заезда
  distance_meters?: number;
  avg_heart_rate?: number;
  max_heart_rate?: number;
  peak_speed_kmh?: number;
  avg_cadence?: number;
  calories?: number;
  ride_date?: string; // YYYY-MM-DD
  idle_time_sec?: number;
  idle_time?: string; // ЧЧ:ММ:СС
  avg_speed_kmh?: number;
  avg_moving_speed_kmh?: number;
  peak_avg_speed_delta_kmh?: number; // пиковая − средняя скорость, км/ч
  heart_rate_time_product?: number; // общее время (мин) × средний пульс
}

export interface ParticipantDetail extends Participant {
  gifts: Gift[];
}

export interface ParticipantListResponse {
  participants: Participant[];
  total: number;
  page?: number;
  page_size?: number;
}

export interface UserBlacklistEntry {
  telegram_user_id: number;
  reason: string;
  username?: string;
  first_name?: string;
  last_name?: string;
  created_at: string;
  updated_at: string;
}

export interface UserBlacklistListResponse {
  entries: UserBlacklistEntry[];
  total: number;
}

export interface CreateUserBlacklistRequest {
  telegram_user_id: number;
  reason?: string;
}

export interface UpdateUserBlacklistRequest {
  reason?: string;
}

export interface Event {
  id: number;
  name: string;
  description: string;
  participation_conditions: string;
  active: boolean;
  stop_results: boolean;
  stop_gifts: boolean;
  start_date?: string;
  end_date?: string;
  gpx_file_path?: string;
  telegram_texts: EventTelegramTexts;
  created_at: string;
  updated_at: string;
}

export interface EventTelegramTexts {
  gift_gender_step: string;
  gift_bike_step: string;
  gift_description_step: string;
  gift_photo_step: string;
  gift_photo_added: string;
  gift_draft: string;
  gift_draft_value_missing: string;
  gift_draft_description_missing: string;
  gift_draft_description_added: string;
  gift_draft_action_description: string;
  gift_draft_action_photo: string;
  gift_preview: string;
  gift_confirmation_prompt: string;
  gift_success: string;
  gift_cancelled: string;
  gift_session_error: string;
  gift_callback_continue: string;
  gift_callback_add_description: string;
  gift_callback_review_draft: string;
  gift_callback_confirm: string;
  gift_callback_open_menu: string;
  result_prompt: string;
  result_invalid_link: string;
  result_success: string;
  result_already_sent: string;
  result_not_registered: string;
  result_start_missing: string;
  result_not_started: string;
}

export interface EventListResponse {
  events: Event[];
  total: number;
}

export interface GiftAttachment {
  id: number;
  gift_id: number;
  telegram_file_id: string;
  file_type: FileType;
}

export type GiftPlaceRuleType = 'places' | 'last_n';

export interface GiftPlaceRule {
  type: GiftPlaceRuleType;
  places?: number[];
  last_count?: number;
}

export interface Gift {
  id: number;
  user_id: number;
  username?: string;
  first_name?: string;
  last_name?: string;
  event_id: number;
  description: string;
  gender_filter?: GenderFilter;
  bike_type_filter?: BikeTypeFilter;
  review_status: GiftReviewStatus;
  place?: number;
  place_rule?: GiftPlaceRule | null;
  attachments?: GiftAttachment[];
  criteria?: Criteria[];
  // Public metadata only. Recipient identity is available exclusively through
  // the protected manual-gift management responses below.
  manual_distribution?: boolean;
  manual_assignment?: ManualGift;
  created_at: string;
}

export interface GiftListResponse {
  gifts: Gift[];
  total: number;
  page?: number;
  page_size?: number;
  status_counts?: Record<string, number>;
  participant_count?: number;
}

export interface ManualGiftRecipient {
  id: number;
  display_name: string;
  username?: string;
  status: ParticipantStatus;
}

// Protected owner/admin read model. It intentionally omits Telegram user IDs,
// notes, results, and registration metadata.
export interface ManualGift {
  id: number;
  event_id: number;
  description: string;
  gender_filter?: GenderFilter;
  bike_type_filter?: BikeTypeFilter;
  review_status: GiftReviewStatus;
  manual_distribution: boolean;
  place?: number;
  place_rule?: GiftPlaceRule | null;
  attachments?: GiftAttachment[];
  criteria?: Criteria[];
  recipient?: ManualGiftRecipient;
  created_at: string;
}

export interface ManualGiftListResponse {
  gifts: ManualGift[];
}

export interface MiniappParticipantOption {
  id: number;
  display_name: string;
  username?: string;
  status: ParticipantStatus;
  has_prize: boolean;
}

export interface MiniappParticipantOptionsResponse {
  participants: MiniappParticipantOption[];
  total: number;
}

export interface MiniappTelegramUser {
  id: number;
  username?: string;
  first_name?: string;
  last_name?: string;
  language_code?: string;
  photo_url?: string;
  is_premium: boolean;
}

export interface MiniappEvent {
  id: number;
  name: string;
  description: string;
}

export interface MiniappSessionResponse {
  user: MiniappTelegramUser;
  event: MiniappEvent;
  has_my_gifts: boolean;
  my_result_participant_id?: number;
}

// Публичная запись лидерборда Mini App. Зеркалит backend
// MiniappLeaderboardEntryDTO — только публичные поля участника (без user_id,
// notes, has_gift, призов и даты регистрации).
export interface MiniappLeaderboardEntry {
  id: number;
  name: string;
  gender: Gender;
  bike_type: BikeType;
  status: ParticipantStatus;
  is_finished: boolean;
  place: number; // место в абсолютном зачёте (0 если нет)

  elapsed_time?: string; // полное время, ЧЧ:ММ:СС
  elapsed_time_sec?: number;
  moving_time?: string; // чистое время, ЧЧ:ММ:СС
  moving_time_sec?: number;
  idle_time?: string; // простой, ЧЧ:ММ:СС
  prev_elapsed_delta?: string; // прошлый год − общее время, плюс = быстрее
  prev_elapsed_delta_sec?: number;

  result_link?: string; // ссылка на результат (Strava)
  submitted_at?: string; // дата отправки результата (ISO 8601)

  ride_date?: string; // YYYY-MM-DD
  distance_meters?: number;
  avg_speed_kmh?: number;
  avg_moving_speed_kmh?: number;
  peak_speed_kmh?: number;
  avg_heart_rate?: number;
  max_heart_rate?: number;
  avg_cadence?: number;
  calories?: number;
}

export interface MiniappLeaderboardResponse {
  participants: MiniappLeaderboardEntry[];
  total: number;
}

export type PageSize = number | 'all';

export interface Nomination {
  id: number;
  event_id: number;
  name: string;
  description: string;
  gender_filter: GenderFilter;
  bike_type_filter: BikeTypeFilter;
  sort_order: number;
  is_active: boolean;
}

export interface NominationListResponse {
  nominations: Nomination[];
  total: number;
}

// Устаревшие типы для обратной совместимости (будут удалены)
export interface PrizeAssignment {
  id: number;
  participant_id: number;
  gift_id: number;
  comment?: string;
  assigned_at: string;
  gift?: Gift;
}

export interface PrizeAssignmentListResponse {
  prize_assignments: PrizeAssignment[];
  total: number;
}

export interface CreatePrizeAssignmentRequest {
  participant_id: number;
  gift_id: number;
  comment?: string;
}

export interface Stats {
  event_id: number;
  event_name: string;
  participants_count: number;
  finished_count: number;
  gifts_count: number;
  prizes_assigned_count: number;
  participants_with_prizes_count: number;
  by_gender: Record<string, number>;
  by_bike_type: Record<string, number>;
}

export interface StatsListResponse {
  stats: Stats[];
  total: number;
}

export interface DailyCount {
  date: string; // YYYY-MM-DD
  count: number;
}

export interface EventDailyStats {
  event_id: number;
  event_name: string;
  start_date: string | null;
  registrations: DailyCount[]; // новые участники по дате регистрации
  finishes: DailyCount[]; // проехавшие по дате отправки результата
}

export interface CreateEventRequest {
  name: string;
  description: string;
  participation_conditions?: string;
  active: boolean;
  stop_results?: boolean;
  stop_gifts?: boolean;
  start_date?: string;
  end_date?: string;
  gpx_file_path?: string;
  telegram_texts?: EventTelegramTexts;
}

export interface UpdateEventRequest {
  name?: string;
  description?: string;
  participation_conditions?: string;
  active?: boolean;
  stop_results?: boolean;
  stop_gifts?: boolean;
  start_date?: string;
  end_date?: string;
  gpx_file_path?: string;
  telegram_texts?: EventTelegramTexts;
}

export interface CreateNominationRequest {
  event_id: number;
  name: string;
  description: string;
  gender_filter: GenderFilter;
  bike_type_filter: BikeTypeFilter;
  sort_order: number;
  is_active: boolean;
}

export interface UpdateNominationRequest {
  name?: string;
  description?: string;
  gender_filter?: GenderFilter;
  bike_type_filter?: BikeTypeFilter;
  sort_order?: number;
  is_active?: boolean;
}

export interface UpdateParticipantRequest {
  bike_type?: BikeType;
  gender?: Gender;
  notes?: string;
  status?: ParticipantStatus;
  prev_elapsed_time_sec?: number; // ручное время прошлого года; 0 = удалить ручное значение
}

// LockStatus — состояние блокировки редактирования участника (in-memory лок на бэкенде).
// Возвращается эндпоинтами /api/participants/{id}/lock и приходит в теле 409-конфликта,
// когда другой администратор уже редактирует запись.
export interface LockStatus {
  participant_id: number;
  locked: boolean;
  locked_by_user_id?: number;
  locked_by_username?: string;
  acquired_at?: string;
  expires_at?: string;
  is_mine: boolean;
}

export interface Result {
  id: number;
  participant_id: number;
  result_link?: string;
  elapsed_time_sec?: number;
  moving_time_sec?: number;
  elapsed_time?: string; // формат ЧЧ:ММ:СС
  moving_time?: string; // формат ЧЧ:ММ:СС
  is_current: boolean;
  submitted_at: string;
  criteria?: Criteria[]; // критерии результата

  // Метрики заезда (вводятся вручную; опциональны)
  started_at?: string; // ISO 8601
  finished_at?: string; // ISO 8601
  distance_meters?: number;
  avg_heart_rate?: number;
  max_heart_rate?: number;
  peak_speed_kmh?: number;
  avg_cadence?: number;
  calories?: number;

  // Вычисляемые поля (только для чтения; считаются на сервере)
  ride_date?: string; // YYYY-MM-DD
  idle_time_sec?: number;
  idle_time?: string; // формат ЧЧ:ММ:СС
  avg_speed_kmh?: number;
  avg_moving_speed_kmh?: number;
  peak_avg_speed_delta_kmh?: number; // пиковая − средняя скорость, км/ч
  heart_rate_time_product?: number; // общее время (мин) × средний пульс
}

export interface ResultListResponse {
  results: Result[];
  total: number;
}

// Поля ввода метрик заезда (общие для создания и обновления результата).
// Общее время задаётся через elapsed_time_sec ИЛИ пару started_at+finished_at.
export interface ResultMetricsInput {
  elapsed_time_sec?: number;
  moving_time_sec?: number;
  started_at?: string; // ISO 8601 (зона Минск)
  finished_at?: string; // ISO 8601 (зона Минск)
  distance_meters?: number;
  avg_heart_rate?: number;
  max_heart_rate?: number;
  peak_speed_kmh?: number;
  avg_cadence?: number;
  calories?: number;
}

export interface CreateResultRequest extends ResultMetricsInput {
  result_link?: string;
}

export type UpdateResultRequest = ResultMetricsInput;

export interface CreateGiftRequest {
  user_id: number;
  description: string;
  gender_filter?: GenderFilter;
  bike_type_filter?: BikeTypeFilter;
}

export interface Criteria {
  id: number;
  name: string;
  description: string;
  criteria_type: CriteriaType;
  created_at: string;
}

export interface CriteriaListResponse {
  criteria: Criteria[];
  total: number;
  page: number;
  page_size: number;
}

export interface CreateCriteriaRequest {
  name: string;
  description: string;
  criteria_type: CriteriaType;
}

export interface UpdateCriteriaRequest {
  name?: string;
  description?: string;
  criteria_type?: CriteriaType;
}

export interface UpdateGiftRequest {
  description?: string;
  gender_filter?: GenderFilter;
  bike_type_filter?: BikeTypeFilter;
  review_status?: GiftReviewStatus;
  place?: number | null;
  place_rule?: GiftPlaceRule | null;
  criteria_ids?: number[];
  manual_distribution?: boolean;
  // Omitted preserves the existing recipient; null explicitly clears it.
  manual_recipient_participant_id?: number | null;
}

export interface PrizeDistribution {
  participant_id: number;
  participant_name: string;
  gender: string;
  bike_type: string;
  status: ParticipantStatus;
  place_absolute: number;
  place_by_gender: number;
  place_by_gender_bike: number;
  result_criteria: Criteria[];
  matched_gifts?: Gift[];
  matched_gift_assignments?: PrizeGiftAssignment[];
  match_reason: string; // "criteria", "place", "no_match"
}

export interface PrizeGiftAssignment {
  gift: Gift;
  gift_id: number;
  rule_type: 'none' | GiftPlaceRuleType;
  target_rank?: number;
  assigned_rank: number;
  is_fallback: boolean;
  fallback_reason?: string;
  match_reason: string;
}

export interface UnassignedPrizeSlot {
  gift_id: number;
  gift?: Gift;
  rule_type: GiftPlaceRuleType;
  target_rank?: number;
  reason: string;
  fallback_reason?: string;
}

export interface PrizeDistributionStats {
  total_participants: number;
  with_prizes: number;
  without_prizes: number;
  prize_slots: number;
}

export interface PrizeDistributionListResponse {
  distribution: PrizeDistribution[];
  unassigned_slots?: UnassignedPrizeSlot[];
  total: number;
  page?: number;
  page_size?: number;
  stats?: PrizeDistributionStats;
}

export interface ApiError {
  error: string;
  message?: string;
}
