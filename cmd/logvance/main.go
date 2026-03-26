package main

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/yourusername/logvance/config"
	"github.com/yourusername/logvance/internal/analyzer"
	"github.com/yourusername/logvance/internal/api"
	"github.com/yourusername/logvance/internal/auth"
	"github.com/yourusername/logvance/internal/geoip"
	"github.com/yourusername/logvance/internal/parser"
	"github.com/yourusername/logvance/internal/storage"
	"github.com/yourusername/logvance/internal/tailer"
)

const batchSize = 1000

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("[logvance] starting...")

	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := storage.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	defer db.Close()

	if err := db.CreateSecuritySchema(); err != nil {
		log.Fatalf("security schema: %v", err)
	}

	// Auth
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "logvance-dev-secret-change-in-production"
		log.Println("[auth] WARNING: using default JWT secret")
	}
	jwtManager := auth.NewJWTManager(jwtSecret)
	userStore := auth.NewUserStore(db.GetConn())
	if err := userStore.Init(); err != nil {
		log.Fatalf("user store: %v", err)
	}
	count, _ := userStore.Count()
	if count == 0 {
		log.Println("[auth] No users — POST /api/auth/setup")
	} else {
		log.Printf("[auth] %d user(s) registered", count)
	}

	// TOTP 2FA
	totpManager := auth.NewTOTPManager(db.GetConn())
	if err := totpManager.Init(); err != nil {
		log.Printf("[totp] WARNING: %v", err)
	}

	// GeoIP
	var geo *geoip.Resolver
	geoPath := "data/geoip/GeoLite2-City.mmdb"
	if _, err := os.Stat(geoPath); err == nil {
		geo, err = geoip.New(geoPath)
		if err != nil {
			log.Printf("[geoip] WARNING: %v", err)
		}
		if geo != nil {
			defer geo.Close()
		}
	} else {
		log.Println("[geoip] database not found, world map disabled")
	}

	log.Printf("[logvance] database: %s", cfg.Database.Path)

	t, err := tailer.New(cfg.Logs.NginxAccess)
	if err != nil {
		log.Fatalf("tailer: %v", err)
	}
	defer t.Close()

	if err := t.Start(0); err != nil {
		log.Fatalf("tail start: %v", err)
	}
	log.Printf("[logvance] tailing: %s", cfg.Logs.NginxAccess)

	numWorkers := runtime.NumCPU()
	log.Printf("[logvance] workers: %d", numWorkers)
	parsed := make(chan *parser.LogEntry, 50000)

	for i := 0; i < numWorkers; i++ {
		go func() {
			for line := range t.Lines() {
				entry, err := parser.ParseNginxLine(line)
				if err != nil {
					continue
				}
				parsed <- entry
			}
		}()
	}

	go func() {
		batch := make([]*parser.LogEntry, 0, batchSize)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		total := 0

		flush := func() {
			if len(batch) == 0 {
				return
			}
			if err := db.InsertBatch(batch); err != nil {
				log.Printf("[writer] error: %v", err)
			} else {
				total += len(batch)
				log.Printf("[writer] flushed %d | total: %d", len(batch), total)
			}
			batch = batch[:0]
		}

		for {
			select {
			case entry := <-parsed:
				batch = append(batch, entry)
				threats := analyzer.AnalyzeRequest(entry.Path, entry.UserAgent, entry.IP, entry.StatusCode)
				for _, threat := range threats {
					evt := storage.SecurityEvent{
						IP: entry.IP, Timestamp: entry.Time,
						Path: entry.Path, UserAgent: entry.UserAgent,
						ThreatType: string(threat.Type), Severity: threat.Severity,
						Description: threat.Description, Score: threat.Score,
					}
					if err := db.InsertSecurityEvent(evt); err != nil {
						log.Printf("[security] error: %v", err)
					} else {
						log.Printf("[security] THREAT: %s from %s — %s (score:%d)",
							threat.Type, entry.IP, entry.Path, threat.Score)
					}
				}
				if len(batch) >= batchSize {
					flush()
				}
			case <-ticker.C:
				flush()
			}
		}
	}()

	srv := api.New(db, cfg.Server.Port, jwtManager, userStore)
	srv.SetTOTP(totpManager)
	if geo != nil {
		srv.SetGeo(geo)
	}

	go srv.BroadcastUpdate()

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("api server: %v", err)
		}
	}()

	log.Printf("[logvance] ready — http://%s:%d", cfg.Server.Host, cfg.Server.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("[logvance] shutdown: %v", sig)
	log.Println("[logvance] stopped cleanly")
}
