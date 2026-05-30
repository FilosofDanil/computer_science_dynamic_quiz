package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"basics/internal/apperrors"
	"basics/internal/quiz"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGTestStore is a PostgreSQL-backed TestStore. Each test is stored as one row
// whose questions live in a JSONB column, so a whole test set is a single
// JSON document in the database.
type PGTestStore struct {
	pool *pgxpool.Pool
}

const createSchemaSQL = `
CREATE TABLE IF NOT EXISTS tests (
    id          BIGSERIAL PRIMARY KEY,
    owner_chat  BIGINT,
    title       TEXT        NOT NULL,
    data        JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS tests_owner_idx ON tests (owner_chat);`

// NewPGTestStore connects to the database using connString, verifies the
// connection, and ensures the schema exists. The returned store is safe for
// concurrent use; call Close when done.
func NewPGTestStore(ctx context.Context, connString string) (*PGTestStore, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("storage: connect postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("storage: ping postgres: %w", err)
	}
	if _, err := pool.Exec(ctx, createSchemaSQL); err != nil {
		pool.Close()
		return nil, fmt.Errorf("storage: ensure schema: %w", err)
	}
	return &PGTestStore{pool: pool}, nil
}

// Close releases the underlying connection pool.
func (s *PGTestStore) Close() {
	s.pool.Close()
}

// questionsDoc is the shape stored in the JSONB data column.
type questionsDoc struct {
	Questions []quiz.Topic `json:"questions"`
}

func scanTest(row pgx.Row) (Test, error) {
	var (
		t   Test
		raw []byte
	)
	if err := row.Scan(&t.ID, &t.OwnerChat, &t.Title, &raw); err != nil {
		return Test{}, err
	}
	var doc questionsDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Test{}, fmt.Errorf("storage: decode test %d: %w", t.ID, err)
	}
	t.Questions = doc.Questions
	return t, nil
}

func (s *PGTestStore) queryTests(ctx context.Context, sql string, args ...any) ([]Test, error) {
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("storage: query tests: %w", err)
	}
	defer rows.Close()

	var out []Test
	for rows.Next() {
		t, err := scanTest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: iterate tests: %w", err)
	}
	return out, nil
}

// ListAvailable returns all global tests plus the tests owned by chatID.
func (s *PGTestStore) ListAvailable(chatID int64) ([]Test, error) {
	const sql = `SELECT id, owner_chat, title, data
	             FROM tests
	             WHERE owner_chat IS NULL OR owner_chat = $1
	             ORDER BY owner_chat NULLS FIRST, title`
	return s.queryTests(context.Background(), sql, chatID)
}

// ListOwned returns only the tests owned by chatID.
func (s *PGTestStore) ListOwned(chatID int64) ([]Test, error) {
	const sql = `SELECT id, owner_chat, title, data
	             FROM tests
	             WHERE owner_chat = $1
	             ORDER BY title`
	return s.queryTests(context.Background(), sql, chatID)
}

// Get returns a single test by id.
func (s *PGTestStore) Get(id int64) (Test, error) {
	const sql = `SELECT id, owner_chat, title, data FROM tests WHERE id = $1`
	t, err := scanTest(s.pool.QueryRow(context.Background(), sql, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Test{}, fmt.Errorf("%w: id %d", apperrors.ErrTestNotFound, id)
	}
	if err != nil {
		return Test{}, fmt.Errorf("storage: get test %d: %w", id, err)
	}
	return t, nil
}

// Create inserts a new test and returns its generated id.
func (s *PGTestStore) Create(t Test) (int64, error) {
	raw, err := json.Marshal(questionsDoc{Questions: t.Questions})
	if err != nil {
		return 0, fmt.Errorf("storage: encode test: %w", err)
	}
	const sql = `INSERT INTO tests (owner_chat, title, data)
	             VALUES ($1, $2, $3) RETURNING id`
	var id int64
	if err := s.pool.QueryRow(context.Background(), sql, t.OwnerChat, t.Title, raw).Scan(&id); err != nil {
		return 0, fmt.Errorf("storage: create test: %w", err)
	}
	return id, nil
}

// Update replaces the title and questions of a test owned by t.OwnerChat.
func (s *PGTestStore) Update(t Test) error {
	if t.OwnerChat == nil {
		return fmt.Errorf("%w: cannot update a global test", apperrors.ErrTestNotFound)
	}
	raw, err := json.Marshal(questionsDoc{Questions: t.Questions})
	if err != nil {
		return fmt.Errorf("storage: encode test: %w", err)
	}
	const sql = `UPDATE tests
	             SET title = $1, data = $2, updated_at = NOW()
	             WHERE id = $3 AND owner_chat = $4`
	tag, err := s.pool.Exec(context.Background(), sql, t.Title, raw, t.ID, *t.OwnerChat)
	if err != nil {
		return fmt.Errorf("storage: update test %d: %w", t.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: id %d", apperrors.ErrTestNotFound, t.ID)
	}
	return nil
}

// Delete removes a test owned by ownerChat.
func (s *PGTestStore) Delete(id, ownerChat int64) error {
	const sql = `DELETE FROM tests WHERE id = $1 AND owner_chat = $2`
	tag, err := s.pool.Exec(context.Background(), sql, id, ownerChat)
	if err != nil {
		return fmt.Errorf("storage: delete test %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: id %d", apperrors.ErrTestNotFound, id)
	}
	return nil
}

// CountGlobal returns the number of curated/global tests. Used by the
// migration command to decide whether seeding is needed.
func (s *PGTestStore) CountGlobal(ctx context.Context) (int, error) {
	const sql = `SELECT COUNT(*) FROM tests WHERE owner_chat IS NULL`
	var n int
	if err := s.pool.QueryRow(ctx, sql).Scan(&n); err != nil {
		return 0, fmt.Errorf("storage: count global tests: %w", err)
	}
	return n, nil
}

// GlobalTitleExists reports whether a global test with the given title exists.
func (s *PGTestStore) GlobalTitleExists(ctx context.Context, title string) (bool, error) {
	const sql = `SELECT EXISTS (SELECT 1 FROM tests WHERE owner_chat IS NULL AND title = $1)`
	var exists bool
	if err := s.pool.QueryRow(ctx, sql, title).Scan(&exists); err != nil {
		return false, fmt.Errorf("storage: check global title: %w", err)
	}
	return exists, nil
}

// compile-time check that PGTestStore satisfies TestStore.
var _ TestStore = (*PGTestStore)(nil)
