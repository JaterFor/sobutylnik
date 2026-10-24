CREATE TABLE swipes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    swiper_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    liked       BOOLEAN NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT swipes_no_self CHECK (swiper_id <> target_id),
    UNIQUE (swiper_id, target_id)
);

CREATE INDEX idx_swipes_swiper ON swipes(swiper_id);
CREATE INDEX idx_swipes_target ON swipes(target_id);

CREATE TABLE matches (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_a_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_b_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status        TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'unmatched')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    unmatched_at  TIMESTAMPTZ,
    unmatched_by  UUID REFERENCES users(id),

    CONSTRAINT matches_no_self CHECK (user_a_id <> user_b_id),
    CONSTRAINT matches_ordered_pair CHECK (user_a_id < user_b_id),
    UNIQUE (user_a_id, user_b_id)
);

CREATE INDEX idx_matches_user_a ON matches(user_a_id);
CREATE INDEX idx_matches_user_b ON matches(user_b_id);

CREATE TABLE blocks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    blocker_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT blocks_no_self CHECK (blocker_id <> blocked_id),
    UNIQUE (blocker_id, blocked_id)
);

CREATE INDEX idx_blocks_blocker ON blocks(blocker_id);
CREATE INDEX idx_blocks_blocked ON blocks(blocked_id);

CREATE TABLE reports (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reported_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason      TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT reports_no_self CHECK (reporter_id <> reported_id),
    UNIQUE (reporter_id, reported_id)
);

CREATE INDEX idx_reports_reported ON reports(reported_id);

ALTER TABLE users ADD COLUMN restricted BOOLEAN NOT NULL DEFAULT false;
