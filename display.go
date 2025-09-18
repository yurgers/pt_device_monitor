package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/term"
)

type DisplayManager struct {
	config       *Config
	lastData     *GroupedDevices
	errorMessage string
	termWidth    int
	termHeight   int
	startRow     int
	linesDrawn   int
}

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
)

func NewDisplayManager(config *Config) *DisplayManager {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, height = 120, 50
	}

	dm := &DisplayManager{
		config:     config,
		termWidth:  width,
		termHeight: height,
		startRow:   -1, // Will be set on first render
		linesDrawn: 0,
	}

	return dm
}

func (dm *DisplayManager) StartFullScreenMode() {
	dm.initFullScreen()
}

func (dm *DisplayManager) initFullScreen() {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		// Clear entire screen
		fmt.Print("\033[2J")
		// Move cursor to top-left
		fmt.Print("\033[H")
		// Hide cursor for cleaner display
		fmt.Print("\033[?25l")
		// Enable alternate screen buffer (like top/htop)
		fmt.Print("\033[?1049h")
	}
}

func (dm *DisplayManager) ClearScreen() {
	// Clear entire screen and move cursor to top-left
	fmt.Print("\033[2J\033[H")
	dm.linesDrawn = 0
}

func (dm *DisplayManager) MoveCursor() {
	fmt.Print("\033[H")
}
func (dm *DisplayManager) RestoreTerminal() {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		// Disable alternate screen buffer (return to normal terminal)
		fmt.Print("\033[?1049l")
		// Show cursor
		fmt.Print("\033[?25h")
		// Reset all terminal attributes
		fmt.Print("\033[0m")
		// Move to a new line
		fmt.Print("\n")
	}
}

func (dm *DisplayManager) UpdateTerminalSize() {
	if width, height, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		dm.termWidth = width
		dm.termHeight = height
	}
}

func (dm *DisplayManager) printLine(text string) {
	fmt.Println(text)
	dm.linesDrawn++
}

func (dm *DisplayManager) printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)

	for _, char := range format {
		if char == '\n' {
			dm.linesDrawn++
		}
	}
}

// displayWidth calculates the actual display width of a string, excluding ANSI escape sequences
func displayWidth(s string) int {
	// Remove ANSI escape sequences using regex
	ansiRegex := regexp.MustCompile(`\033\[[0-9;]*[a-zA-Z]`)
	cleanString := ansiRegex.ReplaceAllString(s, "")
	// Use UTF-8 rune count instead of byte length to handle Unicode characters correctly
	return utf8.RuneCountInString(cleanString)
}

// stripColors removes all ANSI color codes from a string
func stripColors(s string) string {
	ansiRegex := regexp.MustCompile(`\033\[[0-9;]*[a-zA-Z]`)
	return ansiRegex.ReplaceAllString(s, "")
}

// Render renders the complete display
func (dm *DisplayManager) Render(data *GroupedDevices, err error) {
	dm.ClearScreen()

	if err != nil {
		dm.errorMessage = err.Error()
	} else {
		dm.errorMessage = ""
		dm.lastData = data
	}

	dm.renderHeader()

	if dm.errorMessage != "" {
		dm.renderError()
		if dm.lastData != nil {
			lastUpdateTime := dm.lastData.LastUpdated.Format("2006-01-02 15:04:05")
			message := fmt.Sprintf("Last known data (from %s):", lastUpdateTime)
			dm.renderSubheader(message)
			dm.renderDeviceGroups(dm.lastData)
		}
	} else if data != nil {
		dm.renderDeviceGroups(data)
	} else {
		dm.renderMessage("Waiting for data...")
	}

	dm.renderFooter()
}

// renderHeader renders the application header
func (dm *DisplayManager) renderHeader() {
	// Use actual terminal width or fallback to configured width
	tableWidth := dm.termWidth

	border := strings.Repeat("─", tableWidth-2) // -2 for border chars
	dm.printf("┌%s┐\n", border)

	title := "Physical Devices Monitor"
	if dm.config.ShowTimestamp {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		totalDevices := 0
		if dm.lastData != nil {
			totalDevices = dm.lastData.TotalDevices
		}

		title = fmt.Sprintf("%s - Last Updated: %s (Total: %d)",
			title, timestamp, totalDevices)
	}

	padding := tableWidth - displayWidth(title) - 4 // -4 for "│ " and " │"
	if padding < 0 {
		padding = 0
	}
	line := fmt.Sprintf("│ %s%s │", title, strings.Repeat(" ", padding))
	dm.printLine(line)

	dm.printf("├%s┤\n", border)
}

// simplifyErrorMessage extracts the essential part of an error message
func (dm *DisplayManager) simplifyErrorMessage(errorMsg string) string {
	// Define error patterns and their simplified messages
	errorPatterns := map[string]string{
		"context deadline exceeded": "Connection timeout",
		"connection refused":        "Connection refused",
		"no such host":              "Host not found",
		"invalid credentials":       "Invalid credentials",
		"unauthorized":              "Authentication failed",
		"forbidden":                 "Access denied",
		"not found":                 "Endpoint not found",
		"internal server error":     "Server error",
		"bad gateway":               "Bad gateway",
		"service unavailable":       "Service unavailable",
		"network is unreachable":    "Network unreachable",
		"certificate":               "Certificate error",
		"tls":                       "TLS/SSL error",
		"timeout":                   "Connection timeout",
		"connection reset":          "Connection reset",
		"broken pipe":               "Connection broken",
	}

	errorLower := strings.ToLower(errorMsg)

	// Check for known patterns
	for pattern, simplifiedMsg := range errorPatterns {
		if strings.Contains(errorLower, pattern) {
			return simplifiedMsg
		}
	}

	// If no pattern matches, try to extract the last meaningful part
	parts := strings.Split(errorMsg, ": ")
	if len(parts) > 1 {
		lastPart := parts[len(parts)-1]
		// Clean up common prefixes and suffixes
		cleanPrefixes := []string{"failed to ", "error: ", "unable to ", "cannot "}
		for _, prefix := range cleanPrefixes {
			lastPart = strings.TrimPrefix(lastPart, prefix)
		}

		// Capitalize first letter if it's a letter
		if len(lastPart) > 0 {
			firstChar := strings.ToUpper(string(lastPart[0]))
			if len(lastPart) > 1 {
				lastPart = firstChar + lastPart[1:]
			} else {
				lastPart = firstChar
			}
		}

		re := regexp.MustCompile(`\((.*?)\)`)
		matches := re.FindStringSubmatch(lastPart)

		if len(matches) > 1 {
			return (matches[1])
		}

		return lastPart
	}

	// If all else fails, return the original message (truncated if too long)
	if len(errorMsg) > 80 {
		return errorMsg[:77] + "..."
	}
	return errorMsg
}

func (dm *DisplayManager) renderError() {
	errorColor := dm.getColor(ColorRed)
	resetColor := dm.getColor(ColorReset)

	// Simplify the error message
	simplifiedError := dm.simplifyErrorMessage(dm.errorMessage)

	errorText := fmt.Sprintf("%sERROR: %s%s", errorColor, simplifiedError, resetColor)
	tableWidth := dm.termWidth

	padding := tableWidth - displayWidth(fmt.Sprintf("ERROR: %s", simplifiedError)) - 4
	if padding < 0 {
		padding = 0
	}
	paddedLine := fmt.Sprintf("│ %s%s │", errorText, strings.Repeat(" ", padding))
	dm.printLine(paddedLine)
	// Empty line
	emptyLine := fmt.Sprintf("│%s│", strings.Repeat(" ", tableWidth-2))
	dm.printLine(emptyLine)
}

func (dm *DisplayManager) renderSubheader(message string) {
	tableWidth := dm.termWidth

	padding := tableWidth - len(message) - 4 // -4 for "│ " and " │"
	if padding < 0 {
		padding = 0
	}
	line := fmt.Sprintf("│ %s%s │", message, strings.Repeat(" ", padding))
	dm.printLine(line)
}

func (dm *DisplayManager) renderMessage(message string) {
	tableWidth := dm.termWidth

	padding := tableWidth - len(message) - 4 // -4 for "│ " and " │"
	if padding < 0 {
		padding = 0
	}
	line := fmt.Sprintf("│ %s%s │", message, strings.Repeat(" ", padding))
	dm.printLine(line)
}

func (dm *DisplayManager) renderDeviceGroups(data *GroupedDevices) {
	if len(data.LogicalDeviceGroups) == 0 {
		dm.renderMessage("No devices found")
		return
	}

	// Sort groups by logical device name
	groups := make([]LogicalDeviceGroup, len(data.LogicalDeviceGroups))
	copy(groups, data.LogicalDeviceGroups)
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].LogicalDevice.Name < groups[j].LogicalDevice.Name
	})

	for i, group := range groups {
		if i > 0 {

			tableWidth := dm.termWidth

			emptyLine := fmt.Sprintf("│%s│", strings.Repeat(" ", tableWidth-2))
			dm.printLine(emptyLine)
		}
		dm.renderLogicalDeviceGroup(&group)
	}
}

func (dm *DisplayManager) renderLogicalDeviceGroup(group *LogicalDeviceGroup) {

	topologyColor := dm.getColor(ColorBlue)
	boldColor := dm.getColor(ColorBold)
	resetColor := dm.getColor(ColorReset)

	topology := group.GetTopologyDisplayName()
	header := fmt.Sprintf("%sLOGICAL DEVICE: %s %s(%s)%s",
		boldColor, group.LogicalDevice.Name, topologyColor, topology, resetColor)

	contexts := group.GetVirtualContextsDisplay()
	if contexts != "" {
		header += fmt.Sprintf(" - Contexts: %s", contexts)
	}

	tableWidth := dm.termWidth

	padding := tableWidth - len(fmt.Sprintf("LOGICAL DEVICE: %s (%s)", group.LogicalDevice.Name, topology)) - 4
	if contexts != "" {
		padding -= len(fmt.Sprintf(" - Contexts: %s", contexts))
	}
	if padding < 0 {
		padding = 0
	}

	line := fmt.Sprintf("│ %s%s │", header, strings.Repeat(" ", padding))
	dm.printLine(line)

	for i, device := range group.PhysicalDevices {
		isLast := i == len(group.PhysicalDevices)-1
		dm.renderPhysicalDevice(&device, isLast)
	}
}

func (dm *DisplayManager) renderTableHeaders() {
	colWidths := dm.calculateColumnWidths()

	treeCol := padString("", colWidths[0], true)
	nameCol := padString("Device Name", colWidths[1], true)
	modelCol := padString("Model", colWidths[2], true)
	statusCol := padString("Status", colWidths[3], true)
	addressCol := padString("Address", colWidths[4], true)
	priorityCol := padString("Priority", colWidths[5], true)
	versionCol := padString("Version", colWidths[6], true)

	headerRow := fmt.Sprintf("│ %s %s │ %s │ %s │ %s │ %s │ %s │",
		treeCol, nameCol, modelCol, statusCol, addressCol, priorityCol, versionCol)
	dm.printLine(headerRow)

	separator := "├" + strings.Repeat("─", colWidths[0]+2) + "┼" +
		strings.Repeat("─", colWidths[1]+2) + "┼" +
		strings.Repeat("─", colWidths[2]+2) + "┼" +
		strings.Repeat("─", colWidths[3]+2) + "┼" +
		strings.Repeat("─", colWidths[4]+2) + "┼" +
		strings.Repeat("─", colWidths[5]+2) + "┼" +
		strings.Repeat("─", colWidths[6]+2) + "┤"
	dm.printLine(separator)
}

func (dm *DisplayManager) calculateColumnWidths() []int {
	// Base column widths
	baseWidths := []int{3, 25, 15, 15, 12, 13, 8} // Tree, Name, Model, Status, Address, Priority, LastConnected

	totalBase := 0
	for _, w := range baseWidths {
		totalBase += w + 3 // +3 for " │ "
	}

	// If terminal is wider, expand name and address columns proportionally

	extraSpace := dm.termWidth - totalBase
	baseWidths[1] += int(float64(extraSpace) * 0.2)
	baseWidths[2] += int(float64(extraSpace) * 0.1)
	baseWidths[3] += int(float64(extraSpace) * 0.1)
	baseWidths[4] += int(float64(extraSpace) * 0.2)
	baseWidths[5] += int(float64(extraSpace) * 0.1)
	baseWidths[6] += int(float64(extraSpace) * 0.3)

	for i := range baseWidths {
		if baseWidths[i] < 0 {
			baseWidths[i] = 0
		}
	}

	return baseWidths
}

// padString pads a string to a specific width, handling ANSI color codes properly
// This ensures proper column alignment when strings contain color escape sequences
// which would otherwise disrupt fmt.Sprintf alignment with %-*s
func padString(s string, width int, leftAlign bool) string {

	currentWidth := displayWidth(s)
	if currentWidth >= width {
		return s
	}

	padding := strings.Repeat(" ", width-currentWidth)
	if leftAlign {
		return s + padding
	}
	return padding + s
}

// renderPhysicalDevice renders a single physical device with fixed columns
func (dm *DisplayManager) renderPhysicalDevice(device *PhysicalDevice, isLast bool) {
	// Tree character
	treeChar := "├─"
	if isLast {
		treeChar = "└─"
	}

	// Connection state color
	connColor := dm.getConnectionStateColor(device.ConnectionState)
	resetColor := dm.getColor(ColorReset)

	// Format device info with fixed column widths
	role := device.GetRoleDisplay()
	deviceName := device.Name
	if role != "" {
		// Add color to role in brackets
		roleColor := dm.getRoleColor(role)
		deviceName += fmt.Sprintf(" [%s%s%s]", roleColor, role, resetColor)
	}

	connectionState := device.GetConnectionStateDisplay()
	productVersion := device.GetProductVersionDisplay()

	// Get column widths from term library calculation
	colWidths := dm.calculateColumnWidths()

	// Priority for cluster nodes
	priority := "-"
	if device.AsNode != nil {
		if colWidths[5] < 12 {
			priority = fmt.Sprintf("%d", device.AsNode.Priority)
		} else {
			priority = fmt.Sprintf("Priority: %d", device.AsNode.Priority)
		}

	}

	// Fixed column widths using calculated sizes with proper color-aware padding
	treeCol := padString(treeChar, colWidths[0], true)
	nameCol := padString(truncateString(deviceName, colWidths[1]), colWidths[1], true)
	modelCol := padString(truncateString(device.Model, colWidths[2]), colWidths[2], true)
	statusCol := padString(truncateString(connectionState, colWidths[3]), colWidths[3], true)
	addressCol := padString(truncateString(device.Address, colWidths[4]), colWidths[4], true)
	priorityCol := padString(truncateString(priority, colWidths[5]), colWidths[5], true)
	versionCol := padString(truncateString(productVersion, colWidths[6]), colWidths[6], true)

	deviceRow := fmt.Sprintf(" %s %s │ %s │ %s%s%s │ %s │ %s │ %s",
		treeCol,
		nameCol,
		modelCol,
		connColor, statusCol, resetColor,
		addressCol,
		priorityCol,
		versionCol,
	)

	padding := dm.termWidth - displayWidth(deviceRow) - 4 // -4 for "│ " and " │"

	if padding < 1 {
		padding = 0
	}

	line := fmt.Sprintf("│ %s%s │", deviceRow, strings.Repeat(" ", padding))

	dm.printLine(line)

}

// truncateString truncates a string to a maximum length, adding "..." if needed
// Handles ANSI color codes properly by using display width instead of byte length
func truncateString(s string, maxLen int) string {
	displayLen := displayWidth(s)
	if displayLen <= maxLen {
		return s
	}

	if maxLen <= 3 {
		// For very short lengths, strip colors and truncate
		clean := stripColors(s)
		if len(clean) <= maxLen {
			return clean
		}
		return clean[:maxLen]
	}

	// Need to truncate while preserving color codes
	clean := stripColors(s)
	if len(clean) <= maxLen-3 {
		return s // Original fits with ellipsis
	}

	// Extract color codes and text, then reconstruct
	ansiRegex := regexp.MustCompile(`\033\[[0-9;]*[a-zA-Z]`)
	colorCodes := ansiRegex.FindAllString(s, -1)
	textParts := ansiRegex.Split(s, -1)

	// Build truncated string with colors
	var result strings.Builder
	colorIndex := 0
	textLen := 0
	targetLen := maxLen - 3 // Reserve space for "..."

	for _, part := range textParts {
		if textLen >= targetLen {
			break
		}

		remaining := targetLen - textLen
		if len(part) <= remaining {
			result.WriteString(part)
			textLen += len(part)
		} else {
			result.WriteString(part[:remaining])
			textLen += remaining
			break
		}

		// Add color code if available
		if colorIndex < len(colorCodes) {
			result.WriteString(colorCodes[colorIndex])
			colorIndex++
		}
	}

	result.WriteString("...")
	return result.String() + ColorReset
}

// renderFooter renders the application footer
func (dm *DisplayManager) renderFooter() {
	var color string
	resetColor := dm.getColor(ColorReset)

	// Use dynamic width
	tableWidth := dm.termWidth

	border := strings.Repeat("─", tableWidth-2)
	dm.printf("├%s┤\n", border)

	if dm.errorMessage != "" {
		color = dm.getColor(ColorRed)
	} else {
		color = dm.getColor(ColorGreen)
	}

	footerInfo := fmt.Sprintf("Poll Interval: %v │ Press Ctrl+C to exit │ MGMT: %s%s%s",
		dm.config.PollInterval,
		color,
		extractHostFromURL(dm.config.BaseURL),
		resetColor,
	)

	padding := tableWidth - displayWidth(footerInfo) - 4 // -4 for "│ " and " │"
	if padding < 0 {
		padding = 0
	}
	line := fmt.Sprintf("│ %s%s │", footerInfo, strings.Repeat(" ", padding))
	dm.printLine(line)

	dm.printf("└%s┘\n", border)
}

// getColor returns color code if color output is enabled
func (dm *DisplayManager) getColor(color string) string {
	if dm.config.ColorOutput {
		return color
	}
	return ""
}

// getConnectionStateColor returns appropriate color for connection state
func (dm *DisplayManager) getConnectionStateColor(state string) string {
	if !dm.config.ColorOutput {
		return ""
	}

	switch state {
	case "PHYSICAL_DEVICE_CONNECTION_STATE_CONNECTED":
		return ColorGreen
	case "PHYSICAL_DEVICE_CONNECTION_STATE_DISCONNECTED":
		return ColorRed
	default:
		return ColorYellow
	}
}

// getRoleColor returns appropriate color for cluster role
func (dm *DisplayManager) getRoleColor(role string) string {
	if !dm.config.ColorOutput {
		return ""
	}

	switch role {
	case "ACTIVE":
		return ColorGreen
	case "STANDBY":
		return ColorYellow
	default:
		return ColorRed
	}
}

// extractHostFromURL extracts hostname from URL for display
func extractHostFromURL(url string) string {
	if strings.HasPrefix(url, "https://") {
		url = url[8:]
	} else if strings.HasPrefix(url, "http://") {
		url = url[7:]
	}

	if idx := strings.Index(url, "/"); idx != -1 {
		url = url[:idx]
	}

	return url
}

func GroupDevicesByLogicalDevice(response *APIResponse) *GroupedDevices {
	groupMap := make(map[string]*LogicalDeviceGroup)

	for _, device := range response.PhysicalDevices {
		logicalID := device.LogicalDevice.ID

		if group, exists := groupMap[logicalID]; exists {
			group.PhysicalDevices = append(group.PhysicalDevices, device)
		} else {
			groupMap[logicalID] = &LogicalDeviceGroup{
				LogicalDevice:   device.LogicalDevice,
				PhysicalDevices: []PhysicalDevice{device},
			}
		}
	}

	var groups []LogicalDeviceGroup
	for _, group := range groupMap {
		// Analyze topology
		group.IsCluster = group.LogicalDevice.TopologyType == "TOPOLOGY_TYPE_ACTIVE_STANDBY" ||
			group.LogicalDevice.TopologyType == "TOPOLOGY_TYPE_CLUSTER"

		// Find active and standby nodes for cluster topologies
		if group.IsCluster {
			for i := range group.PhysicalDevices {
				device := &group.PhysicalDevices[i]
				if device.AsNode != nil && device.AsNode.Role == "ACTIVE_STANDBY_ROLE_ACTIVE" {
					group.ActiveNode = device
				} else if device.AsNode != nil && device.AsNode.Role == "ACTIVE_STANDBY_ROLE_STANDBY" {
					group.StandbyNodes = append(group.StandbyNodes, *device)
				}
			}
		}

		groups = append(groups, *group)
	}

	return &GroupedDevices{
		LogicalDeviceGroups: groups,
		TotalDevices:        len(response.PhysicalDevices),
		LastUpdated:         time.Now(),
	}
}
