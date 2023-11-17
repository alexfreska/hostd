package settings_test

import (
	"path/filepath"
	"testing"
	"time"

	"go.sia.tech/core/types"
	"go.sia.tech/hostd/alerts"
	"go.sia.tech/hostd/host/settings"
	"go.sia.tech/hostd/internal/test"
	"go.sia.tech/hostd/persist/sqlite"
	"go.uber.org/zap/zaptest"
	"lukechampine.com/frand"
)

func TestAutoAnnounce(t *testing.T) {
	hostKey := types.NewPrivateKeyFromSeed(frand.Bytes(32))
	dir := t.TempDir()
	log := zaptest.NewLogger(t)
	node, err := test.NewWallet(hostKey, dir, log.Named("wallet"))
	if err != nil {
		t.Fatal(err)
	}
	defer node.Close()

	// fund the wallet
	if err := node.MineBlocks(node.Address(), 99); err != nil {
		t.Fatal(err)
	}

	db, err := sqlite.OpenDatabase(filepath.Join(dir, "hostd.db"), log.Named("sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	a := alerts.NewManager()
	manager, err := settings.NewConfigManager(dir, hostKey, "localhost:9882", db, node.ChainManager(), node.TPool(), node, a, log.Named("settings"))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	settings := settings.DefaultSettings
	settings.NetAddress = "foo.bar:1234"
	manager.UpdateSettings(settings)

	// trigger an auto-announce
	if err := node.MineBlocks(node.Address(), 1); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)

	// confirm the announcement
	if err := node.MineBlocks(node.Address(), 2); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)

	lastAnnouncement, err := manager.LastAnnouncement()
	if err != nil {
		t.Fatal(err)
	} else if lastAnnouncement.Index.Height != 101 {
		t.Fatalf("expected height 100, got %v", lastAnnouncement.Index.Height)
	} else if lastAnnouncement.Address != "foo.bar:1234" {
		t.Fatal("announcement not updated")
	}
	lastHeight := lastAnnouncement.Index.Height

	// mine until right before the next announcement to ensure that the
	// announcement is not triggered early
	if err := node.MineBlocks(node.Address(), 99); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)

	lastAnnouncement, err = manager.LastAnnouncement()
	if err != nil {
		t.Fatal(err)
	} else if lastAnnouncement.Index.Height != lastHeight {
		t.Fatal("announcement triggered unexpectedly")
	}

	// trigger an auto-announce
	if err := node.MineBlocks(node.Address(), 1); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)

	// confirm the announcement
	if err := node.MineBlocks(node.Address(), 2); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)

	nextHeight := lastHeight + 1 + 100 // off-by-one because the announcement is mined in the next block
	lastAnnouncement, err = manager.LastAnnouncement()
	if err != nil {
		t.Fatal(err)
	} else if lastAnnouncement.Index.Height != nextHeight {
		t.Fatalf("expected height %v, got %v", nextHeight, lastAnnouncement.Index.Height)
	} else if lastAnnouncement.Address != "foo.bar:1234" {
		t.Fatal("announcement not updated")
	}
}
