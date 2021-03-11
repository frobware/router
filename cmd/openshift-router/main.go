package main

import (
	"flag"
	"fmt"
	"math/rand"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/klog"

	"github.com/openshift/router/pkg/cmd/infra/router"

	"github.com/pkg/profile"
)

func main() {
	// defer serviceability.BehaviorOnPanic(os.Getenv("OPENSHIFT_ON_PANIC"), version.Get())()
	// defer serviceability.Profile(os.Getenv("OPENSHIFT_PROFILE")).Stop()
	defer profile.Start(profile.CPUProfile).Stop()
	rand.Seed(time.Now().UTC().UnixNano())

	cmd := CommandFor(filepath.Base(os.Args[0]))
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	logFlags := flag.FlagSet{}
	klog.InitFlags(&logFlags)
	cmd.PersistentFlags().AddGoFlagSet(&logFlags)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// CommandFor returns the appropriate command for this base name,
// or the OpenShift CLI command.
func CommandFor(basename string) *cobra.Command {
	var cmd *cobra.Command

	switch basename {
	case "openshift-router", "openshift-haproxy-router":
		cmd = router.NewCommandTemplateRouter(basename)
	default:
		fmt.Printf("unknown command name: %s\n", basename)
		os.Exit(1)
	}

	return cmd
}
