import React, { useState } from 'react';
import { View, ActivityIndicator } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import { AuthProvider, useAuth } from '../context/AuthContext';
import { LoginScreen } from '../screens/LoginScreen';
import { TelegramLoginScreen } from '../screens/TelegramLoginScreen';
import { MainTabs } from './MainTabs';
import { colors } from '../theme/colors';
import { common } from '../theme/common';

function RootContent() {
  const { isAuthenticated, isLoading } = useAuth();
  const [showTelegramLogin, setShowTelegramLogin] = useState(false);

  if (isLoading) {
    return (
      <View style={[common.screen, { alignItems: 'center', justifyContent: 'center' }]}>
        <ActivityIndicator color={colors.accent} />
      </View>
    );
  }

  if (!isAuthenticated) {
    if (showTelegramLogin) {
      return <TelegramLoginScreen onDone={() => setShowTelegramLogin(false)} />;
    }
    return <LoginScreen onOpenTelegramLogin={() => setShowTelegramLogin(true)} />;
  }

  return <MainTabs />;
}

export function RootNavigator() {
  return (
    <AuthProvider>
      <StatusBar style="light" />
      <RootContent />
    </AuthProvider>
  );
}
