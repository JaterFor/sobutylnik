import React, { useEffect, useState, useCallback } from 'react';
import { View, Text, FlatList, TouchableOpacity, StyleSheet, ActivityIndicator, Image } from 'react-native';
import { getMatches } from '../api/matches';
import { MatchSummary } from '../api/types';
import { common } from '../theme/common';
import { colors } from '../theme/colors';

interface Props {
  onOpenChat: (matchId: string) => void;
}

export function MatchesScreen({ onOpenChat }: Props) {
  const [matches, setMatches] = useState<MatchSummary[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setMatches(await getMatches());
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  if (loading) {
    return (
      <View style={[common.screen, styles.centered]}>
        <ActivityIndicator color={colors.accent} />
      </View>
    );
  }

  return (
    <View style={common.screen}>
      <View style={common.content}>
        <Text style={common.title}>Мэтчи</Text>
        {matches.length === 0 ? (
          <Text style={styles.emptyText}>Пока нет мэтчей. Начните свайпать в ленте!</Text>
        ) : (
          <FlatList
            data={matches}
            keyExtractor={(m) => m.id}
            onRefresh={load}
            refreshing={loading}
            renderItem={({ item }) => (
              <TouchableOpacity style={styles.row} onPress={() => onOpenChat(item.id)}>
                {item.other_user_photo ? (
                  <Image source={{ uri: item.other_user_photo }} style={styles.avatar} />
                ) : (
                  <View style={styles.avatar} />
                )}
                <View style={styles.rowInfo}>
                  <Text style={styles.rowTitle}>{item.other_user_name}</Text>
                  <Text style={styles.rowSubtitle}>Открыть чат</Text>
                </View>
              </TouchableOpacity>
            )}
          />
        )}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  centered: { alignItems: 'center', justifyContent: 'center' },
  emptyText: { color: colors.textSecondary, fontSize: 15 },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 14,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
  },
  avatar: { width: 50, height: 50, borderRadius: 25, backgroundColor: colors.surfaceAlt, marginRight: 14 },
  rowInfo: { flex: 1 },
  rowTitle: { color: colors.textPrimary, fontSize: 16, fontWeight: '600' },
  rowSubtitle: { color: colors.textSecondary, fontSize: 13, marginTop: 2 },
});
