// Типы для API

export type BikeType = 'gravel' | 'mtb' | 'road' | 'single_speed' | 'tandem';
export type Gender = 'male' | 'female';
export type GenderFilter = 'all' | 'male' | 'female';
export type BikeTypeFilter = 'all' | 'gravel' | 'mtb' | 'road' | 'single_speed' | 'tandem';
export type FileType = 'photo' | 'document';
export type CriteriaType = 'speed' | 'photo' | 'beer' | 'custom';

export interface User {
  id: number;
  username: string;
  role: string;
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
  result_link?: string;
  is_finished: boolean;
  elapsed_time?: string; // формат ЧЧ:ММ:СС
  moving_time?: string; // формат ЧЧ:ММ:СС
  elapsed_time_sec?: number;
  moving_time_sec?: number;
  notes?: string;
  registered_at: string;
  finished_at?: string;
  place?: number; // место в зачёте (0 если нет) - устаревшее, используйте place_absolute
  place_absolute?: number; // место в абсолютном зачёте
  place_by_gender?: number; // место в зачёте по гендеру
  place_by_gender_bike?: number; // место в зачёте по гендеру+тип велосипеда
  has_gift: boolean;
  prizes_count: number;
  matched_gifts?: Gift[]; // все подобранные подарки
}

export interface ParticipantDetail extends Participant {
  gifts: Gift[];
}

export interface ParticipantListResponse {
  participants: Participant[];
  total: number;
}

export interface Event {
  id: number;
  name: string;
  description: string;
  active: boolean;
  start_date?: string;
  end_date?: string;
  gpx_file_path?: string;
  created_at: string;
  updated_at: string;
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
  place?: number;
  attachments?: GiftAttachment[];
  criteria?: Criteria[];
  created_at: string;
}

export interface GiftListResponse {
  gifts: Gift[];
  total: number;
}

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
  by_gender: Record<string, number>;
  by_bike_type: Record<string, number>;
}

export interface StatsListResponse {
  stats: Stats[];
  total: number;
}

export interface CreateEventRequest {
  name: string;
  description: string;
  active: boolean;
  start_date?: string;
  end_date?: string;
  gpx_file_path?: string;
}

export interface UpdateEventRequest {
  name?: string;
  description?: string;
  active?: boolean;
  start_date?: string;
  end_date?: string;
  gpx_file_path?: string;
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
}

export interface ResultListResponse {
  results: Result[];
  total: number;
}

export interface CreateResultRequest {
  result_link: string;
}

export interface UpdateResultRequest {
  elapsed_time_sec?: number;
  moving_time_sec?: number;
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

export interface CreateGiftRequest {
  user_id: number;
  description: string;
  attachments?: GiftAttachmentRequest[];
}

export interface UpdateGiftRequest {
  description?: string;
  gender_filter?: GenderFilter;
  bike_type_filter?: BikeTypeFilter;
  place?: number | null;
  criteria_ids?: number[];
}

export interface GiftAttachmentRequest {
  telegram_file_id: string;
  file_type: FileType;
}

export interface PrizeDistribution {
  participant_id: number;
  participant_name: string;
  gender: string;
  bike_type: string;
  place_absolute: number;
  place_by_gender: number;
  result_criteria: Criteria[];
  matched_gifts?: Gift[];
  match_reason: string; // "criteria", "place", "no_match"
}

export interface PrizeDistributionListResponse {
  distribution: PrizeDistribution[];
  total: number;
}

export interface ApiError {
  error: string;
  message?: string;
}
