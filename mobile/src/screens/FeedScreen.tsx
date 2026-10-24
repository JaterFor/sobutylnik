import React, { useEffect, useState, useCallback, useRef } from 'react';
import { View, Text, TouchableOpacity, StyleSheet, ActivityIndicator, Animated, PanResponder } from 'react-native';
import { getFeed, swipe, FeedFilter } from '../api/feed';
import { reportUser, blockUser } from '../api/moderation';
import { Candidate } from '../api/types';
import { ApiError } from '../api/client';
import { common } from '../theme/common';
import { colors } from '../theme/colors';

const SWIPE_THRESHOLD = 120;

interface Props {
  onMatch: (matchId: string) => void;
}

export function FeedScreen({ onMatch }: Props) {
  const [candidates, setCandidates] = useState<Candidate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<FeedFilter>({ radiusKm: 5 });
  const position = useRef(new Animated.ValueXY()).current;

  const load = useCallback(async (f: FeedFilter) => {
    setLoading(true);
    setError(null);
    try {
      const data = await getFeed(f);
      setCandidates(data);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Не удалось загрузить ленту');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load(filter);
  }, [filter, load]);

  const current = candidates[0];

  const handleDecision = useCallback(
    async (liked: boolean) => {
      if (!current) return;
      const target = current;
      setCandidates((prev) => prev.slice(1));
      position.setValue({ x: 0, y: 0 });
      try {
        const result = await swipe(target.user_id, liked);
        if (result.matched && result.match_id) {
          onMatch(result.match_id);
        }
      } catch (e) {
        if (!(e instanceof ApiError && e.status === 409)) {
          setError('Не удалось сохранить решение, попробуйте ещё раз');
        }
      }
    },
    [current, onMatch, position]
  );

  const panResponder = useRef(
    PanResponder.create({
      onStartShouldSetPanResponder: () => true,
      onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
      onPanResponderRelease: (_, gesture) => {
        if (gesture.dx > SWIPE_THRESHOLD) {
          Animated.timing(position, { toValue: { x: 500, y: gesture.dy }, duration: 200, useNativeDriver: false }).start(
            () => handleDecision(true)
          );
        } else if (gesture.dx < -SWIPE_THRESHOLD) {
          Animated.timing(position, { toValue: { x: -500, y: gesture.dy }, duration: 200, useNativeDriver: false }).start(
            () => handleDecision(false)
          );
        } else {
          Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
        }
      },
    })
  ).current;

  const handleReport = async () => {
    if (!current) return;
    try {
      await reportUser(current.user_id, 'inappropriate_behavior');
    } catch {
      // best-effort; errors are not surfaced for moderation actions on MVP
    }
  };

  const handleBlock = async () => {
    if (!current) return;
    const target = current;
    setCandidates((prev) => prev.slice(1));
    try {
      await blockUser(target.user_id);
    } catch {
      // best-effort
    }
  };

  const rotate = position.x.interpolate({
    inputRange: [-200, 0, 200],
    outputRange: ['-15deg', '0deg', '15deg'],
  });

  if (loading) {
    return (
      <View style={[common.screen, styles.centered]}>
        <ActivityIndicator color={colors.accent} />
      </View>
    );
  }

  return (
    <View style={common.screen}>
      <View style={styles.filterRow}>
        {[2, 5, 15, 50].map((r) => (
          <TouchableOpacity
            key={r}
            style={[styles.filterChip, filter.radiusKm === r && styles.filterChipActive]}
            onPress={() => setFilter({ ...filter, radiusKm: r })}
          >
            <Text style={filter.radiusKm === r ? styles.filterChipTextActive : styles.filterChipText}>{r} км</Text>
          </TouchableOpacity>
        ))}
      </View>

      <View style={styles.cardArea}>
        {error && <Text style={common.errorText}>{error}</Text>}
        {!current && !error && (
          <Text style={styles.emptyText}>Никого не найдено рядом. Попробуйте увеличить радиус.</Text>
        )}
        {current && (
          <Animated.View
            {...panResponder.panHandlers}
            style={[styles.card, { transform: [...position.getTranslateTransform(), { rotate }] }]}
          >
            <View style={styles.cardImagePlaceholder}>
              {current.photos[0] ? (
                <Animated.Image source={{ uri: current.photos[0] }} style={styles.cardImage} />
              ) : (
                <Text style={styles.cardImagePlaceholderText}>Нет фото</Text>
              )}
            </View>
            <View style={styles.cardInfo}>
              <Text style={styles.cardName}>
                {current.name}, {current.age}
              </Text>
              <Text style={styles.cardMeta}>
                {current.distance_km.toFixed(1)} км · {readinessLabel(current.readiness_status)}
              </Text>
              {!!current.bio && <Text style={styles.cardBio}>{current.bio}</Text>}
            </View>
          </Animated.View>
        )}
      </View>

      {current && (
        <View style={styles.actionsRow}>
          <TouchableOpacity style={[styles.actionButton, styles.dislikeButton]} onPress={() => handleDecision(false)}>
            <Text style={styles.actionButtonText}>✕</Text>
          </TouchableOpacity>
          <TouchableOpacity style={styles.moderationButton} onPress={handleReport}>
            <Text style={styles.moderationButtonText}>Жалоба</Text>
          </TouchableOpacity>
          <TouchableOpacity style={styles.moderationButton} onPress={handleBlock}>
            <Text style={styles.moderationButtonText}>Блок</Text>
          </TouchableOpacity>
          <TouchableOpacity style={[styles.actionButton, styles.likeButton]} onPress={() => handleDecision(true)}>
            <Text style={styles.actionButtonText}>♥</Text>
          </TouchableOpacity>
        </View>
      )}
    </View>
  );
}

function readinessLabel(status: string): string {
  switch (status) {
    case 'now':
      return 'Готов сейчас';
    case 'today':
      return 'Сегодня';
    case 'this_week':
      return 'На этой неделе';
    default:
      return status;
  }
}

const styles = StyleSheet.create({
  centered: { alignItems: 'center', justifyContent: 'center' },
  filterRow: { flexDirection: 'row', gap: 8, paddingHorizontal: 20, paddingTop: 16 },
  filterChip: {
    paddingHorizontal: 14,
    paddingVertical: 8,
    borderRadius: 16,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border,
  },
  filterChipActive: { backgroundColor: colors.accentMuted, borderColor: colors.accent },
  filterChipText: { color: colors.textSecondary, fontSize: 13 },
  filterChipTextActive: { color: colors.textPrimary, fontSize: 13, fontWeight: '600' },
  cardArea: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 20 },
  emptyText: { color: colors.textSecondary, textAlign: 'center', fontSize: 16 },
  card: {
    width: '100%',
    height: '85%',
    backgroundColor: colors.surface,
    borderRadius: 16,
    overflow: 'hidden',
  },
  cardImagePlaceholder: { flex: 3, backgroundColor: colors.surfaceAlt, alignItems: 'center', justifyContent: 'center' },
  cardImage: { width: '100%', height: '100%' },
  cardImagePlaceholderText: { color: colors.textMuted },
  cardInfo: { flex: 1, padding: 16 },
  cardName: { color: colors.textPrimary, fontSize: 22, fontWeight: '700' },
  cardMeta: { color: colors.textSecondary, fontSize: 14, marginTop: 4 },
  cardBio: { color: colors.textSecondary, fontSize: 14, marginTop: 8 },
  actionsRow: {
    flexDirection: 'row',
    justifyContent: 'space-evenly',
    alignItems: 'center',
    paddingVertical: 20,
  },
  actionButton: {
    width: 56,
    height: 56,
    borderRadius: 28,
    alignItems: 'center',
    justifyContent: 'center',
  },
  dislikeButton: { backgroundColor: colors.surfaceAlt, borderWidth: 1, borderColor: colors.border },
  likeButton: { backgroundColor: colors.accent },
  actionButtonText: { color: colors.textPrimary, fontSize: 24 },
  moderationButton: { paddingHorizontal: 10, paddingVertical: 8 },
  moderationButtonText: { color: colors.textMuted, fontSize: 12 },
});
