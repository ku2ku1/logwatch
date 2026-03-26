package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/yourusername/logvance/internal/parser"
)

type DB struct {
	conn *sql.DB
}

func New(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	conn, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.createSchema(); err != nil {
		return nil, fmt.Errorf("create schema: %w", err)
	}
	return db, nil
}

func (db *DB) createSchema() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS nginx_logs (
			id           BIGINT PRIMARY KEY,
			ip           VARCHAR,
			ts           TIMESTAMP,
			method       VARCHAR,
			path         VARCHAR,
			protocol     VARCHAR,
			status_code  INTEGER,
			bytes_sent   BIGINT,
			referer      VARCHAR,
			user_agent   VARCHAR,
			response_time DOUBLE,
			created_at   TIMESTAMP DEFAULT NOW()
		);

		CREATE SEQUENCE IF NOT EXISTS nginx_logs_seq START 1;

		CREATE TABLE IF NOT EXISTS checkpoints (
			source   VARCHAR PRIMARY KEY,
			position BIGINT,
			updated  TIMESTAMP DEFAULT NOW()
		);
	`)
	return err
}

func (db *DB) InsertBatch(entries []*parser.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO nginx_logs
			(id, ip, ts, method, path, protocol, status_code, bytes_sent, referer, user_agent, response_time)
		VALUES (nextval('nginx_logs_seq'), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range entries {
		_, err = stmt.Exec(
			e.IP, e.Time, e.Method, e.Path, e.Protocol,
			e.StatusCode, e.BytesSent, e.Referer, e.UserAgent, e.ResponseTime,
		)
		if err != nil {
			return fmt.Errorf("insert: %w", err)
		}
	}

	return tx.Commit()
}

func (db *DB) SaveCheckpoint(source string, position int64) error {
	_, err := db.conn.Exec(`
		INSERT INTO checkpoints (source, position, updated)
		VALUES (?, ?, ?)
		ON CONFLICT (source) DO UPDATE SET position = excluded.position, updated = excluded.updated
	`, source, position, time.Now())
	return err
}

func (db *DB) GetCheckpoint(source string) (int64, error) {
	var pos int64
	err := db.conn.QueryRow(
		`SELECT position FROM checkpoints WHERE source = ?`, source,
	).Scan(&pos)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return pos, err
}

// Stats queries
type Stats struct {
	TotalRequests int64
	UniqueIPs     int64
	TotalBytes    int64
	AvgResponse   float64
	ErrorRate     float64
}

func (db *DB) GetStats(since time.Time) (*Stats, error) {
	row := db.conn.QueryRow(`
		SELECT
			COUNT(*)                                         AS total,
			COUNT(DISTINCT ip)                               AS unique_ips,
			COALESCE(SUM(bytes_sent), 0)                     AS total_bytes,
			COALESCE(AVG(response_time), 0)                  AS avg_resp,
			COALESCE(
				SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END) * 100.0 / COUNT(*),
				0
			)                                                AS error_rate
		FROM nginx_logs
		WHERE ts >= ?
	`, since)

	s := &Stats{}
	return s, row.Scan(&s.TotalRequests, &s.UniqueIPs, &s.TotalBytes, &s.AvgResponse, &s.ErrorRate)
}

type TopEntry struct {
	Key   string
	Count int64
}

func (db *DB) GetTopIPs(since time.Time, limit int) ([]TopEntry, error) {
	return db.queryTop(`
		SELECT ip AS key, COUNT(*) AS cnt
		FROM nginx_logs WHERE ts >= ?
		GROUP BY ip ORDER BY cnt DESC LIMIT ?
	`, since, limit)
}

func (db *DB) GetTopPaths(since time.Time, limit int) ([]TopEntry, error) {
	return db.queryTop(`
		SELECT path AS key, COUNT(*) AS cnt
		FROM nginx_logs WHERE ts >= ?
		GROUP BY path ORDER BY cnt DESC LIMIT ?
	`, since, limit)
}

func (db *DB) GetStatusCodes(since time.Time) ([]TopEntry, error) {
	return db.queryTop(`
		SELECT CAST(status_code AS VARCHAR) AS key, COUNT(*) AS cnt
		FROM nginx_logs WHERE ts >= ?
		GROUP BY status_code ORDER BY cnt DESC
	`, since, 20)
}

func (db *DB) queryTop(query string, args ...any) ([]TopEntry, error) {
	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TopEntry
	for rows.Next() {
		var e TopEntry
		if err := rows.Scan(&e.Key, &e.Count); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if result == nil {
		return []TopEntry{}, nil
	}
	return result, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Exec(query string, args ...any) (sql.Result, error) {
	return db.conn.Exec(query, args...)
}

func (db *DB) GetConn() *sql.DB {
	return db.conn
}

// Security tables
func (db *DB) CreateSecuritySchema() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS security_events (
			id          BIGINT PRIMARY KEY,
			ip          VARCHAR,
			ts          TIMESTAMP,
			path        VARCHAR,
			user_agent  VARCHAR,
			threat_type VARCHAR,
			severity    VARCHAR,
			description VARCHAR,
			score       INTEGER
		);

		CREATE SEQUENCE IF NOT EXISTS sec_events_seq START 1;

		CREATE TABLE IF NOT EXISTS ip_threat_scores (
			ip          VARCHAR PRIMARY KEY,
			total_score INTEGER DEFAULT 0,
			event_count INTEGER DEFAULT 0,
			last_seen   TIMESTAMP,
			blocked     BOOLEAN DEFAULT false
		);
	`)
	return err
}

type SecurityEvent struct {
	IP          string
	Timestamp   interface{}
	Path        string
	UserAgent   string
	ThreatType  string
	Severity    string
	Description string
	Score       int
}

func (db *DB) InsertSecurityEvent(e SecurityEvent) error {
	_, err := db.conn.Exec(`
		INSERT INTO security_events
			(id, ip, ts, path, user_agent, threat_type, severity, description, score)
		VALUES (nextval('sec_events_seq'), ?, ?, ?, ?, ?, ?, ?, ?)
	`, e.IP, e.Timestamp, e.Path, e.UserAgent, e.ThreatType, e.Severity, e.Description, e.Score)
	if err != nil {
		return err
	}

	// IP threat score update
	_, err = db.conn.Exec(`
		INSERT INTO ip_threat_scores (ip, total_score, event_count, last_seen)
		VALUES (?, ?, 1, NOW())
		ON CONFLICT (ip) DO UPDATE SET
			total_score = ip_threat_scores.total_score + excluded.total_score,
			event_count = ip_threat_scores.event_count + 1,
			last_seen   = NOW()
	`, e.IP, e.Score)
	return err
}

type SecurityStats struct {
	TotalEvents    int64
	CriticalEvents int64
	UniqueAttackers int64
	TopThreatType  string
}

func (db *DB) GetSecurityStats() (*SecurityStats, error) {
	s := &SecurityStats{}
	err := db.conn.QueryRow(`
		SELECT
			COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN severity = 'critical' THEN 1 ELSE 0 END), 0) AS critical,
			COALESCE(COUNT(DISTINCT ip), 0) AS unique_ips
		FROM security_events
		WHERE ts >= CAST(CURRENT_TIMESTAMP AS TIMESTAMP) - INTERVAL 24 HOURS
	`).Scan(&s.TotalEvents, &s.CriticalEvents, &s.UniqueAttackers)
	if err != nil {
		return s, err
	}

	// Top threat type
	db.conn.QueryRow(`
		SELECT threat_type FROM security_events
		WHERE ts >= CAST(CURRENT_TIMESTAMP AS TIMESTAMP) - INTERVAL 24 HOURS
		GROUP BY threat_type ORDER BY COUNT(*) DESC LIMIT 1
	`).Scan(&s.TopThreatType)

	return s, nil
}

func (db *DB) GetRecentThreats(limit int) ([]SecurityEvent, error) {
	rows, err := db.conn.Query(`
		SELECT ip, ts, path, user_agent, threat_type, severity, description, score
		FROM security_events
		ORDER BY ts DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []SecurityEvent
	for rows.Next() {
		var e SecurityEvent
		if err := rows.Scan(&e.IP, &e.Timestamp, &e.Path, &e.UserAgent,
			&e.ThreatType, &e.Severity, &e.Description, &e.Score); err != nil {
			continue
		}
		events = append(events, e)
	}
	return events, nil
}

func (db *DB) GetTopAttackers(limit int) ([]TopEntry, error) {
	return db.queryTop(`
		SELECT ip AS key, CAST(COALESCE(SUM(score), 0) AS BIGINT) AS cnt
		FROM security_events
		WHERE ts >= CAST(CURRENT_TIMESTAMP AS TIMESTAMP) - INTERVAL 24 HOURS
		GROUP BY ip ORDER BY cnt DESC LIMIT ?
	`, limit)
}

func (db *DB) GetSecurityStatsFixed() (*SecurityStats, error) {
	s := &SecurityStats{}
	err := db.conn.QueryRow(`
		SELECT
			COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN severity = 'critical' THEN 1 ELSE 0 END), 0) AS critical,
			COALESCE(COUNT(DISTINCT ip), 0) AS unique_ips
		FROM security_events
		WHERE ts >= CAST(CURRENT_TIMESTAMP AS TIMESTAMP) - INTERVAL 24 HOURS
	`).Scan(&s.TotalEvents, &s.CriticalEvents, &s.UniqueAttackers)
	if err != nil {
		return s, err
	}
	db.conn.QueryRow(`
		SELECT threat_type FROM security_events
		WHERE ts >= CAST(CURRENT_TIMESTAMP AS TIMESTAMP) - INTERVAL 24 HOURS
		GROUP BY threat_type ORDER BY COUNT(*) DESC LIMIT 1
	`).Scan(&s.TopThreatType)
	return s, nil
}

func (db *DB) GetTopAttackersFixed(limit int) ([]TopEntry, error) {
	return db.queryTop(`
		SELECT ip AS key, CAST(COALESCE(SUM(score), 0) AS BIGINT) AS cnt
		FROM security_events
		WHERE ts >= CAST(CURRENT_TIMESTAMP AS TIMESTAMP) - INTERVAL 24 HOURS
		GROUP BY ip ORDER BY cnt DESC LIMIT ?
	`, limit)
}


func (db *DB) GetTopIPsWithGeo(since time.Time, limit int) ([]TopEntry, error) {
	return db.GetTopIPs(since, limit)
}

type GeoEntry struct {
	IP          string  `json:"ip"`
	Count       int64   `json:"count"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
}
