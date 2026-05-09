package core

type InterfaceInfo struct {
	PlatformName string
	Name         string
	Index        string
	MTU          int
	Gateway      string
	LocalAddress string
	Description  string
}

type ClashTarget struct {
	Group      string
	Leaf       string
	Server     string
	Port       int
	ResolvedIP string
	ConfigPath string
	Source     string
}

type ProbeSelection struct {
	ProbeIP string
	Target  *ClashTarget
	Warning string
}

type TestTarget struct {
	Name     string
	URL      string
	Enabled  bool
	Profiles []string
	Order    int
}

type SavedSettings struct {
	Version       int
	RouteProbe    string
	FallbackProbe string
	HTTPProxy     string
	ClashAPI      string
	ProxyGroup    string
	ConfigPath    string
	BrowserPath   string
	TestProfile   string
	TestTargets   []TestTarget
	SweepMTUs     string
	TargetMTU     int
}

type DetectRequest struct {
	Probe         string
	FallbackProbe string
	Controller    string
	Secret        string
	Group         string
	ConfigPath    string
	ClashCurrent  bool
}

type DetectResult struct {
	Selection   ProbeSelection
	Interface   InterfaceInfo
	OriginalMTU int
	Candidates  []InterfaceInfo
}

type InterfaceCommandResult struct {
	Interface   InterfaceInfo
	Output      string
	OriginalMTU int
}

type ResolveTargetRequest struct {
	Controller string
	Secret     string
	Group      string
	ConfigPath string
}

type TestRunRequest struct {
	Interface   InterfaceInfo
	HTTPProxy   string
	TestProfile string
	BrowserPath string
	TestTargets []TestTarget
	Rounds      int
	Concurrency int
}

type SweepRunRequest struct {
	Interface   InterfaceInfo
	HTTPProxy   string
	TestProfile string
	BrowserPath string
	TestTargets []TestTarget
	SweepMTUs   string
	Rounds      int
	Concurrency int
}

type CheckResult struct {
	OK        bool
	Code      int
	Elapsed   float64
	Error     string
	BytesRead int
}

type TestSummary struct {
	Profile        string
	Concurrency    int
	PlannedTotal   int
	Total          int
	OK             int
	Failures       int
	Avg            float64
	P95            float64
	Bytes          int
	FailByName     map[string]int
	FailByError    map[string]int
	FirstError     string
	Browser        string
	ProbeTransport string
	Cancelled      bool
}

type SweepRow struct {
	MTU          int
	Effective    int
	Profile      string
	PlannedTotal int
	Total        int
	OK           int
	Failures     int
	FirstError   string
	Cancelled    bool
}

type SweepResult struct {
	Rows        []SweepRow
	OutputPath  string
	StartMTU    int
	RestoredMTU int
	Cancelled   bool
}

type TaskEvent interface {
	taskEvent()
}

type TaskState struct {
	Kind            string `json:"kind"`
	Status          string `json:"status"`
	CancelRequested bool   `json:"cancel_requested"`
}

type TaskProgress struct {
	Kind  string `json:"kind"`
	Done  int    `json:"done"`
	Total int    `json:"total"`
	Label string `json:"label"`
}

type TaskLog struct {
	Kind string `json:"kind"`
	Line string `json:"line"`
	TS   string `json:"ts"`
}

func (TaskState) taskEvent() {}

func (TaskProgress) taskEvent() {}

func (TaskLog) taskEvent() {}

type SystemStatus struct {
	PlatformName          string
	IsAdmin               bool
	SupportsPersistentMTU bool
	Busy                  bool
	CurrentTaskKind       string
	CurrentTaskStatus     string
}
