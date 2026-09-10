package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"thingspan/internal/api"
	"thingspan/internal/config"
	"thingspan/internal/database"
	"thingspan/internal/security"
	"thingspan/internal/services"
)

func runHealthCheck(port int) {
	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/healthz", port)

	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Health check returned status: %d\n", resp.StatusCode)
		os.Exit(1)
	}

	os.Exit(0)
}

func main() {
	healthcheckFlag := flag.Bool("healthcheck", false, "Run local container health check and exit")
	portFlag := flag.Int("port", 0, "Server port (overrides PORT env)")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	port := cfg.Port
	if *portFlag > 0 {
		port = *portFlag
	}

	// 1. Health check CLI mode
	if *healthcheckFlag {
		runHealthCheck(port)
		return
	}

	// 2. Normal startup mode: validate secrets first
	if err := cfg.ValidateSecrets(); err != nil {
		log.Fatalf("启动失败: %v", err)
	}

	log.Printf("Thingspan 启动中 (时区: %s, 数据目录: %s)...", cfg.TZ, cfg.DataDir)

	// Initialize SQLite Database
	db, err := database.InitDB(cfg.DatabasePath())
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer db.Close()

	// Run Database Migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}
	log.Println("数据库初始化与迁移完成")

	// Initialize Security
	pm, err := security.NewPasswordManager(cfg.AppPassword)
	if err != nil {
		log.Fatalf("密码哈希初始化失败: %v", err)
	}
	jwtm := security.NewJWTManager(cfg.JWTSecret, cfg.AccessTokenExpireMinutes, cfg.RefreshTokenExpireDays)
	rl := security.NewRateLimiter()

	// Initialize Email and Reminder Scanner
	emailSender := services.NewEmailSender(cfg)
	scanner := services.NewReminderScanner(db, cfg, emailSender)

	// Start Background Scheduler
	scheduler := services.NewScheduler(scanner, cfg)
	scheduler.Start()
	defer scheduler.Stop()

	// Build Router
	router := api.NewRouter(cfg, db, pm, jwtm, rl, scanner)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown handling
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Thingspan 启动完成，正在监听端口 :%d", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP 服务异常退出: %v", err)
		}
	}()

	<-stopChan
	log.Println("正在关闭 Thingspan 服务...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("服务优雅关闭超时: %v", err)
	}
	log.Println("Thingspan 已安全退出")
}

