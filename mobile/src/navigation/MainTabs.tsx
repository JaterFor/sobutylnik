import React, { useState, useEffect } from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
import { FeedScreen } from '../screens/FeedScreen';
import { MatchesScreen } from '../screens/MatchesScreen';
import { ProfileEditScreen } from '../screens/ProfileEditScreen';
import { ChatScreen } from '../screens/ChatScreen';
import { getMatches } from '../api/matches';
import { colors } from '../theme/colors';
import { common } from '../theme/common';
import { registerForPushNotifications } from '../notifications/registerPush';

type Tab = 'feed' | 'matches' | 'profile';

export function MainTabs() {
  const [tab, setTab] = useState<Tab>('feed');
  const [openChat, setOpenChat] = useState<{ matchId: string; otherUserId: string } | null>(null);

  useEffect(() => {
    registerForPushNotifications().catch(() => {
      // best-effort — see notifications spec: missing permission is not an error
    });
  }, []);

  const openChatByMatchId = async (matchId: string) => {
    const matches = await getMatches();
    const match = matches.find((m) => m.id === matchId);
    if (match) {
      setOpenChat({ matchId, otherUserId: match.other_user_id });
    }
  };

  if (openChat) {
    return (
      <ChatScreen
        matchId={openChat.matchId}
        otherUserId={openChat.otherUserId}
        onUnmatched={() => setOpenChat(null)}
      />
    );
  }

  return (
    <View style={common.screen}>
      <View style={styles.content}>
        {tab === 'feed' && <FeedScreen onMatch={(matchId) => openChatByMatchId(matchId)} />}
        {tab === 'matches' && <MatchesScreen onOpenChat={(matchId) => openChatByMatchId(matchId)} />}
        {tab === 'profile' && <ProfileEditScreen onSaved={() => setTab('feed')} />}
      </View>

      <View style={styles.tabBar}>
        <TabButton label="Лента" active={tab === 'feed'} onPress={() => setTab('feed')} />
        <TabButton label="Мэтчи" active={tab === 'matches'} onPress={() => setTab('matches')} />
        <TabButton label="Профиль" active={tab === 'profile'} onPress={() => setTab('profile')} />
      </View>
    </View>
  );
}

function TabButton({ label, active, onPress }: { label: string; active: boolean; onPress: () => void }) {
  return (
    <TouchableOpacity style={styles.tabButton} onPress={onPress}>
      <Text style={active ? styles.tabLabelActive : styles.tabLabel}>{label}</Text>
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  content: { flex: 1 },
  tabBar: {
    flexDirection: 'row',
    borderTopWidth: 1,
    borderTopColor: colors.border,
    backgroundColor: colors.surface,
    paddingVertical: 10,
  },
  tabButton: { flex: 1, alignItems: 'center' },
  tabLabel: { color: colors.textMuted, fontSize: 13 },
  tabLabelActive: { color: colors.accent, fontSize: 13, fontWeight: '700' },
});
