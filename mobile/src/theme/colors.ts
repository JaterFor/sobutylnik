// Dark theme only on MVP (see design.md: "Тёмная тема по умолчанию").
// No light-theme variant or system-theme detection is wired in.
export const colors = {
  background: '#121212',
  surface: '#1E1E1E',
  surfaceAlt: '#2A2A2A',
  border: '#333333',
  textPrimary: '#F5F5F5',
  textSecondary: '#B0B0B0',
  textMuted: '#777777',
  accent: '#E85D4A',
  accentMuted: '#7A3328',
  danger: '#E5484D',
  success: '#4CAF80',
  overlay: 'rgba(0,0,0,0.6)',
};

export type Colors = typeof colors;
