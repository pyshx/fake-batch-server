package api

import "time"

type JobState string

const (
	JobStateUnspecified JobState = "STATE_UNSPECIFIED"
	JobStateQueued      JobState = "QUEUED"
	JobStateScheduled   JobState = "SCHEDULED"
	JobStateRunning     JobState = "RUNNING"
	JobStateSucceeded   JobState = "SUCCEEDED"
	JobStateFailed      JobState = "FAILED"
	JobStateDeleting    JobState = "DELETING"
	JobStateDeleted     JobState = "DELETED"
)

type TaskState string

const (
	TaskStateUnspecified TaskState = "STATE_UNSPECIFIED"
	TaskStatePending     TaskState = "PENDING"
	TaskStateAssigned    TaskState = "ASSIGNED"
	TaskStateRunning     TaskState = "RUNNING"
	TaskStateSucceeded   TaskState = "SUCCEEDED"
	TaskStateFailed      TaskState = "FAILED"
	TaskStateAborted     TaskState = "ABORTED"
)

type Job struct {
	Name            string              `json:"name"`
	UID             string              `json:"uid"`
	Priority        int32               `json:"priority,omitempty"`
	State           JobState            `json:"state"`
	CreateTime      time.Time           `json:"createTime"`
	UpdateTime      time.Time           `json:"updateTime"`
	Labels          map[string]string   `json:"labels,omitempty"`
	TaskGroups      []*TaskGroup        `json:"taskGroups"`
	AllocationPolicy *AllocationPolicy   `json:"allocationPolicy,omitempty"`
	LogsPolicy      *LogsPolicy         `json:"logsPolicy,omitempty"`
	Status          *JobStatus          `json:"status,omitempty"`
}

type TaskGroup struct {
	Name             string            `json:"name"`
	TaskSpec         *TaskSpec         `json:"taskSpec"`
	TaskCount        int64             `json:"taskCount,omitempty"`
	TaskCountPerNode int64             `json:"taskCountPerNode,omitempty"`
	Parallelism      int64             `json:"parallelism,omitempty"`
	SchedulingPolicy string            `json:"schedulingPolicy,omitempty"`
	TaskEnvironments []*Environment    `json:"taskEnvironments,omitempty"`
}

type TaskSpec struct {
	ComputeResource  *ComputeResource  `json:"computeResource,omitempty"`
	Runnables        []*Runnable       `json:"runnables"`
	MaxRunDuration   string            `json:"maxRunDuration,omitempty"`
	MaxRetryCount    int32             `json:"maxRetryCount,omitempty"`
	Volumes          []*Volume         `json:"volumes,omitempty"`
	Environment      *Environment      `json:"environment,omitempty"`
}

type Runnable struct {
	Container      *Container     `json:"container,omitempty"`
	Script         *Script        `json:"script,omitempty"`
	Barrier        *Barrier       `json:"barrier,omitempty"`
	DisplayName    string         `json:"displayName,omitempty"`
	IgnoreExitStatus bool         `json:"ignoreExitStatus,omitempty"`
	Background     bool           `json:"background,omitempty"`
	AlwaysRun      bool           `json:"alwaysRun,omitempty"`
	Environment    *Environment   `json:"environment,omitempty"`
	Timeout        string         `json:"timeout,omitempty"`
}

type Container struct {
	ImageURI         string   `json:"imageUri"`
	Commands         []string `json:"commands,omitempty"`
	Entrypoint       string   `json:"entrypoint,omitempty"`
	Volumes          []string `json:"volumes,omitempty"`
	Options          string   `json:"options,omitempty"`
	BlockExternalNetwork bool `json:"blockExternalNetwork,omitempty"`
}

type Script struct {
	Path string `json:"path,omitempty"`
	Text string `json:"text,omitempty"`
}

type Barrier struct {
	Name string `json:"name"`
}

type ComputeResource struct {
	CPUMilli    int64  `json:"cpuMilli,omitempty"`
	MemoryMib   int64  `json:"memoryMib,omitempty"`
	GPUCount    int64  `json:"gpuCount,omitempty"`
	BootDiskMib int64  `json:"bootDiskMib,omitempty"`
}

type Volume struct {
	NFS       *NFS      `json:"nfs,omitempty"`
	GCS       *GCS      `json:"gcs,omitempty"`
	DeviceName string   `json:"deviceName,omitempty"`
	MountPath  string   `json:"mountPath"`
	MountOptions []string `json:"mountOptions,omitempty"`
}

type NFS struct {
	Server     string `json:"server"`
	RemotePath string `json:"remotePath"`
}

type GCS struct {
	RemotePath string `json:"remotePath"`
}

type Environment struct {
	Variables       map[string]string `json:"variables,omitempty"`
	SecretVariables map[string]string `json:"secretVariables,omitempty"`
}

type AllocationPolicy struct {
	Location    *LocationPolicy    `json:"location,omitempty"`
	Instances   []*InstancePolicy  `json:"instances,omitempty"`
	ServiceAccount *ServiceAccount `json:"serviceAccount,omitempty"`
	Labels      map[string]string  `json:"labels,omitempty"`
	Network     *NetworkPolicy     `json:"network,omitempty"`
}

type LocationPolicy struct {
	AllowedLocations []string `json:"allowedLocations,omitempty"`
}

type InstancePolicy struct {
	MachineType      string            `json:"machineType,omitempty"`
	ProvisioningModel string           `json:"provisioningModel,omitempty"`
	Accelerators     []*Accelerator    `json:"accelerators,omitempty"`
	Disks            []*AttachedDisk   `json:"disks,omitempty"`
}

type Accelerator struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
}

type AttachedDisk struct {
	NewDisk      *Disk  `json:"newDisk,omitempty"`
	ExistingDisk string `json:"existingDisk,omitempty"`
	DeviceName   string `json:"deviceName,omitempty"`
}

type Disk struct {
	Type   string `json:"type,omitempty"`
	SizeGb int64  `json:"sizeGb,omitempty"`
}

type ServiceAccount struct {
	Email  string   `json:"email,omitempty"`
	Scopes []string `json:"scopes,omitempty"`
}

type NetworkPolicy struct {
	NetworkInterfaces []*NetworkInterface `json:"networkInterfaces,omitempty"`
}

type NetworkInterface struct {
	Network         string `json:"network,omitempty"`
	Subnetwork      string `json:"subnetwork,omitempty"`
	NoExternalIPAddress bool `json:"noExternalIpAddress,omitempty"`
}

type LogsPolicy struct {
	Destination     string `json:"destination,omitempty"`
	LogsPath        string `json:"logsPath,omitempty"`
}

type JobStatus struct {
	State          JobState               `json:"state"`
	StatusEvents   []*StatusEvent         `json:"statusEvents,omitempty"`
	TaskGroups     map[string]*TaskGroupStatus `json:"taskGroups,omitempty"`
	RunDuration    string                 `json:"runDuration,omitempty"`
}

type StatusEvent struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	EventTime   time.Time `json:"eventTime"`
}

type TaskGroupStatus struct {
	Counts map[string]int64 `json:"counts"`
}

type Task struct {
	Name       string    `json:"name"`
	Status     *TaskStatus `json:"status"`
}

type TaskStatus struct {
	State        TaskState      `json:"state"`
	StatusEvents []*StatusEvent `json:"statusEvents,omitempty"`
}

type ListJobsResponse struct {
	Jobs          []*Job `json:"jobs"`
	NextPageToken string `json:"nextPageToken,omitempty"`
}

type ListTasksResponse struct {
	Tasks         []*Task `json:"tasks"`
	NextPageToken string  `json:"nextPageToken,omitempty"`
}
