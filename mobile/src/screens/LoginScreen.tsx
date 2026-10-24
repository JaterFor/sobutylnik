import React, { useState } from 'react';
import { View, Text, TextInput, TouchableOpacity, StyleSheet } from 'react-native';
import { common } from '../theme/common';
import { colors } from '../theme/colors';
import { useAuth } from '../context/AuthContext';
import { ApiError } from '../api/client';

type Mode = 'login' | 'register';

export function LoginScreen({ onOpenTelegramLogin }: { onOpenTelegramLogin: () => void }) {
  const { loginWithEmail, registerWithEmail } = useAuth();
  const [mode, setMode] = useState<Mode>('login');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async () => {
    setError(null);
    setSubmitting(true);
    try {
      if (mode === 'login') {
        await loginWithEmail(email.trim(), password);
      } else {
        await registerWithEmail(email.trim(), password);
      }
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Что-то пошло не так, попробуйте ещё раз');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <View style={common.screen}>
      <View style={common.content}>
        <Text style={common.title}>Собутыльник</Text>

        <TouchableOpacity style={styles.telegramButton} onPress={onOpenTelegramLogin}>
          <Text style={styles.telegramButtonText}>Войти через Telegram</Text>
        </TouchableOpacity>

        <View style={styles.divider}>
          <View style={styles.dividerLine} />
          <Text style={styles.dividerText}>или</Text>
          <View style={styles.dividerLine} />
        </View>

        <Text style={common.label}>Email</Text>
        <TextInput
          style={common.input}
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          keyboardType="email-address"
          placeholder="you@example.com"
          placeholderTextColor={colors.textMuted}
        />

        <Text style={common.label}>Пароль</Text>
        <TextInput
          style={common.input}
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          placeholder="Минимум 8 символов"
          placeholderTextColor={colors.textMuted}
        />

        {error && <Text style={common.errorText}>{error}</Text>}

        <TouchableOpacity style={common.button} onPress={handleSubmit} disabled={submitting}>
          <Text style={common.buttonText}>{mode === 'login' ? 'Войти' : 'Зарегистрироваться'}</Text>
        </TouchableOpacity>

        <Text style={common.link} onPress={() => setMode(mode === 'login' ? 'register' : 'login')}>
          {mode === 'login' ? 'Нет аккаунта? Зарегистрироваться' : 'Уже есть аккаунт? Войти'}
        </Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  telegramButton: {
    backgroundColor: '#2AABEE',
    borderRadius: 10,
    paddingVertical: 14,
    alignItems: 'center',
  },
  telegramButtonText: { color: '#FFFFFF', fontSize: 16, fontWeight: '600' },
  divider: { flexDirection: 'row', alignItems: 'center', marginVertical: 20 },
  dividerLine: { flex: 1, height: 1, backgroundColor: colors.border },
  dividerText: { color: colors.textMuted, marginHorizontal: 10 },
});
