import React, { useState } from 'react';
import { View, Text, StyleSheet, ActivityIndicator } from 'react-native';
import { WebView, WebViewMessageEvent } from 'react-native-webview';
import { colors } from '../theme/colors';
import { common } from '../theme/common';
import { TELEGRAM_BOT_USERNAME } from '../api/config';
import { useAuth } from '../context/AuthContext';

// Loads Telegram's official Login Widget script inside a WebView. The widget
// posts the signed auth payload (id, first_name, auth_date, hash, ...) via
// window.Telegram.WebviewProxy / postMessage, which we forward verbatim to
// backend's POST /auth/telegram for HMAC verification server-side.
const widgetHtml = `
<!DOCTYPE html>
<html>
<head>
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>
    body { background: ${colors.background}; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; }
  </style>
</head>
<body>
  <script async src="https://telegram.org/js/telegram-widget.js?22"
    data-telegram-login="${TELEGRAM_BOT_USERNAME}"
    data-size="large"
    data-radius="10"
    data-onauth="onTelegramAuth(user)"
    data-request-access="write"></script>
  <script>
    function onTelegramAuth(user) {
      window.ReactNativeWebView.postMessage(JSON.stringify(user));
    }
  </script>
</body>
</html>
`;

interface Props {
  onDone: () => void;
}

export function TelegramLoginScreen({ onDone }: Props) {
  const { loginWithTelegram } = useAuth();
  const [error, setError] = useState<string | null>(null);

  const handleMessage = async (event: WebViewMessageEvent) => {
    try {
      const payload = JSON.parse(event.nativeEvent.data);
      await loginWithTelegram(payload);
      onDone();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Не удалось войти через Telegram');
    }
  };

  return (
    <View style={common.screen}>
      <WebView
        source={{ html: widgetHtml }}
        onMessage={handleMessage}
        style={styles.webview}
        startInLoadingState
        renderLoading={() => (
          <View style={styles.loading}>
            <ActivityIndicator color={colors.accent} />
          </View>
        )}
      />
      {error && (
        <View style={styles.errorBar}>
          <Text style={common.errorText}>{error}</Text>
        </View>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  webview: { flex: 1, backgroundColor: colors.background },
  loading: { flex: 1, alignItems: 'center', justifyContent: 'center' },
  errorBar: { padding: 12 },
});
