package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"entropy/internal/node"
)

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
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func runNode(arguments []string) error {
	flags := flag.NewFlagSet("node", flag.ContinueOnError)
	data := flags.String("data", "", "data directory (default: user config directory)")
	listen := flags.String("listen", node.DefaultListenAddress, "P2P listen address")
	peer := flags.String("peer", "", "peer URL, for example http://192.168.1.20:47821")
	mine := flags.Bool("mine", false, "start mining immediately")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	service, err := node.New(node.Config{DataDirectory: *data, ListenAddress: *listen})
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := service.Start(ctx); err != nil {
		return err
	}
	if *peer != "" {
		if err := service.AddPeer(*peer); err != nil {
			return err
		}
	}
	if *mine {
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

func usage() {
	fmt.Fprintln(os.Stderr, "usage: entropy <node|status|mine-one> [options]")
}
