# sobutylnik mobile

React Native + Expo client for the Sobutylnik MVP. See `/openspec/changes/mvp-core/` for specs and design.

## Running

```bash
npm install
npm run web      # browser (no emulator available in this environment, but useful for quick checks)
npm run android  # requires Android emulator or device
npm run ios      # requires macOS + Xcode
```

The app talks to the deployed backend at `http://195.209.221.226` (see `src/api/config.ts`) — no local backend needed for client-only testing, but `/profile/photos/presign` uploads will fail until real Cloudflare R2 credentials are set on the server, and Telegram login needs a real bot (`TELEGRAM_BOT_USERNAME` in `src/api/config.ts`, `TELEGRAM_BOT_TOKEN` in the server's `.env`).

## Structure

- `src/api/` — typed HTTP client + per-capability request functions
- `src/context/AuthContext.tsx` — auth state, token refresh
- `src/storage/` — SecureStore token persistence
- `src/screens/` — one file per screen
- `src/navigation/` — root auth/main switch (`RootNavigator`) + the main tab-like screen (`MainTabs`)
- `src/theme/` — dark-only theme (see design.md: "Тёмная тема по умолчанию")
- `src/hooks/useChatSocket.ts` — WebSocket client for realtime chat

## Known gaps (MVP scope)

- Telegram Login Widget requires a real bot username/token — currently a placeholder on both client and server.
- Photo upload requires real Cloudflare R2 credentials on the server.
- No TLS yet (backend has no domain) — the app talks to the VPS over plain HTTP.
- UI was verified via TypeScript compilation and a successful Expo web bundle build, not by interactively running the app — this environment has no device/emulator or browser screenshot tooling.
