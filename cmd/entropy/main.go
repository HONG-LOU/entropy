package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"entropy/internal/node"
)

type stringListFlag struct {
	values []string
}

func (f *stringListFlag) String() string {
	return strings.Join(f.values, ",")
}

func (f *stringListFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("value must not be empty")
	}
	f.values = append(f.values, value)
	return nil
}

type nodeCLIOptions struct {
	dataDirectory         string
	listenAddress         string
	peer                  string
	mine                  bool
	seedMode              bool
	pruneDepth            uint64
	pruneDepthSet         bool
	disableDiscovery      bool
	bootstrapManifestURLs []string
	trustLoopbackProxy    bool
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "node":
		err = runNode(os.Args[2:])
	case "status":
		err = showStatus(os.Args[2:])
	case "mine-one":
		err = mineOne(os.Args[2:])
	case "history":
		err = showHistory(os.Args[2:])
	case "wallet-backup":
		err = backupWallet(os.Args[2:])
	case "wallet-migrate":
		err = migrateWallet(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func runNode(arguments []string) error {
	options, err := parseNodeOptions(arguments)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	service, err := node.NewContext(ctx, node.Config{
		DataDirectory:         options.dataDirectory,
		ListenAddress:         options.listenAddress,
		PruneDepth:            options.pruneDepth,
		PruneDepthSet:         options.pruneDepthSet,
		DisableDiscovery:      options.disableDiscovery,
		SeedMode:              options.seedMode,
		BootstrapManifestURLs: options.bootstrapManifestURLs,
		TrustLoopbackProxy:    options.trustLoopbackProxy,
	})
	if err != nil {
		return err
	}
	if err := service.Start(ctx); err != nil {
		return err
	}
	if options.peer != "" {
		if err := service.AddPeer(options.peer); err != nil {
			return err
		}
	}
	if options.mine {
		if err := service.StartMining(); err != nil {
			return err
		}
	}
	dashboard, err := service.Dashboard()
	if err != nil {
		return err
	}
	fmt.Printf("Entropy node running\naddress: %s\nlisten:  %s\nheight:  %d\n", dashboard.Address, dashboard.ListenAddress, dashboard.Height)
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return service.Close(shutdown)
}

func parseNodeOptions(arguments []string) (nodeCLIOptions, error) {
	flags := flag.NewFlagSet("node", flag.ContinueOnError)
	data := flags.String("data", "", "data directory (default: user config directory)")
	listen := flags.String("listen", node.DefaultListenAddress, "P2P listen address")
	peer := flags.String("peer", "", "peer URL, for example http://192.168.1.20:47821")
	mine := flags.Bool("mine", false, "start mining immediately")
	seedMode := flags.Bool("seed-mode", false, "run an archive relay with an ephemeral non-financial identity")
	pruneDepth := flags.Uint64("prune-depth", 0, "retain this many recent block bodies (0 = archive)")
	noDiscovery := flags.Bool("no-discovery", false, "disable LAN multicast peer discovery")
	noBootstrap := flags.Bool("no-bootstrap", false, "disable public bootstrap manifests")
	trustLoopbackProxy := flags.Bool("trust-loopback-proxy", false, "trust one proxy client IP header from a loopback TCP peer")
	var bootstrapManifests stringListFlag
	flags.Var(&bootstrapManifests, "bootstrap-manifest", "HTTPS bootstrap manifest URL; repeat to configure fallback sources")
	if err := flags.Parse(arguments); err != nil {
		return nodeCLIOptions{}, err
	}
	pruneDepthSet := false
	flags.Visit(func(item *flag.Flag) {
		if item.Name == "prune-depth" {
			pruneDepthSet = true
		}
	})
	if *noBootstrap && len(bootstrapManifests.values) > 0 {
		return nodeCLIOptions{}, fmt.Errorf("--no-bootstrap cannot be combined with --bootstrap-manifest")
	}
	if *seedMode && *mine {
		return nodeCLIOptions{}, fmt.Errorf("--seed-mode cannot be combined with --mine")
	}
	if *seedMode && pruneDepthSet && *pruneDepth != 0 {
		return nodeCLIOptions{}, fmt.Errorf("--seed-mode requires --prune-depth 0")
	}
	manifestURLs := node.DefaultBootstrapManifestURLs()
	if *noBootstrap {
		manifestURLs = nil
	} else if len(bootstrapManifests.values) > 0 {
		manifestURLs = append([]string(nil), bootstrapManifests.values...)
	}
	return nodeCLIOptions{
		dataDirectory:         *data,
		listenAddress:         *listen,
		peer:                  *peer,
		mine:                  *mine,
		seedMode:              *seedMode,
		pruneDepth:            *pruneDepth,
		pruneDepthSet:         pruneDepthSet,
		disableDiscovery:      *noDiscovery,
		bootstrapManifestURLs: manifestURLs,
		trustLoopbackProxy:    *trustLoopbackProxy,
	}, nil
}

func showStatus(arguments []string) error {
	flags := flag.NewFlagSet("status", flag.ContinueOnError)
	data := flags.String("data", "", "data directory")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	service, err := node.New(node.Config{DataDirectory: *data, ListenAddress: "127.0.0.1:0"})
	if err != nil {
		return err
	}
	defer func() { _ = service.Close(context.Background()) }()
	dashboard, err := service.Dashboard()
	if err != nil {
		return err
	}
	fmt.Printf("address:   %s\nbalance:   %s ENT\nheight:    %d\npending:   %d\npeers:     %d\nissued:    %s / %s ENT\n", dashboard.Address, dashboard.ConfirmedBalance, dashboard.Height, dashboard.PendingCount, dashboard.PeerCount, dashboard.Issued, dashboard.MaxSupply)
	return nil
}

func mineOne(arguments []string) error {
	flags := flag.NewFlagSet("mine-one", flag.ContinueOnError)
	data := flags.String("data", "", "data directory")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	service, err := node.New(node.Config{DataDirectory: *data, ListenAddress: "127.0.0.1:0"})
	if err != nil {
		return err
	}
	defer func() { _ = service.Close(context.Background()) }()
	block, err := service.MineOnce(context.Background())
	if err != nil {
		return err
	}
	fmt.Printf("mined block %d %s\n", block.Height, block.Hash)
	return nil
}

func showHistory(arguments []string) error {
	flags := flag.NewFlagSet("history", flag.ContinueOnError)
	data := flags.String("data", "", "data directory")
	limit := flags.Int("limit", 20, "maximum history rows")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	service, err := node.New(node.Config{DataDirectory: *data, ListenAddress: "127.0.0.1:0"})
	if err != nil {
		return err
	}
	defer func() { _ = service.Close(context.Background()) }()
	history, err := service.TransactionHistory(*limit)
	if err != nil {
		return err
	}
	for _, transaction := range history {
		state := fmt.Sprintf("%d confirmations", transaction.Confirmations)
		if transaction.Pending {
			state = "pending"
		}
		fmt.Printf("%s  received=%s sent=%s  %s\n", transaction.ID, transaction.Received, transaction.Sent, state)
	}
	return nil
}

func backupWallet(arguments []string) error {
	flags := flag.NewFlagSet("wallet-backup", flag.ContinueOnError)
	data := flags.String("data", "", "data directory")
	output := flags.String("output", "", "destination .entwallet file")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *output == "" {
		return fmt.Errorf("--output is required")
	}
	password := os.Getenv("ENTROPY_WALLET_PASSWORD")
	if password == "" {
		return fmt.Errorf("ENTROPY_WALLET_PASSWORD is required")
	}
	service, err := node.New(node.Config{DataDirectory: *data, ListenAddress: "127.0.0.1:0"})
	if err != nil {
		return err
	}
	defer func() { _ = service.Close(context.Background()) }()
	return service.ExportWalletBackup(*output, password)
}

func migrateWallet(arguments []string) error {
	flags := flag.NewFlagSet("wallet-migrate", flag.ContinueOnError)
	data := flags.String("data", "", "data directory")
	output := flags.String("output", "", "required portable .entwallet backup")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *output == "" {
		return fmt.Errorf("--output is required")
	}
	password := os.Getenv("ENTROPY_WALLET_PASSWORD")
	if password == "" {
		return fmt.Errorf("ENTROPY_WALLET_PASSWORD is required")
	}
	return node.MigrateLegacyWallet(*data, *output, password)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: entropy <node|status|mine-one|history|wallet-backup|wallet-migrate> [options]")
}
