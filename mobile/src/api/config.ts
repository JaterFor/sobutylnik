// No domain yet (TLS still pending, see design.md), so the client talks to
// the VPS directly over HTTP. Swap for https://<domain> once DNS + certbot
// are set up.
export const API_BASE_URL = 'http://195.209.221.226';
export const WS_BASE_URL = 'ws://195.209.221.226';

// TODO: set the real bot username once the Telegram bot is created
// (matches backend's TELEGRAM_BOT_TOKEN, currently a placeholder too).
export const TELEGRAM_BOT_USERNAME = 'sobutylnik_bot';
