package proxytest

import (
	"reflect"
	"testing"

	"mtu-tuner/internal/core"
)

func TestPlanSuiteUsesCurrentDefaultTargets(t *testing.T) {
	t.Parallel()

	browserPlan, err := planSuite(core.TestRunRequest{
		TestProfile: "browser",
		TestTargets: core.DefaultSavedSettings().TestTargets,
	})
	if err != nil {
		t.Fatalf("planSuite(browser) error = %v", err)
	}
	if got := len(browserPlan.specs); got != len(core.BrowserTestSpecs) {
		t.Fatalf("len(planSuite(browser).specs) = %d, want %d", got, len(core.BrowserTestSpecs))
	}
	if browserPlan.specs[0].name != "yt_page#1" {
		t.Fatalf("planSuite(browser).specs[0].name = %q, want yt_page#1", browserPlan.specs[0].name)
	}

	stressPlan, err := planSuite(core.TestRunRequest{
		TestProfile: "stress",
		TestTargets: core.DefaultSavedSettings().TestTargets,
	})
	if err != nil {
		t.Fatalf("planSuite(stress) error = %v", err)
	}
	if got := len(stressPlan.specs); got != len(core.StressTestSpecs)*core.DefaultStressRounds {
		t.Fatalf("len(planSuite(stress).specs) = %d, want %d", got, len(core.StressTestSpecs)*core.DefaultStressRounds)
	}

	quickPlan, err := planSuite(core.TestRunRequest{
		TestProfile: "quick",
		TestTargets: core.DefaultSavedSettings().TestTargets,
	})
	if err != nil {
		t.Fatalf("planSuite(quick) error = %v", err)
	}
	if got := len(quickPlan.specs); got != 6 {
		t.Fatalf("len(planSuite(quick).specs) = %d, want 6", got)
	}

	chromePlan, err := planSuite(core.TestRunRequest{
		TestProfile: "chrome",
		TestTargets: core.DefaultSavedSettings().TestTargets,
	})
	if err != nil {
		t.Fatalf("planSuite(chrome) error = %v", err)
	}
	if got := len(chromePlan.chromeTargets); got != len(core.ChromeProbeTargets) {
		t.Fatalf("len(planSuite(chrome).chromeTargets) = %d, want %d", got, len(core.ChromeProbeTargets))
	}
}

func TestPlanSuiteFiltersConfiguredTargetsByProfileAndOrder(t *testing.T) {
	t.Parallel()

	plan, err := planSuite(core.TestRunRequest{
		TestProfile: "browser",
		TestTargets: []core.TestTarget{
			{Name: "later", URL: "https://www.google.com/", Enabled: true, Profiles: []string{"browser"}, Order: 20},
			{Name: "disabled", URL: "https://www.gstatic.com/generate_204", Enabled: false, Profiles: []string{"browser"}, Order: 5},
			{Name: "wrong-profile", URL: "https://example.com/only-chrome", Enabled: true, Profiles: []string{"chrome"}, Order: 1},
			{Name: "earlier", URL: "https://www.youtube.com/", Enabled: true, Profiles: []string{"browser", "quick"}, Order: 10},
		},
	})
	if err != nil {
		t.Fatalf("planSuite(browser) error = %v", err)
	}

	gotNames := []string{plan.specs[0].spec.Name, plan.specs[1].spec.Name}
	wantNames := []string{"earlier", "later"}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("planSuite(browser) names = %#v, want %#v", gotNames, wantNames)
	}
}
