export interface TokenPair {
  access_token: string;
  refresh_token: string;
}

export interface ProfilePhoto {
  id: string;
  url: string;
  is_primary: boolean;
}

export interface Profile {
  user_id: string;
  name: string;
  age: number;
  gender: 'male' | 'female' | 'other';
  bio: string;
  preferred_drink?: string;
  readiness_status: 'now' | 'today' | 'this_week';
  has_location: boolean;
  photos: ProfilePhoto[];
}

export interface Candidate {
  user_id: string;
  name: string;
  age: number;
  gender: string;
  bio: string;
  readiness_status: string;
  distance_km: number;
  photos: string[];
}

export interface SwipeResult {
  matched: boolean;
  match_id?: string;
}

export interface MatchSummary {
  id: string;
  other_user_id: string;
  other_user_name: string;
  other_user_photo?: string;
  last_activity_at: string;
}

export interface ChatMessage {
  id: string;
  sender_id: string;
  body: string;
  read_at?: string;
  created_at: string;
}
