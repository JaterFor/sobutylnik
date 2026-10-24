import React, { useEffect, useState, useCallback, useRef } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  FlatList,
  StyleSheet,
  ActivityIndicator,
  KeyboardAvoidingView,
  Platform,
} from 'react-native';
import { getChatHistory, sendMessage, unmatch } from '../api/matches';
import { reportUser, blockUser } from '../api/moderation';
import { ChatMessage } from '../api/types';
import { useChatSocket } from '../hooks/useChatSocket';
import { getAccessToken } from '../storage/tokens';
import { decodeUserIdFromToken } from '../storage/jwt';
import { common } from '../theme/common';
import { colors } from '../theme/colors';
import { ApiError } from '../api/client';

interface Props {
  matchId: string;
  otherUserId: string;
  onUnmatched: () => void;
}

export function ChatScreen({ matchId, otherUserId, onUnmatched }: Props) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [myUserId, setMyUserId] = useState<string | null>(null);
  const [draft, setDraft] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const listRef = useRef<FlatList>(null);

  const load = useCallback(async () => {
    try {
      const history = await getChatHistory(matchId);
      setMessages(history);
    } catch (e) {
      if (e instanceof ApiError && e.status === 403) {
        onUnmatched();
      } else {
        setError('Не удалось загрузить историю');
      }
    } finally {
      setLoading(false);
    }
  }, [matchId, onUnmatched]);

  useEffect(() => {
    getAccessToken().then((token) => {
      if (token) setMyUserId(decodeUserIdFromToken(token));
    });
    load();
  }, [load]);

  useChatSocket({
    onNewMessage: (msg) => {
      setMessages((prev) => [...prev, msg]);
    },
    onMessagesRead: (messageIds) => {
      setMessages((prev) =>
        prev.map((m) => (messageIds.includes(m.id) ? { ...m, read_at: new Date().toISOString() } : m))
      );
    },
  });

  const handleSend = async () => {
    const body = draft.trim();
    if (!body) return;
    setDraft('');
    try {
      const sent = await sendMessage(matchId, body);
      setMessages((prev) => [...prev, sent]);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Не удалось отправить сообщение');
    }
  };

  const handleUnmatch = async () => {
    try {
      await unmatch(matchId);
      onUnmatched();
    } catch {
      setError('Не удалось разорвать мэтч');
    }
  };

  const handleReport = async () => {
    try {
      await reportUser(otherUserId, 'inappropriate_behavior');
    } catch {
      setError('Не удалось отправить жалобу');
    }
  };

  const handleBlock = async () => {
    try {
      await blockUser(otherUserId);
      onUnmatched();
    } catch {
      setError('Не удалось заблокировать пользователя');
    }
  };

  if (loading) {
    return (
      <View style={[common.screen, styles.centered]}>
        <ActivityIndicator color={colors.accent} />
      </View>
    );
  }

  return (
    <KeyboardAvoidingView
      style={common.screen}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      keyboardVerticalOffset={80}
    >
      <View style={styles.headerActions}>
        <TouchableOpacity style={styles.headerButton} onPress={handleReport}>
          <Text style={styles.headerButtonText}>Жалоба</Text>
        </TouchableOpacity>
        <TouchableOpacity style={styles.headerButton} onPress={handleBlock}>
          <Text style={styles.headerButtonText}>Блокировать</Text>
        </TouchableOpacity>
        <TouchableOpacity style={styles.headerButton} onPress={handleUnmatch}>
          <Text style={styles.unmatchButtonText}>Разорвать мэтч</Text>
        </TouchableOpacity>
      </View>

      <FlatList
        ref={listRef}
        data={messages}
        keyExtractor={(m) => m.id}
        contentContainerStyle={styles.list}
        onContentSizeChange={() => listRef.current?.scrollToEnd({ animated: true })}
        renderItem={({ item }) => {
          const isMine = item.sender_id === myUserId;
          return (
            <View style={[styles.bubble, isMine ? styles.bubbleMine : styles.bubbleTheirs]}>
              <Text style={styles.bubbleText}>{item.body}</Text>
              {isMine && <Text style={styles.readStatus}>{item.read_at ? 'прочитано' : 'отправлено'}</Text>}
            </View>
          );
        }}
      />

      {error && <Text style={[common.errorText, styles.errorMargin]}>{error}</Text>}

      <View style={styles.inputRow}>
        <TextInput
          style={styles.input}
          value={draft}
          onChangeText={setDraft}
          placeholder="Сообщение..."
          placeholderTextColor={colors.textMuted}
          multiline
        />
        <TouchableOpacity style={styles.sendButton} onPress={handleSend}>
          <Text style={styles.sendButtonText}>➤</Text>
        </TouchableOpacity>
      </View>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  centered: { alignItems: 'center', justifyContent: 'center' },
  headerActions: { flexDirection: 'row', justifyContent: 'flex-end', gap: 16, padding: 12 },
  headerButton: {},
  headerButtonText: { color: colors.textMuted, fontSize: 13 },
  unmatchButtonText: { color: colors.danger, fontSize: 13 },
  list: { padding: 16, gap: 8 },
  bubble: { maxWidth: '75%', borderRadius: 14, paddingHorizontal: 14, paddingVertical: 10, marginBottom: 8 },
  bubbleMine: { backgroundColor: colors.accent, alignSelf: 'flex-end' },
  bubbleTheirs: { backgroundColor: colors.surface, alignSelf: 'flex-start' },
  bubbleText: { color: colors.textPrimary, fontSize: 15 },
  readStatus: { color: 'rgba(255,255,255,0.7)', fontSize: 10, marginTop: 4, textAlign: 'right' },
  errorMargin: { marginHorizontal: 16 },
  inputRow: { flexDirection: 'row', padding: 12, gap: 8, alignItems: 'flex-end' },
  input: {
    flex: 1,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 20,
    paddingHorizontal: 16,
    paddingVertical: 10,
    color: colors.textPrimary,
    maxHeight: 100,
  },
  sendButton: {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: colors.accent,
    alignItems: 'center',
    justifyContent: 'center',
  },
  sendButtonText: { color: '#FFFFFF', fontSize: 18 },
});
