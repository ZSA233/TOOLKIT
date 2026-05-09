package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	projectapp "mtu-tuner/internal/app"
	"mtu-tuner/internal/core"
)

func main() {
	os.Exit(run())
}

func run() int {
	service, err := projectapp.NewDefaultService("", nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	projectapp.SetDefaultService(service)

	var (
		noGUI        bool
		probe        = flag.String("probe", core.DefaultProbe, "Route probe IP/domain.")
		fallback     = flag.String("fallback-probe", core.DefaultFallbackProbe, "Fallback route probe.")
		proxy        = flag.String("proxy", core.DefaultHTTPProxy, "HTTP proxy URL.")
		controller   = flag.String("controller", core.DefaultController, "Clash controller URL.")
		secret       = flag.String("secret", "", "Clash API secret.")
		group        = flag.String("group", core.DefaultClashGroup, "Clash proxy group.")
		configPath   = flag.String("config", "", "Clash config path.")
		clashCurrent = flag.Bool("clash-current", false, "Force current Clash node as route probe.")
		detect       = flag.Bool("detect", false, "Detect interface.")
		quickTest    = flag.Bool("quick-test", false, "Run connectivity test.")
		testProfile  = flag.String("test-profile", core.DefaultTestProfile, "Test profile.")
		rounds       = flag.Int("rounds", 0, "Override rounds.")
		concurrency  = flag.Int("concurrency", 0, "Override concurrency.")
		browser      = flag.String("browser", "", "Browser executable path.")
		setActive    = flag.Int("set-active", 0, "Set active MTU.")
		setPersistent = flag.Int("set-persistent", 0, "Set persistent MTU.")
	)
	flag.BoolVar(&noGUI, "nogui", false, "Compatibility flag.")
	flag.Parse()
	_ = noGUI

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	request := core.DetectRequest{
		Probe:         *probe,
		FallbackProbe: *fallback,
		Controller:    *controller,
		Secret:        *secret,
		Group:         *group,
		ConfigPath:    *configPath,
		ClashCurrent:  *clashCurrent,
	}
	detectResult, err := service.DetectInterface(ctx, request)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if detectResult.Selection.Target != nil {
		if err := printJSON(map[string]any{
			"group_chain": detectResult.Selection.Target.Group,
			"leaf":        detectResult.Selection.Target.Leaf,
			"server":      detectResult.Selection.Target.Server,
			"port":        detectResult.Selection.Target.Port,
			"resolved_ip": detectResult.Selection.Target.ResolvedIP,
			"source":      detectResult.Selection.Target.Source,
			"config_path": detectResult.Selection.Target.ConfigPath,
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	} else if detectResult.Selection.Warning != "" {
		fmt.Fprintln(os.Stderr, "WARNING:", detectResult.Selection.Warning)
	}

	if *detect || *clashCurrent || core.IsAutoProbe(*probe) || (!*quickTest && *setActive == 0 && *setPersistent == 0) {
		if err := printJSON(detectResult.Interface); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}

	if *setActive > 0 {
		result, err := service.SetActiveMTU(ctx, detectResult.Interface, *setActive)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if result.Output != "" {
			fmt.Println(result.Output)
		}
		fmt.Printf("Effective MTU: %d\n", result.Interface.MTU)
	}
	if *setPersistent > 0 {
		result, err := service.SetPersistentMTU(ctx, detectResult.Interface, *setPersistent)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if result.Output != "" {
			fmt.Println(result.Output)
		}
		fmt.Printf("Effective MTU: %d\n", result.Interface.MTU)
	}
	if *quickTest {
		summary, err := service.RunTestSync(ctx, core.TestRunRequest{
			Interface:   detectResult.Interface,
			HTTPProxy:   *proxy,
			TestProfile: *testProfile,
			BrowserPath: *browser,
			Rounds:      *rounds,
			Concurrency: *concurrency,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if err := printJSON(summary); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	return 0
}

func printJSON(value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(payload))
	return nil
}
