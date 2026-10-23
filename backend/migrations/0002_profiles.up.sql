CREATE TABLE profiles (
    user_id          UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    name             TEXT NOT NULL,
    age              SMALLINT NOT NULL CHECK (age >= 18),
    gender           TEXT NOT NULL CHECK (gender IN ('male', 'female', 'other')),
    bio              TEXT NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
    preferred_drink  TEXT,
    readiness_status TEXT NOT NULL DEFAULT 'now' CHECK (readiness_status IN ('now', 'today', 'this_week')),
    location         GEOGRAPHY(Point, 4326),
    location_updated_at TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_profiles_location ON profiles USING GIST (location);

CREATE TABLE profile_photos (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    object_key TEXT NOT NULL,
    position   SMALLINT NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (user_id, position)
);

CREATE INDEX idx_profile_photos_user_id ON profile_photos(user_id);
