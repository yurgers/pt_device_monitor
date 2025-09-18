package main

import (
	"strings"
	"time"
)

type APIResponse struct {
	PhysicalDevices []PhysicalDevice `json:"physicalDevices"`
	Total           int              `json:"total"`
}

type PhysicalDevice struct {
	ID                  string        `json:"id"`
	LogicalDevice       LogicalDevice `json:"logicalDevice"`
	Name                string        `json:"name"`
	Description         string        `json:"description"`
	Model               string        `json:"model"`
	SerialNumber        string        `json:"serialNumber"`
	ConnectionState     string        `json:"connectionState"` // PHYSICAL_DEVICE_CONNECTION_STATE_UNSPECIFIED
	Address             string        `json:"address"`
	AddressType         string        `json:"addressType"`     // PHYSICAL_DEVICE_ADDRESS_TYPE_UNSPECIFIED
	LastConnectedAt     string        `json:"lastConnectedAt"` // RFC3339 format: "2019-08-24T14:15:22Z"
	CreatedAt           string        `json:"createdAt"`       // RFC3339 format: "2019-08-24T14:15:22Z"
	UpdatedAt           string        `json:"updatedAt"`       // RFC3339 format: "2019-08-24T14:15:22Z"
	AsNode              *AsNode       `json:"asNode,omitempty"`
	SoftwareVersion     string        `json:"softwareVersion"`
	TopologyType        string        `json:"topologyType"`        // TOPOLOGY_TYPE_UNSPECIFIED
	HealthStatus        string        `json:"healthStatus"`        // PHYSICAL_DEVICE_HEALTH_STATUS_UNSPECIFIED
	ConfigurationStatus string        `json:"configurationStatus"` // PHYSICAL_DEVICE_CONFIGURATION_STATUS_UNSPECIFIED
	ProductVersion      string        `json:"productVersion"`
	LogicalDeviceChange string        `json:"logicalDeviceChange"` // LOGICAL_DEVICE_CHANGE_UNSPECIFIED
}

type LogicalDevice struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	TopologyType    string           `json:"topologyType"` // TOPOLOGY_TYPE_UNSPECIFIED
	VirtualContexts []VirtualContext `json:"virtualContexts"`
}

type VirtualContext struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"isDefault"`
}

type AsNode struct {
	SyncLinkIP   string `json:"syncLinkIp"`
	SyncLinkPort int    `json:"syncLinkPort"`
	Priority     int    `json:"priority"`
	Role         string `json:"role"`        // ACTIVE_STANDBY_ROLE_UNSPECIFIED
	SuspendMode  string `json:"suspendMode"` // PHYSICAL_DEVICE_SUSPEND_MODE_UNSPECIFIED
}

type Config struct {
	BaseURL        string        `json:"base_url"`
	APIEndpoint    string        `json:"api_endpoint"`
	PollInterval   time.Duration `json:"poll_interval"`
	RequestTimeout time.Duration `json:"request_timeout"`
	ShowTimestamp  bool          `json:"show_timestamp"`
	ColorOutput    bool          `json:"color_output"`
	Username       string        `json:"username"`
	Password       string        `json:"password"`
}

type GroupedDevices struct {
	LogicalDeviceGroups []LogicalDeviceGroup `json:"groups"`
	TotalDevices        int                  `json:"total_devices"`
	LastUpdated         time.Time            `json:"last_updated"`
}

type LogicalDeviceGroup struct {
	LogicalDevice   LogicalDevice    `json:"logical_device"`
	PhysicalDevices []PhysicalDevice `json:"physical_devices"`
	IsCluster       bool             `json:"is_cluster"`
	ActiveNode      *PhysicalDevice  `json:"active_node,omitempty"`
	StandbyNodes    []PhysicalDevice `json:"standby_nodes,omitempty"`
}

func (g *LogicalDeviceGroup) GetTopologyDisplayName() string {
	switch g.LogicalDevice.TopologyType {
	case "TOPOLOGY_TYPE_STANDALONE":
		return "STANDALONE"
	case "TOPOLOGY_TYPE_ACTIVE_STANDBY":
		return "ACTIVE_STANDBY"
	default:
		return "UNSPECIFIED"
	}
}

func (g *LogicalDeviceGroup) GetVirtualContextsDisplay() string {
	var contexts []string
	for _, vc := range g.LogicalDevice.VirtualContexts {
		if vc.IsDefault {
			contexts = append(contexts, vc.Name+" (default)")
		} else {
			contexts = append(contexts, vc.Name)
		}
	}
	return strings.Join(contexts, ", ")
}

func (pd *PhysicalDevice) GetRoleDisplay() string {
	if pd.AsNode != nil {
		switch pd.AsNode.Role {
		case "ACTIVE_STANDBY_ROLE_ACTIVE":
			return "ACTIVE"
		case "ACTIVE_STANDBY_ROLE_STANDBY":
			return "STANDBY"
		default:
			return "UNSPECIFIED"
		}
	}
	return ""
}

func (pd *PhysicalDevice) GetConnectionStateDisplay() string {
	switch pd.ConnectionState {
	case "PHYSICAL_DEVICE_CONNECTION_STATE_CONNECTED":
		return "CONNECTED"
	case "PHYSICAL_DEVICE_CONNECTION_STATE_CONNECTING":
		return "CONNECTING"
	case "PHYSICAL_DEVICE_CONNECTION_STATE_DISCONNECTED":
		return "DISCONNECTED"
	default:
		return "UNSPECIFIED"
	}
}

// GetHealthStatusDisplay returns a human-readable health status
func (pd *PhysicalDevice) GetHealthStatusDisplay() string {
	switch pd.HealthStatus {
	case "PHYSICAL_DEVICE_HEALTH_STATUS_HEALTHY":
		return "HEALTHY"
	case "PHYSICAL_DEVICE_HEALTH_STATUS_WARNING":
		return "WARNING"
	case "PHYSICAL_DEVICE_HEALTH_STATUS_CRITICAL":
		return "CRITICAL"
	default:
		return "UNSPECIFIED"
	}
}

func (pd *PhysicalDevice) GetLastConnectedDisplay() string {
	if pd.LastConnectedAt == "" {
		return "Never"
	}

	t, err := time.Parse(time.RFC3339, pd.LastConnectedAt)
	if err != nil {
		return "Invalid"
	}

	return t.Format("2006-01-02 15:04")
}

func (pd *PhysicalDevice) GetProductVersionDisplay() string {
	if pd.ProductVersion == "" {
		return "-"
	}

	return string(pd.ProductVersion)
}

func (pd *PhysicalDevice) GetPriorityDisplay() string {
	if pd.AsNode != nil {
		return string(rune(pd.AsNode.Priority + '0'))
	}
	return ""
}
