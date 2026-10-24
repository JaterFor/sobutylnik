import { api } from './client';
import { MatchSummary, ChatMessage } from './types';

export function getMatches() {
  return api.get<MatchSummary[]>('/matches/');
}

export function unmatch(matchId: string) {
  return api.post<void>(`/matches/${matchId}/unmatch`);
}

export function getChatHistory(matchId: string) {
  return api.get<ChatMessage[]>(`/chat/${matchId}/messages`);
}

export function sendMessage(matchId: string, body: string) {
  return api.post<ChatMessage>(`/chat/${matchId}/messages`, { body });
}
