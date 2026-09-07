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

func (s *Store) AllowMessage(ctx context.Context, senderID, recipientID int64) (bool, error) {
	if s.rateWindow <= 0 || s.rateMax <= 0 {
		return true, nil
	}

	bucket := time.Now().UTC().Unix() / int64(s.rateWindow.Seconds())

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

func isUniqueViolation(err error) bool {
	var serr *sqlitedrv.Error
	if !errors.As(err, &serr) {
		return false
	}
	code := serr.Code()
	return code == sqlite3.SQLITE_CONSTRAINT_UNIQUE || code == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY
}
