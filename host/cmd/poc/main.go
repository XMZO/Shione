package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"shione/host/internal/app"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	backend := app.NewDefault()

	switch os.Args[1] {
	case "frp":
		runFRPCommand(backend, os.Args[2:])
	case "easytier":
		runEasyTierCommand(backend, os.Args[2:])
	case "stack":
		runStackCommand(backend, os.Args[2:])
	case "snapshot":
		printSnapshot(backend)
	default:
		usage()
		os.Exit(2)
	}
}

func runFRPCommand(backend *app.App, args []string) {
	if len(args) < 1 {
		frpUsage()
		os.Exit(2)
	}

	switch args[0] {
	case "validate":
		fs := flag.NewFlagSet("frp validate", flag.ExitOnError)
		configPath := fs.String("config", "", "Path to frpc TOML/YAML/JSON/INI config")
		_ = fs.Parse(args[1:])
		requireConfig(*configPath)

		if err := backend.ValidateFRP(*configPath); err != nil {
			fatalf("frp config invalid: %v", err)
		}
		fmt.Printf("frp config is valid: %s\n", *configPath)

	case "smoke":
		fs := flag.NewFlagSet("frp smoke", flag.ExitOnError)
		configPath := fs.String("config", "", "Path to frpc config")
		runFor := fs.Duration("run-for", 3*time.Second, "How long to keep frp running before stopping; 0 waits for Ctrl+C")
		graceful := fs.Duration("graceful", 500*time.Millisecond, "Graceful shutdown duration to pass to frp")
		stopWait := fs.Duration("stop-wait", 5*time.Second, "How long to wait for frp to stop")
		_ = fs.Parse(args[1:])
		requireConfig(*configPath)

		if err := backend.ValidateFRP(*configPath); err != nil {
			fatalf("frp config invalid: %v", err)
		}
		fmt.Println("validated frp config")

		if err := backend.StartFRP(*configPath); err != nil {
			fatalf("start frp failed: %v", err)
		}
		fmt.Println("started frp client")
		printSnapshot(backend)

		if *runFor == 0 {
			waitForSignal()
		} else {
			time.Sleep(*runFor)
		}

		if err := backend.StopFRP(*graceful, *stopWait); err != nil {
			fatalf("stop frp failed: %v", err)
		}
		fmt.Println("stopped frp client")
		printSnapshot(backend)

	default:
		frpUsage()
		os.Exit(2)
	}
}

func runEasyTierCommand(backend *app.App, args []string) {
	if len(args) < 1 {
		easyTierUsage()
		os.Exit(2)
	}

	switch args[0] {
	case "start":
		fs := flag.NewFlagSet("easytier start", flag.ExitOnError)
		configPath := fs.String("config", "", "Path to an EasyTier config file")
		_ = fs.Parse(args[1:])
		requireConfig(*configPath)

		if err := backend.StartEasyTier(*configPath); err != nil {
			fatalf("start EasyTier failed: %v", err)
		}
		fmt.Println("started EasyTier")
		printSnapshot(backend)

	case "smoke":
		fs := flag.NewFlagSet("easytier smoke", flag.ExitOnError)
		configPath := fs.String("config", "", "Path to an EasyTier config file")
		runFor := fs.Duration("run-for", 3*time.Second, "How long to keep EasyTier running before stopping; 0 waits for Ctrl+C")
		_ = fs.Parse(args[1:])
		requireConfig(*configPath)

		if err := backend.StartEasyTier(*configPath); err != nil {
			fatalf("start EasyTier failed: %v", err)
		}
		fmt.Println("started EasyTier")
		printSnapshot(backend)

		if *runFor == 0 {
			waitForSignal()
		} else {
			time.Sleep(*runFor)
		}

		if err := backend.StopEasyTier(); err != nil {
			fatalf("stop EasyTier failed: %v", err)
		}
		fmt.Println("stopped EasyTier")
		printSnapshot(backend)

	case "stop":
		if err := backend.StopEasyTier(); err != nil {
			fatalf("stop EasyTier failed: %v", err)
		}
		fmt.Println("stopped EasyTier")
		printSnapshot(backend)

	case "status":
		printSnapshot(backend)

	default:
		easyTierUsage()
		os.Exit(2)
	}
}

func runStackCommand(backend *app.App, args []string) {
	if len(args) < 1 {
		stackUsage()
		os.Exit(2)
	}

	switch args[0] {
	case "smoke":
		fs := flag.NewFlagSet("stack smoke", flag.ExitOnError)
		frpConfig := fs.String("frp-config", "", "Path to frpc config")
		easyTierConfig := fs.String("easytier-config", "", "Path to EasyTier config")
		runFor := fs.Duration("run-for", 3*time.Second, "How long to keep both components running before stopping; 0 waits for Ctrl+C")
		graceful := fs.Duration("graceful", 500*time.Millisecond, "Graceful shutdown duration to pass to frp")
		stopWait := fs.Duration("stop-wait", 5*time.Second, "How long to wait for frp to stop")
		_ = fs.Parse(args[1:])

		if *frpConfig == "" {
			fatalf("missing --frp-config")
		}
		if *easyTierConfig == "" {
			fatalf("missing --easytier-config")
		}

		if err := backend.ValidateFRP(*frpConfig); err != nil {
			fatalf("frp config invalid: %v", err)
		}
		fmt.Println("validated frp config")

		if err := backend.StartEasyTier(*easyTierConfig); err != nil {
			fatalf("start EasyTier failed: %v", err)
		}
		fmt.Println("started EasyTier")

		if err := backend.StartFRP(*frpConfig); err != nil {
			_ = backend.StopEasyTier()
			fatalf("start frp failed: %v", err)
		}
		fmt.Println("started frp client")
		printSnapshot(backend)

		if *runFor == 0 {
			waitForSignal()
		} else {
			time.Sleep(*runFor)
		}

		if err := backend.StopFRP(*graceful, *stopWait); err != nil {
			fatalf("stop frp failed: %v", err)
		}
		fmt.Println("stopped frp client")

		if err := backend.StopEasyTier(); err != nil {
			fatalf("stop EasyTier failed: %v", err)
		}
		fmt.Println("stopped EasyTier")
		printSnapshot(backend)

	default:
		stackUsage()
		os.Exit(2)
	}
}

func requireConfig(configPath string) {
	if configPath == "" {
		fatalf("missing --config")
	}
}

func printSnapshot(backend *app.App) {
	data, err := backend.SnapshotJSON()
	if err != nil {
		fatalf("render snapshot failed: %v", err)
	}
	fmt.Println(string(data))
}

func waitForSignal() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	<-sigCh
}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  poc frp validate --config <path>")
	fmt.Println("  poc frp smoke --config <path> [--run-for 3s] [--graceful 500ms] [--stop-wait 5s]")
	fmt.Println("  poc easytier start --config <path>")
	fmt.Println("  poc easytier smoke --config <path> [--run-for 3s]")
	fmt.Println("  poc easytier stop")
	fmt.Println("  poc easytier status")
	fmt.Println("  poc stack smoke --frp-config <path> --easytier-config <path> [--run-for 3s]")
	fmt.Println("  poc snapshot")
}

func frpUsage() {
	fmt.Println("Usage:")
	fmt.Println("  poc frp validate --config <path>")
	fmt.Println("  poc frp smoke --config <path> [--run-for 3s]")
}

func easyTierUsage() {
	fmt.Println("Usage:")
	fmt.Println("  poc easytier start --config <path>")
	fmt.Println("  poc easytier smoke --config <path> [--run-for 3s]")
	fmt.Println("  poc easytier stop")
	fmt.Println("  poc easytier status")
}

func stackUsage() {
	fmt.Println("Usage:")
	fmt.Println("  poc stack smoke --frp-config <path> --easytier-config <path> [--run-for 3s]")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
