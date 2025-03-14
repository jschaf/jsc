// publish deploys the contents of the public directory to firebase.
package main

import (
	"flag"
	"fmt"
	"golang.org/x/oauth2/google"
	"log/slog"
	"os"
	"time"

	"github.com/jschaf/jsc/pkg/dirs"
	"github.com/jschaf/jsc/pkg/firebase"
	"github.com/jschaf/jsc/pkg/log"
	"github.com/jschaf/jsc/pkg/process"
	"golang.org/x/net/context"
	hosting "google.golang.org/api/firebasehosting/v1beta1"
)

const (
	siteName   = "jschaf"
	siteParent = "sites/" + siteName
)

func main() {
	process.RunMain(runMain)
}

// servingConfig returns the known fields for the Firebase hosting config. This
// corresponds to the hosting field in firebase.json.
func servingConfig() *hosting.ServingConfig {
	return &hosting.ServingConfig{
		TrailingSlashBehavior: "REMOVE",
		Rewrites: []*hosting.Rewrite{
			{
				Glob: "/_/heap/**",
				Run: &hosting.CloudRunRewrite{
					Region:    "us-west2",
					ServiceId: "track-server",
				},
			},
		},
	}
}

func runMain(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*60)
	defer cancel()

	fset := flag.CommandLine
	logLevel := log.DefineFlags(fset)
	if err := fset.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	slog.SetDefault(slog.New(log.NewDevHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})))

	slog.Info("start deployment")
	start := time.Now()

	creds, err := google.FindDefaultCredentials(ctx, hosting.FirebaseScope)
	if err != nil {
		return fmt.Errorf("find default credentials: %w", err)
	}

	svc, err := hosting.NewService(ctx)
	if err != nil {
		return fmt.Errorf("new hosting service: %w", err)
	}
	versionSvc := svc.Projects.Sites.Versions

	// Create the version: we'll eventually release this version.
	createVersionStart := time.Now()
	createVersion := versionSvc.Create(siteParent, &hosting.Version{
		Config: servingConfig(),
	})
	createVersion.Context(ctx)
	version, err := createVersion.Do()
	if err != nil {
		return fmt.Errorf("create site version: %w", err)
	}
	slog.Info("create new version", "version", version.Name, "duration", time.Since(createVersionStart))

	// Populate files: get the SHA256 hash of all gzipped files in the public
	// directory, send them to Firebase with the URL that serves the file.
	// Firebase returns the SHA256 hashes of the files we need to upload to
	// firebase.
	siteHashes := firebase.NewSiteHashes()
	if err := siteHashes.PopulateFromDir(dirs.Dist); err != nil {
		return fmt.Errorf("populate from dir: %w", err)
	}
	popFilesStart := time.Now()
	popFilesReq := hosting.PopulateVersionFilesRequest{Files: siteHashes.HashesByURL()}
	popFiles := versionSvc.PopulateFiles(version.Name, &popFilesReq)
	popFiles.Context(ctx)
	popFilesResp, err := popFiles.Do()
	if err != nil {
		return fmt.Errorf("populate files: %w", err)
	}
	slog.Info("populate files response requests", "count", len(popFilesResp.UploadRequiredHashes), "duration", time.Since(popFilesStart))

	// Upload files: only upload files that have a SHA256 hash in the
	// populateFiles response.
	filesToUpload, err := siteHashes.FindFilesForHashes(popFilesResp.UploadRequiredHashes)
	if err != nil {
		return fmt.Errorf("find files for hashes: %w", err)
	}
	uploader := firebase.NewUploader(siteHashes, popFilesResp.UploadUrl, creds.TokenSource)
	if err := uploader.UploadAll(ctx, filesToUpload); err != nil {
		return fmt.Errorf("upload all: %w", err)
	}

	// Finalize the version: prevent adding any new resources.
	versionFinal := hosting.Version{Status: "FINALIZED"}
	patchVersion := versionSvc.Patch(version.Name, &versionFinal)
	patchVersion.Context(ctx)
	patchVersionResp, err := patchVersion.Do()
	if err != nil {
		return fmt.Errorf("finalize version: %w", err)
	}
	if patchVersionResp.Status != "FINALIZED" {
		return fmt.Errorf("finalize version status not 'FINALIZED', got %q", patchVersionResp.Status)
	}

	// Release version: promote a version to release so it's shown on the website.
	release := hosting.Release{}
	createRelease := svc.Sites.Releases.Create(siteParent, &release)
	createRelease.Context(ctx)
	createRelease.VersionName(patchVersionResp.Name)
	createReleaseResp, err := createRelease.Do()
	if err != nil {
		return fmt.Errorf("create release: %w", err)
	}
	slog.Info("created release", "name", createReleaseResp.Name)

	slog.Info("completed deployment", "duration", time.Since(start))
	return nil
}
