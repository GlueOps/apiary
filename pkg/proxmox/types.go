package proxmox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/copier"
)

var (
	isFloat = regexp.MustCompile(`^[0-9.]*$`)
)

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Otp      string `json:"otp,omitempty"`
	Path     string `json:"path,omitempty"`
	Privs    string `json:"privs,omitempty"`
	Realm    string `json:"realm,omitempty"`
}

type Permission map[string]IntOrBool
type Permissions map[string]Permission

type PermissionsOptions struct {
	Path   string
	UserID string
}

type Session struct {
	Username            string `json:"username"`
	CSRFPreventionToken string `json:"CSRFPreventionToken,omitempty"`

	Cap         map[string]map[string]int `json:"cap,omitempty"`
	ClusterName string                    `json:"clustername,omitempty"`
	Ticket      string                    `json:"ticket,omitempty"`
}

type Version struct {
	Release string `json:"release"`
	RepoID  string `json:"repoid"`
	Version string `json:"version"`
}

type NodeService struct {
	client *Client
	Node   string `json:"-"`

	Service string `json:"service"`
	Name    string `json:"name,omitempty"`
	Desc    string `json:"desc,omitempty"`

	Status      string `json:"state,omitempty"`
	ActiveState string `json:"active-state,omitempty"`
	UnitState   string `json:"unit-state,omitempty"`
}

type NodeTime struct {
	Time      int64  `json:"time"`
	Localtime int64  `json:"localtime"`
	Timezone  string `json:"timezone"`
}

type Subscription struct {
	Status         string `json:"status,omitempty"`
	Key            string `json:"key,omitempty"`
	Level          string `json:"level,omitempty"`
	ProductName    string `json:"productname,omitempty"`
	RegDate        string `json:"regdate,omitempty"`
	NextDueDate    string `json:"nextduedate,omitempty"`
	Validdirectory string `json:"validdirectory,omitempty"`
	Sockets        int    `json:"sockets,omitempty"`
	Checktime      int64  `json:"checktime,omitempty"`
	ServerID       string `json:"serverid,omitempty"`
	URL            string `json:"url,omitempty"`
	Message        string `json:"message,omitempty"`
	Signature      string `json:"signature,omitempty"`
}

type LogEntry struct {
	N int    `json:"n"`
	T string `json:"t"`
}

type NodeReplicationJob struct {
	client *Client

	Node string `json:"-"`

	ID        string  `json:"id"`
	Type      string  `json:"type,omitempty"`
	Source    string  `json:"source,omitempty"`
	Target    string  `json:"target,omitempty"`
	Guest     int     `json:"guest,omitempty"`
	JobNum    int     `json:"jobnum,omitempty"`
	Schedule  string  `json:"schedule,omitempty"`
	LastSync  int64   `json:"last_sync,omitempty"`
	LastTry   int64   `json:"last_try,omitempty"`
	NextSync  int64   `json:"next_sync,omitempty"`
	Duration  float64 `json:"duration,omitempty"`
	FailCount int     `json:"fail_count,omitempty"`
	Error     string  `json:"error,omitempty"`
	PID       int     `json:"pid,omitempty"`
	State     string  `json:"state,omitempty"`
}

type APTIndexEntry struct {
	ID string `json:"id"`
}

type APTUpdate struct {
	Package      string `json:"Package"`
	Title        string `json:"Title"`
	Description  string `json:"Description"`
	Version      string `json:"Version"`
	OldVersion   string `json:"OldVersion,omitempty"`
	Origin       string `json:"Origin"`
	Arch         string `json:"Arch"`
	Section      string `json:"Section"`
	Priority     string `json:"Priority"`
	NotifyStatus string `json:"NotifyStatus,omitempty"`
}

type APTPackageVersion struct {
	Package        string `json:"Package"`
	Title          string `json:"Title"`
	Description    string `json:"Description"`
	Version        string `json:"Version"`
	OldVersion     string `json:"OldVersion,omitempty"`
	Origin         string `json:"Origin"`
	Arch           string `json:"Arch"`
	Section        string `json:"Section"`
	Priority       string `json:"Priority"`
	CurrentState   string `json:"CurrentState"`
	ManagerVersion string `json:"ManagerVersion,omitempty"`
	RunningKernel  string `json:"RunningKernel,omitempty"`
	NotifyStatus   string `json:"NotifyStatus,omitempty"`
}

type APTRepositories struct {
	Digest        string                `json:"digest"`
	Files         []*APTRepositoryFile  `json:"files,omitempty"`
	Errors        []*APTRepositoryError `json:"errors,omitempty"`
	Infos         []*APTRepositoryInfo  `json:"infos,omitempty"`
	StandardRepos []*APTStandardRepo    `json:"standard-repos,omitempty"`
}

type APTRepositoryFile struct {
	Path         string           `json:"path"`
	FileType     string           `json:"file-type"`
	Digest       []int            `json:"digest,omitempty"`
	Repositories []*APTRepository `json:"repositories,omitempty"`
}

type APTRepository struct {
	Enabled    bool                   `json:"Enabled"`
	FileType   string                 `json:"FileType"`
	Types      []string               `json:"Types"`
	URIs       []string               `json:"URIs"`
	Suites     []string               `json:"Suites"`
	Components []string               `json:"Components,omitempty"`
	Options    []*APTRepositoryOption `json:"Options,omitempty"`
	Comment    string                 `json:"Comment,omitempty"`
}

type APTRepositoryOption struct {
	Key    string   `json:"Key"`
	Values []string `json:"Values"`
}

type APTRepositoryError struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

type APTRepositoryInfo struct {
	Path     string `json:"path"`
	Index    string `json:"index"`
	Kind     string `json:"kind"`
	Message  string `json:"message"`
	Property string `json:"property,omitempty"`
}

type APTStandardRepo struct {
	Handle string `json:"handle"`
	Name   string `json:"name"`
	Status *bool  `json:"status,omitempty"`
}

type Term struct {
	Port   StringOrInt
	Ticket string
	UPID   string
	User   string
}

type VNCConfig struct {
	GeneratePassword bool `json:"generate-password,omitempty"`
	Websocket        bool `json:"websocket,omitempty"`
}

type VNC struct {
	Cert     string
	Port     StringOrInt
	Ticket   string
	UPID     string
	User     string
	Password string `json:",omitempty"`
}

type Cluster struct {
	client  *Client
	Version int
	Quorate int
	Nodes   NodeStatuses
	Name    string
	ID      string
}

func (cl *Cluster) UnmarshalJSON(b []byte) error {
	var tmp []map[string]interface{}
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	for _, d := range tmp {
		t, ok := d["type"]
		if !ok {
			break
		}

		switch t.(string) {
		case "cluster":
			if v, ok := d["id"]; ok {
				cl.ID = v.(string)
			}
			if v, ok := d["name"]; ok {
				cl.Name = v.(string)
			}
			if v, ok := d["version"]; ok {
				cl.Version = int(v.(float64))
			}
			if v, ok := d["quorate"]; ok {
				cl.Quorate = int(v.(float64))
			}
		case "node":
			ns := NodeStatus{
				Status: "offline",
				Type:   "node",
			}
			if v, ok := d["name"]; ok {
				ns.Name = v.(string)
			}
			if v, ok := d["level"]; ok {
				ns.Level = v.(string)
			}
			if v, ok := d["online"]; ok {
				ns.Online = int(v.(float64))
				if ns.Online == 1 {
					ns.Status = "online"
				}
			}
			if v, ok := d["id"]; ok {
				ns.ID = v.(string)
			}
			if v, ok := d["ip"]; ok {
				ns.IP = v.(string)
			}
			if v, ok := d["local"]; ok {
				ns.Local = int(v.(float64))
			}

			cl.Nodes = append(cl.Nodes, &ns)
		}
	}

	return nil
}

type ClusterResources []*ClusterResource

type ClusterResource struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	CGroupMode uint64  `json:"cgroup-mode,omitempty"`
	Content    string  `json:",omitempty"`
	CPU        float64 `json:",omitempty"`
	Disk       uint64  `json:",omitempty"`
	DiskRead   uint64  `json:",omitempty"`
	DiskWrite  uint64  `json:",omitempty"`
	HAstate    string  `json:",omitempty"`
	Level      string  `json:",omitempty"`
	MaxCPU     uint64  `json:",omitempty"`
	MaxDisk    uint64  `json:",omitempty"`
	MaxMem     uint64  `json:",omitempty"`
	Mem        uint64  `json:",omitempty"`
	Name       string  `json:",omitempty"`
	NetIn      uint64  `json:",omitempty"`
	NetOut     uint64  `json:",omitempty"`
	Node       string  `json:",omitempty"`
	PluginType string  `json:",omitempty"`
	Pool       string  `json:",omitempty"`
	Shared     uint64  `json:",omitempty"`
	Status     string  `json:",omitempty"`
	Storage    string  `json:",omitempty"`
	Tags       string  `json:",omitempty"`
	Template   uint64  `json:",omitempty"`
	Uptime     uint64  `json:",omitempty"`
	VMID       uint64  `json:",omitempty"`
}

type Ceph struct {
	client *Client
}

type ClusterCephStatus struct {
	ElectionEpoch  int            `json:"election_epoch"`
	Fsid           string         `json:"fsid"`
	Fsmap          CephFsMap      `json:"fsmap"`
	Health         CephHealth     `json:"health"`
	Mgrmap         CephMgrMap     `json:"mgrmap"`
	Monmap         CephMonMap     `json:"monmap"`
	Osdmap         CephOsdMap     `json:"osdmap"`
	Pgmap          CephPgMap      `json:"pgmap"`
	ProgressEvents struct{}       `json:"progress_events"`
	Quorum         []int          `json:"quorum"`
	QuorumAge      int            `json:"quorum_age"`
	QuorumNames    []string       `json:"quorum_names"`
	Servicemap     CephServiceMap `json:"servicemap"`
}

type CephHealthCheckName string
type CephHealthCheckDetail struct {
	Message string `json:"message"`
}
type CephHealthCheckSummary struct {
	Count   int    `json:"count"`
	Message string `json:"message"`
}
type CephHealthCheck struct {
	Detail   []CephHealthCheckDetail `json:"detail"`
	Muted    bool                    `json:"muted"`
	Severity string                  `json:"severity"`
	Summary  CephHealthCheckSummary  `json:"summary"`
}

type CephHealth struct {
	Checks map[CephHealthCheckName]CephHealthCheck `json:"checks"`
	Mutes  []interface{}                           `json:"mutes"`
	Status string                                  `json:"status"`
}

type CephOsdMap struct {
	Epoch          int `json:"epoch"`
	NumInOsds      int `json:"num_in_osds"`
	NumOsds        int `json:"num_osds"`
	NumRemappedPgs int `json:"num_remapped_pgs"`
	NumUpOsds      int `json:"num_up_osds"`
	OsdInSince     int `json:"osd_in_since"`
	OsdUpSince     int `json:"osd_up_since"`
}

type CephPgMap struct {
	BytesAvail int64 `json:"bytes_avail"`
	BytesTotal int64 `json:"bytes_total"`
	BytesUsed  int64 `json:"bytes_used"`
	DataBytes  int64 `json:"data_bytes"`
	NumObjects int   `json:"num_objects"`
	NumPgs     int   `json:"num_pgs"`
	NumPools   int   `json:"num_pools"`
	PgsByState []struct {
		Count     int    `json:"count"`
		StateName string `json:"state_name"`
	} `json:"pgs_by_state"`
	ReadBytesSec  int `json:"read_bytes_sec"`
	ReadOpPerSec  int `json:"read_op_per_sec"`
	WriteBytesSec int `json:"write_bytes_sec"`
	WriteOpPerSec int `json:"write_op_per_sec"`
}

type CephMonMap struct {
	Created           time.Time        `json:"created"`
	DisallowedLeaders string           `json:"disallowed_leaders"`
	ElectionStrategy  int              `json:"election_strategy"`
	Epoch             int              `json:"epoch"`
	Features          CephMonFeatures  `json:"features"`
	Fsid              string           `json:"fsid"`
	MinMonRelease     int              `json:"min_mon_release"`
	MinMonReleaseName string           `json:"min_mon_release_name"`
	Modified          time.Time        `json:"modified"`
	Mons              []ClusterCephMon `json:"mons"`
	Quorum            []int            `json:"quorum"`
	RemovedRanks      string           `json:"removed_ranks"`
	StretchMode       bool             `json:"stretch_mode"`
	TiebreakerMon     string           `json:"tiebreaker_mon"`
}

type ClusterCephMon struct {
	Addr          string `json:"addr"`
	CrushLocation string `json:"crush_location"`
	Name          string `json:"name"`
	Priority      int    `json:"priority"`
	Rank          int    `json:"rank"`
	Weight        int    `json:"weight"`
	PublicAddr    string `json:"public_addr"`
	PublicAddrs   struct {
		Addrvec []CephMgrAddrVector `json:"addrvec"`
	} `json:"public_addrs"`
}

type CephMonFeatures struct {
	Optional   []interface{} `json:"optional"`
	Persistent []string      `json:"persistent"`
}

type CephFsMap struct {
	ByRank []struct {
		FilesystemID int    `json:"filesystem_id"`
		Gid          int    `json:"gid"`
		Name         string `json:"name"`
		Rank         int    `json:"rank"`
		Status       string `json:"status"`
	} `json:"by_rank"`
	Epoch     int `json:"epoch"`
	ID        int `json:"id"`
	In        int `json:"in"`
	Max       int `json:"max"`
	Up        int `json:"up"`
	UpStandby int `json:"up:standby"`
}

type CephServiceMap struct {
	Epoch    int      `json:"epoch"`
	Modified string   `json:"modified"`
	Services struct{} `json:"services"`
}

type CephMgrMap struct {
	ActiveAddr          string                   `json:"active_addr"`
	ActiveAddrs         CephMgrActiveAddresses   `json:"active_addrs"`
	ActiveChange        string                   `json:"active_change"`
	ActiveClients       []CephMgrActiveClient    `json:"active_clients"`
	ActiveGid           int                      `json:"active_gid"`
	ActiveMgrFeatures   int64                    `json:"active_mgr_features"`
	ActiveName          string                   `json:"active_name"`
	AlwaysOnModules     CephMgrAlwaysOnModules   `json:"always_on_modules"`
	Available           bool                     `json:"available"`
	AvailableModules    []CephMgrAvailableModule `json:"available_modules"`
	Epoch               int                      `json:"epoch"`
	LastFailureOsdEpoch int                      `json:"last_failure_osd_epoch"`
	Modules             []string                 `json:"modules"`
	Services            CephMgrServices          `json:"services"`
	Standbys            []CephMgrStandby         `json:"standbys"`
}

type CephMgrAvailableModule struct {
	CanRun        bool                          `json:"can_run"`
	ErrorString   string                        `json:"error_string"`
	ModuleOptions CephMgrAvailableModuleOptions `json:"module_options"`
	Name          string                        `json:"name"`
}

type CephMgrAvailableModuleOptions struct {
	Interval          CephMgrAvailableModuleOption `json:"interval"`
	LogLevel          CephMgrAvailableModuleOption `json:"log_level"`
	LogToCluster      CephMgrAvailableModuleOption `json:"log_to_cluster"`
	LogToClusterLevel CephMgrAvailableModuleOption `json:"log_to_cluster_level"`
	LogToFile         CephMgrAvailableModuleOption `json:"log_to_file"`
	SMTPDestination   CephMgrAvailableModuleOption `json:"smtp_destination"`
	SMTPFromName      CephMgrAvailableModuleOption `json:"smtp_from_name"`
	SMTPHost          CephMgrAvailableModuleOption `json:"smtp_host"`
	SMTPPassword      CephMgrAvailableModuleOption `json:"smtp_password"`
	SMTPPort          CephMgrAvailableModuleOption `json:"smtp_port"`
	SMTPSender        CephMgrAvailableModuleOption `json:"smtp_sender"`
	SMTPSsl           CephMgrAvailableModuleOption `json:"smtp_ssl"`
	SMTPUser          CephMgrAvailableModuleOption `json:"smtp_user"`
}

type CephMgrAvailableModuleOption struct {
	DefaultValue string        `json:"default_value"`
	Desc         string        `json:"desc"`
	EnumAllowed  []string      `json:"enum_allowed"`
	Flags        int           `json:"flags"`
	Level        string        `json:"level"`
	LongDesc     string        `json:"long_desc"`
	Max          string        `json:"max"`
	Min          string        `json:"min"`
	Name         string        `json:"name"`
	SeeAlso      []interface{} `json:"see_also"`
	Tags         []interface{} `json:"tags"`
	Type         string        `json:"type"`
}

type CephMgrServices struct {
	Dashboard  string `json:"dashboard"`
	Prometheus string `json:"prometheus"`
}

type CephMgrStandby struct {
	AvailableModules []CephMgrAvailableModule `json:"available_modules"`
	Gid              int                      `json:"gid"`
	MgrFeatures      int64                    `json:"mgr_features"`
	Name             string                   `json:"name"`
}

type CephMgrActiveAddresses struct {
	Addrvec []CephMgrAddrVector `json:"addrvec"`
}

type CephMgrAddrVector struct {
	Addr  string `json:"addr"`
	Nonce int    `json:"nonce"`
	Type  string `json:"type"`
}

type CephMgrActiveClient struct {
	Addrvec []CephMgrAddrVector `json:"addrvec"`
	Name    string              `json:"name"`
}

type CephMgrAlwaysOnModules struct {
	Octopus []string `json:"octopus"`
	Pacific []string `json:"pacific"`
	Quincy  []string `json:"quincy"`
	Reef    []string `json:"reef"`
	Squid   []string `json:"squid"`
}

type CephFS struct {
	client         *Client
	Node           string   `json:"-"`
	Name           string   `json:"name"`
	MetadataPool   string   `json:"metadata_pool"`
	MetadataPoolID int      `json:"metadata_pool_id,omitempty"`
	DataPool       string   `json:"data_pool"`
	DataPools      []string `json:"data_pools,omitempty"`
	DataPoolIDs    []int    `json:"data_pool_ids,omitempty"`
}

type CephFSOptions struct {
	PgNum      int       `json:"pg_num,omitempty"`
	AddStorage IntOrBool `json:"add-storage,omitempty"`
}

type CephCfgDBEntry struct {
	Section            string    `json:"section"`
	Name               string    `json:"name"`
	Value              string    `json:"value"`
	Level              string    `json:"level,omitempty"`
	Mask               string    `json:"mask,omitempty"`
	CanUpdateAtRuntime IntOrBool `json:"can_update_at_runtime,omitempty"`
}

type CephCfgValue map[string]map[string]string

type CephIndexEntry struct {
	Subdir string `json:"subdir,omitempty"`
}

type CephInitOptions struct {
	Network string `json:"network,omitempty"`

	ClusterNetwork string `json:"cluster-network,omitempty"`

	Size int `json:"size,omitempty"`

	MinSize int `json:"min_size,omitempty"`

	PGBits int `json:"pg_bits,omitempty"`

	DisableCephx bool `json:"disable_cephx,omitempty"`
}

type CephLogEntry struct {
	N int    `json:"n"`
	T string `json:"t"`
}

type CephRule struct {
	Name string `json:"name"`
}

type CephCmdSafety struct {
	Safe   bool   `json:"safe"`
	Status string `json:"status,omitempty"`
}

type CephPool struct {
	client              *Client
	Node                string         `json:"-"`
	ApplicationMetadata map[string]any `json:"application_metadata,omitempty"`
	AutoscaleStatus     map[string]any `json:"autoscale_status,omitempty"`
	BytesUsed           uint64         `json:"bytes_used,omitempty"`
	CrushRule           int            `json:"crush_rule"`
	CrushRuleName       string         `json:"crush_rule_name,omitempty"`
	MinSize             int            `json:"min_size"`
	PercentUsed         float64        `json:"percent_used,omitempty"`
	PgAutoscaleMode     string         `json:"pg_autoscale_mode,omitempty"`
	PgNum               int            `json:"pg_num"`
	PgNumFinal          int            `json:"pg_num_final,omitempty"`
	PgNumMin            int            `json:"pg_num_min,omitempty"`
	Pool                int            `json:"pool"`
	PoolName            string         `json:"pool_name"`
	Size                int            `json:"size"`
	TargetSize          uint64         `json:"target_size,omitempty"`
	TargetSizeRatio     float64        `json:"target_size_ratio,omitempty"`
	Type                string         `json:"type"`
}

type CephPoolSubdir struct {
	Subdir string `json:"subdir,omitempty"`
}

type CephPoolStatus struct {
	Application          string         `json:"application,omitempty"`
	ApplicationList      []string       `json:"application_list,omitempty"`
	AutoscaleStatus      map[string]any `json:"autoscale_status,omitempty"`
	CrushRule            string         `json:"crush_rule,omitempty"`
	FastRead             bool           `json:"fast_read"`
	HashPSPool           bool           `json:"hashpspool"`
	ID                   int            `json:"id"`
	MinSize              int            `json:"min_size,omitempty"`
	Name                 string         `json:"name"`
	NoDeepScrub          bool           `json:"nodeep-scrub"`
	NoDelete             bool           `json:"nodelete"`
	NoPGChange           bool           `json:"nopgchange"`
	NoScrub              bool           `json:"noscrub"`
	NoSizeChange         bool           `json:"nosizechange"`
	PgAutoscaleMode      string         `json:"pg_autoscale_mode,omitempty"`
	PgNum                int            `json:"pg_num,omitempty"`
	PgNumMin             int            `json:"pg_num_min,omitempty"`
	PgpNum               int            `json:"pgp_num"`
	Size                 int            `json:"size,omitempty"`
	Statistics           map[string]any `json:"statistics,omitempty"`
	TargetSize           string         `json:"target_size,omitempty"`
	TargetSizeRatio      float64        `json:"target_size_ratio,omitempty"`
	UseGMTHitset         bool           `json:"use_gmt_hitset"`
	WriteFadviseDontneed bool           `json:"write_fadvise_dontneed"`
}

type CephPoolErasureCoding struct {
	K             int
	M             int
	DeviceClass   string
	FailureDomain string
	Profile       string
}

func (ec *CephPoolErasureCoding) String() string {
	if ec == nil {
		return ""
	}
	parts := []string{
		fmt.Sprintf("k=%d", ec.K),
		fmt.Sprintf("m=%d", ec.M),
	}
	if ec.DeviceClass != "" {
		parts = append(parts, "device-class="+ec.DeviceClass)
	}
	if ec.FailureDomain != "" {
		parts = append(parts, "failure-domain="+ec.FailureDomain)
	}
	if ec.Profile != "" {
		parts = append(parts, "profile="+ec.Profile)
	}
	return strings.Join(parts, ",")
}

type CephPoolOptions struct {
	Name            string                 `json:"name,omitempty"`
	AddStorages     *bool                  `json:"add_storages,omitempty"`
	Application     string                 `json:"application,omitempty"`
	CrushRule       string                 `json:"crush_rule,omitempty"`
	ErasureCoding   *CephPoolErasureCoding `json:"-"`
	MinSize         *int                   `json:"min_size,omitempty"`
	PgAutoscaleMode string                 `json:"pg_autoscale_mode,omitempty"`
	PgNum           *int                   `json:"pg_num,omitempty"`
	PgNumMin        *int                   `json:"pg_num_min,omitempty"`
	Size            *int                   `json:"size,omitempty"`
	TargetSize      string                 `json:"target_size,omitempty"`
	TargetSizeRatio *float64               `json:"target_size_ratio,omitempty"`
}

type NodeStatuses []*NodeStatus
type NodeStatus struct {
	Status string `json:",omitempty"`
	Level  string `json:",omitempty"`
	ID     string `json:",omitempty"`

	Node           string  `json:",omitempty"`
	Type           string  `json:",omitempty"`
	MaxCPU         int     `json:",omitempty"`
	MaxMem         uint64  `json:",omitempty"`
	Disk           uint64  `json:",omitempty"`
	SSLFingerprint string  `json:"ssl_fingerprint,omitempty"`
	MaxDisk        uint64  `json:",omitempty"`
	Mem            uint64  `json:",omitempty"`
	CPU            float64 `json:",omitempty"`
	Uptime         uint64  `json:",omitempty"`

	NodeID int    `json:",omitempty"`
	Name   string `json:",omitempty"`
	IP     string `json:",omitempty"`
	Online int    `json:",omitempty"`
	Local  int    `json:",omitempty"`
}

type NodeStartAllOptions struct {
	Force IntOrBool `json:"force,omitempty"`
	VMs   string    `json:"vms,omitempty"`
}

type NodeStopAllOptions struct {
	ForceStop IntOrBool `json:"force-stop,omitempty"`
	Timeout   uint64    `json:"timeout,omitempty"`
	VMs       string    `json:"vms,omitempty"`
}

type NodeSuspendAllOptions struct {
	VMs string `json:"vms,omitempty"`
}

type NodeMigrateAllOptions struct {
	Target         string    `json:"target"`
	MaxWorkers     uint64    `json:"maxworkers,omitempty"`
	VMs            string    `json:"vms,omitempty"`
	WithLocalDisks IntOrBool `json:"with-local-disks,omitempty"`
}

type Node struct {
	Name       string
	client     *Client
	Kversion   string
	LoadAvg    []string
	CPU        float64
	RootFS     RootFS
	PVEVersion string
	CPUInfo    CPUInfo
	Swap       Memory
	Idle       int
	Memory     Memory
	Ksm        Ksm
	Uptime     uint64
	Wait       float64
}

type VirtualMachines []*VirtualMachine
type VirtualMachine struct {
	client               *Client
	VirtualMachineConfig *VirtualMachineConfig

	Name string
	Node string

	Agent          IntOrBool
	Spice          IntOrBool
	NetIn          uint64
	CPUs           int
	DiskWrite      uint64
	Status         string
	Lock           string `json:",omitempty"`
	VMID           StringOrUint64
	PID            StringOrUint64
	Netout         uint64
	Disk           uint64
	Mem            uint64
	CPU            float64
	MaxMem         uint64
	MaxDisk        uint64
	DiskRead       uint64
	QMPStatus      string `json:"qmpstatus,omitempty"`
	RunningMachine string `json:"running-machine,omitempty"`
	RunningQemu    string `json:"running-qemu,omitempty"`
	Tags           string `json:"tags,omitempty"`
	Uptime         uint64
	Template       IsTemplate
	HA             HA `json:",omitempty"`
}

type HA struct {
	Managed int
}

type RootFS struct {
	Avail uint64
	Total uint64
	Free  uint64
	Used  uint64
}

type CPUInfo struct {
	UserHz  int `json:"user_hz"`
	MHZ     StringOrInt
	Model   string
	Cores   int
	Sockets int
	Flags   string
	CPUs    int
	HVM     string
}

type Memory struct {
	Used  uint64
	Free  uint64
	Total uint64
}

type Ksm struct {
	Shared int64
}

type Time struct {
	Timezone  string
	Time      uint64
	Localtime uint64
}

type Timeframe string

const (
	TimeframeHour  = Timeframe("hour")
	TimeframeDay   = Timeframe("day")
	TimeframeWeek  = Timeframe("week")
	TimeframeMonth = Timeframe("month")
	TimeframeYear  = Timeframe("year")
)

type ConsolidationFunction string

const (
	AVERAGE = ConsolidationFunction("AVERAGE")
	MAX     = ConsolidationFunction("MAX")
)

type RRDData struct {
	Time      uint64
	CPU       float64
	MaxCPU    int
	Mem       float64
	MaxMem    uint64
	Disk      int
	MaxDisk   uint64
	DiskRead  float64
	DiskWrite float64
	NetIn     float64
	NetOut    float64
}

type VirtualMachineOptions []*VirtualMachineOption

type VirtualMachineOption struct {
	Name  string
	Value interface{}
}

type VirtualMachineConfig struct {
	Digest      string `json:"digest"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Meta        string `json:"meta,omitempty"`

	VMGenID    *string `json:"vmgenid,omitempty"`
	Hookscript string  `json:"hookscript,omitempty"`

	Hotplug   *string   `json:"hotplug,omitempty"`
	Template  IntOrBool `json:"template,omitempty"`
	Agent     string    `json:"agent,omitempty"`
	Autostart IntOrBool `json:"autostart,omitempty"`

	Tablet *IntOrBool `json:"tablet,omitempty"`

	KVM *IntOrBool `json:"kvm,omitempty"`

	Tags      string   `json:"tags,omitempty"`
	TagsSlice []string `json:"-"`

	Protection IntOrBool `json:"protection,omitempty"`
	Lock       string    `json:"lock,omitempty"`

	Boot   string    `json:"boot,omitempty"`
	OnBoot IntOrBool `json:"onboot,omitempty"`

	OSType  *string `json:"ostype,omitempty"`
	Machine string  `json:"machine,omitempty"`
	Args    string  `json:"args,omitempty"`

	Bios     *string `json:"bios,omitempty"`
	EFIDisk0 string  `json:"efidisk0,omitempty"`
	SMBios1  string  `json:"smbios1,omitempty"`

	Acpi *IntOrBool `json:"acpi,omitempty"`

	Sockets *int `json:"sockets,omitempty"`

	Cores    *int             `json:"cores,omitempty"`
	CPU      string           `json:"cpu,omitempty"`
	CPULimit *StringOrFloat64 `json:"cpulimit,omitempty"`

	CPUUnits *int   `json:"cpuunits,omitempty"`
	Vcpus    int    `json:"vcpus,omitempty"`
	Affinity string `json:"affinity,omitempty"`

	Numa      IntOrBool   `json:"numa,omitempty"`
	Memory    StringOrInt `json:"memory,omitempty"`
	Hugepages string      `json:"hugepages,omitempty"`
	Balloon   int         `json:"balloon,omitempty"`

	VGA string `json:"vga,omitempty"`

	SCSIHW    *string `json:"scsihw,omitempty"`
	TPMState0 string  `json:"tpmstate0,omitempty"`
	Rng0      string  `json:"rng0,omitempty"`
	Audio0    string  `json:"audio0,omitempty"`

	IDEs      map[string]string `json:"-"`
	SCSIs     map[string]string `json:"-"`
	SATAs     map[string]string `json:"-"`
	VirtIOs   map[string]string `json:"-"`
	Unuseds   map[string]string `json:"-"`
	Nets      map[string]string `json:"-"`
	Numas     map[string]string `json:"-"`
	HostPCIs  map[string]string `json:"-"`
	Serials   map[string]string `json:"-"`
	USBs      map[string]string `json:"-"`
	Parallels map[string]string `json:"-"`
	IPConfigs map[string]string `json:"-"`

	CIType       string `json:"citype,omitempty"`
	CIUser       string `json:"ciuser,omitempty"`
	CIPassword   string `json:"cipassword,omitempty"`
	Nameserver   string `json:"nameserver,omitempty"`
	Searchdomain string `json:"searchdomain,omitempty"`

	SSHKeys  string `json:"sshkeys,omitempty"`
	CICustom string `json:"cicustom,omitempty"`

	CIUpgrade *IntOrBool `json:"ciupgrade,omitempty"`
}

func (vmc *VirtualMachineConfig) indexedDeviceMaps() map[string]*map[string]string {
	return map[string]*map[string]string{
		"ide":      &vmc.IDEs,
		"scsi":     &vmc.SCSIs,
		"sata":     &vmc.SATAs,
		"virtio":   &vmc.VirtIOs,
		"unused":   &vmc.Unuseds,
		"net":      &vmc.Nets,
		"numa":     &vmc.Numas,
		"hostpci":  &vmc.HostPCIs,
		"serial":   &vmc.Serials,
		"usb":      &vmc.USBs,
		"parallel": &vmc.Parallels,
		"ipconfig": &vmc.IPConfigs,
	}
}

func (vmc *VirtualMachineConfig) UnmarshalJSON(data []byte) error {
	type tmpVirtualMachineConfig VirtualMachineConfig

	tmp := &struct {
		*tmpVirtualMachineConfig
	}{
		tmpVirtualMachineConfig: (*tmpVirtualMachineConfig)(vmc),
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	vmc.TagsSlice = strings.Split(vmc.Tags, TagSeperator)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	routes := vmc.indexedDeviceMaps()
	for k, v := range raw {
		prefix, ok := indexedDeviceKey(k)
		if !ok {
			continue
		}
		target, ok := routes[prefix]
		if !ok {
			continue
		}
		var s string
		if err := json.Unmarshal(v, &s); err != nil {

			continue
		}
		if *target == nil {
			*target = make(map[string]string)
		}
		(*target)[k] = s
	}

	return nil
}

func indexedDeviceKey(k string) (prefix string, ok bool) {
	for i := 0; i < len(k); i++ {
		c := k[i]
		if c >= '0' && c <= '9' {
			if i == 0 {
				return "", false
			}

			for j := i + 1; j < len(k); j++ {
				if k[j] < '0' || k[j] > '9' {
					return "", false
				}
			}
			return k[:i], true
		}
	}
	return "", false
}

type VirtualMachineFeature struct {
	HasFeature bool     `json:"hasFeature"`
	Nodes      []string `json:"nodes,omitempty"`
}

type VirtualMachineMigrateOptions struct {
	Target string `json:"target"`

	BWLimit          *uint64   `json:"bwlimit,omitempty"`
	Force            IntOrBool `json:"force,omitempty"`
	MigrationNetwork string    `json:"migration_network,omitempty"`
	MigrationType    string    `json:"migration_type,omitempty"`
	Online           IntOrBool `json:"online,omitempty"`
	TargetStorage    string    `json:"targetstorage,omitempty"`
	WithLocalDisks   IntOrBool `json:"with-local-disks,omitempty"`
}

type ContainerMigrateOptions struct {
	Target string `json:"target"`

	BWLimit *uint64   `json:"bwlimit,omitempty"`
	Online  IntOrBool `json:"online,omitempty"`
	Restart IntOrBool `json:"restart,omitempty"`
}

type ContainerDeleteOptions struct {
	Force IntOrBool `json:"force,omitempty"`

	Purge IntOrBool `json:"purge,omitempty"`

	DestroyUnreferencedDisks IntOrBool `json:"destroy-unreferenced-disks,omitempty"`
}

type ContainerRemoteMigrateOptions struct {
	TargetEndpoint string    `json:"target-endpoint"`
	TargetBridge   string    `json:"target-bridge"`
	TargetStorage  string    `json:"target-storage"`
	TargetVMID     int       `json:"target-vmid,omitempty"`
	BWLimit        uint64    `json:"bwlimit,omitempty"`
	Delete         IntOrBool `json:"delete,omitempty"`
	Online         IntOrBool `json:"online,omitempty"`
	Restart        IntOrBool `json:"restart,omitempty"`
	Timeout        uint64    `json:"timeout,omitempty"`
}

type ContainerPending struct {
	Key     string      `json:"key"`
	Value   interface{} `json:"value,omitempty"`
	Pending interface{} `json:"pending,omitempty"`
	Delete  int         `json:"delete,omitempty"`
}

type ContainerRRD struct {
	Filename string `json:"filename"`
}

type ContainerDirIndexEntry struct {
	Subdir string `json:"subdir,omitempty"`
}

type ContainerStatusIndexEntry struct {
	Subdir string `json:"subdir,omitempty"`
}

type ContainerMigratePreconditions struct {
	Running         bool                                                          `json:"running"`
	AllowedNodes    []string                                                      `json:"allowed_nodes,omitempty"`
	NotAllowedNodes map[string]*VirtualMachineMigratePreconditionsNotAllowedNodes `json:"not_allowed_nodes,omitempty"`
	LocalDisks      []*VirtualMachineMigratePreconditionsLocalDisk                `json:"local_disks,omitempty"`
	LocalResources  []string                                                      `json:"local_resources,omitempty"`
}

type ContainerMigrationTunnel struct {
	Socket string `json:"socket,omitempty"`
	Ticket string `json:"ticket,omitempty"`
	UPID   string `json:"upid,omitempty"`
}

type ContainerMigrationTunnelOptions struct {
	Bridges string `json:"bridges,omitempty"`

	Storages string `json:"storages,omitempty"`
}

type VirtualMachineRRD struct {
	Filename string `json:"filename"`
}

type VirtualMachineRemoteMigrateOptions struct {
	TargetEndpoint string    `json:"target-endpoint"`
	TargetBridge   string    `json:"target-bridge"`
	TargetStorage  string    `json:"target-storage"`
	TargetVMID     int       `json:"target-vmid,omitempty"`
	BWLimit        uint64    `json:"bwlimit,omitempty"`
	Delete         IntOrBool `json:"delete,omitempty"`
	Online         IntOrBool `json:"online,omitempty"`
}

type VirtualMachineMigratePreconditionsLocalDisk struct {
	VolID    string `json:"volid"`
	Size     uint64 `json:"size,omitempty"`
	CDROM    bool   `json:"cdrom,omitempty"`
	IsUnused bool   `json:"is_unused,omitempty"`
}

type VirtualMachineMigratePreconditionsBlockingHAResource struct {
	SID   string `json:"sid"`
	Cause string `json:"cause"`
}

type VirtualMachineMigratePreconditionsNotAllowedNodes struct {
	UnavailableStorages []string                                                `json:"unavailable_storages,omitempty"`
	BlockingHAResources []*VirtualMachineMigratePreconditionsBlockingHAResource `json:"blocking-ha-resources,omitempty"`
}

type VirtualMachineMigratePreconditions struct {
	Running              bool                                                          `json:"running"`
	HasDBusVMState       bool                                                          `json:"has-dbus-vmstate"`
	AllowedNodes         []string                                                      `json:"allowed_nodes,omitempty"`
	NotAllowedNodes      map[string]*VirtualMachineMigratePreconditionsNotAllowedNodes `json:"not_allowed_nodes,omitempty"`
	LocalDisks           []*VirtualMachineMigratePreconditionsLocalDisk                `json:"local_disks,omitempty"`
	LocalResources       []string                                                      `json:"local_resources,omitempty"`
	MappedResources      []string                                                      `json:"mapped-resources,omitempty"`
	MappedResourceInfo   map[string]interface{}                                        `json:"mapped-resource-info,omitempty"`
	DependentHAResources []string                                                      `json:"dependent-ha-resources,omitempty"`
}

type VirtualMachineDeleteOptions struct {
	SkipLock                 IntOrBool `json:"skiplock,omitempty"`
	Purge                    IntOrBool `json:"purge,omitempty"`
	DestroyUnreferencedDisks IntOrBool `json:"destroy-unreferenced-disks,omitempty"`
}

type SpiceProxy struct {
	Type             string `json:"type"`
	Host             string `json:"host"`
	Port             string `json:"port,omitempty"`
	Password         string `json:"password,omitempty"`
	Proxy            string `json:"proxy,omitempty"`
	Title            string `json:"title,omitempty"`
	TLSPort          string `json:"tls-port,omitempty"`
	CA               string `json:"ca,omitempty"`
	HostSubject      string `json:"host-subject,omitempty"`
	DeleteThisFile   string `json:"delete-this-file,omitempty"`
	SecureAttention  string `json:"secure-attention,omitempty"`
	ReleaseCursor    string `json:"release-cursor,omitempty"`
	ToggleFullscreen string `json:"toggle-fullscreen,omitempty"`
}

type FirewallLogEntry struct {
	LineNum int    `json:"n"`
	Text    string `json:"t"`
}

func (f *FirewallLogEntry) UnmarshalJSON(b []byte) error {

	var tuple []interface{}
	if err := json.Unmarshal(b, &tuple); err == nil && len(tuple) == 2 {
		if n, ok := tuple[0].(float64); ok {
			f.LineNum = int(n)
		}
		if t, ok := tuple[1].(string); ok {
			f.Text = t
		}
		return nil
	}

	aux := struct {
		N int    `json:"n"`
		T string `json:"t"`
	}{}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	f.LineNum = aux.N
	f.Text = aux.T
	return nil
}

type FirewallRef struct {
	Name    string `json:"name"`
	Ref     string `json:"ref,omitempty"`
	Scope   string `json:"scope,omitempty"`
	Type    string `json:"type"`
	Comment string `json:"comment,omitempty"`
}

type ContainerSnapshotUpdateOptions struct {
	Description string `json:"description,omitempty"`
}

type VirtualMachineCloneOptions struct {
	NewID int `json:"newid"`

	BWLimit     *uint64   `json:"bwlimit,omitempty"`
	Description string    `json:"description,omitempty"`
	Format      string    `json:"format,omitempty"`
	Full        IntOrBool `json:"full,omitempty"`
	Name        string    `json:"name,omitempty"`
	Pool        string    `json:"pool,omitempty"`
	SnapName    string    `json:"snapname,omitempty"`
	Storage     string    `json:"storage,omitempty"`
	Target      string    `json:"target,omitempty"`
}

type VirtualMachineMoveDiskOptions struct {
	Disk string `json:"disk"`

	BWLimit      *uint64   `json:"bwlimit,omitempty"`
	Delete       IntOrBool `json:"delete,omitempty"`
	Digest       string    `json:"digest,omitempty"`
	Format       string    `json:"format,omitempty"`
	Storage      string    `json:"storage,omitempty"`
	TargetDigest string    `json:"target-digest,omitempty"`
	TargetDisk   string    `json:"target-disk,omitempty"`
	TargetVMID   int       `json:"target-vmid,omitempty"`
}

type UPID string

type Tasks []*Task
type Task struct {
	client       *Client
	UPID         UPID
	ID           string
	Type         string
	User         string
	Status       string
	Node         string
	PID          uint64 `json:",omitempty"`
	PStart       uint64 `json:",omitempty"`
	Saved        string `json:",omitempty"`
	ExitStatus   string `json:",omitempty"`
	IsCompleted  bool
	IsRunning    bool
	IsFailed     bool
	IsSuccessful bool
	StartTime    time.Time     `json:"-"`
	EndTime      time.Time     `json:"-"`
	Duration     time.Duration `json:"-"`
}

func (t *Task) UnmarshalJSON(b []byte) error {
	var tmp map[string]interface{}
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	type TempTask Task
	var task TempTask
	if err := json.Unmarshal(b, &task); err != nil {
		return err
	}

	if starttime, ok := tmp["starttime"]; ok {
		task.StartTime = time.Unix(int64(starttime.(float64)), 0)
	}

	if endtime, ok := tmp["endtime"]; ok {
		task.EndTime = time.Unix(int64(endtime.(float64)), 0)
	}

	if !task.StartTime.IsZero() && !task.EndTime.IsZero() {
		task.Duration = task.EndTime.Sub(task.StartTime)
	}

	c := Task(task)
	return copier.Copy(t, &c)
}

type Log map[int]string

func (l *Log) UnmarshalJSON(b []byte) error {
	var data []map[string]interface{}
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	log := make(map[int]string, len(data))
	for _, row := range data {
		if n, ok := row["n"]; ok {
			if t, ok := row["t"]; ok {
				log[int(n.(float64))-1] = t.(string)
			}
		}
	}

	return copier.Copy(l, Log(log))
}

type Containers []*Container
type Container struct {
	client          *Client
	ContainerConfig *ContainerConfig

	CPUs    int
	MaxDisk uint64
	MaxMem  uint64
	MaxSwap uint64
	Name    string
	Node    string
	Status  string
	Tags    string
	Uptime  uint64
	VMID    StringOrUint64
}

type ContainerInterfaces []*ContainerInterface

type ContainerInterface struct {
	HWAddr string `json:"hwaddr,omitempty"`
	Name   string `json:"name,omitempty"`
	Inet   string `json:"inet,omitempty"`
	Inet6  string `json:"inet6,omitempty"`
}

type ContainerCloneOptions struct {
	NewID int `json:"newid"`

	BWLimit     *uint64   `json:"bwlimit,omitempty"`
	Description string    `json:"description,omitempty"`
	Full        IntOrBool `json:"full,omitempty"`
	Hostname    string    `json:"hostname,omitempty"`
	Pool        string    `json:"pool,omitempty"`
	SnapName    string    `json:"snapname,omitempty"`
	Storage     string    `json:"storage,omitempty"`
	Target      string    `json:"target,omitempty"`
}

type ContainerConfig struct {
	Arch *string `json:"arch,omitempty"`

	CMode *string `json:"cmode,omitempty"`

	Console  *IntOrBool `json:"console,omitempty"`
	Cores    int        `json:"cores,omitempty"`
	CPULimit int        `json:"cpulimit,omitempty"`

	CPUUnits    *int      `json:"cpuunits,omitempty"`
	Debug       IntOrBool `json:"debug,omitempty"`
	Description string    `json:"description,omitempty"`

	Devs    map[string]string `json:"-"`
	Mps     map[string]string `json:"-"`
	Nets    map[string]string `json:"-"`
	Unuseds map[string]string `json:"-"`

	Digest     string     `json:"digest"`
	Features   string     `json:"features,omitempty"`
	HookScript string     `json:"hookscript,omitempty"`
	LXC        [][]string `json:"lxc,omitempty"`
	Hostname   string     `json:"hostname,omitempty"`
	Lock       string     `json:"lock,omitempty"`

	Memory       *int      `json:"memory,omitempty"`
	Nameserver   string    `json:"nameserver,omitempty"`
	OnBoot       IntOrBool `json:"onboot,omitempty"`
	OSType       string    `json:"ostype,omitempty"`
	Protection   IntOrBool `json:"protection,omitempty"`
	RootFS       string    `json:"rootfs,omitempty"`
	SearchDomain string    `json:"searchdomain,omitempty"`
	Startup      string    `json:"startup,omitempty"`

	Swap      *int      `json:"swap,omitempty"`
	TagsSlice []string  `json:"-"`
	Tags      string    `json:"tags,omitempty"`
	Template  IntOrBool `json:"template,omitempty"`
	Timezone  string    `json:"timezone,omitempty"`

	TTY          *int      `json:"tty,omitempty"`
	Unprivileged IntOrBool `json:"unprivileged,omitempty"`
}

func (cc *ContainerConfig) indexedDeviceMaps() map[string]*map[string]string {
	return map[string]*map[string]string{
		"dev":    &cc.Devs,
		"mp":     &cc.Mps,
		"net":    &cc.Nets,
		"unused": &cc.Unuseds,
	}
}

func (cc *ContainerConfig) UnmarshalJSON(data []byte) error {
	type tmpContainerConfig ContainerConfig

	tmp := &struct {
		*tmpContainerConfig
	}{
		tmpContainerConfig: (*tmpContainerConfig)(cc),
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	cc.TagsSlice = strings.Split(cc.Tags, TagSeperator)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	routes := cc.indexedDeviceMaps()
	for k, v := range raw {
		prefix, ok := indexedDeviceKey(k)
		if !ok {
			continue
		}
		target, ok := routes[prefix]
		if !ok {
			continue
		}
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			continue
		}
		if *target == nil {
			*target = make(map[string]string)
		}
		(*target)[k] = s
	}

	return nil
}

type ContainerOptions []*ContainerOption
type ContainerOption struct {
	Name  string
	Value interface{}
}

type Appliances []*Appliance
type Appliance struct {
	client       *Client
	Node         string `json:",omitempty"`
	Os           string
	Source       string
	Type         string
	SHA512Sum    string
	Package      string
	Template     string
	Architecture string
	InfoPage     string
	Description  string
	ManageURL    string
	Version      string
	Section      string
	Headline     string
}

type Storages []*Storage
type Storage struct {
	client       *Client
	Node         string
	Name         string `json:"storage"`
	Enabled      int
	UsedFraction float64 `json:"used_fraction"`
	Active       int
	Content      string
	Shared       int
	Avail        uint64
	Type         string
	Used         uint64
	Total        uint64
	Storage      string `json:"-"`
}

func (s *Storage) UnmarshalJSON(b []byte) error {

	aux := &struct {
		Node         string      `json:"node,omitempty"`
		Name         string      `json:"storage,omitempty"`
		Enabled      json.Number `json:"enabled,omitempty"`
		UsedFraction json.Number `json:"used_fraction,omitempty"`
		Active       json.Number `json:"active,omitempty"`
		Content      string      `json:"content,omitempty"`
		Shared       json.Number `json:"shared,omitempty"`
		Avail        json.Number `json:"avail,omitempty"`
		Type         string      `json:"type,omitempty"`
		Used         json.Number `json:"used,omitempty"`
		Total        json.Number `json:"total,omitempty"`
	}{}

	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.UseNumber()
	if err := decoder.Decode(&aux); err != nil {
		return err
	}

	s.Node = aux.Node
	s.Name = aux.Name
	s.Storage = aux.Name
	s.Content = aux.Content
	s.Type = aux.Type

	if aux.Enabled != "" {
		if val, err := aux.Enabled.Int64(); err == nil {
			s.Enabled = int(val)
		}
	}

	if aux.Active != "" {
		if val, err := aux.Active.Int64(); err == nil {
			s.Active = int(val)
		}
	}

	if aux.Shared != "" {
		if val, err := aux.Shared.Int64(); err == nil {
			s.Shared = int(val)
		}
	}

	if aux.Avail != "" {

		if val, err := aux.Avail.Int64(); err == nil {
			s.Avail = clampToUint64(val)
		} else if val, err := aux.Avail.Float64(); err == nil {
			s.Avail = clampFloatToUint64(val)
		}
	}

	if aux.Used != "" {

		if val, err := aux.Used.Int64(); err == nil {
			s.Used = clampToUint64(val)
		} else if val, err := aux.Used.Float64(); err == nil {
			s.Used = clampFloatToUint64(val)
		}
	}

	if aux.Total != "" {

		if val, err := aux.Total.Int64(); err == nil {
			s.Total = clampToUint64(val)
		} else if val, err := aux.Total.Float64(); err == nil {
			s.Total = clampFloatToUint64(val)
		}
	}

	if aux.UsedFraction != "" {
		if val, err := aux.UsedFraction.Float64(); err == nil {
			s.UsedFraction = val
		}
	}

	return nil
}

func (storages *Storages) UnmarshalJSON(b []byte) error {
	var items []*Storage
	if err := json.Unmarshal(b, &items); err != nil {
		return err
	}
	*storages = items
	return nil
}

type ClusterBackups []*ClusterBackup

type ClusterBackup struct {
	client *Client

	ID               string    `json:"id,omitempty"`
	Schedule         string    `json:"schedule,omitempty"`
	Enabled          IntOrBool `json:"enabled,omitempty"`
	RepeatMissed     IntOrBool `json:"repeat-missed,omitempty"`
	All              IntOrBool `json:"all,omitempty"`
	NotesTemplate    string    `json:"notes-template,omitempty"`
	MailNotification string    `json:"mailnotification,omitempty"`
	MailTo           string    `json:"mailto,omitempty"`
	Mode             string    `json:"mode,omitempty"`
	Type             string    `json:"type,omitempty"`
	NextRun          uint64    `json:"next-run,omitempty"`
	Storage          string    `json:"storage,omitempty"`
	VMID             string    `json:"vmid,omitempty"`
	Exclude          string    `json:"exclude,omitempty"`
	Node             string    `json:"node,omitempty"`
	Pool             string    `json:"pool,omitempty"`
	BwLimit          uint64    `json:"bwlimit,omitempty"`
	Comment          string    `json:"comment,omitempty"`
	PruneBackups     string    `json:"prune-backups,omitempty"`
}

type ClusterBackupOptions struct {
	All                bool   `json:"all,omitempty"`
	BwLimit            uint64 `json:"bwlimit,omitempty"`
	Comment            string `json:"comment,omitempty"`
	Compress           string `json:"compress,omitempty"`
	Dow                string `json:"dow,omitempty"`
	DumpDir            string `json:"dumpdir,omitempty"`
	Enabled            bool   `json:"enabled,omitempty"`
	Exclude            string `json:"exclude,omitempty"`
	ExcludePath        string `json:"exclude-path,omitempty"`
	ID                 string `json:"id,omitempty"`
	IoNice             uint   `json:"ionice,omitempty"`
	LockWait           uint   `json:"lockwait,omitempty"`
	MailNotification   string `json:"mailnotification,omitempty"`
	MailTo             string `json:"mailto,omitempty"`
	MaxFiles           uint   `json:"maxfiles,omitempty"`
	Mode               string `json:"mode,omitempty"`
	Node               string `json:"node,omitempty"`
	NotesTemplate      string `json:"notes-template,omitempty"`
	NotificationMode   string `json:"notification-mode,omitempty"`
	NotificationPolicy string `json:"notification-policy,omitempty"`
	NotificationTarget string `json:"notification-target,omitempty"`
	Performance        string `json:"performance,omitempty"`
	Pigz               int    `json:"pigz,omitempty"`
	Pool               string `json:"pool,omitempty"`
	Protected          bool   `json:"protected,omitempty"`
	PruneBackups       string `json:"prune-backups,omitempty"`
	Quiet              bool   `json:"quiet,omitempty"`
	Remove             bool   `json:"remove,omitempty"`
	RepeatMissed       bool   `json:"repeat-missed,omitempty"`
	Schedule           string `json:"schedule,omitempty"`
	Script             string `json:"script,omitempty"`
	StdExcludes        bool   `json:"stdexcludes,omitempty"`
	Stop               bool   `json:"stop,omitempty"`
	StopWait           uint   `json:"stopwait,omitempty"`
	Storage            string `json:"storage,omitempty"`
	TmpDir             string `json:"tmpdir,omitempty"`
	VMID               string `json:"vmid,omitempty"`
	Zstd               uint   `json:"zstd,omitempty"`
}

type ClusterStorages []*ClusterStorage

type ClusterStorage struct {
	client   *Client
	Content  string
	Digest   string
	Storage  string
	Type     string
	Shared   int    `json:",omitempty"`
	Nodes    string `json:",omitempty"`
	Thinpool string `json:",omitempty"`
	Path     string `json:",omitempty"`
	VgName   string `json:",omitempty"`
}

type ClusterStorageOptions struct {
	Name  string
	Value string
}
type Volume interface {
	Delete() error
}

type ISOs []*ISO
type ISO struct{ Content }

type VzTmpls []*VzTmpl
type VzTmpl struct{ Content }

type Imports []*Import
type Import struct{ Content }

type Backups []*Backup
type Backup struct{ Content }

type Content struct {
	client  *Client
	URL     string
	Node    string
	Storage string `json:",omitempty"`
	Content string `json:",omitempty"`
	VolID   string `json:",omitempty"`
	CTime   uint64 `json:",omitempty"`
	Format  string
	Size    StringOrUint64
	Used    StringOrUint64 `json:",omitempty"`
	Path    string         `json:",omitempty"`
	Notes   string         `json:",omitempty"`
}

type IsTemplate bool

func (it *IsTemplate) UnmarshalJSON(b []byte) error {
	*it = true
	if string(b) == "\"\"" {
		*it = false
	}

	return nil
}

type StringOrInt int

func (d *StringOrInt) UnmarshalJSON(b []byte) error {
	str := strings.ReplaceAll(string(b), "\"", "")

	if str == "" || str == "null" {
		*d = StringOrInt(0)
		return nil
	}

	if !isFloat.MatchString(str) {
		return fmt.Errorf("failed to match %s: %s", isFloat.String(), str)
	}

	parsed, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return err
	}

	*d = StringOrInt(math.Trunc(parsed))
	return nil
}

type StringOrUint64 uint64

func (d *StringOrUint64) UnmarshalJSON(b []byte) error {
	str := strings.ReplaceAll(string(b), "\"", "")
	if str == "" || str == "null" {
		*d = StringOrUint64(0)
		return nil
	}

	if !isFloat.MatchString(str) {
		return fmt.Errorf("failed to match %s: %s", isFloat.String(), str)
	}

	parsed, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return err
	}

	*d = StringOrUint64(math.Trunc(parsed))

	return nil
}

type StringOrFloat64 float64

func (d *StringOrFloat64) UnmarshalJSON(b []byte) error {
	str := strings.ReplaceAll(string(b), "\"", "")
	if str == "" || str == "null" {
		*d = StringOrFloat64(0)
		return nil
	}

	if !isFloat.MatchString(str) {
		return fmt.Errorf("failed to match %s: %s", isFloat.String(), str)
	}

	parsed, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return err
	}
	*d = StringOrFloat64(parsed)
	return nil
}

type IntOrBool bool

func (b *IntOrBool) UnmarshalJSON(i []byte) error {
	parsed, err := strconv.ParseBool(string(i))
	if err != nil {
		return err
	}
	*b = IntOrBool(parsed)
	return nil
}

func (b *IntOrBool) MarshalJSON() ([]byte, error) {
	if *b {
		return []byte("1"), nil
	}
	return []byte("0"), nil
}

type CSV []string

func (c *CSV) UnmarshalJSON(b []byte) error {
	var list []string
	if err := json.Unmarshal(b, &list); err == nil {
		*c = CSV(list)
		return nil
	}

	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	if s == "" {
		*c = nil
		return nil
	}

	parts := strings.Split(s, ",")
	list = make([]string, 0, len(parts))
	for _, part := range parts {
		if v := strings.TrimSpace(part); v != "" {
			list = append(list, v)
		}
	}

	*c = CSV(list)
	return nil
}

func (c CSV) MarshalJSON() ([]byte, error) {
	if c == nil {
		return []byte("null"), nil
	}
	return json.Marshal(strings.Join([]string(c), ","))
}

type NodeNetworks []*NodeNetwork
type NodeNetwork struct {
	client  *Client
	Node    string `json:"-"`
	NodeAPI *Node  `json:"-"`

	Iface     string `json:"iface,omitempty"`
	Autostart int    `json:"autostart,omitempty"`

	CIDR               string `json:"cidr,omitempty"`
	CIDR6              string `json:"cidr6,omitempty"`
	Gateway            string `json:"gateway,omitempty"`
	Gateway6           string `json:"gateway6,omitempty"`
	MTU                string `json:"mtu,omitempty"`
	Netmask            string `json:"netmask,omitempty"`
	Netmask6           string `json:"netmask6,omitempty"`
	VLANID             string `json:"vlan-id,omitempty"`
	VLANRawDevice      string `json:"vlan-raw-device,omitempty"`
	BridgeVLANAware    int    `json:"bridge_vlan_aware,omitempty"`
	BridgePorts        string `json:"bridge_ports,omitempty"`
	BridgeStp          string `json:"bridge_stp,omitempty"`
	BridgeFd           string `json:"bridge_fd,omitempty"`
	Comments           string `json:"comments,omitempty"`
	Comments6          string `json:"comments6,omitempty"`
	BondPrimary        string `json:"bond-primary,omitempty"`
	BondMode           string `json:"bond_mode,omitempty"`
	BondXmit           string `json:"bond_xmit,omitempty"`
	BondXmitHashPolicy string `json:"bond_xmit_hash_policy,omitempty"`

	OVSBonds   string `json:"ovs_bonds,omitempty"`
	OVSBridge  string `json:"ovs_bridge,omitempty"`
	OVSOptions string `json:"ovs_options,omitempty"`
	OVSPorts   string `json:"ovs_ports,omitempty"`
	OVSTags    string `json:"ovs_tag,omitempty"`

	Slaves   string      `json:"slaves,omitempty"`
	Address  string      `json:"address,omitempty"`
	Address6 string      `json:"address6,omitempty"`
	Type     string      `json:"type,omitempty"`
	Active   StringOrInt `json:"active,omitempty"`
	Method   string      `json:"method,omitempty"`
	Method6  string      `json:"method6,omitempty"`
	Priority int         `json:"priority,omitempty"`
}

type AgentNetworkIPAddress struct {
	IPAddressType string `json:"ip-address-type"`
	IPAddress     string `json:"ip-address"`
	Prefix        int    `json:"prefix"`

	MacAddress string `json:"mac-address,omitempty"`
}

type AgentHostName struct {
	HostName string `json:"host-name"`
}

type AgentNetworkIface struct {
	Name            string                   `json:"name"`
	HardwareAddress string                   `json:"hardware-address"`
	IPAddresses     []*AgentNetworkIPAddress `json:"ip-addresses"`
}

type AgentOsInfo struct {
	Version       string `json:"version"`
	VersionID     string `json:"version-id"`
	ID            string `json:"id"`
	Machine       string `json:"machine"`
	PrettyName    string `json:"pretty-name"`
	Name          string `json:"name"`
	KernelRelease string `json:"kernel-release"`
	KernelVersion string `json:"kernel-version"`
}

type AgentExecStatus struct {
	Exited       int       `json:"exited"`
	ErrData      string    `json:"err-data"`
	ErrTruncated bool      `json:"err-truncated"`
	ExitCode     int       `json:"exitcode"`
	OutData      string    `json:"out-data"`
	OutTruncated IntOrBool `json:"out-truncated"`
	Signal       bool      `json:"signal"`
}

type AgentFileRead struct {
	Content   string    `json:"content"`
	Truncated IntOrBool `json:"truncated,omitempty"`
}

type AgentFsInfo struct {
	Name       string             `json:"name"`
	Mountpoint string             `json:"mountpoint"`
	Type       string             `json:"type"`
	UsedBytes  uint64             `json:"used-bytes,omitempty"`
	TotalBytes uint64             `json:"total-bytes,omitempty"`
	Disk       []*AgentFsInfoDisk `json:"disk,omitempty"`
}

type AgentFsInfoDisk struct {
	Serial        string        `json:"serial,omitempty"`
	BusType       string        `json:"bus-type,omitempty"`
	Bus           int           `json:"bus,omitempty"`
	Unit          int           `json:"unit,omitempty"`
	Target        int           `json:"target,omitempty"`
	PciController *AgentPciCtrl `json:"pci-controller,omitempty"`
	Dev           string        `json:"dev,omitempty"`
}

type AgentPciCtrl struct {
	Domain   int `json:"domain"`
	Bus      int `json:"bus"`
	Slot     int `json:"slot"`
	Function int `json:"function"`
}

type AgentTime int64

type AgentUser struct {
	User      string  `json:"user"`
	Domain    string  `json:"domain,omitempty"`
	LoginTime float64 `json:"login-time"`
}

type AgentVCPU struct {
	LogicalID  int  `json:"logical-id"`
	Online     bool `json:"online"`
	CanOffline bool `json:"can-offline,omitempty"`
}

type AgentInfo struct {
	Version           string              `json:"version"`
	SupportedCommands []*AgentCommandInfo `json:"supported_commands,omitempty"`
}

type AgentCommandInfo struct {
	Name            string `json:"name"`
	Enabled         bool   `json:"enabled"`
	SuccessResponse bool   `json:"success-response"`
}

type AgentMemoryBlock struct {
	PhysIndex  int  `json:"phys-index"`
	Online     bool `json:"online"`
	CanOffline bool `json:"can-offline,omitempty"`
}

type AgentMemoryBlockInfo struct {
	Size uint64 `json:"size"`
}

type AgentCommandIndexEntry struct {
	Name string `json:"name,omitempty"`
}

type AgentFsfreezeStatus string

type FirewallSecurityGroup struct {
	client  *Client
	Group   string          `json:"group,omitempty"`
	Comment string          `json:"comment,omitempty"`
	Rules   []*FirewallRule `json:"rules,omitempty"`
}

type fwRuleKind uint8

const (
	fwRuleKindUnknown fwRuleKind = iota
	fwRuleKindNode
	fwRuleKindQemu
	fwRuleKindLXC
)

type FirewallRule struct {
	client *Client
	kind   fwRuleKind
	node   string
	vmid   uint64

	Type     string `json:"type,omitempty"`
	Action   string `json:"action,omitempty"`
	Pos      int    `json:"pos,omitempty"`
	Comment  string `json:"comment,omitempty"`
	Dest     string `json:"dest,omitempty"`
	Dport    string `json:"dport,omitempty"`
	Enable   int    `json:"enable,omitempty"`
	IcmpType string `json:"icmp-type,omitempty"`
	Iface    string `json:"iface,omitempty"`
	Log      string `json:"log,omitempty"`
	Macro    string `json:"macro,omitempty"`
	Proto    string `json:"proto,omitempty"`
	Source   string `json:"source,omitempty"`
	Sport    string `json:"sport,omitempty"`
}

func (r *FirewallRule) IsEnable() bool {
	return r.Enable == 1
}

var errFirewallRuleNoParent = fmt.Errorf("firewall rule has no parent context; obtain via Node/VirtualMachine/Container.FirewallRule")

func (r *FirewallRule) Get(ctx context.Context) error {
	if r.client == nil {
		return errFirewallRuleNoParent
	}
	switch r.kind {
	case fwRuleKindNode:
		return r.client.Get(ctx, fmt.Sprintf("/nodes/%s/firewall/rules/%d", r.node, r.Pos), r)
	case fwRuleKindQemu:
		return r.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/firewall/rules/%d", r.node, r.vmid, r.Pos), r)
	case fwRuleKindLXC:
		return r.client.Get(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/rules/%d", r.node, r.vmid, r.Pos), r)
	}
	return errFirewallRuleNoParent
}

func (r *FirewallRule) Update(ctx context.Context) error {
	if r.client == nil {
		return errFirewallRuleNoParent
	}
	switch r.kind {
	case fwRuleKindNode:
		return r.client.Put(ctx, fmt.Sprintf("/nodes/%s/firewall/rules/%d", r.node, r.Pos), r, nil)
	case fwRuleKindQemu:
		return r.client.Put(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/firewall/rules/%d", r.node, r.vmid, r.Pos), r, nil)
	case fwRuleKindLXC:
		return r.client.Put(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/rules/%d", r.node, r.vmid, r.Pos), r, nil)
	}
	return errFirewallRuleNoParent
}

func (r *FirewallRule) Delete(ctx context.Context) error {
	if r.client == nil {
		return errFirewallRuleNoParent
	}
	switch r.kind {
	case fwRuleKindNode:
		return r.client.Delete(ctx, fmt.Sprintf("/nodes/%s/firewall/rules/%d", r.node, r.Pos), nil)
	case fwRuleKindQemu:
		return r.client.Delete(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/firewall/rules/%d", r.node, r.vmid, r.Pos), nil)
	case fwRuleKindLXC:
		return r.client.Delete(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/rules/%d", r.node, r.vmid, r.Pos), nil)
	}
	return errFirewallRuleNoParent
}

type FirewallNodeOption struct {
	Enable         *IntOrBool `json:"enable,omitempty"`
	LogLevelIn     string     `json:"log_level_in,omitempty"`
	LogLevelOut    string     `json:"log_level_out,omitempty"`
	LogNfConntrack IntOrBool  `json:"log_nf_conntrack,omitempty"`

	NDP IntOrBool `json:"ndp,omitempty"`

	Ntp                     IntOrBool `json:"-"`
	NFConntrackAllowInvalid IntOrBool `json:"nf_conntrack_allow_invalid,omitempty"`

	NFConntrackMax *int `json:"nf_conntrack_max,omitempty"`

	NFConntrackTCPTimeoutEstablished *int `json:"nf_conntrack_tcp_timeout_established,omitempty"`

	NFConntrackTCPTimeoutSynRecv *int      `json:"nf_conntrack_tcp_timeout_syn_recv,omitempty"`
	Nosmurfs                     IntOrBool `json:"nosmurfs,omitempty"`
	ProtectionSynflood           IntOrBool `json:"protection_synflood,omitempty"`

	ProtectionSynfloodBurst *int `json:"protection_synflood_burst,omitempty"`

	ProtectionSynfloodRate *int      `json:"protection_synflood_rate,omitempty"`
	SmurfLogLevel          string    `json:"smurf_log_level,omitempty"`
	TCPFlagsLogLevel       string    `json:"tcp_flags_log_level,omitempty"`
	TCPflags               IntOrBool `json:"tcpflags,omitempty"`
}

type FirewallVirtualMachineOption struct {
	Enable      IntOrBool `json:"enable,omitempty"`
	Dhcp        IntOrBool `json:"dhcp,omitempty"`
	Ipfilter    IntOrBool `json:"ipfilter,omitempty"`
	LogLevelIn  string    `json:"log_level_in,omitempty"`
	LogLevelOut string    `json:"log_level_out,omitempty"`

	Macfilter *IntOrBool `json:"macfilter,omitempty"`

	NDP IntOrBool `json:"ndp,omitempty"`

	Ntp       IntOrBool `json:"-"`
	PolicyIn  string    `json:"policy_in,omitempty"`
	PolicyOut string    `json:"policy_out,omitempty"`
	Radv      IntOrBool `json:"radv,omitempty"`
}

type VirtualMachineSnapshot struct {
	client *Client `json:"-"`

	Node string `json:"-"`

	VMID int `json:"-"`

	Name        string
	Vmstate     int
	Description string
	Snaptime    int64
	Parent      string
	Snapstate   string
}

type Pools []*Pool
type Pool struct {
	client  *Client
	PoolID  string            `json:"poolid,omitempty"`
	Comment string            `json:"comment,omitempty"`
	Members []ClusterResource `json:"members,omitempty"`
}

type PoolUpdateOption struct {
	Comment string `json:"comment,omitempty"`

	Delete IntOrBool `json:"delete,omitempty"`

	AllowMove IntOrBool `json:"allow-move,omitempty"`

	Storage string `json:"storage,omitempty"`

	VirtualMachines string `json:"vms,omitempty"`
}

type DomainType string

const (
	DomainTypeAD     = DomainType("ad")
	DomainTypeLDAP   = DomainType("ldap")
	DomainTypeOpenID = DomainType("openid")
	DomainTypePam    = DomainType("pam")
	DomainTypePVE    = DomainType("pve")
)

type Domains []*Domain
type Domain struct {
	client *Client
	Realm  string `json:",omitempty"`
	Type   string `json:",omitempty"`

	ACRValues      string    `json:"acr-values,omitempty"`
	AutoCreate     IntOrBool `json:"autocreate,omitempty"`
	BaseDN         string    `json:"base_dn,omitempty"`
	BindDN         string    `json:"bind_dn,omitempty"`
	CAPath         string    `json:"capath,omitempty"`
	CaseSensitive  IntOrBool `json:"case-sensitive,omitempty"`
	Cert           string    `json:"cert,omitempty"`
	CertKey        string    `json:"certkey,omitempty"`
	ClientID       string    `json:"client-id,omitempty"`
	ClientKey      string    `json:"client-key,omitempty"`
	Comment        string    `json:"comment,omitempty"`
	Default        IntOrBool `json:"default,omitempty"`
	DeleteList     string    `json:"delete,omitempty"`
	Digest         string    `json:"digest,omitempty"`
	Domain         string    `json:"domain,omitempty"`
	Filter         string    `json:"filter,omitempty"`
	GroupClasses   string    `json:"group_classes,omitempty"`
	GroupDN        string    `json:"group_dn,omitempty"`
	GroupFilter    string    `json:"group_filter,omitempty"`
	GroupName      string    `json:"group_name,omitempty"`
	IssuerURL      string    `json:"issuer-url,omitempty"`
	Mode           string    `json:"mode,omitempty"`
	Password       string    `json:"password,omitempty"`
	Port           int       `json:"port,omitempty"`
	Prompt         string    `json:"prompt,omitempty"`
	Scopes         string    `json:"scopes,omitempty"`
	Secure         IntOrBool `json:"secure,omitempty"`
	Server1        string    `json:"server1,omitempty"`
	Server2        string    `json:"server2,omitempty"`
	SSLVersion     string    `json:"sslversion,omitempty"`
	SyncDefaults   string    `json:"sync-defaults,omitempty"`
	SyncAttributes string    `json:"sync_attributes,omitempty"`
	TFA            string    `json:"tfa,omitempty"`
	UserAttr       string    `json:"user_attr,omitempty"`
	UserClasses    string    `json:"user_classes,omitempty"`
	Verify         IntOrBool `json:"verify"`
}

type DomainSyncOptions struct {
	DryRun IntOrBool `json:"dry-run,omitempty"`

	EnableNew *IntOrBool `json:"enable-new,omitempty"`

	RemoveVanished *string `json:"remove-vanished,omitempty"`
	Scope          string  `json:"scope,omitempty"`
}

type Groups []*Group
type Group struct {
	client  *Client
	GroupID string   `json:"groupid,omitempty"`
	Comment string   `json:"comment,omitempty"`
	Users   string   `json:"users,omitempty"`
	Members []string `json:"members,omitempty"`
}

type Users []*User
type User struct {
	client         *Client
	UserID         string           `json:"userid,omitempty"`
	Comment        string           `json:"comment,omitempty"`
	Email          string           `json:"email,omitempty"`
	Enable         IntOrBool        `json:"enable"`
	Expire         int              `json:"expire,omitempty"`
	Firstname      string           `json:"firstname,omitempty"`
	Lastname       string           `json:"lastname,omitempty"`
	Groups         CSV              `json:"groups,omitempty"`
	Keys           string           `json:"keys,omitempty"`
	Tokens         map[string]Token `json:"tokens,omitempty"`
	RealmType      string           `json:"realm-type,omitempty"`
	TFALockedUntil string           `json:"tfa-locked-until,omitempty"`
	TOTPLocked     IntOrBool        `json:"totp-locked,omitempty"`
}

type UserOptions struct {
	Append  IntOrBool `json:"append,omitempty"`
	Comment string    `json:"comment,omitempty"`
	Email   string    `json:"email,omitempty"`

	Enable    *IntOrBool `json:"enable,omitempty"`
	Expire    int        `json:"expire,omitempty"`
	Firstname string     `json:"firstname,omitempty"`
	Groups    CSV        `json:"groups,omitempty"`
	Keys      string     `json:"keys,omitempty"`
	Lastname  string     `json:"lastname,omitempty"`
}

type Tokens []*Token
type Token struct {
	TokenID string    `json:"tokenid,omitempty"`
	Comment string    `json:"comment,omitempty"`
	Expire  int       `json:"expire,omitempty"`
	Privsep IntOrBool `json:"privsep"`
}

type Roles []*Role
type Role struct {
	client  *Client
	RoleID  string    `json:"roleid,omitempty"`
	Privs   string    `json:"privs,omitempty"`
	Special IntOrBool `json:"special,omitempty"`
}

type ACLs []*ACL
type ACL struct {
	Path      string    `json:"path,omitempty"`
	RoleID    string    `json:"roleid,omitempty"`
	Type      string    `json:"type,omitempty"`
	UGID      string    `json:"ugid,omitempty"`
	Propagate IntOrBool `json:"propagate,omitempty"`
}

type ACLOptions struct {
	Path      string    `json:"path,omitempty"`
	Roles     string    `json:"roles,omitempty"`
	Groups    string    `json:"groups,omitempty"`
	Users     string    `json:"users,omitempty"`
	Tokens    string    `json:"tokens,omitempty"`
	Propagate IntOrBool `json:"propagate"`
	Delete    IntOrBool `json:"delete,omitempty"`
}

type StorageDownloadURLOptions struct {
	Content           string `json:"content,omitempty"`
	Filename          string `json:"filename,omitempty"`
	Node              string `json:"node,omitempty"`
	Storage           string `json:"storage,omitempty"`
	URL               string `json:"url,omitempty"`
	Checksum          string `json:"checksum,omitempty"`
	ChecksumAlgorithm string `json:"checksum-algorithm,omitempty"`
	Compression       string `json:"compression,omitempty"`

	VerifyCertificates *IntOrBool `json:"verify-certificates,omitempty"`
}

type StorageContentVerification struct {
	State string `json:"state,omitempty"`
	UPID  string `json:"upid,omitempty"`
}

type StorageContent struct {
	Format string         `json:"format,omitempty"`
	Size   uint64         `json:"size,omitempty"`
	Volid  string         `json:"volid,omitempty"`
	Ctime  StringOrUint64 `json:"ctime,omitempty"`

	Encrypted  string    `json:"encrypted,omitempty"`
	Notes      string    `json:"notes,omitempty"`
	Parent     string    `json:"parent,omitempty"`
	Protection IntOrBool `json:"protection,omitempty"`
	Used       uint64    `json:"used,omitempty"`

	Verification *StorageContentVerification `json:"verification,omitempty"`
	VMID         uint64                      `json:"vmid,omitempty"`
}

type StoragePruneBackupsOptions struct {
	PruneBackups string

	Type string

	VMID uint64
}

type PruneBackupItem struct {
	Volid string         `json:"volid"`
	Ctime StringOrUint64 `json:"ctime"`
	Mark  string         `json:"mark"`
	Type  string         `json:"type"`
	VMID  uint64         `json:"vmid,omitempty"`
}

type ImportMetadata struct {
	Type       string                  `json:"type"`
	Source     string                  `json:"source"`
	CreateArgs map[string]interface{}  `json:"create-args"`
	Disks      map[string]string       `json:"disks,omitempty"`
	Net        map[string]interface{}  `json:"net,omitempty"`
	Warnings   []ImportMetadataWarning `json:"warnings,omitempty"`
}

type ImportMetadataWarning struct {
	Type  string `json:"type"`
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type NodeDNS struct {
	Search string `json:"search,omitempty"`
	DNS1   string `json:"dns1,omitempty"`
	DNS2   string `json:"dns2,omitempty"`
	DNS3   string `json:"dns3,omitempty"`
}

type NodeCertificates []*NodeCertificate

type NodeCertificate struct {
	Filename      string   `json:"filename,omitempty"`
	Fingerprint   string   `json:"fingerprint,omitempty"`
	Issuer        string   `json:"issuer,omitempty"`
	NotAfter      string   `json:"not-after,omitempty"`
	NotBefore     string   `json:"not-before,omitempty"`
	Pem           string   `json:"pem,omitempty"`
	PublicKeyBits int      `json:"public-key-bits,omitempty"`
	PublicKeyType string   `json:"public-key-type,omitempty"`
	San           []string `json:"san,omitempty"`
	Subject       string   `json:"subject,omitempty"`
}

type CustomCertificate struct {
	Certificates string `json:"certificates,omitempty"`
	Force        bool   `json:"force,omitempty"`
	Key          string `json:"key,omitempty"`
	Restart      bool   `json:"restart,omitempty"`
}

type NewUser struct {
	UserID    string   `json:"userid"`
	Comment   string   `json:"comment,omitempty"`
	Email     string   `json:"email,omitempty"`
	Enable    bool     `json:"enable"`
	Expire    int      `json:"expire,omitempty"`
	Firstname string   `json:"firstname,omitempty"`
	Groups    CSV      `json:"groups,omitempty"`
	Keys      []string `json:"keys,omitempty"`
	Lastname  string   `json:"lastname,omitempty"`
	Password  string   `json:"password,omitempty"`
}

type TFA struct {
	Realm string   `json:"realm,omitempty"`
	Types []string `json:"types,omitempty"`
	User  string   `json:"user,omitempty"`
}

type NewAPIToken struct {
	FullTokenID string      `json:"full-tokenid,omitempty"`
	Info        interface{} `json:"info,omitempty"`
	Value       string      `json:"value,omitempty"`
}

type VNCProxyOptions struct {
	Websocket string `json:"websocket,omitempty"`
	Height    int    `json:"height,omitempty"`
	Width     int    `json:"width,omitempty"`
}

type ContainerSnapshot struct {
	client *Client `json:"-"`

	Node string `json:"-"`

	VMID int `json:"-"`

	Description          string `json:"description,omitempty"`
	Name                 string `json:"name,omitempty"`
	Parent               string `json:"parent,omitempty"`
	SnapshotCreationTime int64  `json:"snaptime,omitempty"`
}

type Firewall struct {
	Aliases []*FirewallAlias    `json:"aliases,omitempty"`
	Ipset   []*FirewallIPSet    `json:"ipset,omitempty"`
	Rules   []*FirewallRule     `json:"rules,omitempty"`
	Options *FirewallNodeOption `json:"options,omitempty"`
}

type FirewallAlias struct {
	Cidr    string `json:"cidr,omitempty"`
	Digest  string `json:"digest,omitempty"`
	Name    string `json:"name,omitempty"`
	Comment string `json:"comment,omitempty"`
}

type FirewallIPSet struct {
	Name    string `json:"name,omitempty"`
	Digest  string `json:"digest,omitempty"`
	Comment string `json:"comment,omitempty"`
}

type FirewallIPSetCreationOption struct {
	Name    string `json:"name"`
	Digest  string `json:"digest,omitempty"`
	Comment string `json:"comment,omitempty"`
	Rename  string `json:"rename,omitempty"`
}

type FirewallIPSetEntry struct {
	CIDR    string `json:"cidr,omitempty"`
	Digest  string `json:"digest,omitempty"`
	Comment string `json:"comment,omitempty"`
	NoMatch bool   `json:"nomatch,omitempty"`
}

type FirewallIPSetEntryCreationOption struct {
	CIDR    string `json:"cidr"`
	Comment string `json:"comment,omitempty"`
	NoMatch bool   `json:"nomatch,omitempty"`
}

type FirewallIPSetEntryUpdateOption struct {
	Comment string `json:"comment,omitempty"`
	Digest  string `json:"digest,omitempty"`
	NoMatch bool   `json:"nomatch,omitempty"`
}

type FirewallClusterOption struct {
	Enable        int        `json:"enable,omitempty"`
	Ebtables      *IntOrBool `json:"ebtables,omitempty"`
	LogRatelimit  string     `json:"log_ratelimit,omitempty"`
	PolicyForward string     `json:"policy_forward,omitempty"`
	PolicyIn      string     `json:"policy_in,omitempty"`
	PolicyOut     string     `json:"policy_out,omitempty"`
}

type FirewallClusterOptionUpdateOption struct {
	Delete        string     `json:"delete,omitempty"`
	Digest        string     `json:"digest,omitempty"`
	Enable        int        `json:"enable,omitempty"`
	Ebtables      *IntOrBool `json:"ebtables,omitempty"`
	LogRatelimit  string     `json:"log_ratelimit,omitempty"`
	PolicyForward string     `json:"policy_forward,omitempty"`
	PolicyIn      string     `json:"policy_in,omitempty"`
	PolicyOut     string     `json:"policy_out,omitempty"`
}

type FirewallAliasCreateOption struct {
	CIDR    string `json:"cidr"`
	Name    string `json:"name"`
	Comment string `json:"comment,omitempty"`
}

type FirewallAliasUpdateOption struct {
	CIDR    string `json:"cidr,omitempty"`
	Comment string `json:"comment,omitempty"`
	Digest  string `json:"digest,omitempty"`
	Rename  string `json:"rename,omitempty"`
}

type FirewallMacro struct {
	Macro string `json:"macro"`
	Descr string `json:"descr,omitempty"`
}

type HAGroup struct {
	Group      string    `json:"group"`
	Type       string    `json:"type,omitempty"`
	Nodes      string    `json:"nodes,omitempty"`
	Comment    string    `json:"comment,omitempty"`
	Digest     string    `json:"digest,omitempty"`
	NoFailback IntOrBool `json:"nofailback,omitempty"`
	Restricted IntOrBool `json:"restricted,omitempty"`
}

type HAGroupCreateOption struct {
	Group      string    `json:"group"`
	Nodes      string    `json:"nodes"`
	Type       string    `json:"type,omitempty"`
	Comment    string    `json:"comment,omitempty"`
	NoFailback IntOrBool `json:"nofailback,omitempty"`
	Restricted IntOrBool `json:"restricted,omitempty"`
}

type HAGroupUpdateOption struct {
	Delete     string    `json:"delete,omitempty"`
	Digest     string    `json:"digest,omitempty"`
	Comment    string    `json:"comment,omitempty"`
	Nodes      string    `json:"nodes,omitempty"`
	NoFailback IntOrBool `json:"nofailback,omitempty"`
	Restricted IntOrBool `json:"restricted,omitempty"`
}

type HAResource struct {
	SID         string     `json:"sid"`
	Type        string     `json:"type,omitempty"`
	Group       string     `json:"group,omitempty"`
	Comment     string     `json:"comment,omitempty"`
	Digest      string     `json:"digest,omitempty"`
	State       *string    `json:"state,omitempty"`
	Failback    *IntOrBool `json:"failback,omitempty"`
	MaxRelocate *int       `json:"max_relocate,omitempty"`
	MaxRestart  *int       `json:"max_restart,omitempty"`
}

type HAResourceCreateOption struct {
	SID         string     `json:"sid"`
	Type        string     `json:"type,omitempty"`
	Group       string     `json:"group,omitempty"`
	Comment     string     `json:"comment,omitempty"`
	State       *string    `json:"state,omitempty"`
	Failback    *IntOrBool `json:"failback,omitempty"`
	MaxRelocate *int       `json:"max_relocate,omitempty"`
	MaxRestart  *int       `json:"max_restart,omitempty"`
}

type HAResourceUpdateOption struct {
	Delete      string     `json:"delete,omitempty"`
	Digest      string     `json:"digest,omitempty"`
	Group       string     `json:"group,omitempty"`
	Comment     string     `json:"comment,omitempty"`
	State       *string    `json:"state,omitempty"`
	Failback    *IntOrBool `json:"failback,omitempty"`
	MaxRelocate *int       `json:"max_relocate,omitempty"`
	MaxRestart  *int       `json:"max_restart,omitempty"`
}

type HARule struct {
	Rule      string    `json:"rule"`
	Type      string    `json:"type,omitempty"`
	Affinity  string    `json:"affinity,omitempty"`
	Comment   string    `json:"comment,omitempty"`
	Digest    string    `json:"digest,omitempty"`
	Nodes     string    `json:"nodes,omitempty"`
	Resources string    `json:"resources,omitempty"`
	Disable   IntOrBool `json:"disable,omitempty"`
	Strict    IntOrBool `json:"strict,omitempty"`
}

type HARuleCreateOption struct {
	Rule      string    `json:"rule"`
	Type      string    `json:"type"`
	Resources string    `json:"resources"`
	Affinity  string    `json:"affinity,omitempty"`
	Comment   string    `json:"comment,omitempty"`
	Nodes     string    `json:"nodes,omitempty"`
	Disable   IntOrBool `json:"disable,omitempty"`
	Strict    IntOrBool `json:"strict,omitempty"`
}

type HARuleUpdateOption struct {
	Delete    string    `json:"delete,omitempty"`
	Digest    string    `json:"digest,omitempty"`
	Type      string    `json:"type,omitempty"`
	Affinity  string    `json:"affinity,omitempty"`
	Comment   string    `json:"comment,omitempty"`
	Nodes     string    `json:"nodes,omitempty"`
	Resources string    `json:"resources,omitempty"`
	Disable   IntOrBool `json:"disable,omitempty"`
	Strict    IntOrBool `json:"strict,omitempty"`
}

type HAStatusEntry struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Status       string `json:"status,omitempty"`
	Node         string `json:"node,omitempty"`
	Quorate      int    `json:"quorate,omitempty"`
	CRMState     string `json:"crm_state,omitempty"`
	LRMState     string `json:"lrm_state,omitempty"`
	Service      string `json:"service,omitempty"`
	ServiceState string `json:"state,omitempty"`
	Request      string `json:"request_state,omitempty"`
	Comment      string `json:"comment,omitempty"`
	Timestamp    int64  `json:"timestamp,omitempty"`
}

type ReplicationJob struct {
	ID        string           `json:"id"`
	Target    string           `json:"target,omitempty"`
	Type      string           `json:"type,omitempty"`
	Schedule  *string          `json:"schedule,omitempty"`
	Comment   string           `json:"comment,omitempty"`
	Disable   IntOrBool        `json:"disable,omitempty"`
	Rate      *StringOrFloat64 `json:"rate,omitempty"`
	RemoveJob string           `json:"remove_job,omitempty"`
	Source    string           `json:"source,omitempty"`
	Guest     int              `json:"guest,omitempty"`
	JobNum    int              `json:"jobnum,omitempty"`
}

type ReplicationJobOptions struct {
	ID        string           `json:"id"`
	Target    string           `json:"target"`
	Type      string           `json:"type"`
	Schedule  *string          `json:"schedule,omitempty"`
	Comment   string           `json:"comment,omitempty"`
	Disable   IntOrBool        `json:"disable,omitempty"`
	Rate      *StringOrFloat64 `json:"rate,omitempty"`
	RemoveJob string           `json:"remove_job,omitempty"`
	Source    string           `json:"source,omitempty"`
}

type ReplicationJobUpdateOption struct {
	Delete    string           `json:"delete,omitempty"`
	Digest    string           `json:"digest,omitempty"`
	Schedule  *string          `json:"schedule,omitempty"`
	Comment   string           `json:"comment,omitempty"`
	Disable   IntOrBool        `json:"disable,omitempty"`
	Rate      *StringOrFloat64 `json:"rate,omitempty"`
	RemoveJob string           `json:"remove_job,omitempty"`
	Source    string           `json:"source,omitempty"`
}

type HAManagerStatus struct {
	ManagerStatus map[string]any            `json:"manager_status,omitempty"`
	NodeStatus    map[string]string         `json:"node_status,omitempty"`
	ServiceStatus map[string]map[string]any `json:"service_status,omitempty"`
	Quorum        map[string]any            `json:"quorum,omitempty"`
}

type (
	VirtualMachineBackupMode               = string
	VirtualMachineBackupCompress           = string
	VirtualMachineBackupNotificationPolicy = string
)

const (
	VirtualMachineBackupModeSnapshot = VirtualMachineBackupMode("snapshot")
	VirtualMachineBackupModeSuspend  = VirtualMachineBackupMode("suspend")
	VirtualMachineBackupModeStop     = VirtualMachineBackupMode("stop")

	VirtualMachineBackupCompressZero = VirtualMachineBackupCompress("0")
	VirtualMachineBackupCompressOne  = VirtualMachineBackupCompress("1")
	VirtualMachineBackupCompressGzip = VirtualMachineBackupCompress("gzip")
	VirtualMachineBackupCompressLzo  = VirtualMachineBackupCompress("lzo")
	VirtualMachineBackupCompressZstd = VirtualMachineBackupCompress("zstd")

	VirtualMachineBackupNotificationPolicyAlways  = VirtualMachineBackupNotificationPolicy("always")
	VirtualMachineBackupNotificationPolicyFailure = VirtualMachineBackupNotificationPolicy("failure")
	VirtualMachineBackupNotificationPolicyNever   = VirtualMachineBackupNotificationPolicy("never")
)

type VirtualMachineBackupOptions struct {
	All         IntOrBool                    `json:"all,omitempty"`
	BwLimit     uint                         `json:"bwlimit,omitempty"`
	Compress    VirtualMachineBackupCompress `json:"compress,omitempty"`
	DumpDir     string                       `json:"dumpDir,omitempty"`
	Exclude     string                       `json:"exclude,omitempty"`
	ExcludePath []string                     `json:"exclude-path,omitempty"`

	IoNice *uint `json:"ionice,omitempty"`

	LockWait           *uint                                  `json:"lockwait,omitempty"`
	MailTo             string                                 `json:"mailto,omitempty"`
	Mode               VirtualMachineBackupMode               `json:"mode,omitempty"`
	Node               string                                 `json:"node,omitempty"`
	NotesTemplate      string                                 `json:"notes-template,omitempty"`
	NotificationPolicy VirtualMachineBackupNotificationPolicy `json:"notification-policy,omitempty"`
	NotificationTarget string                                 `json:"notification-target,omitempty"`
	Performance        string                                 `json:"performance,omitempty"`
	Pigz               int                                    `json:"pigz,omitempty"`
	Pool               string                                 `json:"pool,omitempty"`
	Protected          string                                 `json:"protected,omitempty"`

	PruneBackups *string   `json:"prune-backups,omitempty"`
	Quiet        IntOrBool `json:"quiet,omitempty"`

	Remove *IntOrBool `json:"remove,omitempty"`
	Script string     `json:"script,omitempty"`

	StdExcludes *IntOrBool `json:"stdexcludes,omitempty"`
	StdOut      IntOrBool  `json:"stdout,omitempty"`
	Stop        IntOrBool  `json:"stop,omitempty"`

	StopWait *uint  `json:"stopwait,omitempty"`
	Storage  string `json:"storage,omitempty"`
	TmpDir   string `json:"tmpdir,omitempty"`
	VMID     uint64 `json:"vmid,omitempty"`

	Zstd *uint `json:"zstd,omitempty"`
}

type Separator = string

const (
	StringSeparator = Separator("\n")
	FieldSeparator  = Separator(":")
	SpaceSeparator  = Separator(" ")
)

type VzdumpConfig struct {
	Boot       string `json:"boot"`
	CiPassword string `json:"cipassword"`
	CiUser     string `json:"ciuser"`
	Cores      uint64 `json:"cores,string"`
	Memory     uint64 `json:"memory,string"`
	Meta       string `json:"meta"`
	Numa       string `json:"numa"`
	OsType     string `json:"ostype"`
	Scsihw     string `json:"scsihw"`
	Sockets    uint64 `json:"sockets,string"`

	SSHKeys string `json:"sshkeys"`
	VmgenID string `json:"vmgenid"`

	IDE0 string `json:"ide0,omitempty"`
	IDE1 string `json:"ide1,omitempty"`
	IDE2 string `json:"ide2,omitempty"`
	IDE3 string `json:"ide3,omitempty"`

	SCSI0  string `json:"scsi0,omitempty"`
	SCSI1  string `json:"scsi1,omitempty"`
	SCSI2  string `json:"scsi2,omitempty"`
	SCSI3  string `json:"scsi3,omitempty"`
	SCSI4  string `json:"scsi4,omitempty"`
	SCSI5  string `json:"scsi5,omitempty"`
	SCSI6  string `json:"scsi6,omitempty"`
	SCSI7  string `json:"scsi7,omitempty"`
	SCSI8  string `json:"scsi8,omitempty"`
	SCSI9  string `json:"scsi9,omitempty"`
	SCSI10 string `json:"scsi10,omitempty"`
	SCSI11 string `json:"scsi11,omitempty"`
	SCSI12 string `json:"scsi12,omitempty"`
	SCSI13 string `json:"scsi13,omitempty"`
	SCSI14 string `json:"scsi14,omitempty"`
	SCSI15 string `json:"scsi15,omitempty"`
	SCSI16 string `json:"scsi16,omitempty"`
	SCSI17 string `json:"scsi17,omitempty"`
	SCSI18 string `json:"scsi18,omitempty"`
	SCSI19 string `json:"scsi19,omitempty"`
	SCSI20 string `json:"scsi20,omitempty"`
	SCSI21 string `json:"scsi21,omitempty"`
	SCSI22 string `json:"scsi22,omitempty"`
	SCSI23 string `json:"scsi23,omitempty"`
	SCSI24 string `json:"scsi24,omitempty"`
	SCSI25 string `json:"scsi25,omitempty"`
	SCSI26 string `json:"scsi26,omitempty"`
	SCSI27 string `json:"scsi27,omitempty"`
	SCSI28 string `json:"scsi28,omitempty"`
	SCSI29 string `json:"scsi29,omitempty"`
	SCSI30 string `json:"scsi30,omitempty"`

	SATA0 string `json:"sata0,omitempty"`
	SATA1 string `json:"sata1,omitempty"`
	SATA2 string `json:"sata2,omitempty"`
	SATA3 string `json:"sata3,omitempty"`
	SATA4 string `json:"sata4,omitempty"`
	SATA5 string `json:"sata5,omitempty"`

	VirtIO0  string `json:"virtio0,omitempty"`
	VirtIO1  string `json:"virtio1,omitempty"`
	VirtIO2  string `json:"virtio2,omitempty"`
	VirtIO3  string `json:"virtio3,omitempty"`
	VirtIO4  string `json:"virtio4,omitempty"`
	VirtIO5  string `json:"virtio5,omitempty"`
	VirtIO6  string `json:"virtio6,omitempty"`
	VirtIO7  string `json:"virtio7,omitempty"`
	VirtIO8  string `json:"virtio8,omitempty"`
	VirtIO9  string `json:"virtio9,omitempty"`
	VirtIO10 string `json:"virtio10,omitempty"`
	VirtIO11 string `json:"virtio11,omitempty"`
	VirtIO12 string `json:"virtio12,omitempty"`
	VirtIO13 string `json:"virtio13,omitempty"`
	VirtIO14 string `json:"virtio14,omitempty"`
	VirtIO15 string `json:"virtio15,omitempty"`

	Unused0 string `json:"unused0,omitempty"`
	Unused1 string `json:"unused1,omitempty"`
	Unused2 string `json:"unused2,omitempty"`
	Unused3 string `json:"unused3,omitempty"`
	Unused4 string `json:"unused4,omitempty"`
	Unused5 string `json:"unused5,omitempty"`
	Unused6 string `json:"unused6,omitempty"`
	Unused7 string `json:"unused7,omitempty"`
	Unused8 string `json:"unused8,omitempty"`
	Unused9 string `json:"unused9,omitempty"`

	Net0 string `json:"net0,omitempty"`
	Net1 string `json:"net1,omitempty"`
	Net2 string `json:"net2,omitempty"`
	Net3 string `json:"net3,omitempty"`
	Net4 string `json:"net4,omitempty"`
	Net5 string `json:"net5,omitempty"`
	Net6 string `json:"net6,omitempty"`
	Net7 string `json:"net7,omitempty"`
	Net8 string `json:"net8,omitempty"`
	Net9 string `json:"net9,omitempty"`

	Numa0 string `json:"numa0,omitempty"`
	Numa1 string `json:"numa1,omitempty"`
	Numa2 string `json:"numa2,omitempty"`
	Numa3 string `json:"numa3,omitempty"`
	Numa4 string `json:"numa4,omitempty"`
	Numa5 string `json:"numa5,omitempty"`
	Numa6 string `json:"numa6,omitempty"`
	Numa7 string `json:"numa7,omitempty"`
	Numa8 string `json:"numa8,omitempty"`
	Numa9 string `json:"numa9,omitempty"`

	HostPCI0 string `json:"hostpci0,omitempty"`
	HostPCI1 string `json:"hostpci1,omitempty"`
	HostPCI2 string `json:"hostpci2,omitempty"`
	HostPCI3 string `json:"hostpci3,omitempty"`
	HostPCI4 string `json:"hostpci4,omitempty"`
	HostPCI5 string `json:"hostpci5,omitempty"`
	HostPCI6 string `json:"hostpci6,omitempty"`
	HostPCI7 string `json:"hostpci7,omitempty"`
	HostPCI8 string `json:"hostpci8,omitempty"`
	HostPCI9 string `json:"hostpci9,omitempty"`

	Serial0 string `json:"serial0,omitempty"`
	Serial1 string `json:"serial1,omitempty"`
	Serial2 string `json:"serial2,omitempty"`
	Serial3 string `json:"serial3,omitempty"`

	USB0  string `json:"usb0,omitempty"`
	USB1  string `json:"usb1,omitempty"`
	USB2  string `json:"usb2,omitempty"`
	USB3  string `json:"usb3,omitempty"`
	USB4  string `json:"usb4,omitempty"`
	USB5  string `json:"usb5,omitempty"`
	USB6  string `json:"usb6,omitempty"`
	USB7  string `json:"usb7,omitempty"`
	USB8  string `json:"usb8,omitempty"`
	USB9  string `json:"usb9,omitempty"`
	USB10 string `json:"usb10,omitempty"`
	USB11 string `json:"usb11,omitempty"`
	USB12 string `json:"usb12,omitempty"`
	USB13 string `json:"usb13,omitempty"`
	USB14 string `json:"usb14,omitempty"`

	Parallel0 string `json:"parallel0,omitempty"`
	Parallel1 string `json:"parallel1,omitempty"`
	Parallel2 string `json:"parallel2,omitempty"`

	IPConfig0 string `json:"ipconfig0,omitempty"`
	IPConfig1 string `json:"ipconfig1,omitempty"`
	IPConfig2 string `json:"ipconfig2,omitempty"`
	IPConfig3 string `json:"ipconfig3,omitempty"`
	IPConfig4 string `json:"ipconfig4,omitempty"`
	IPConfig5 string `json:"ipconfig5,omitempty"`
	IPConfig6 string `json:"ipconfig6,omitempty"`
	IPConfig7 string `json:"ipconfig7,omitempty"`
	IPConfig8 string `json:"ipconfig8,omitempty"`
	IPConfig9 string `json:"ipconfig9,omitempty"`
}

type PendingConfiguration []PendingConfigItem

type PendingConfigItem struct {
	Key    string `json:"key,omitempty"`
	Delete *int   `json:"delete,omitempty"`

	Pending interface{} `json:"pending,omitempty"`
	Value   interface{} `json:"value,omitempty"`
}

type VNet struct {
	client *Client `json:"-"`

	Name      string `json:"vnet,omitempty"`
	Type      string `json:"type,omitempty"`
	Zone      string `json:"zone,omitempty"`
	Alias     string `json:"alias,omitempty"`
	VlanAware int    `json:"vlanaware,omitempty"`
	Tag       uint32 `json:"tag,omitempty"`
}

type VNetOptions struct {
	Name         string    `json:"vnet"`
	Zone         string    `json:"zone"`
	Alias        string    `json:"alias,omitempty"`
	IsolatePorts IntOrBool `json:"isolate-ports,omitempty"`
	Tag          uint32    `json:"tag,omitempty"`
	Type         string    `json:"type,omitempty"`
	VlanAware    IntOrBool `json:"vlanaware,omitempty"`
}
type NetRange struct {
	StartAddress string `json:"start-address,omitempty"`
	EndAddress   string `json:"end-address,omitempty"`
}
type VNetSubnet struct {
	client *Client `json:"-"`

	CIDR      string     `json:"cidr,omitempty"`
	Gateway   string     `json:"gateway,omitempty"`
	Netmask   string     `json:"mask,omitempty"`
	Type      string     `json:"type,omitempty"`
	Zone      string     `json:"zone,omitempty"`
	VNet      string     `json:"vnet,omitempty"`
	SNAT      int        `json:"snat,omitempty"`
	Network   string     `json:"network,omitempty"`
	ID        string     `json:"id,omitempty"`
	DhcpRange []NetRange `json:"dhcp-range,omitempty"`
}
type IPAM struct {
	Hostname string `json:"hostname,omitempty"`
	IP       string `json:"ip,omitempty"`
	Mac      string `json:"mac,omitempty"`
	Subnet   string `json:"subnet,omitempty"`
	VMID     string `json:"vmid,omitempty"`
	VNet     string `json:"vnet,omitempty"`
	Zone     string `json:"zone,omitempty"`
	Gateway  int    `json:"gateway,omitempty"`
}

type SDNZone struct {
	Name       string `json:"zone"`
	Type       string `json:"type"`
	DHCP       string `json:"dhcp,omitempty"`
	DNS        string `json:"dns,omitempty"`
	DNSZone    string `json:"dnszone,omitempty"`
	IPAM       string `json:"ipam,omitempty"`
	MTU        int    `json:"mtu,omitempty"`
	Nodes      CSV    `json:"nodes,omitempty"`
	Peers      CSV    `json:"peers,omitempty"`
	Pending    bool   `json:"pending,omitempty"`
	ReverseDNS string `json:"reversedns,omitempty"`
	State      string `json:"state,omitempty"`
}

type SDNZoneOptions struct {
	Name                     string    `json:"zone"`
	Type                     string    `json:"type"`
	AdvertiseSubnets         IntOrBool `json:"advertise-subnets,omitempty"`
	Bridge                   string    `json:"bridge,omitempty"`
	BridgeDisableMACLearning IntOrBool `json:"bridge-disable-mac-learning,omitempty"`
	Controller               string    `json:"controller,omitempty"`
	DHCP                     string    `json:"dhcp,omitempty"`
	DisableARPNDSuppression  IntOrBool `json:"disable-arp-nd-suppression,omitempty"`
	DNS                      string    `json:"dns,omitempty"`
	DNSZone                  string    `json:"dnszone,omitempty"`
	DPID                     int       `json:"dpid,omitempty"`
	ExitNodes                string    `json:"exit-nodes,omitempty"`
	ExitNodesLocalRouting    bool      `json:"exit-nodes-local-routing,omitempty"`
	ExitNodesPrimary         string    `json:"exit-nodes-primary,omitempty"`
	Fabric                   string    `json:"fabric,omitempty"`
	IPAM                     string    `json:"ipam,omitempty"`
	MAC                      string    `json:"mac,omitempty"`
	MTU                      int       `json:"mtu,omitempty"`
	Nodes                    string    `json:"nodes,omitempty"`
	Peers                    string    `json:"peers,omitempty"`
	ReverseDNS               string    `json:"reversedns,omitempty"`
	RTImport                 string    `json:"rt-import,omitempty"`
	Tag                      uint      `json:"tag,omitempty"`

	VLANProtocol *string `json:"vlan-protocol,omitempty"`
	VRFVXLAN     int     `json:"vrf-vxlan,omitempty"`

	VXLANPort *uint16 `json:"vxlan-port,omitempty"`
}

type SDNController struct {
	client *Client `json:"-"`

	Controller          string `json:"controller,omitempty"`
	Type                string `json:"type,omitempty"`
	ASN                 uint32 `json:"asn,omitempty"`
	BGPMode             string `json:"bgp-mode,omitempty"`
	BGPMultipathASRelax bool   `json:"bgp-multipath-as-relax,omitempty"`
	EBGP                bool   `json:"ebgp,omitempty"`
	EBGPMultihop        int    `json:"ebgp-multihop,omitempty"`
	ISISDomain          string `json:"isis-domain,omitempty"`
	ISISIfaces          string `json:"isis-ifaces,omitempty"`
	ISISNet             string `json:"isis-net,omitempty"`
	Loopback            string `json:"loopback,omitempty"`
	Node                string `json:"node,omitempty"`
	Nodes               string `json:"nodes,omitempty"`
	PeerGroupName       string `json:"peer-group-name,omitempty"`
	Peers               string `json:"peers,omitempty"`
	State               string `json:"state,omitempty"`
	Digest              string `json:"digest,omitempty"`
}

type SDNControllerOptions struct {
	Controller              string `json:"controller,omitempty"`
	Type                    string `json:"type,omitempty"`
	ASN                     uint32 `json:"asn,omitempty"`
	BGPMode                 string `json:"bgp-mode,omitempty"`
	BGPMultipathASPathRelax bool   `json:"bgp-multipath-as-path-relax,omitempty"`
	EBGP                    bool   `json:"ebgp,omitempty"`
	EBGPMultihop            int    `json:"ebgp-multihop,omitempty"`
	Fabric                  string `json:"fabric,omitempty"`
	ISISDomain              string `json:"isis-domain,omitempty"`
	ISISIfaces              string `json:"isis-ifaces,omitempty"`
	ISISNet                 string `json:"isis-net,omitempty"`
	Loopback                string `json:"loopback,omitempty"`
	Node                    string `json:"node,omitempty"`
	Nodes                   string `json:"nodes,omitempty"`
	PeerGroupName           string `json:"peer-group-name,omitempty"`
	Peers                   string `json:"peers,omitempty"`
	RouteMapIn              string `json:"route-map-in,omitempty"`
	RouteMapOut             string `json:"route-map-out,omitempty"`
	LockToken               string `json:"lock-token,omitempty"`
	Digest                  string `json:"digest,omitempty"`
	Delete                  string `json:"delete,omitempty"`
}

type SDNDNS struct {
	client *Client `json:"-"`

	DNS           string `json:"dns,omitempty"`
	Type          string `json:"type,omitempty"`
	URL           string `json:"url,omitempty"`
	Key           string `json:"key,omitempty"`
	TTL           int    `json:"ttl,omitempty"`
	ReverseMaskV6 int    `json:"reversemaskv6,omitempty"`
	ReverseV6Mask int    `json:"reversev6mask,omitempty"`
	Fingerprint   string `json:"fingerprint,omitempty"`
	Digest        string `json:"digest,omitempty"`
}

type SDNDNSOptions struct {
	DNS           string `json:"dns,omitempty"`
	Type          string `json:"type,omitempty"`
	URL           string `json:"url,omitempty"`
	Key           string `json:"key,omitempty"`
	TTL           int    `json:"ttl,omitempty"`
	ReverseMaskV6 int    `json:"reversemaskv6,omitempty"`
	ReverseV6Mask int    `json:"reversev6mask,omitempty"`
	Fingerprint   string `json:"fingerprint,omitempty"`
	LockToken     string `json:"lock-token,omitempty"`
	Digest        string `json:"digest,omitempty"`
	Delete        string `json:"delete,omitempty"`
}

type SDNIPAM struct {
	client *Client `json:"-"`

	IPAM        string `json:"ipam,omitempty"`
	Type        string `json:"type,omitempty"`
	URL         string `json:"url,omitempty"`
	Token       string `json:"token,omitempty"`
	Section     int    `json:"section,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	Digest      string `json:"digest,omitempty"`
}

type SDNIPAMOptions struct {
	IPAM        string `json:"ipam,omitempty"`
	Type        string `json:"type,omitempty"`
	URL         string `json:"url,omitempty"`
	Token       string `json:"token,omitempty"`
	Section     int    `json:"section,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	LockToken   string `json:"lock-token,omitempty"`
	Digest      string `json:"digest,omitempty"`
	Delete      string `json:"delete,omitempty"`
}

type SDNFabric struct {
	client *Client `json:"-"`

	ID                  string   `json:"id,omitempty"`
	Protocol            string   `json:"protocol,omitempty"`
	IPPrefix            string   `json:"ip_prefix,omitempty"`
	IP6Prefix           string   `json:"ip6_prefix,omitempty"`
	Area                string   `json:"area,omitempty"`
	HelloInterval       float64  `json:"hello_interval,omitempty"`
	CSNPInterval        float64  `json:"csnp_interval,omitempty"`
	PersistentKeepalive int      `json:"persistent_keepalive,omitempty"`
	Redistribute        []string `json:"redistribute,omitempty"`
	RouteFilter         string   `json:"route_filter,omitempty"`
	Digest              string   `json:"digest,omitempty"`
}

type SDNFabricOptions struct {
	ID                  string   `json:"id,omitempty"`
	Protocol            string   `json:"protocol,omitempty"`
	IPPrefix            string   `json:"ip_prefix,omitempty"`
	IP6Prefix           string   `json:"ip6_prefix,omitempty"`
	Area                string   `json:"area,omitempty"`
	HelloInterval       float64  `json:"hello_interval,omitempty"`
	CSNPInterval        float64  `json:"csnp_interval,omitempty"`
	PersistentKeepalive int      `json:"persistent_keepalive,omitempty"`
	Redistribute        []string `json:"redistribute,omitempty"`
	RouteFilter         string   `json:"route_filter,omitempty"`
	LockToken           string   `json:"lock-token,omitempty"`
	Digest              string   `json:"digest,omitempty"`
	Delete              []string `json:"delete,omitempty"`
}

type SDNFabricNode struct {
	client *Client `json:"-"`

	FabricID   string   `json:"fabric_id,omitempty"`
	NodeID     string   `json:"node_id,omitempty"`
	IP         string   `json:"ip,omitempty"`
	IP6        string   `json:"ip6,omitempty"`
	Interfaces []string `json:"interfaces,omitempty"`
	AllowedIPs []string `json:"allowed_ips,omitempty"`
	Endpoint   string   `json:"endpoint,omitempty"`
	Peers      []string `json:"peers,omitempty"`
	Digest     string   `json:"digest,omitempty"`
}

type SDNFabricNodeOptions struct {
	FabricID   string   `json:"fabric_id,omitempty"`
	NodeID     string   `json:"node_id,omitempty"`
	IP         string   `json:"ip,omitempty"`
	IP6        string   `json:"ip6,omitempty"`
	Interfaces []string `json:"interfaces,omitempty"`
	AllowedIPs []string `json:"allowed_ips,omitempty"`
	Endpoint   string   `json:"endpoint,omitempty"`
	Peers      []string `json:"peers,omitempty"`
	LockToken  string   `json:"lock-token,omitempty"`
	Digest     string   `json:"digest,omitempty"`
	Delete     []string `json:"delete,omitempty"`
}

type SDNFabricsAll struct {
	Fabrics []*SDNFabric     `json:"fabrics,omitempty"`
	Nodes   []*SDNFabricNode `json:"nodes,omitempty"`
}

type SDNPrefixList struct {
	client *Client `json:"-"`

	ID      string                `json:"id,omitempty"`
	Entries []*SDNPrefixListEntry `json:"entries,omitempty"`
	Digest  string                `json:"digest,omitempty"`
}

type SDNPrefixListEntry struct {
	client *Client `json:"-"`

	ID     string `json:"-"`
	Seq    uint32 `json:"seq,omitempty"`
	Action string `json:"action,omitempty"`
	Prefix string `json:"prefix,omitempty"`
	GE     int    `json:"ge,omitempty"`
	LE     int    `json:"le,omitempty"`
	Digest string `json:"digest,omitempty"`
}

type SDNPrefixListOptions struct {
	ID        string                `json:"id,omitempty"`
	Entries   []*SDNPrefixListEntry `json:"entries,omitempty"`
	LockToken string                `json:"lock-token,omitempty"`
	Digest    string                `json:"digest,omitempty"`
	Delete    []string              `json:"delete,omitempty"`
}

type SDNPrefixListEntryOptions struct {
	Seq       uint32   `json:"seq,omitempty"`
	Action    string   `json:"action,omitempty"`
	Prefix    string   `json:"prefix,omitempty"`
	GE        int      `json:"ge,omitempty"`
	LE        int      `json:"le,omitempty"`
	LockToken string   `json:"lock-token,omitempty"`
	Digest    string   `json:"digest,omitempty"`
	Delete    []string `json:"delete,omitempty"`
}

type SDNRouteMapID struct {
	ID string `json:"id,omitempty"`
}

type SDNRouteMapEntry struct {
	client *Client `json:"-"`

	RouteMapID string   `json:"route-map-id,omitempty"`
	Order      uint16   `json:"order,omitempty"`
	Action     string   `json:"action,omitempty"`
	Match      []string `json:"match,omitempty"`
	Set        []string `json:"set,omitempty"`
	Call       string   `json:"call,omitempty"`
	ExitAction string   `json:"exit-action,omitempty"`
	Digest     string   `json:"digest,omitempty"`
}

type SDNRouteMapEntryOptions struct {
	RouteMapID string   `json:"route-map-id,omitempty"`
	Order      uint16   `json:"order,omitempty"`
	Action     string   `json:"action,omitempty"`
	Match      []string `json:"match,omitempty"`
	Set        []string `json:"set,omitempty"`
	Call       string   `json:"call,omitempty"`
	ExitAction string   `json:"exit-action,omitempty"`
	LockToken  string   `json:"lock-token,omitempty"`
	Digest     string   `json:"digest,omitempty"`
	Delete     []string `json:"delete,omitempty"`
}

type SDNLockToken string

type SDNDryRun struct {
	FRRDiff        string `json:"frr-diff,omitempty"`
	InterfacesDiff string `json:"interfaces-diff,omitempty"`
}

type SDNVNetFirewallOptions struct {
	Enable          IntOrBool `json:"enable,omitempty"`
	PolicyForward   string    `json:"policy_forward,omitempty"`
	LogLevelForward string    `json:"log_level_forward,omitempty"`
	Digest          string    `json:"digest,omitempty"`
}

type SDNVNetFirewallOptionsUpdate struct {
	Enable          *IntOrBool `json:"enable,omitempty"`
	PolicyForward   string     `json:"policy_forward,omitempty"`
	LogLevelForward string     `json:"log_level_forward,omitempty"`
	Digest          string     `json:"digest,omitempty"`
	Delete          string     `json:"delete,omitempty"`
}

type SDNVNetFirewallRule struct {
	Pos       int    `json:"pos,omitempty"`
	Type      string `json:"type,omitempty"`
	Action    string `json:"action,omitempty"`
	Enable    int    `json:"enable,omitempty"`
	Comment   string `json:"comment,omitempty"`
	Source    string `json:"source,omitempty"`
	Dest      string `json:"dest,omitempty"`
	Proto     string `json:"proto,omitempty"`
	SPort     string `json:"sport,omitempty"`
	DPort     string `json:"dport,omitempty"`
	IFace     string `json:"iface,omitempty"`
	Log       string `json:"log,omitempty"`
	Macro     string `json:"macro,omitempty"`
	IPVersion int    `json:"ipversion,omitempty"`
	ICMPType  string `json:"icmp-type,omitempty"`
}

type SDNVNetFirewallRuleOptions struct {
	Pos      int    `json:"pos,omitempty"`
	Type     string `json:"type,omitempty"`
	Action   string `json:"action,omitempty"`
	Enable   int    `json:"enable,omitempty"`
	Comment  string `json:"comment,omitempty"`
	Source   string `json:"source,omitempty"`
	Dest     string `json:"dest,omitempty"`
	Proto    string `json:"proto,omitempty"`
	SPort    string `json:"sport,omitempty"`
	DPort    string `json:"dport,omitempty"`
	IFace    string `json:"iface,omitempty"`
	Log      string `json:"log,omitempty"`
	Macro    string `json:"macro,omitempty"`
	ICMPType string `json:"icmp-type,omitempty"`
	MoveTo   int    `json:"moveto,omitempty"`
	Digest   string `json:"digest,omitempty"`
	Delete   string `json:"delete,omitempty"`
}

type SDNVNetIPOptions struct {
	Zone string `json:"zone,omitempty"`
	IP   string `json:"ip,omitempty"`
	MAC  string `json:"mac,omitempty"`
	VMID int    `json:"vmid,omitempty"`
}

type SDNSubnetOptions struct {
	Subnet        string    `json:"subnet,omitempty"`
	Type          string    `json:"type,omitempty"`
	VNet          string    `json:"vnet,omitempty"`
	Gateway       string    `json:"gateway,omitempty"`
	DHCPDNSServer string    `json:"dhcp-dns-server,omitempty"`
	DHCPRange     []string  `json:"dhcp-range,omitempty"`
	DNSZonePrefix string    `json:"dnszoneprefix,omitempty"`
	SNAT          IntOrBool `json:"snat,omitempty"`
	LockToken     string    `json:"lock-token,omitempty"`
	Digest        string    `json:"digest,omitempty"`
	Delete        string    `json:"delete,omitempty"`
}

type ClusterMetricServers []*ClusterMetricServerSummary

type ClusterMetricServerSummary struct {
	ID      string    `json:"id,omitempty"`
	Type    string    `json:"type,omitempty"`
	Server  string    `json:"server,omitempty"`
	Port    int       `json:"port,omitempty"`
	Disable IntOrBool `json:"disable,omitempty"`
}

type ClusterMetricServer struct {
	ID            string    `json:"id,omitempty"`
	Type          string    `json:"type,omitempty"`
	Server        string    `json:"server,omitempty"`
	Port          int       `json:"port,omitempty"`
	Disable       IntOrBool `json:"disable,omitempty"`
	APIPathPrefix string    `json:"api-path-prefix,omitempty"`
	Bucket        string    `json:"bucket,omitempty"`
	InfluxDBProto string    `json:"influxdbproto,omitempty"`
	MaxBodySize   uint64    `json:"max-body-size,omitempty"`
	MTU           uint      `json:"mtu,omitempty"`
	Organization  string    `json:"organization,omitempty"`
	Path          string    `json:"path,omitempty"`
	Proto         string    `json:"proto,omitempty"`
	Timeout       uint      `json:"timeout,omitempty"`
	Token         string    `json:"token,omitempty"`

	OtelCompression        string    `json:"otel-compression,omitempty"`
	OtelHeaders            string    `json:"otel-headers,omitempty"`
	OtelMaxBodySize        uint64    `json:"otel-max-body-size,omitempty"`
	OtelPath               string    `json:"otel-path,omitempty"`
	OtelProtocol           string    `json:"otel-protocol,omitempty"`
	OtelResourceAttributes string    `json:"otel-resource-attributes,omitempty"`
	OtelTimeout            uint      `json:"otel-timeout,omitempty"`
	OtelVerifySSL          IntOrBool `json:"otel-verify-ssl,omitempty"`
	VerifyCertificate      IntOrBool `json:"verify-certificate,omitempty"`
	Digest                 string    `json:"digest,omitempty"`
}

type ClusterMetricServerOptions struct {
	ID            string `json:"id,omitempty"`
	Type          string `json:"type,omitempty"`
	Server        string `json:"server,omitempty"`
	Port          int    `json:"port,omitempty"`
	Disable       *bool  `json:"disable,omitempty"`
	APIPathPrefix string `json:"api-path-prefix,omitempty"`
	Bucket        string `json:"bucket,omitempty"`
	InfluxDBProto string `json:"influxdbproto,omitempty"`
	MaxBodySize   uint64 `json:"max-body-size,omitempty"`
	MTU           uint   `json:"mtu,omitempty"`
	Organization  string `json:"organization,omitempty"`
	Path          string `json:"path,omitempty"`
	Proto         string `json:"proto,omitempty"`
	Timeout       uint   `json:"timeout,omitempty"`
	Token         string `json:"token,omitempty"`

	OtelCompression        string `json:"otel-compression,omitempty"`
	OtelHeaders            string `json:"otel-headers,omitempty"`
	OtelMaxBodySize        uint64 `json:"otel-max-body-size,omitempty"`
	OtelPath               string `json:"otel-path,omitempty"`
	OtelProtocol           string `json:"otel-protocol,omitempty"`
	OtelResourceAttributes string `json:"otel-resource-attributes,omitempty"`
	OtelTimeout            uint   `json:"otel-timeout,omitempty"`
	OtelVerifySSL          *bool  `json:"otel-verify-ssl,omitempty"`
	VerifyCertificate      *bool  `json:"verify-certificate,omitempty"`
	Digest                 string `json:"digest,omitempty"`
	Delete                 string `json:"delete,omitempty"`
}

type ClusterJobIndexEntry struct {
	SubDir string `json:"subdir,omitempty"`
}

type ClusterScheduleEvent struct {
	Timestamp int64  `json:"timestamp,omitempty"`
	UTC       string `json:"utc,omitempty"`
}

type ClusterRealmSyncJob struct {
	ID             string    `json:"id,omitempty"`
	Comment        string    `json:"comment,omitempty"`
	EnableNew      IntOrBool `json:"enable-new,omitempty"`
	Enabled        IntOrBool `json:"enabled,omitempty"`
	LastRun        int64     `json:"last-run,omitempty"`
	NextRun        int64     `json:"next-run,omitempty"`
	Realm          string    `json:"realm,omitempty"`
	RemoveVanished string    `json:"remove-vanished,omitempty"`
	Schedule       string    `json:"schedule,omitempty"`
	Scope          string    `json:"scope,omitempty"`
}

type ClusterRealmSyncJobOptions struct {
	Comment        string `json:"comment,omitempty"`
	EnableNew      *bool  `json:"enable-new,omitempty"`
	Enabled        *bool  `json:"enabled,omitempty"`
	Realm          string `json:"realm,omitempty"`
	RemoveVanished string `json:"remove-vanished,omitempty"`
	Schedule       string `json:"schedule,omitempty"`
	Scope          string `json:"scope,omitempty"`
	Delete         string `json:"delete,omitempty"`
}

type Disk struct {
	DevPath      string    `json:"devpath,omitempty"`
	Used         string    `json:"used,omitempty"`
	GPT          IntOrBool `json:"gpt,omitempty"`
	Size         uint64    `json:"size,omitempty"`
	Health       string    `json:"health,omitempty"`
	Model        string    `json:"model,omitempty"`
	Serial       string    `json:"serial,omitempty"`
	Type         string    `json:"type,omitempty"`
	Vendor       string    `json:"vendor,omitempty"`
	WWN          string    `json:"wwn,omitempty"`
	ByIDLink     string    `json:"by_id_link,omitempty"`
	Wearout      string    `json:"wearout,omitempty"`
	OSDID        int       `json:"osdid,omitempty"`
	OSDEncrypted IntOrBool `json:"osdencrypted,omitempty"`
	Parent       string    `json:"parent,omitempty"`
	RPM          int       `json:"rpm,omitempty"`
	BLKSize      int       `json:"blocksize,omitempty"`
	MountPoint   string    `json:"mounted,omitempty"`
	Vendor2      string    `json:"vendor2,omitempty"`
}

type DiskSMART struct {
	Health     string           `json:"health,omitempty"`
	Type       string           `json:"type,omitempty"`
	Text       string           `json:"text,omitempty"`
	Attributes []map[string]any `json:"attributes,omitempty"`
}

type NodeDirectory struct {
	Device  string `json:"device,omitempty"`
	Options string `json:"options,omitempty"`
	Path    string `json:"path,omitempty"`
	Type    string `json:"type,omitempty"`
	UUID    string `json:"unitfile,omitempty"`
}

type NodeDirectoryOptions struct {
	Name       string    `json:"name"`
	Device     string    `json:"device"`
	Filesystem string    `json:"filesystem,omitempty"`
	AddStorage IntOrBool `json:"add_storage,omitempty"`
}

type NodeLVMTree struct {
	Children []NodeLVMVolumeGroup `json:"children,omitempty"`
	Leaf     IntOrBool            `json:"leaf,omitempty"`
}

type NodeLVMVolumeGroup struct {
	Name     string            `json:"name,omitempty"`
	Size     uint64            `json:"size,omitempty"`
	Free     uint64            `json:"free,omitempty"`
	Leaf     IntOrBool         `json:"leaf,omitempty"`
	Children []NodeLVMPhysical `json:"children,omitempty"`
}

type NodeLVMPhysical struct {
	Name string    `json:"name,omitempty"`
	Size uint64    `json:"size,omitempty"`
	Free uint64    `json:"free,omitempty"`
	Leaf IntOrBool `json:"leaf,omitempty"`
}

type NodeLVMOptions struct {
	Name       string    `json:"name"`
	Device     string    `json:"device"`
	AddStorage IntOrBool `json:"add_storage,omitempty"`
}

type NodeLVMThin struct {
	LV           string `json:"lv,omitempty"`
	LVSize       uint64 `json:"lv_size,omitempty"`
	MetadataSize uint64 `json:"metadata_size,omitempty"`
	MetadataUsed uint64 `json:"metadata_used,omitempty"`
	Used         uint64 `json:"used,omitempty"`
}

type NodeLVMThinOptions struct {
	Name       string    `json:"name"`
	Device     string    `json:"device"`
	AddStorage IntOrBool `json:"add_storage,omitempty"`
}

type NodeZFSPoolSummary struct {
	Name   string  `json:"name,omitempty"`
	Health string  `json:"health,omitempty"`
	Size   uint64  `json:"size,omitempty"`
	Alloc  uint64  `json:"alloc,omitempty"`
	Free   uint64  `json:"free,omitempty"`
	Frag   int     `json:"frag,omitempty"`
	Dedup  float64 `json:"dedup,omitempty"`
}

type NodeZFSPool struct {
	Name     string        `json:"name,omitempty"`
	State    string        `json:"state,omitempty"`
	Status   string        `json:"status,omitempty"`
	Action   string        `json:"action,omitempty"`
	Scan     string        `json:"scan,omitempty"`
	Errors   string        `json:"errors,omitempty"`
	Children []NodeZFSVdev `json:"children,omitempty"`
}

type NodeZFSVdev struct {
	Name     string        `json:"name,omitempty"`
	State    string        `json:"state,omitempty"`
	Read     uint64        `json:"read,omitempty"`
	Write    uint64        `json:"write,omitempty"`
	Cksum    uint64        `json:"cksum,omitempty"`
	Msg      string        `json:"msg,omitempty"`
	Children []NodeZFSVdev `json:"children,omitempty"`
	Leaf     IntOrBool     `json:"leaf,omitempty"`
}

type NodeZFSPoolOptions struct {
	Name        string    `json:"name"`
	Devices     string    `json:"devices"`
	RaidLevel   string    `json:"raidlevel"`
	Ashift      int       `json:"ashift,omitempty"`
	Compression string    `json:"compression,omitempty"`
	DraidConfig string    `json:"draid-config,omitempty"`
	AddStorage  IntOrBool `json:"add_storage,omitempty"`
}

type CephOSD struct {
	client *Client
	Node   string `json:"-"`
	ID     int    `json:"-"`
}

type CephOSDTree struct {
	Flags string                 `json:"flags,omitempty"`
	Root  map[string]interface{} `json:"root,omitempty"`
}

type CephOSDDetails struct {
	OSD     CephOSDMetadata `json:"osd"`
	Devices []CephOSDDevice `json:"devices,omitempty"`
}

type CephOSDMetadata struct {
	BackAddr       string `json:"back_addr,omitempty"`
	Encrypted      bool   `json:"encrypted,omitempty"`
	FrontAddr      string `json:"front_addr,omitempty"`
	HBBackAddr     string `json:"hb_back_addr,omitempty"`
	HBFrontAddr    string `json:"hb_front_addr,omitempty"`
	Hostname       string `json:"hostname,omitempty"`
	ID             int    `json:"id"`
	MemUsage       int64  `json:"mem_usage,omitempty"`
	OSDData        string `json:"osd_data,omitempty"`
	OSDObjectStore string `json:"osd_objectstore,omitempty"`
	PID            int    `json:"pid,omitempty"`
	Version        string `json:"version,omitempty"`
}

type CephOSDDevice struct {
	DevNode        string `json:"dev_node,omitempty"`
	Device         string `json:"device,omitempty"`
	PhysicalDevice string `json:"physical_device,omitempty"`
	Size           uint64 `json:"size,omitempty"`
	SupportDiscard bool   `json:"support_discard,omitempty"`
	Type           string `json:"type,omitempty"`
}

type CephOSDLVInfo struct {
	CreationTime string `json:"creation_time,omitempty"`
	LVName       string `json:"lv_name,omitempty"`
	LVPath       string `json:"lv_path,omitempty"`
	LVSize       uint64 `json:"lv_size,omitempty"`
	LVUUID       string `json:"lv_uuid,omitempty"`
	VGName       string `json:"vg_name,omitempty"`
}

type CephOSDCreateOptions struct {
	Dev              string    `json:"dev"`
	CrushDeviceClass string    `json:"crush-device-class,omitempty"`
	DBDev            string    `json:"db_dev,omitempty"`
	DBDevSize        float64   `json:"db_dev_size,omitempty"`
	Encrypted        IntOrBool `json:"encrypted,omitempty"`
	OSDsPerDevice    int       `json:"osds-per-device,omitempty"`
	WALDev           string    `json:"wal_dev,omitempty"`
	WALDevSize       float64   `json:"wal_dev_size,omitempty"`
}

type ACMEDirectory struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

type ACMEMeta struct {
	CAAIdentities           []string  `json:"caaIdentities,omitempty"`
	ExternalAccountRequired IntOrBool `json:"externalAccountRequired,omitempty"`
	TermsOfService          string    `json:"termsOfService,omitempty"`
	Website                 string    `json:"website,omitempty"`
}

type ACMEChallengeSchema struct {
	ID     string                 `json:"id,omitempty"`
	Name   string                 `json:"name,omitempty"`
	Type   string                 `json:"type,omitempty"`
	Schema map[string]interface{} `json:"schema,omitempty"`
}

type ACMEAccountIndex struct {
	Name string `json:"name,omitempty"`
}

type ACMEAccount struct {
	Account   map[string]interface{} `json:"account,omitempty"`
	Directory string                 `json:"directory,omitempty"`
	Location  string                 `json:"location,omitempty"`
	TOS       string                 `json:"tos,omitempty"`
}

type ACMEAccountOptions struct {
	Contact    string `json:"contact"`
	Directory  string `json:"directory,omitempty"`
	EABKid     string `json:"eab-kid,omitempty"`
	EABHMACKey string `json:"eab-hmac-key,omitempty"`
	Name       string `json:"name,omitempty"`
	TOSURL     string `json:"tos_url,omitempty"`
}

type ACMEPlugin struct {
	ID              string    `json:"plugin,omitempty"`
	Type            string    `json:"type,omitempty"`
	API             string    `json:"api,omitempty"`
	Data            string    `json:"data,omitempty"`
	Disable         IntOrBool `json:"disable,omitempty"`
	Nodes           string    `json:"nodes,omitempty"`
	ValidationDelay int       `json:"validation-delay,omitempty"`
	Digest          string    `json:"digest,omitempty"`
}

type ACMEPluginOptions struct {
	ID              string `json:"id,omitempty"`
	Type            string `json:"type,omitempty"`
	API             string `json:"api,omitempty"`
	Data            string `json:"data,omitempty"`
	Disable         *bool  `json:"disable,omitempty"`
	Nodes           string `json:"nodes,omitempty"`
	ValidationDelay *int   `json:"validation-delay,omitempty"`
	Digest          string `json:"digest,omitempty"`
	Delete          string `json:"delete,omitempty"`
}

type ClusterMappings []*ClusterMappingIndexEntry

type ClusterMappingIndexEntry struct {
	Name string `json:"name,omitempty"`
}

type ClusterMappingCheck struct {
	Message  string `json:"message,omitempty"`
	Severity string `json:"severity,omitempty"`
}

type ClusterDirMappings []*ClusterDirMapping

type ClusterDirMapping struct {
	ID          string                 `json:"id,omitempty"`
	Description string                 `json:"description,omitempty"`
	Map         []string               `json:"map,omitempty"`
	Checks      []*ClusterMappingCheck `json:"checks,omitempty"`
	Digest      string                 `json:"digest,omitempty"`
}

type ClusterDirMappingOptions struct {
	ID          string   `json:"id,omitempty"`
	Description string   `json:"description,omitempty"`
	Map         []string `json:"map,omitempty"`
	Digest      string   `json:"digest,omitempty"`
	Delete      string   `json:"delete,omitempty"`
}

type ClusterPCIMappings []*ClusterPCIMapping

type ClusterPCIMapping struct {
	ID                   string                 `json:"id,omitempty"`
	Description          string                 `json:"description,omitempty"`
	Map                  []string               `json:"map,omitempty"`
	Checks               []*ClusterMappingCheck `json:"checks,omitempty"`
	MDev                 IntOrBool              `json:"mdev,omitempty"`
	LiveMigrationCapable IntOrBool              `json:"live-migration-capable,omitempty"`
	Digest               string                 `json:"digest,omitempty"`
}

type ClusterPCIMappingOptions struct {
	ID                   string   `json:"id,omitempty"`
	Description          string   `json:"description,omitempty"`
	Map                  []string `json:"map,omitempty"`
	MDev                 bool     `json:"mdev,omitempty"`
	LiveMigrationCapable bool     `json:"live-migration-capable,omitempty"`
	Digest               string   `json:"digest,omitempty"`
	Delete               string   `json:"delete,omitempty"`
}

type ClusterUSBMappings []*ClusterUSBMapping

type ClusterUSBMapping struct {
	ID          string                 `json:"id,omitempty"`
	Description string                 `json:"description,omitempty"`
	Map         []string               `json:"map,omitempty"`
	Error       []*ClusterMappingCheck `json:"error,omitempty"`
	Digest      string                 `json:"digest,omitempty"`
}

type ClusterUSBMappingOptions struct {
	ID          string   `json:"id,omitempty"`
	Description string   `json:"description,omitempty"`
	Map         []string `json:"map,omitempty"`
	Digest      string   `json:"digest,omitempty"`
	Delete      string   `json:"delete,omitempty"`
}

type ClusterNotificationIndex []*ClusterNotificationIndexEntry

type ClusterNotificationIndexEntry struct {
	Name string `json:"name,omitempty"`
}

type ClusterNotificationMatcherField struct {
	Name string `json:"name,omitempty"`
}

type ClusterNotificationMatcherFieldValue struct {
	Field   string `json:"field,omitempty"`
	Value   string `json:"value,omitempty"`
	Comment string `json:"comment,omitempty"`
}

type ClusterNotificationTarget struct {
	Name    string    `json:"name,omitempty"`
	Type    string    `json:"type,omitempty"`
	Comment string    `json:"comment,omitempty"`
	Origin  string    `json:"origin,omitempty"`
	Disable IntOrBool `json:"disable,omitempty"`
}

type ClusterNotificationMatcher struct {
	Name          string    `json:"name,omitempty"`
	Comment       string    `json:"comment,omitempty"`
	Mode          string    `json:"mode,omitempty"`
	Disable       IntOrBool `json:"disable,omitempty"`
	InvertMatch   IntOrBool `json:"invert-match,omitempty"`
	MatchCalendar []string  `json:"match-calendar,omitempty"`
	MatchField    []string  `json:"match-field,omitempty"`
	MatchSeverity []string  `json:"match-severity,omitempty"`
	Target        []string  `json:"target,omitempty"`
	Origin        string    `json:"origin,omitempty"`
	Digest        string    `json:"digest,omitempty"`
}

type ClusterNotificationMatcherOptions struct {
	Name          string   `json:"name,omitempty"`
	Comment       string   `json:"comment,omitempty"`
	Mode          string   `json:"mode,omitempty"`
	Disable       *bool    `json:"disable,omitempty"`
	InvertMatch   *bool    `json:"invert-match,omitempty"`
	MatchCalendar []string `json:"match-calendar,omitempty"`
	MatchField    []string `json:"match-field,omitempty"`
	MatchSeverity []string `json:"match-severity,omitempty"`
	Target        []string `json:"target,omitempty"`
	Digest        string   `json:"digest,omitempty"`
	Delete        []string `json:"delete,omitempty"`
}

type ClusterNotificationGotifyEndpoint struct {
	Name    string    `json:"name,omitempty"`
	Server  string    `json:"server,omitempty"`
	Comment string    `json:"comment,omitempty"`
	Disable IntOrBool `json:"disable,omitempty"`
	Origin  string    `json:"origin,omitempty"`
	Digest  string    `json:"digest,omitempty"`
}

type ClusterNotificationGotifyOptions struct {
	Name    string   `json:"name,omitempty"`
	Server  string   `json:"server,omitempty"`
	Token   string   `json:"token,omitempty"`
	Comment string   `json:"comment,omitempty"`
	Disable *bool    `json:"disable,omitempty"`
	Digest  string   `json:"digest,omitempty"`
	Delete  []string `json:"delete,omitempty"`
}

type ClusterNotificationSendmailEndpoint struct {
	Name        string    `json:"name,omitempty"`
	Author      string    `json:"author,omitempty"`
	FromAddress string    `json:"from-address,omitempty"`
	MailTo      []string  `json:"mailto,omitempty"`
	MailToUser  []string  `json:"mailto-user,omitempty"`
	Comment     string    `json:"comment,omitempty"`
	Disable     IntOrBool `json:"disable,omitempty"`
	Origin      string    `json:"origin,omitempty"`
	Digest      string    `json:"digest,omitempty"`
}

type ClusterNotificationSendmailOptions struct {
	Name        string   `json:"name,omitempty"`
	Author      string   `json:"author,omitempty"`
	FromAddress string   `json:"from-address,omitempty"`
	MailTo      []string `json:"mailto,omitempty"`
	MailToUser  []string `json:"mailto-user,omitempty"`
	Comment     string   `json:"comment,omitempty"`
	Disable     *bool    `json:"disable,omitempty"`
	Digest      string   `json:"digest,omitempty"`
	Delete      []string `json:"delete,omitempty"`
}

type ClusterNotificationSMTPEndpoint struct {
	Name        string    `json:"name,omitempty"`
	Server      string    `json:"server,omitempty"`
	Port        int       `json:"port,omitempty"`
	Mode        string    `json:"mode,omitempty"`
	Username    string    `json:"username,omitempty"`
	FromAddress string    `json:"from-address,omitempty"`
	Author      string    `json:"author,omitempty"`
	MailTo      []string  `json:"mailto,omitempty"`
	MailToUser  []string  `json:"mailto-user,omitempty"`
	Comment     string    `json:"comment,omitempty"`
	Disable     IntOrBool `json:"disable,omitempty"`
	Origin      string    `json:"origin,omitempty"`
	Digest      string    `json:"digest,omitempty"`
}

type ClusterNotificationSMTPOptions struct {
	Name        string   `json:"name,omitempty"`
	Server      string   `json:"server,omitempty"`
	Port        int      `json:"port,omitempty"`
	Mode        string   `json:"mode,omitempty"`
	Username    string   `json:"username,omitempty"`
	Password    string   `json:"password,omitempty"`
	FromAddress string   `json:"from-address,omitempty"`
	Author      string   `json:"author,omitempty"`
	MailTo      []string `json:"mailto,omitempty"`
	MailToUser  []string `json:"mailto-user,omitempty"`
	Comment     string   `json:"comment,omitempty"`
	Disable     *bool    `json:"disable,omitempty"`
	Digest      string   `json:"digest,omitempty"`
	Delete      []string `json:"delete,omitempty"`
}

type ClusterNotificationWebhookEndpoint struct {
	Name    string    `json:"name,omitempty"`
	URL     string    `json:"url,omitempty"`
	Method  string    `json:"method,omitempty"`
	Header  []string  `json:"header,omitempty"`
	Body    string    `json:"body,omitempty"`
	Secret  []string  `json:"secret,omitempty"`
	Comment string    `json:"comment,omitempty"`
	Disable IntOrBool `json:"disable,omitempty"`
	Origin  string    `json:"origin,omitempty"`
	Digest  string    `json:"digest,omitempty"`
}

type ClusterNotificationWebhookOptions struct {
	Name    string   `json:"name,omitempty"`
	URL     string   `json:"url,omitempty"`
	Method  string   `json:"method,omitempty"`
	Header  []string `json:"header,omitempty"`
	Body    string   `json:"body,omitempty"`
	Secret  []string `json:"secret,omitempty"`
	Comment string   `json:"comment,omitempty"`
	Disable *bool    `json:"disable,omitempty"`
	Digest  string   `json:"digest,omitempty"`
	Delete  []string `json:"delete,omitempty"`
}

type VirtualMachineDirIndexEntry struct {
	Subdir string `json:"subdir,omitempty"`
}

type VirtualMachineStatusIndexEntry struct {
	Subdir string `json:"subdir,omitempty"`
}

type VirtualMachineSnapshotIndexEntry struct {
	Subdir string `json:"subdir,omitempty"`
}

type ContainerSnapshotIndexEntry struct {
	Subdir string `json:"subdir,omitempty"`
}

type VirtualMachineMigrationTunnel struct {
	Socket string `json:"socket,omitempty"`
	Ticket string `json:"ticket,omitempty"`
	UPID   string `json:"upid,omitempty"`
}

type VirtualMachineMigrationTunnelOptions struct {
	Bridges string `json:"bridges,omitempty"`

	Storages string `json:"storages,omitempty"`
}

type CephMon struct {
	client           *Client
	Node             string    `json:"-"`
	Addr             string    `json:"addr,omitempty"`
	CephVersion      string    `json:"ceph_version,omitempty"`
	CephVersionShort string    `json:"ceph_version_short,omitempty"`
	DirExists        IntOrBool `json:"direxists,omitempty"`
	Host             string    `json:"host,omitempty"`
	Name             string    `json:"name,omitempty"`
	Quorum           IntOrBool `json:"quorum,omitempty"`
	Rank             int       `json:"rank,omitempty"`
	Service          IntOrBool `json:"service,omitempty"`
	State            string    `json:"state,omitempty"`
}

type CephMonOptions struct {
	MonAddress string `json:"mon-address,omitempty"`
}

type CephMgr struct {
	client           *Client
	Node             string    `json:"-"`
	Addr             string    `json:"addr,omitempty"`
	CephVersion      string    `json:"ceph_version,omitempty"`
	CephVersionShort string    `json:"ceph_version_short,omitempty"`
	DirExists        IntOrBool `json:"direxists,omitempty"`
	Host             string    `json:"host,omitempty"`
	Name             string    `json:"name,omitempty"`
	Service          IntOrBool `json:"service,omitempty"`
	State            string    `json:"state,omitempty"`
}

type CephMDS struct {
	client           *Client
	Node             string    `json:"-"`
	Addr             string    `json:"addr,omitempty"`
	CephVersion      string    `json:"ceph_version,omitempty"`
	CephVersionShort string    `json:"ceph_version_short,omitempty"`
	DirExists        IntOrBool `json:"direxists,omitempty"`
	FSName           string    `json:"fs_name,omitempty"`
	Host             string    `json:"host,omitempty"`
	Name             string    `json:"name,omitempty"`
	Rank             int       `json:"rank,omitempty"`
	Service          IntOrBool `json:"service,omitempty"`
	StandbyReplay    IntOrBool `json:"standby_replay,omitempty"`
	State            string    `json:"state,omitempty"`
}

type CephMDSOptions struct {
	HotStandby IntOrBool `json:"hotstandby,omitempty"`
}

type SDNZoneStatus struct {
	Zone   string `json:"zone"`
	Status string `json:"status,omitempty"`
}

type SDNZoneContent struct {
	VNet      string `json:"vnet"`
	Status    string `json:"status,omitempty"`
	StatusMsg string `json:"statusmsg,omitempty"`
}

type SDNZoneBridge struct {
	Name          string           `json:"name"`
	Ports         []*SDNBridgePort `json:"ports,omitempty"`
	VLANFiltering string           `json:"vlan_filtering,omitempty"`
}

type SDNBridgePort struct {
	Name        string   `json:"name"`
	Index       string   `json:"index,omitempty"`
	PrimaryVLAN float64  `json:"primary_vlan,omitempty"`
	VLANs       []string `json:"vlans,omitempty"`
	VMID        float64  `json:"vmid,omitempty"`
}

type SDNIPVRFEntry struct {
	IP       string   `json:"ip"`
	Metric   int      `json:"metric,omitempty"`
	Nexthops []string `json:"nexthops,omitempty"`
	Protocol string   `json:"protocol,omitempty"`
}

type SDNMACVRFEntry struct {
	IP      string `json:"ip,omitempty"`
	MAC     string `json:"mac,omitempty"`
	NextHop string `json:"nexthop,omitempty"`
}

type SDNFabricInterface struct {
	Name  string `json:"name"`
	State string `json:"state,omitempty"`
	Type  string `json:"type,omitempty"`
}

type SDNFabricNeighbor struct {
	Neighbor string `json:"neighbor"`
	Status   string `json:"status,omitempty"`
	Uptime   string `json:"uptime,omitempty"`
}

type SDNFabricRoute struct {
	Route string   `json:"route"`
	Via   []string `json:"via,omitempty"`
}

type ScanZFSPool struct {
	Pool string `json:"pool"`
}

type ScanLVMVG struct {
	VG string `json:"vg"`
}

type ScanLVMThinPool struct {
	LV string `json:"lv"`
}

type ScanNFSExport struct {
	Path    string `json:"path"`
	Options string `json:"options,omitempty"`
}

type ScanCIFSShare struct {
	Share       string `json:"share"`
	Description string `json:"description,omitempty"`
}

type ScanPBSStore struct {
	Store   string `json:"store"`
	Comment string `json:"comment,omitempty"`
}

type ScanISCSITarget struct {
	Target string `json:"target"`
	Portal string `json:"portal,omitempty"`
}

type QEMUCPUModel struct {
	Name     string `json:"name"`
	Vendor   string `json:"vendor"`
	Custom   bool   `json:"custom"`
	Abstract bool   `json:"abstract,omitempty"`
}

type QEMUCPUFlag struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	SupportedOn []string `json:"supported-on,omitempty"`
}

type QEMUMachineType struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Version string `json:"version"`
	Changes string `json:"changes,omitempty"`
}

type QEMUMigrationCapabilities struct {
	HasDbusVMState bool `json:"has-dbus-vmstate"`
}

type PCIDevice struct {
	client *Client
	Node   string `json:"-"`

	ID                  string `json:"id"`
	Class               string `json:"class,omitempty"`
	Vendor              string `json:"vendor,omitempty"`
	VendorName          string `json:"vendor_name,omitempty"`
	Device              string `json:"device,omitempty"`
	DeviceName          string `json:"device_name,omitempty"`
	SubsystemVendor     string `json:"subsystem_vendor,omitempty"`
	SubsystemVendorName string `json:"subsystem_vendor_name,omitempty"`
	SubsystemDevice     string `json:"subsystem_device,omitempty"`
	SubsystemDeviceName string `json:"subsystem_device_name,omitempty"`
	IOMMUGroup          int    `json:"iommugroup,omitempty"`
	MdevCapable         bool   `json:"mdev,omitempty"`
}

type PCIMdevType struct {
	Type        string `json:"type"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Available   int    `json:"available"`
}

type USBDevice struct {
	BusNum       int    `json:"busnum"`
	DevNum       int    `json:"devnum"`
	Port         int    `json:"port"`
	Level        int    `json:"level"`
	Class        int    `json:"class"`
	VendID       string `json:"vendid"`
	ProdID       string `json:"prodid"`
	Speed        string `json:"speed"`
	Manufacturer string `json:"manufacturer,omitempty"`
	Product      string `json:"product,omitempty"`
	Serial       string `json:"serial,omitempty"`
	USBPath      string `json:"usbpath,omitempty"`
}

type NodeConfig struct {
	Acme                string `json:"acme,omitempty"`
	AcmeDomain0         string `json:"acmedomain0,omitempty"`
	AcmeDomain1         string `json:"acmedomain1,omitempty"`
	AcmeDomain2         string `json:"acmedomain2,omitempty"`
	AcmeDomain3         string `json:"acmedomain3,omitempty"`
	AcmeDomain4         string `json:"acmedomain4,omitempty"`
	AcmeDomain5         string `json:"acmedomain5,omitempty"`
	BallooningTarget    int    `json:"ballooning-target,omitempty"`
	Description         string `json:"description,omitempty"`
	Digest              string `json:"digest,omitempty"`
	Location            string `json:"location,omitempty"`
	StartAllOnBootDelay int    `json:"startall-onboot-delay,omitempty"`
	WakeOnLAN           string `json:"wakeonlan,omitempty"`
}

type NodeConfigOptions struct {
	Acme        string `json:"acme,omitempty"`
	AcmeDomain0 string `json:"acmedomain0,omitempty"`
	AcmeDomain1 string `json:"acmedomain1,omitempty"`
	AcmeDomain2 string `json:"acmedomain2,omitempty"`
	AcmeDomain3 string `json:"acmedomain3,omitempty"`
	AcmeDomain4 string `json:"acmedomain4,omitempty"`
	AcmeDomain5 string `json:"acmedomain5,omitempty"`

	BallooningTarget *int   `json:"ballooning-target,omitempty"`
	Delete           string `json:"delete,omitempty"`
	Description      string `json:"description,omitempty"`
	Digest           string `json:"digest,omitempty"`
	Location         string `json:"location,omitempty"`

	StartAllOnBootDelay *int   `json:"startall-onboot-delay,omitempty"`
	WakeOnLAN           string `json:"wakeonlan,omitempty"`
}

type NodeHosts struct {
	Data   string `json:"data"`
	Digest string `json:"digest,omitempty"`
}

type NodeRRDImage struct {
	Filename string `json:"filename"`
}

type NodeURLMetadata struct {
	Filename string `json:"filename,omitempty"`
	MimeType string `json:"mimetype,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

type StorageIdentity struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type ClusterJoinInfo struct {
	ConfigDigest  string                 `json:"config_digest,omitempty"`
	PreferredNode string                 `json:"preferred_node,omitempty"`
	NodeList      []*ClusterJoinNodeInfo `json:"nodelist,omitempty"`

	Totem map[string]any `json:"totem,omitempty"`
}

type ClusterJoinNodeInfo struct {
	Name        string `json:"name,omitempty"`
	NodeID      int    `json:"nodeid,omitempty"`
	PVEAddr     string `json:"pve_addr,omitempty"`
	PVEFP       string `json:"pve_fp,omitempty"`
	QuorumVotes int    `json:"quorum_votes,omitempty"`
	Ring0Addr   string `json:"ring0_addr,omitempty"`
}

type ClusterJoinOptions struct {
	Hostname    string `json:"hostname,omitempty"`
	Password    string `json:"password,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`

	Force  *bool  `json:"force,omitempty"`
	NodeID int    `json:"nodeid,omitempty"`
	Votes  int    `json:"votes,omitempty"`
	Link0  string `json:"link0,omitempty"`
	Link1  string `json:"link1,omitempty"`
	Link2  string `json:"link2,omitempty"`
	Link3  string `json:"link3,omitempty"`
	Link4  string `json:"link4,omitempty"`
	Link5  string `json:"link5,omitempty"`
	Link6  string `json:"link6,omitempty"`
	Link7  string `json:"link7,omitempty"`
}

type ClusterConfigNodeEntry struct {
	Node string `json:"node,omitempty"`
}

type ClusterCreateOptions struct {
	ClusterName      string `json:"clustername,omitempty"`
	NodeID           int    `json:"nodeid,omitempty"`
	Votes            int    `json:"votes,omitempty"`
	TokenCoefficient int    `json:"token-coefficient,omitempty"`
	Link0            string `json:"link0,omitempty"`
	Link1            string `json:"link1,omitempty"`
	Link2            string `json:"link2,omitempty"`
	Link3            string `json:"link3,omitempty"`
	Link4            string `json:"link4,omitempty"`
	Link5            string `json:"link5,omitempty"`
	Link6            string `json:"link6,omitempty"`
	Link7            string `json:"link7,omitempty"`
}

type ClusterAddNodeOptions struct {
	APIVersion int    `json:"apiversion,omitempty"`
	NewNodeIP  string `json:"new_node_ip,omitempty"`
	NodeID     int    `json:"nodeid,omitempty"`
	Votes      int    `json:"votes,omitempty"`

	Force *bool  `json:"force,omitempty"`
	Link0 string `json:"link0,omitempty"`
	Link1 string `json:"link1,omitempty"`
	Link2 string `json:"link2,omitempty"`
	Link3 string `json:"link3,omitempty"`
	Link4 string `json:"link4,omitempty"`
	Link5 string `json:"link5,omitempty"`
	Link6 string `json:"link6,omitempty"`
	Link7 string `json:"link7,omitempty"`
}

type ClusterAddNodeResult struct {
	CorosyncAuthkey string   `json:"corosync_authkey,omitempty"`
	CorosyncConf    string   `json:"corosync_conf,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
}

type CustomCPUModel struct {
	client *Client `json:"-"`

	CPUType       string `json:"cputype,omitempty"`
	Flags         string `json:"flags,omitempty"`
	GuestPhysBits int    `json:"guest-phys-bits,omitempty"`
	Hidden        int    `json:"hidden,omitempty"`
	HVVendorID    string `json:"hv-vendor-id,omitempty"`
	Level         int    `json:"level,omitempty"`
	PhysBits      string `json:"phys-bits,omitempty"`
	ReportedModel string `json:"reported-model,omitempty"`
	Digest        string `json:"digest,omitempty"`
}

type ClusterLogEntry struct {
	Node string `json:"node,omitempty"`
	Time int64  `json:"time,omitempty"`
	UID  int    `json:"uid,omitempty"`
	User string `json:"user,omitempty"`
	Pri  int    `json:"pri,omitempty"`
	Tag  string `json:"tag,omitempty"`
	Pid  int    `json:"pid,omitempty"`
	Msg  string `json:"msg,omitempty"`
	UPID string `json:"upid,omitempty"`
}

type ClusterOptionsResponse struct {
	BWLimit     string `json:"bwlimit,omitempty"`
	ConsentText string `json:"consent-text,omitempty"`
	Console     string `json:"console,omitempty"`
	Description string `json:"description,omitempty"`
	EmailFrom   string `json:"email_from,omitempty"`
	Fencing     string `json:"fencing,omitempty"`
	HTTPProxy   string `json:"http_proxy,omitempty"`
	Keyboard    string `json:"keyboard,omitempty"`
	Language    string `json:"language,omitempty"`
	MACPrefix   string `json:"mac_prefix,omitempty"`
	MaxWorkers  int    `json:"max_workers,omitempty"`
	Migration   string `json:"migration,omitempty"`

	MigrationUnsecure *bool  `json:"migration_unsecure,omitempty"`
	RegisteredTags    string `json:"registered-tags,omitempty"`

	Extra map[string]any `json:"-"`
}

type ClusterOptionsUpdate struct {
	BWLimit           string `json:"bwlimit,omitempty"`
	ConsentText       string `json:"consent-text,omitempty"`
	Console           string `json:"console,omitempty"`
	Description       string `json:"description,omitempty"`
	EmailFrom         string `json:"email_from,omitempty"`
	Fencing           string `json:"fencing,omitempty"`
	HTTPProxy         string `json:"http_proxy,omitempty"`
	Keyboard          string `json:"keyboard,omitempty"`
	Language          string `json:"language,omitempty"`
	MACPrefix         string `json:"mac_prefix,omitempty"`
	MaxWorkers        int    `json:"max_workers,omitempty"`
	Migration         string `json:"migration,omitempty"`
	MigrationUnsecure *bool  `json:"migration_unsecure,omitempty"`
	RegisteredTags    string `json:"registered-tags,omitempty"`

	Delete string `json:"delete,omitempty"`

	Extra map[string]any `json:"-"`
}

type MetricsExportResponse struct {
	Data []*MetricsExportEntry `json:"data,omitempty"`
}

type MetricsExportEntry struct {
	ID        string  `json:"id"`
	Metric    string  `json:"metric"`
	Timestamp int64   `json:"timestamp"`
	Type      string  `json:"type"`
	Value     float64 `json:"value"`
}

type BackupGuestEntry struct {
	VMID int    `json:"vmid"`
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
}

type BackupIncludedVolumesRoot struct {
	Children []*BackupIncludedVolumesGuest `json:"children,omitempty"`
}

type BackupIncludedVolumesGuest struct {
	ID       int                            `json:"id"`
	Name     string                         `json:"name,omitempty"`
	Type     string                         `json:"type,omitempty"`
	Children []*BackupIncludedVolumesVolume `json:"children,omitempty"`
}

type BackupIncludedVolumesVolume struct {
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	Included bool   `json:"included"`
	Reason   string `json:"reason,omitempty"`
}

type CephFlag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Value       bool   `json:"value"`
}

type CephFlagsUpdateOptions struct {
	NoBackfill  *bool `json:"nobackfill,omitempty"`
	NoDeepScrub *bool `json:"nodeep-scrub,omitempty"`
	NoDown      *bool `json:"nodown,omitempty"`
	NoIn        *bool `json:"noin,omitempty"`
	NoOut       *bool `json:"noout,omitempty"`
	NoRebalance *bool `json:"norebalance,omitempty"`
	NoRecover   *bool `json:"norecover,omitempty"`
	NoScrub     *bool `json:"noscrub,omitempty"`
	NoTierAgent *bool `json:"notieragent,omitempty"`
	NoUp        *bool `json:"noup,omitempty"`
	Pause       *bool `json:"pause,omitempty"`
}

type CephMetadata struct {
	Version *CephMetadataVersion      `json:"version,omitempty"`
	OSD     []map[string]any          `json:"osd,omitempty"`
	MON     []map[string]any          `json:"mon,omitempty"`
	MGR     []map[string]any          `json:"mgr,omitempty"`
	MDS     []map[string]any          `json:"mds,omitempty"`
	Node    map[string]map[string]any `json:"node,omitempty"`
}

type CephMetadataVersion struct {
	Version     string `json:"version,omitempty"`
	Buildcommit string `json:"buildcommit,omitempty"`
}

type BulkStartOptions struct {
	VMIDs      []int `json:"vms,omitempty"`
	MaxWorkers int   `json:"max-workers,omitempty"`
	Timeout    int   `json:"timeout,omitempty"`
}

type BulkShutdownOptions struct {
	VMIDs      []int `json:"vms,omitempty"`
	MaxWorkers int   `json:"max-workers,omitempty"`
	Timeout    int   `json:"timeout,omitempty"`

	ForceStop *bool `json:"force-stop,omitempty"`
}

type BulkSuspendOptions struct {
	VMIDs      []int `json:"vms,omitempty"`
	MaxWorkers int   `json:"max-workers,omitempty"`

	ToDisk       bool   `json:"to-disk,omitempty"`
	StateStorage string `json:"statestorage,omitempty"`
}

type BulkMigrateOptions struct {
	Target         string `json:"target,omitempty"`
	VMIDs          []int  `json:"vms,omitempty"`
	MaxWorkers     int    `json:"max-workers,omitempty"`
	Online         *bool  `json:"online,omitempty"`
	WithLocalDisks *bool  `json:"with-local-disks,omitempty"`
}

type CustomCPUModelOptions struct {
	CPUType       string `json:"cputype,omitempty"`
	Flags         string `json:"flags,omitempty"`
	GuestPhysBits int    `json:"guest-phys-bits,omitempty"`

	Hidden     int    `json:"hidden,omitempty"`
	HVVendorID string `json:"hv-vendor-id,omitempty"`
	Level      int    `json:"level,omitempty"`
	PhysBits   string `json:"phys-bits,omitempty"`

	ReportedModel string `json:"reported-model,omitempty"`
	Digest        string `json:"digest,omitempty"`
	Delete        string `json:"delete,omitempty"`
}
