import * as Notifications from 'expo-notifications';
import * as Device from 'expo-device';
import { registerPushToken } from '../api/notifications';

Notifications.setNotificationHandler({
  handleNotification: async () => ({
    shouldShowAlert: true,
    shouldPlaySound: false,
    shouldSetBadge: false,
    shouldShowBanner: true,
    shouldShowList: true,
  }),
});

// Requests permission and registers this device's Expo push token with the
// backend. Silently no-ops on simulators/emulators and when permission is
// denied (notifications spec: absence of permission is not an error).
export async function registerForPushNotifications(): Promise<void> {
  if (!Device.isDevice) return;

  const existing = await Notifications.getPermissionsAsync();
  let status = existing.status;
  if (status !== 'granted') {
    const requested = await Notifications.requestPermissionsAsync();
    status = requested.status;
  }
  if (status !== 'granted') return;

  const tokenResponse = await Notifications.getExpoPushTokenAsync();
  await registerPushToken(tokenResponse.data);
}
