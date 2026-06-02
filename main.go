// Command cactus probes HTTP endpoints on a configurable schedule and sends
// alerts through configured receivers when a probe changes state.
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"cactus/config"
	"cactus/notifier"
	"cactus/prober"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.Probes) == 0 {
		log.Fatal("no probes configured")
	}

	n := buildNotifier(cfg.Receivers)

	stop := make(chan struct{})
	var wg sync.WaitGroup

	for i := range cfg.Probes {
		p := &cfg.Probes[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			prober.Run(p, n, stop)
		}()
	}

	log.Printf("cactus started with %d probe(s)", len(cfg.Probes))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("shutting down...")
	close(stop)
	wg.Wait()
	log.Println("done")
}

func buildNotifier(cfg config.ReceiversConfig) notifier.Notifier {
	var nn []notifier.Notifier
	if cfg.Telegram != nil {
		nn = append(nn, notifier.NewTelegram(*cfg.Telegram))
	}
	return notifier.Multi(nn...)
}
