CREATE TABLE IF NOT EXISTS scenarios (
    id         UUID PRIMARY KEY,
    doc        JSONB NOT NULL,
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    slug       TEXT     GENERATED ALWAYS AS (doc ->> 'slug') STORED,
    role       TEXT     GENERATED ALWAYS AS (doc ->> 'role') STORED,
    title      TEXT     GENERATED ALWAYS AS (doc ->> 'title') STORED,
    difficulty SMALLINT GENERATED ALWAYS AS ((doc ->> 'difficulty')::smallint) STORED,

    CONSTRAINT scenarios_role_check CHECK (role IN ('buyer', 'seller')),
    CONSTRAINT scenarios_difficulty_check CHECK (difficulty BETWEEN 1 AND 4)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_scenarios_slug ON scenarios (slug);
CREATE INDEX IF NOT EXISTS idx_scenarios_role ON scenarios (role, difficulty) WHERE is_active;

CREATE TABLE IF NOT EXISTS attempts (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    scenario_id UUID NOT NULL REFERENCES scenarios (id) ON DELETE CASCADE,
    status      TEXT NOT NULL DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'finished')),
    state       JSONB NOT NULL,
    score       SMALLINT CHECK (score BETWEEN 0 AND 100),
    outcome     TEXT CHECK (outcome IN ('safe', 'partial', 'scammed')),
    started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_attempts_user ON attempts (user_id, started_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_attempts_active
    ON attempts (user_id, scenario_id) WHERE status = 'in_progress';

CREATE TABLE IF NOT EXISTS attempt_steps (
    id         BIGSERIAL PRIMARY KEY,
    attempt_id UUID NOT NULL REFERENCES attempts (id) ON DELETE CASCADE,
    scene_id   TEXT NOT NULL,
    option_id  TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_steps_attempt ON attempt_steps (attempt_id, id);
