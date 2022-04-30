// Copyright 2022 Blockdaemon Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package fetch provides the `fetch` command.
package fetch

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
	"go.blockdaemon.com/solana/cluster-manager/internal/fetch"
	"go.blockdaemon.com/solana/cluster-manager/internal/ledger"
	"go.blockdaemon.com/solana/cluster-manager/internal/logger"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var Cmd = cobra.Command{
	Use:   "fetch",
	Short: "Snapshot downloader",
	Long:  "Fetches a snapshot from another node using the tracker API.",
	Run: func(_ *cobra.Command, _ []string) {
		run()
	},
}

var (
	ledgerDir  string
	trackerURL string
	minSnapAge uint64
	maxSnapAge uint64
)

func init() {
	flags := Cmd.Flags()
	flags.StringVar(&ledgerDir, "ledger", "", "Path to ledger dir")
	flags.StringVar(&trackerURL, "tracker", "", "Tracker URL")
	flags.Uint64Var(&minSnapAge, "min-slots", 500, "Download only snapshots <n> slots newer than local")
	flags.Uint64Var(&maxSnapAge, "max-slots", 10000, "Refuse to download <n> slots older than the newest")
}

func run() {
	log := logger.GetConsoleLogger()
	ctx := context.TODO()

	// Check what snapshots we have locally.
	localSnaps, err := ledger.ListSnapshots(os.DirFS(ledgerDir))
	if err != nil {
		log.Fatal("Failed to check existing snapshots", zap.Error(err))
	}

	// Ask tracker for best snapshots.
	trackerClient := fetch.NewTrackerClient(trackerURL)
	remoteSnaps, err := trackerClient.GetBestSnapshots(ctx, -1)
	if err != nil {
		log.Fatal("Failed to request snapshot info", zap.Error(err))
	}

	_, advice := fetch.ShouldFetchSnapshot(localSnaps, remoteSnaps, minSnapAge, maxSnapAge)
	switch advice {
	case fetch.AdviceNothingFound:
		log.Error("No snapshots available remotely")
		return
	case fetch.AdviceUpToDate:
		log.Info("Existing snapshot is recent enough, no download needed",
			zap.Uint64("existing_slot", localSnaps[0].Slot))
		return
	case fetch.AdviceFetch:
	}

	// Print snapshot to user.
	snap := &remoteSnaps[0]
	buf, _ := json.MarshalIndent(snap, "", "\t")
	log.Info("Downloading a snapshot", zap.ByteString("snap", buf))

	// Setup progress bars for download.
	bars := mpb.New()
	sidecarClient := fetch.NewSidecarClient(snap.Target)
	sidecarClient.SetProxyReaderFunc(func(name string, size int64, rd io.Reader) io.ReadCloser {
		bar := bars.New(
			size,
			mpb.BarStyle(),
			mpb.PrependDecorators(decor.Name(name)),
			mpb.AppendDecorators(
				decor.AverageSpeed(decor.UnitKB, "% .1f"),
				decor.Percentage(),
			),
		)
		return bar.ProxyReader(rd)
	})

	// Download.
	beforeDownload := time.Now()
	group, ctx := errgroup.WithContext(ctx)
	for _, file := range snap.Files {
		file_ := file
		group.Go(func() error {
			err := sidecarClient.DownloadSnapshotFile(ctx, ".", file_.FileName)
			if err != nil {
				log.Error("Download failed",
					zap.String("snapshot", file_.FileName))
			}
			return err
		})
	}
	downloadErr := group.Wait()
	downloadDuration := time.Since(beforeDownload)

	if downloadErr == nil {
		log.Info("Download completed", zap.Duration("download_time", downloadDuration))
	} else {
		log.Info("Aborting download", zap.Duration("download_time", downloadDuration))
	}
}
