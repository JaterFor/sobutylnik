import React, { useEffect, useState, useCallback } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  ScrollView,
  Image,
  StyleSheet,
  ActivityIndicator,
} from 'react-native';
import * as ImagePicker from 'expo-image-picker';
import * as Location from 'expo-location';
import { common } from '../theme/common';
import { colors } from '../theme/colors';
import { getProfile, upsertProfile, uploadPhoto, deletePhoto, updateLocation } from '../api/profile';
import { Profile } from '../api/types';
import { ApiError } from '../api/client';

const GENDERS: Array<{ value: Profile['gender']; label: string }> = [
  { value: 'male', label: 'Мужчина' },
  { value: 'female', label: 'Женщина' },
  { value: 'other', label: 'Другое' },
];

const READINESS: Array<{ value: Profile['readiness_status']; label: string }> = [
  { value: 'now', label: 'Готов сейчас' },
  { value: 'today', label: 'Сегодня' },
  { value: 'this_week', label: 'На этой неделе' },
];

export function ProfileEditScreen({ onSaved }: { onSaved?: () => void }) {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [name, setName] = useState('');
  const [age, setAge] = useState('');
  const [gender, setGender] = useState<Profile['gender']>('male');
  const [bio, setBio] = useState('');
  const [readiness, setReadiness] = useState<Profile['readiness_status']>('now');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [locationStatus, setLocationStatus] = useState<'unknown' | 'granted' | 'denied' | 'set'>('unknown');

  const load = useCallback(async () => {
    try {
      const p = await getProfile();
      setProfile(p);
      setName(p.name);
      setAge(String(p.age));
      setGender(p.gender);
      setBio(p.bio);
      setReadiness(p.readiness_status);
      setLocationStatus(p.has_location ? 'set' : 'unknown');
    } catch (e) {
      if (!(e instanceof ApiError && e.status === 404)) {
        setError('Не удалось загрузить профиль');
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const handlePickPhoto = async () => {
    const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!permission.granted) {
      setError('Нужен доступ к галерее, чтобы добавить фото');
      return;
    }
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ImagePicker.MediaTypeOptions.Images,
      quality: 0.8,
    });
    if (result.canceled || !result.assets?.[0]) return;

    const asset = result.assets[0];
    setUploading(true);
    setError(null);
    try {
      await uploadPhoto(asset.uri, asset.mimeType ?? 'image/jpeg');
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Не удалось загрузить фото');
    } finally {
      setUploading(false);
    }
  };

  const handleDeletePhoto = async (photoId: string) => {
    setError(null);
    try {
      await deletePhoto(photoId);
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Не удалось удалить фото');
    }
  };

  const handleRequestLocation = async () => {
    const permission = await Location.requestForegroundPermissionsAsync();
    if (!permission.granted) {
      setLocationStatus('denied');
      setError('Без доступа к геопозиции профиль не появится в ленте других пользователей');
      return;
    }
    const position = await Location.getCurrentPositionAsync({});
    await updateLocation(position.coords.latitude, position.coords.longitude);
    setLocationStatus('set');
  };

  const handleSave = async () => {
    setError(null);
    const ageNum = parseInt(age, 10);
    if (!name.trim()) {
      setError('Укажите имя');
      return;
    }
    if (!ageNum || ageNum < 18) {
      setError('Возраст должен быть не меньше 18 лет');
      return;
    }
    if (!profile?.photos.length) {
      setError('Добавьте хотя бы одну фотографию');
      return;
    }
    setSaving(true);
    try {
      await upsertProfile({ name: name.trim(), age: ageNum, gender, bio, readiness_status: readiness });
      onSaved?.();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Не удалось сохранить профиль');
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <View style={[common.screen, styles.centered]}>
        <ActivityIndicator color={colors.accent} />
      </View>
    );
  }

  return (
    <ScrollView style={common.screen} contentContainerStyle={common.content}>
      <Text style={common.title}>Мой профиль</Text>

      <Text style={common.label}>Фото (до 3, долгое нажатие — удалить)</Text>
      <View style={styles.photoRow}>
        {(profile?.photos ?? []).map((photo) => (
          <TouchableOpacity key={photo.id} onLongPress={() => handleDeletePhoto(photo.id)}>
            <Image source={{ uri: photo.url }} style={styles.photo} />
          </TouchableOpacity>
        ))}
        {(profile?.photos?.length ?? 0) < 3 && (
          <TouchableOpacity style={styles.addPhoto} onPress={handlePickPhoto} disabled={uploading}>
            {uploading ? <ActivityIndicator color={colors.accent} /> : <Text style={styles.addPhotoText}>+</Text>}
          </TouchableOpacity>
        )}
      </View>

      <Text style={common.label}>Имя</Text>
      <TextInput style={common.input} value={name} onChangeText={setName} placeholderTextColor={colors.textMuted} />

      <Text style={common.label}>Возраст</Text>
      <TextInput
        style={common.input}
        value={age}
        onChangeText={setAge}
        keyboardType="number-pad"
        placeholderTextColor={colors.textMuted}
      />

      <Text style={common.label}>Пол</Text>
      <View style={styles.optionRow}>
        {GENDERS.map((g) => (
          <TouchableOpacity
            key={g.value}
            style={[styles.option, gender === g.value && styles.optionSelected]}
            onPress={() => setGender(g.value)}
          >
            <Text style={gender === g.value ? styles.optionTextSelected : styles.optionText}>{g.label}</Text>
          </TouchableOpacity>
        ))}
      </View>

      <Text style={common.label}>О себе</Text>
      <TextInput
        style={[common.input, styles.bioInput]}
        value={bio}
        onChangeText={setBio}
        multiline
        maxLength={500}
        placeholderTextColor={colors.textMuted}
      />

      <Text style={common.label}>Готовность встретиться</Text>
      <View style={styles.optionRow}>
        {READINESS.map((r) => (
          <TouchableOpacity
            key={r.value}
            style={[styles.option, readiness === r.value && styles.optionSelected]}
            onPress={() => setReadiness(r.value)}
          >
            <Text style={readiness === r.value ? styles.optionTextSelected : styles.optionText}>{r.label}</Text>
          </TouchableOpacity>
        ))}
      </View>

      <Text style={common.label}>Геопозиция</Text>
      <TouchableOpacity style={common.buttonSecondary} onPress={handleRequestLocation}>
        <Text style={common.buttonSecondaryText}>
          {locationStatus === 'set' ? 'Геопозиция обновлена ✓' : 'Разрешить доступ к геопозиции'}
        </Text>
      </TouchableOpacity>

      {error && <Text style={common.errorText}>{error}</Text>}

      <TouchableOpacity style={common.button} onPress={handleSave} disabled={saving}>
        <Text style={common.buttonText}>{saving ? 'Сохранение...' : 'Сохранить'}</Text>
      </TouchableOpacity>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  centered: { alignItems: 'center', justifyContent: 'center' },
  photoRow: { flexDirection: 'row', gap: 10, marginBottom: 8 },
  photo: { width: 80, height: 80, borderRadius: 10, backgroundColor: colors.surfaceAlt },
  addPhoto: {
    width: 80,
    height: 80,
    borderRadius: 10,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border,
    borderStyle: 'dashed',
    alignItems: 'center',
    justifyContent: 'center',
  },
  addPhotoText: { color: colors.accent, fontSize: 28 },
  optionRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  option: {
    paddingHorizontal: 14,
    paddingVertical: 10,
    borderRadius: 20,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border,
  },
  optionSelected: { backgroundColor: colors.accentMuted, borderColor: colors.accent },
  optionText: { color: colors.textSecondary },
  optionTextSelected: { color: colors.textPrimary, fontWeight: '600' },
  bioInput: { minHeight: 80, textAlignVertical: 'top' },
});
