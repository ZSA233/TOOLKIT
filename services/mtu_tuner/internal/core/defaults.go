package core

const (
	AppName              = "mtu-tuner"
	AppDescription       = "Standalone MTU tuner desktop app"
	AppID                = "com.zsa.toolkit.mtu-tuner"
	ConfigDirName        = "mtu-tuner"
	LegacyConfigDirName  = "mtu-quick-tuner"
	BrowserEnvKey        = "MTU_TUNER_BROWSER"
	ConfigVersion        = 2
	DefaultProbe         = "auto"
	DefaultFallbackProbe = "1.1.1.1"
	DefaultHTTPProxy     = "http://127.0.0.1:7890"
	DefaultController    = "http://127.0.0.1:9097"
	DefaultClashGroup    = "auto"
	DefaultMTUList       = "1500,1480,1460,1440,1420,1400,1380,1360"
	DefaultQuickMTU      = 1400
	DefaultTestProfile   = "chrome"
	DefaultBrowserRounds = 1
	DefaultStressRounds  = 3
	DefaultChromeRounds  = 1
)

var QuickMTUValues = []int{1500, 1460, 1440, 1420, 1400, 1380, 1360}

var CommonClashGroups = []string{
	"节点选择",
	"🚀 节点选择",
	"PROXY",
	"Proxy",
	"proxy",
	"GLOBAL",
}

var TestProfiles = []string{"browser", "stress", "quick", "chrome"}

const BrowserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

type TestSpec struct {
	Name          string
	Host          string
	Method        string
	Path          string
	TimeoutSec    float64
	ReadBodyBytes int
}

type ChromeProbeTarget struct {
	Name string
	Kind string
	URL  string
}

func DefaultTestTargets() []TestTarget {
	return []TestTarget{
		{
			Name:     "yt_page",
			URL:      "https://www.youtube.com/",
			Enabled:  true,
			Profiles: []string{"browser", "stress", "quick"},
			Order:    10,
		},
		{
			Name:     "yt_204",
			URL:      "https://www.youtube.com/generate_204",
			Enabled:  true,
			Profiles: []string{"browser", "stress", "chrome"},
			Order:    20,
		},
		{
			Name:     "yt_thumb",
			URL:      "https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg",
			Enabled:  true,
			Profiles: []string{"browser", "stress", "chrome"},
			Order:    30,
		},
		{
			Name:     "google_page",
			URL:      "https://www.google.com/",
			Enabled:  true,
			Profiles: []string{"browser", "stress", "quick"},
			Order:    40,
		},
		{
			Name:     "google_204",
			URL:      "https://www.google.com/generate_204",
			Enabled:  true,
			Profiles: []string{"browser", "stress", "chrome"},
			Order:    50,
		},
		{
			Name:     "gstatic_204",
			URL:      "https://www.gstatic.com/generate_204",
			Enabled:  true,
			Profiles: []string{"browser", "stress", "chrome"},
			Order:    60,
		},
		{
			Name:     "google_logo",
			URL:      "https://www.google.com/images/branding/googlelogo/2x/googlelogo_color_272x92dp.png",
			Enabled:  true,
			Profiles: []string{"chrome"},
			Order:    65,
		},
		{
			Name:     "connectivity_204",
			URL:      "https://connectivitycheck.gstatic.com/generate_204",
			Enabled:  true,
			Profiles: []string{"browser", "stress"},
			Order:    70,
		},
		{
			Name:     "yt_page_2",
			URL:      "https://www.youtube.com/results?search_query=mtu",
			Enabled:  true,
			Profiles: []string{"stress"},
			Order:    80,
		},
		{
			Name:     "google_search",
			URL:      "https://www.google.com/search?q=mtu",
			Enabled:  true,
			Profiles: []string{"stress"},
			Order:    90,
		},
	}
}

var BrowserTestSpecs = []TestSpec{
	{Name: "yt_page", Host: "www.youtube.com", Method: "GET", Path: "/", TimeoutSec: 16, ReadBodyBytes: 384 * 1024},
	{Name: "yt_204", Host: "www.youtube.com", Method: "GET", Path: "/generate_204", TimeoutSec: 8, ReadBodyBytes: 0},
	{Name: "yt_thumb", Host: "i.ytimg.com", Method: "GET", Path: "/vi/dQw4w9WgXcQ/hqdefault.jpg", TimeoutSec: 10, ReadBodyBytes: 128 * 1024},
	{Name: "google_page", Host: "www.google.com", Method: "GET", Path: "/", TimeoutSec: 12, ReadBodyBytes: 192 * 1024},
	{Name: "google_204", Host: "www.google.com", Method: "GET", Path: "/generate_204", TimeoutSec: 8, ReadBodyBytes: 0},
	{Name: "gstatic_204", Host: "www.gstatic.com", Method: "GET", Path: "/generate_204", TimeoutSec: 8, ReadBodyBytes: 0},
	{Name: "connectivity_204", Host: "connectivitycheck.gstatic.com", Method: "GET", Path: "/generate_204", TimeoutSec: 8, ReadBodyBytes: 0},
}

var StressTestSpecs = append(append([]TestSpec{}, BrowserTestSpecs...),
	TestSpec{Name: "yt_page_2", Host: "www.youtube.com", Method: "GET", Path: "/results?search_query=mtu", TimeoutSec: 16, ReadBodyBytes: 384 * 1024},
	TestSpec{Name: "google_search", Host: "www.google.com", Method: "GET", Path: "/search?q=mtu", TimeoutSec: 12, ReadBodyBytes: 192 * 1024},
)

var ChromeProbeTargets = []ChromeProbeTarget{
	{Name: "youtube_204", Kind: "fetch", URL: "https://www.youtube.com/generate_204"},
	{Name: "google_204", Kind: "fetch", URL: "https://www.google.com/generate_204"},
	{Name: "gstatic_204", Kind: "fetch", URL: "https://www.gstatic.com/generate_204"},
	{Name: "ytimg_thumb", Kind: "image", URL: "https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg"},
	{Name: "google_logo", Kind: "image", URL: "https://www.google.com/images/branding/googlelogo/2x/googlelogo_color_272x92dp.png"},
}
