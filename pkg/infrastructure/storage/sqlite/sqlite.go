package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	sqlitedrv "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"anon3anon/pkg/domain"
	"anon3anon/pkg/token"
)

//go:embed schema.sql
var schema string

const tokenAttempts = 5

type Store struct {
	db         *sql.DB
	rateWindow time.Duration
	rateMax    int
}

func Open(path string, rateWindow time.Duration, rateMax int) (*Store, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)",
		path,
	)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.ExecContext(context.Background(), schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}

	return &Store{db: db, rateWindow: rateWindow, rateMax: rateMax}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Store) UpsertUser(ctx context.Context, tgUserID, chatID int64) (domain.User, error) {
	existing, ok, err := s.UserByID(ctx, tgUserID)
	if err != nil {
		return domain.User{}, err
	}
	if ok {
		if _, err := s.db.ExecContext(ctx,
			`UPDATE users SET chat_id = ? WHERE tg_user_id = ?`, chatID, tgUserID,
		); err != nil {
			return domain.User{}, fmt.Errorf("refresh chat id: %w", err)
		}
		existing.ChatID = chatID
		return existing, nil
	}
	return s.insertUser(ctx, tgUserID, chatID)
}

func (s *Store) insertUser(ctx context.Context, tgUserID, chatID int64) (domain.User, error) {
	now := time.Now().UTC()
	for range tokenAttempts {
		tok, err := token.New()
		if err != nil {
			return domain.User{}, fmt.Errorf("generate token: %w", err)
		}

		_, err = s.db.ExecContext(ctx,
			`INSERT INTO users (tg_user_id, chat_id, link_token, created_at) VALUES (?, ?, ?, ?)`,
			tgUserID, chatID, tok, now.Unix(),
		)
		switch {
		case err == nil:
			return domain.User{TgUserID: tgUserID, ChatID: chatID, LinkToken: tok, CreatedAt: now}, nil
		case isUniqueViolation(err):
			continue
		default:
			return domain.User{}, fmt.Errorf("insert user: %w", err)
		}
	}
	return domain.User{}, errors.New("could not allocate a unique link token")
}

func (s *Store) UserByToken(ctx context.Context, tokenValue string) (domain.User, bool, error) {
	return s.queryUser(ctx,
		`SELECT tg_user_id, chat_id, link_token, created_at FROM users WHERE link_token = ?`,
		tokenValue,
	)
}

func (s *Store) UserByID(ctx context.Context, tgUserID int64) (domain.User, bool, error) {
	return s.queryUser(ctx,
		`SELECT tg_user_id, chat_id, link_token, created_at FROM users WHERE tg_user_id = ?`,
		tgUserID,
	)
}

func (s *Store) queryUser(ctx context.Context, query string, arg any) (domain.User, bool, error) {
	var (
		u       domain.User
		created int64
	)
	err := s.db.QueryRowContext(ctx, query, arg).Scan(&u.TgUserID, &u.ChatID, &u.LinkToken, &created)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return domain.User{}, false, nil
	case err != nil:
		return domain.User{}, false, fmt.Errorf("query user: %w", err)
	}
	u.CreatedAt = time.Unix(created, 0).UTC()
	return u, true, nil
}

func (s *Store) RotateToken(ctx context.Context, tgUserID int64) (string, error) {
	for range tokenAttempts {
		tok, err := token.New()
		if err != nil {
			return "", fmt.Errorf("generate token: %w", err)
		}

		res, err := s.db.ExecContext(ctx,
			`UPDATE users SET link_token = ? WHERE tg_user_id = ?`, tok, tgUserID,
		)
		switch {
		case err == nil:
			affected, aerr := res.RowsAffected()
			if aerr != nil {
				return "", fmt.Errorf("rows affected: %w", aerr)
			}
			if affected == 0 {
				return "", fmt.Errorf("rotate token: %w", sql.ErrNoRows)
			}
			return tok, nil
		case isUniqueViolation(err):
			continue
		default:
			return "", fmt.Errorf("update token: %w", err)
		}
	}
	return "", errors.New("could not allocate a unique link token")
}

func (s *Store) DeleteUser(ctx context.Context, tgUserID int64) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin delete user: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Resolve the account first. Nothing is deleted for a caller who is not a
	// registered recipient: the related-row filters below also match plain
	// senders, and wiping a stranger's live session/relays on a no-op /delete
	// would be silent cross-user state destruction.
	var chatID int64
	err = tx.QueryRowContext(ctx,
		`SELECT chat_id FROM users WHERE tg_user_id = ?`, tgUserID,
	).Scan(&chatID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("lookup user: %w", err)
	}

	// sessions.owner_user_id references users, so sessions must go before the
	// users row. Match both roles: rows pointed at this owner and rows for this
	// user's own chat as a sender. chat_id == tg_user_id for private chats, but
	// use the stored value so this stays correct if that ever changes.
	stmts := []struct {
		query string
		args  []any
	}{
		{`DELETE FROM sessions WHERE owner_user_id = ? OR sender_chat_id = ?`, []any{tgUserID, chatID}},
		{`DELETE FROM relays WHERE owner_user_id = ? OR dest_chat_id = ? OR origin_chat_id = ?`, []any{tgUserID, chatID, chatID}},
		{`DELETE FROM blocks WHERE owner_user_id = ? OR sender_chat_id = ?`, []any{tgUserID, chatID}},
		{`DELETE FROM message_rates WHERE recipient_id = ? OR recipient_id = ? OR sender_id = ?`, []any{tgUserID, chatID, chatID}},
	}
	for _, st := range stmts {
		if _, execErr := tx.ExecContext(ctx, st.query, st.args...); execErr != nil {
			return false, fmt.Errorf("delete user rows: %w", execErr)
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM users WHERE tg_user_id = ?`, tgUserID); err != nil {
		return false, fmt.Errorf("delete user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit delete user: %w", err)
	}
	return true, nil
}

func (s *Store) SetSession(ctx context.Context, senderChatID, ownerUserID int64) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (sender_chat_id, owner_user_id, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT (sender_chat_id) DO UPDATE SET owner_user_id = excluded.owner_user_id,
		                                            updated_at    = excluded.updated_at`,
		senderChatID, ownerUserID, time.Now().UTC().Unix(),
	)
	if err != nil {
		return fmt.Errorf("set session: %w", err)
	}
	return nil
}

func (s *Store) GetSession(
	ctx context.Context, senderChatID int64,
) (ownerUserID int64, ok bool, err error) {
	err = s.db.QueryRowContext(ctx,
		`SELECT owner_user_id FROM sessions WHERE sender_chat_id = ?`, senderChatID,
	).Scan(&ownerUserID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return 0, false, nil
	case err != nil:
		return 0, false, fmt.Errorf("get session: %w", err)
	}
	return ownerUserID, true, nil
}

func (s *Store) TouchSession(ctx context.Context, senderChatID int64) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET updated_at = ? WHERE sender_chat_id = ?`,
		time.Now().UTC().Unix(), senderChatID,
	); err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	return nil
}

func (s *Store) ClearSession(ctx context.Context, senderChatID int64) error {
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE sender_chat_id = ?`, senderChatID,
	); err != nil {
		return fmt.Errorf("clear session: %w", err)
	}
	return nil
}

func (s *Store) ClearSessionsForOwner(ctx context.Context, ownerUserID int64) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE owner_user_id = ?`, ownerUserID,
	)
	if err != nil {
		return 0, fmt.Errorf("clear sessions for owner: %w", err)
	}
	removed, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	return removed, nil
}

func (s *Store) PutRelay(ctx context.Context, r domain.Relay) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO relays (dest_chat_id, dest_msg_id, origin_chat_id, owner_user_id, created_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (dest_chat_id, dest_msg_id) DO UPDATE SET
		     origin_chat_id = excluded.origin_chat_id,
		     owner_user_id  = excluded.owner_user_id,
		     created_at     = excluded.created_at`,
		r.DestChatID, r.DestMsgID, r.OriginChatID, r.OwnerUserID, time.Now().UTC().Unix(),
	)
	if err != nil {
		return fmt.Errorf("put relay: %w", err)
	}
	return nil
}

func (s *Store) LookupRelay(ctx context.Context, destChatID int64, destMsgID int) (domain.Relay, bool, error) {
	r := domain.Relay{DestChatID: destChatID, DestMsgID: destMsgID}
	var created int64
	err := s.db.QueryRowContext(ctx,
		`SELECT origin_chat_id, owner_user_id, created_at FROM relays
		 WHERE dest_chat_id = ? AND dest_msg_id = ?`,
		destChatID, destMsgID,
	).Scan(&r.OriginChatID, &r.OwnerUserID, &created)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return domain.Relay{}, false, nil
	case err != nil:
		return domain.Relay{}, false, fmt.Errorf("lookup relay: %w", err)
	}
	r.CreatedAt = time.Unix(created, 0).UTC()
	return r, true, nil
}

func (s *Store) PurgeExpired(ctx context.Context, cutoff time.Time) (domain.PurgeStats, error) {
	ts := cutoff.UTC().Unix()

	var stats domain.PurgeStats

	byTime := []struct {
		name  string
		query string
		into  *int64
	}{
		{"sessions", `DELETE FROM sessions WHERE updated_at < ?`, &stats.Sessions},
		{"relays", `DELETE FROM relays WHERE created_at < ?`, &stats.Relays},
		{"blocks", `DELETE FROM blocks WHERE created_at < ?`, &stats.Blocks},
	}
	for _, d := range byTime {
		n, err := s.execCount(ctx, d.query, ts)
		if err != nil {
			return stats, fmt.Errorf("purge %s: %w", d.name, err)
		}
		*d.into = n
	}

	if s.rateLimitEnabled() {
		n, err := s.execCount(ctx,
			`DELETE FROM message_rates WHERE bucket < ?`,
			ts/int64(s.rateWindow.Seconds()),
		)
		if err != nil {
			return stats, fmt.Errorf("purge message_rates: %w", err)
		}
		stats.MessageRates = n
	}

	return stats, nil
}

func (s *Store) execCount(ctx context.Context, query string, args ...any) (int64, error) {
	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) AllowMessage(ctx context.Context, senderID, recipientID int64) (bool, error) {
	if !s.rateLimitEnabled() {
		return true, nil
	}

	bucket := s.currentBucket()

	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM message_rates
		 WHERE sender_id = ? AND recipient_id = ? AND bucket < ?`,
		senderID, recipientID, bucket,
	); err != nil {
		return false, fmt.Errorf("prune message rates: %w", err)
	}

	var count int
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO message_rates (sender_id, recipient_id, bucket, count)
		 VALUES (?, ?, ?, 1)
		 ON CONFLICT (sender_id, recipient_id, bucket)
		 DO UPDATE SET count = count + 1
		 RETURNING count`,
		senderID, recipientID, bucket,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("bump message rate: %w", err)
	}
	return count <= s.rateMax, nil
}

func (s *Store) RefundMessage(ctx context.Context, senderID, recipientID int64) error {
	if !s.rateLimitEnabled() {
		return nil
	}

	if _, err := s.db.ExecContext(ctx,
		`UPDATE message_rates SET count = count - 1
		 WHERE sender_id = ? AND recipient_id = ? AND bucket = ? AND count > 0`,
		senderID, recipientID, s.currentBucket(),
	); err != nil {
		return fmt.Errorf("refund message rate: %w", err)
	}
	return nil
}

func (s *Store) Block(ctx context.Context, ownerUserID, senderChatID int64) error {
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO blocks (owner_user_id, sender_chat_id, created_at) VALUES (?, ?, ?)
		 ON CONFLICT (owner_user_id, sender_chat_id) DO NOTHING`,
		ownerUserID, senderChatID, time.Now().UTC().Unix(),
	); err != nil {
		return fmt.Errorf("insert block: %w", err)
	}
	return nil
}

func (s *Store) IsBlocked(ctx context.Context, ownerUserID, senderChatID int64) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx,
		`SELECT 1 FROM blocks WHERE owner_user_id = ? AND sender_chat_id = ?`,
		ownerUserID, senderChatID,
	).Scan(&one)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("query block: %w", err)
	}
	return true, nil
}

func (s *Store) rateLimitEnabled() bool {
	return s.rateWindow >= time.Second && s.rateMax > 0
}

func (s *Store) currentBucket() int64 {
	return time.Now().UTC().Unix() / int64(s.rateWindow.Seconds())
}

func isUniqueViolation(err error) bool {
	var serr *sqlitedrv.Error
	if !errors.As(err, &serr) {
		return false
	}
	code := serr.Code()
	return code == sqlite3.SQLITE_CONSTRAINT_UNIQUE || code == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY
}
