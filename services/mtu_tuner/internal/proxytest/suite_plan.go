package proxytest

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"

	"mtu-tuner/internal/core"
)

const (
	targetKindPageLarge    = "page_large"
	targetKindPageStandard = "page_standard"
	targetKindImage        = "image"
	targetKindNoContent    = "no_content"
)

type suitePlan struct {
	profile       string
	specs         []namedSpec
	chromeTargets []core.ChromeProbeTarget
	concurrency   int
	rounds        int
}

type resolvedTestTarget struct {
	name string
	url  *url.URL
	kind string
}

func planSuite(request core.TestRunRequest) (suitePlan, error) {
	profile := strings.ToLower(strings.TrimSpace(request.TestProfile))
	if profile == "" {
		profile = core.DefaultTestProfile
	}

	targets := configuredTargetsForProfile(profile, request.TestTargets)
	switch profile {
	case "quick":
		return suitePlan{
			profile:     "quick",
			specs:       buildQuickSpecs(targets),
			concurrency: 1,
			rounds:      1,
		}, nil
	case "browser":
		rounds := pickRounds(request.Rounds, core.DefaultBrowserRounds)
		return suitePlan{
			profile:     "browser",
			specs:       expandResolvedTargets(targets, rounds),
			concurrency: pickConcurrency(request.Concurrency, 6),
			rounds:      rounds,
		}, nil
	case "stress":
		rounds := pickRounds(request.Rounds, core.DefaultStressRounds)
		return suitePlan{
			profile:     "stress",
			specs:       expandResolvedTargets(targets, rounds),
			concurrency: pickConcurrency(request.Concurrency, 10),
			rounds:      rounds,
		}, nil
	case "chrome":
		rounds := pickRounds(request.Rounds, core.DefaultChromeRounds)
		return suitePlan{
			profile:       "chrome",
			chromeTargets: buildChromeTargets(targets),
			concurrency:   1,
			rounds:        rounds,
		}, nil
	default:
		return suitePlan{}, fmt.Errorf("unknown test profile: %s", profile)
	}
}

func configuredTargetsForProfile(profile string, configured []core.TestTarget) []resolvedTestTarget {
	targets := configured
	if len(targets) == 0 {
		targets = core.DefaultTestTargets()
	}

	type indexedTarget struct {
		index  int
		target core.TestTarget
	}
	filtered := make([]indexedTarget, 0, len(targets))
	for index, target := range targets {
		if !target.Enabled || !targetSupportsProfile(target, profile) {
			continue
		}
		filtered = append(filtered, indexedTarget{
			index:  index,
			target: target,
		})
	}

	sort.SliceStable(filtered, func(i int, j int) bool {
		if filtered[i].target.Order == filtered[j].target.Order {
			return filtered[i].index < filtered[j].index
		}
		return filtered[i].target.Order < filtered[j].target.Order
	})

	resolved := make([]resolvedTestTarget, 0, len(filtered))
	for _, item := range filtered {
		parsedURL, err := url.Parse(strings.TrimSpace(item.target.URL))
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			continue
		}
		name := strings.TrimSpace(item.target.Name)
		if name == "" {
			name = parsedURL.Hostname()
		}
		resolved = append(resolved, resolvedTestTarget{
			name: name,
			url:  parsedURL,
			kind: classifyTestTarget(parsedURL),
		})
	}
	return resolved
}

func targetSupportsProfile(target core.TestTarget, profile string) bool {
	for _, candidate := range target.Profiles {
		if strings.EqualFold(strings.TrimSpace(candidate), profile) {
			return true
		}
	}
	return false
}

func classifyTestTarget(targetURL *url.URL) string {
	extension := strings.ToLower(path.Ext(targetURL.Path))
	switch extension {
	case ".avif", ".gif", ".jpeg", ".jpg", ".png", ".svg", ".webp":
		return targetKindImage
	}
	if strings.Contains(strings.ToLower(targetURL.Path), "generate_204") {
		return targetKindNoContent
	}
	if strings.Contains(strings.ToLower(targetURL.Hostname()), "youtube.com") {
		return targetKindPageLarge
	}
	return targetKindPageStandard
}

func expandResolvedTargets(targets []resolvedTestTarget, rounds int) []namedSpec {
	specs := make([]namedSpec, 0, len(targets)*maxInt(1, rounds))
	for round := 0; round < maxInt(1, rounds); round++ {
		for _, target := range targets {
			spec := specForResolvedTarget(target, http.MethodGet)
			spec.Name = target.name
			specs = append(specs, namedSpec{
				name: fmt.Sprintf("%s#%d", target.name, round+1),
				spec: spec,
			})
		}
	}
	return specs
}

func buildQuickSpecs(targets []resolvedTestTarget) []namedSpec {
	specs := make([]namedSpec, 0, len(targets)*2)
	for _, target := range targets {
		host := strings.ToLower(target.url.Hostname())
		switch {
		case target.kind == targetKindPageLarge && strings.Contains(host, "youtube.com"):
			specs = append(specs,
				quickNamedSpec("yt_head", specForResolvedTarget(target, http.MethodHead), 1),
				quickNamedSpec("yt_head", specForResolvedTarget(target, http.MethodHead), 2),
				quickNamedSpec("yt_head", specForResolvedTarget(target, http.MethodHead), 3),
				quickNamedSpec("yt_get", specForResolvedTarget(target, http.MethodGet), 1),
			)
		case target.kind == targetKindPageStandard && strings.Contains(host, "google.com"):
			specs = append(specs,
				quickNamedSpec("google_head", specForResolvedTarget(target, http.MethodHead), 1),
				quickNamedSpec("google_head", specForResolvedTarget(target, http.MethodHead), 2),
			)
		case target.kind == targetKindImage:
			specs = append(specs, quickNamedSpec(target.name+"_get", specForResolvedTarget(target, http.MethodGet), 1))
		default:
			specs = append(specs,
				quickNamedSpec(target.name+"_head", specForResolvedTarget(target, http.MethodHead), 1),
				quickNamedSpec(target.name+"_get", specForResolvedTarget(target, http.MethodGet), 1),
			)
		}
	}
	return specs
}

func quickNamedSpec(name string, spec core.TestSpec, round int) namedSpec {
	spec.Name = name
	return namedSpec{
		name: fmt.Sprintf("%s#%d", name, round),
		spec: spec,
	}
}

func specForResolvedTarget(target resolvedTestTarget, method string) core.TestSpec {
	testSpec := core.TestSpec{
		Name:   target.name,
		Host:   target.url.Hostname(),
		Method: method,
		Path:   requestPath(target.url),
	}
	switch target.kind {
	case targetKindImage:
		testSpec.TimeoutSec = 10
		testSpec.ReadBodyBytes = 128 * 1024
	case targetKindNoContent:
		testSpec.TimeoutSec = 8
		testSpec.ReadBodyBytes = 0
	case targetKindPageLarge:
		testSpec.TimeoutSec = 16
		testSpec.ReadBodyBytes = 384 * 1024
	default:
		testSpec.TimeoutSec = 12
		testSpec.ReadBodyBytes = 192 * 1024
	}
	if strings.EqualFold(method, http.MethodHead) {
		testSpec.ReadBodyBytes = 0
	}
	return testSpec
}

func requestPath(targetURL *url.URL) string {
	requestPath := targetURL.EscapedPath()
	if requestPath == "" {
		requestPath = "/"
	}
	if targetURL.RawQuery == "" {
		return requestPath
	}
	return requestPath + "?" + targetURL.RawQuery
}

func buildChromeTargets(targets []resolvedTestTarget) []core.ChromeProbeTarget {
	chromeTargets := make([]core.ChromeProbeTarget, 0, len(targets))
	for _, target := range targets {
		kind := "fetch"
		if target.kind == targetKindImage {
			kind = "image"
		}
		chromeTargets = append(chromeTargets, core.ChromeProbeTarget{
			Name: target.name,
			Kind: kind,
			URL:  target.url.String(),
		})
	}
	return chromeTargets
}
