package sdk

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TableWriter provides ASCII table formatting for CLI output
type TableWriter struct {
	headers       []string
	rows          [][]string
	columnWidths  []int
	borderStyle   BorderStyle
	alignments    []Alignment
	showBorder    bool
	showHeader    bool
	compactMode   bool
}

// BorderStyle defines the table border characters
type BorderStyle struct {
	TopLeft      string
	TopRight     string
	BottomLeft   string
	BottomRight  string
	Horizontal   string
	Vertical     string
	Cross        string
	LeftCross    string
	RightCross   string
	TopCross     string
	BottomCross  string
}

// Alignment for table columns
type Alignment int

const (
	AlignLeft Alignment = iota
	AlignRight
	AlignCenter
)

// Predefined border styles
var (
	// DefaultBorder is the standard ASCII table border
	DefaultBorder = BorderStyle{
		TopLeft: "┌", TopRight: "┐", BottomLeft: "└", BottomRight: "┘",
		Horizontal: "─", Vertical: "│", Cross: "┼",
		LeftCross: "├", RightCross: "┤", TopCross: "┬", BottomCross: "┴",
	}

	// SimpleBorder uses simple characters
	SimpleBorder = BorderStyle{
		TopLeft: "+", TopRight: "+", BottomLeft: "+", BottomRight: "+",
		Horizontal: "-", Vertical: "|", Cross: "+",
		LeftCross: "+", RightCross: "+", TopCross: "+", BottomCross: "+",
	}

	// DoubleBorder uses double-line characters
	DoubleBorder = BorderStyle{
		TopLeft: "╔", TopRight: "╗", BottomLeft: "╚", BottomRight: "╝",
		Horizontal: "═", Vertical: "║", Cross: "╬",
		LeftCross: "╠", RightCross: "╣", TopCross: "╦", BottomCross: "╩",
	}

	// NoBorder for minimal output
	NoBorder = BorderStyle{
		TopLeft: "", TopRight: "", BottomLeft: "", BottomRight: "",
		Horizontal: "", Vertical: " ", Cross: "",
		LeftCross: "", RightCross: "", TopCross: "", BottomCross: "",
	}
)

// NewTable creates a new table with headers
func NewTable(headers ...string) *TableWriter {
	return &TableWriter{
		headers:     headers,
		rows:        [][]string{},
		borderStyle: DefaultBorder,
		alignments:  make([]Alignment, len(headers)),
		showBorder:  true,
		showHeader:  true,
		compactMode: false,
	}
}

// NewCompactTable creates a table without borders
func NewCompactTable(headers ...string) *TableWriter {
	t := NewTable(headers...)
	t.borderStyle = NoBorder
	t.showBorder = false
	t.compactMode = true
	return t
}

// AddRow adds a row to the table
func (t *TableWriter) AddRow(cells ...string) {
	t.rows = append(t.rows, cells)
}

// SetAlignment sets column alignment (applies to all columns if indices not specified)
func (t *TableWriter) SetAlignment(align Alignment, columnIndices ...int) {
	if len(columnIndices) == 0 {
		for i := range t.alignments {
			t.alignments[i] = align
		}
	} else {
		for _, idx := range columnIndices {
			if idx < len(t.alignments) {
				t.alignments[idx] = align
			}
		}
	}
}

// SetBorderStyle changes the border style
func (t *TableWriter) SetBorderStyle(style BorderStyle) {
	t.borderStyle = style
}

// Render returns the formatted table as a string
func (t *TableWriter) Render() string {
	if len(t.rows) == 0 {
		return ""
	}

	t.calculateColumnWidths()

	var output strings.Builder

	// Top border
	if t.showBorder {
		output.WriteString(t.renderTopBorder())
		output.WriteString("\n")
	}

	// Header
	if t.showHeader {
		output.WriteString(t.renderRow(t.headers, true))
		output.WriteString("\n")

		if t.showBorder {
			output.WriteString(t.renderMiddleBorder())
			output.WriteString("\n")
		}
	}

	// Data rows
	for i, row := range t.rows {
		output.WriteString(t.renderRow(row, false))
		if i < len(t.rows)-1 || t.showBorder {
			output.WriteString("\n")
		}
	}

	// Bottom border
	if t.showBorder {
		output.WriteString(t.renderBottomBorder())
	}

	return output.String()
}

// Print prints the table to stdout
func (t *TableWriter) Print() {
	fmt.Println(t.Render())
}

// calculateColumnWidths determines the width needed for each column
func (t *TableWriter) calculateColumnWidths() {
	t.columnWidths = make([]int, len(t.headers))

	// Check headers
	for i, header := range t.headers {
		t.columnWidths[i] = len(header)
	}

	// Check all rows
	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(t.columnWidths) && len(cell) > t.columnWidths[i] {
				t.columnWidths[i] = len(cell)
			}
		}
	}

	// Add padding
	if !t.compactMode {
		for i := range t.columnWidths {
			t.columnWidths[i] += 2
		}
	}
}

// renderRow renders a single row with proper alignment
func (t *TableWriter) renderRow(cells []string, isHeader bool) string {
	var row strings.Builder

	if t.showBorder {
		row.WriteString(t.borderStyle.Vertical)
	}

	for i, cell := range cells {
		if i >= len(t.columnWidths) {
			break
		}

		width := t.columnWidths[i]
		padding := width - len(cell)

		if t.compactMode {
			row.WriteString(cell)
			if i < len(cells)-1 {
				row.WriteString("  ")
			}
		} else {
			// Apply alignment
			align := AlignLeft
			if i < len(t.alignments) {
				align = t.alignments[i]
			}

			switch align {
			case AlignLeft:
				row.WriteString(" ")
				row.WriteString(cell)
				row.WriteString(strings.Repeat(" ", padding-1))
			case AlignRight:
				row.WriteString(strings.Repeat(" ", padding-1))
				row.WriteString(cell)
				row.WriteString(" ")
			case AlignCenter:
				leftPad := padding / 2
				rightPad := padding - leftPad
				row.WriteString(strings.Repeat(" ", leftPad))
				row.WriteString(cell)
				row.WriteString(strings.Repeat(" ", rightPad))
			}

			if t.showBorder {
				row.WriteString(t.borderStyle.Vertical)
			}
		}
	}

	return row.String()
}

// renderTopBorder renders the top border
func (t *TableWriter) renderTopBorder() string {
	var border strings.Builder
	border.WriteString(t.borderStyle.TopLeft)
	for i, width := range t.columnWidths {
		border.WriteString(strings.Repeat(t.borderStyle.Horizontal, width))
		if i < len(t.columnWidths)-1 {
			border.WriteString(t.borderStyle.TopCross)
		}
	}
	border.WriteString(t.borderStyle.TopRight)
	return border.String()
}

// renderMiddleBorder renders the border between header and data
func (t *TableWriter) renderMiddleBorder() string {
	var border strings.Builder
	border.WriteString(t.borderStyle.LeftCross)
	for i, width := range t.columnWidths {
		border.WriteString(strings.Repeat(t.borderStyle.Horizontal, width))
		if i < len(t.columnWidths)-1 {
			border.WriteString(t.borderStyle.Cross)
		}
	}
	border.WriteString(t.borderStyle.RightCross)
	return border.String()
}

// renderBottomBorder renders the bottom border
func (t *TableWriter) renderBottomBorder() string {
	var border strings.Builder
	border.WriteString(t.borderStyle.BottomLeft)
	for i, width := range t.columnWidths {
		border.WriteString(strings.Repeat(t.borderStyle.Horizontal, width))
		if i < len(t.columnWidths)-1 {
			border.WriteString(t.borderStyle.BottomCross)
		}
	}
	border.WriteString(t.borderStyle.BottomRight)
	return border.String()
}

// ============================================================================
// CONFIGHHUB-SPECIFIC TABLE FUNCTIONS
// ============================================================================

// RenderSpacesTable creates a table from ConfigHub spaces
func RenderSpacesTable(spaces []*Space) string {
	table := NewTable("Space", "Display Name", "Labels", "Created", "Version")
	table.SetAlignment(AlignRight, 4) // Version column right-aligned

	for _, space := range spaces {
		labels := formatLabels(space.Labels)
		created := formatTimestamp(space.CreatedAt)

		table.AddRow(
			space.Slug,
			truncate(space.DisplayName, 30),
			truncate(labels, 25),
			created,
			fmt.Sprintf("v%d", space.Version),
		)
	}

	return table.Render()
}

// RenderUnitsTable creates a table from ConfigHub units
func RenderUnitsTable(units []*Unit, showUpstream bool) string {
	headers := []string{"Unit", "Display Name", "Type", "Labels", "Applied"}
	if showUpstream {
		headers = append(headers, "Upstream")
	}
	headers = append(headers, "Version")

	table := NewTable(headers...)
	table.SetAlignment(AlignCenter, 4) // Applied status centered
	table.SetAlignment(AlignRight, len(headers)-1) // Version right-aligned

	for _, unit := range units {
		unitType := unit.Labels["type"]
		if unitType == "" {
			unitType = "unknown"
		}

		labels := formatLabels(unit.Labels)
		applied := "✓"
		if unit.TargetID == nil {
			applied = "-"
		}

		row := []string{
			truncate(unit.Slug, 25),
			truncate(unit.DisplayName, 30),
			unitType,
			truncate(labels, 20),
			applied,
		}

		if showUpstream {
			upstream := "-"
			if unit.UpstreamUnitID != nil {
				upstream = "✓"
			}
			row = append(row, upstream)
		}

		row = append(row, fmt.Sprintf("v%d", unit.Version))

		table.AddRow(row...)
	}

	return table.Render()
}

// RenderSetsTable creates a table from ConfigHub sets
func RenderSetsTable(sets []*Set) string {
	table := NewTable("Set", "Display Name", "Members", "Created")

	for _, set := range sets {
		memberCount := "0"
		if set.UnitIDs != nil {
			memberCount = fmt.Sprintf("%d", len(set.UnitIDs))
		}

		created := formatTimestamp(set.CreatedAt)

		table.AddRow(
			set.Slug,
			truncate(set.DisplayName, 40),
			memberCount,
			created,
		)
	}

	return table.Render()
}

// RenderFiltersTable creates a table from ConfigHub filters
func RenderFiltersTable(filters []*Filter) string {
	table := NewTable("Filter", "Resource Type", "Where Clause", "Created")

	for _, filter := range filters {
		whereClause := filter.Where
		if whereClause == "" {
			whereClause = "(empty)"
		}

		created := formatTimestamp(filter.CreatedAt)

		table.AddRow(
			filter.Slug,
			filter.ResourceType,
			truncate(whereClause, 40),
			created,
		)
	}

	return table.Render()
}

// ============================================================================
// ACTIVITY / AUDIT LOG TABLES
// ============================================================================

// ActivityEvent represents a ConfigHub activity
type ActivityEvent struct {
	Timestamp   time.Time
	User        string
	Action      string
	Resource    string
	Status      string // "success", "failure", "pending"
	Details     string
}

// RenderActivityTable creates a table showing recent ConfigHub activity
func RenderActivityTable(events []ActivityEvent) string {
	table := NewTable("Time", "User", "Action", "Resource", "Status", "Details")
	table.SetAlignment(AlignCenter, 4) // Status centered

	for _, event := range events {
		status := event.Status
		statusIcon := ""
		switch status {
		case "success":
			statusIcon = "✓"
		case "failure":
			statusIcon = "✗"
		case "pending":
			statusIcon = "⏳"
		}

		table.AddRow(
			formatTimestamp(event.Timestamp),
			truncate(event.User, 15),
			truncate(event.Action, 20),
			truncate(event.Resource, 25),
			statusIcon+" "+status,
			truncate(event.Details, 30),
		)
	}

	return table.Render()
}

// RenderSuccessFailureTable creates a summary table of operations
func RenderSuccessFailureTable(operations map[string]bool) string {
	table := NewTable("Operation", "Status", "Result")
	table.SetAlignment(AlignCenter, 1) // Status centered

	for operation, success := range operations {
		status := "✓"
		result := "SUCCESS"
		if !success {
			status = "✗"
			result = "FAILURE"
		}

		table.AddRow(operation, status, result)
	}

	return table.Render()
}

// ============================================================================
// RESOURCE STATE COMPARISON TABLES
// ============================================================================

// ResourceState represents the state of a resource
type ResourceState struct {
	Name              string
	DesiredState      string
	ActualState       string
	Drift             bool
	LastSyncTime      time.Time
	ConfigHubVersion  int64
	KubernetesVersion string
}

// RenderStateComparisonTable shows ConfigHub vs Kubernetes state
func RenderStateComparisonTable(resources []ResourceState) string {
	table := NewTable("Resource", "ConfigHub", "Kubernetes", "Drift", "Last Sync", "Versions")
	table.SetAlignment(AlignCenter, 3) // Drift centered

	for _, resource := range resources {
		drift := "-"
		if resource.Drift {
			drift = "⚠ YES"
		} else {
			drift = "✓ No"
		}

		versions := fmt.Sprintf("CH:v%d K8s:%s",
			resource.ConfigHubVersion,
			resource.KubernetesVersion)

		table.AddRow(
			truncate(resource.Name, 25),
			truncate(resource.DesiredState, 15),
			truncate(resource.ActualState, 15),
			drift,
			formatTimestamp(resource.LastSyncTime),
			versions,
		)
	}

	return table.Render()
}

// RenderKubectlTable formats kubectl output as a table
func RenderKubectlTable(headers []string, rows [][]string) string {
	table := NewTable(headers...)

	for _, row := range rows {
		table.AddRow(row...)
	}

	return table.Render()
}

// ============================================================================
// ENVIRONMENT HIERARCHY TABLE
// ============================================================================

// EnvironmentInfo represents an environment in the hierarchy
type EnvironmentInfo struct {
	Name         string
	SpaceID      uuid.UUID
	UnitCount    int
	Applied      int
	NeedsUpgrade int
	Health       string
}

// RenderEnvironmentHierarchyTable shows the environment hierarchy
func RenderEnvironmentHierarchyTable(envs []EnvironmentInfo) string {
	table := NewTable("Environment", "Space ID", "Units", "Applied", "Needs Upgrade", "Health")
	table.SetAlignment(AlignRight, 2, 3, 4) // Numbers right-aligned
	table.SetAlignment(AlignCenter, 5)      // Health centered

	for _, env := range envs {
		healthIcon := ""
		switch env.Health {
		case "healthy":
			healthIcon = "✓"
		case "degraded":
			healthIcon = "⚠"
		case "unhealthy":
			healthIcon = "✗"
		}

		table.AddRow(
			env.Name,
			env.SpaceID.String()[:8]+"...",
			fmt.Sprintf("%d", env.UnitCount),
			fmt.Sprintf("%d", env.Applied),
			fmt.Sprintf("%d", env.NeedsUpgrade),
			healthIcon+" "+env.Health,
		)
	}

	return table.Render()
}

// ============================================================================
// COST ANALYSIS TABLE
// ============================================================================

// RenderCostAnalysisTable shows cost breakdown
func RenderCostAnalysisTable(units []UnitCostEstimate) string {
	table := NewTable("Unit", "Replicas", "CPU Cost", "Memory Cost", "Storage Cost", "Total/Month")
	table.SetAlignment(AlignRight, 1, 2, 3, 4, 5) // All numeric columns right-aligned

	var totalCost float64

	for _, unit := range units {
		table.AddRow(
			truncate(unit.UnitName, 30),
			fmt.Sprintf("%d", unit.Replicas),
			fmt.Sprintf("$%.2f", unit.Breakdown.CPUCost),
			fmt.Sprintf("$%.2f", unit.Breakdown.MemoryCost),
			fmt.Sprintf("$%.2f", unit.Breakdown.StorageCost),
			fmt.Sprintf("$%.2f", unit.MonthlyCost),
		)
		totalCost += unit.MonthlyCost
	}

	// Add total row
	table.AddRow(
		"TOTAL",
		"",
		"",
		"",
		"",
		fmt.Sprintf("$%.2f", totalCost),
	)

	return table.Render()
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// formatLabels converts a map of labels to a string
func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return "-"
	}

	var pairs []string
	for k, v := range labels {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(pairs, ", ")
}

// formatTimestamp formats a time.Time to a short string
func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	} else if diff < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	}

	return t.Format("2006-01-02")
}

// truncate truncates a string to a maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// ============================================================================
// QUICK HELPER FUNCTIONS
// ============================================================================

// QuickTable creates and renders a simple table in one call
func QuickTable(headers []string, rows [][]string) string {
	table := NewTable(headers...)
	for _, row := range rows {
		table.AddRow(row...)
	}
	return table.Render()
}

// PrintSpaces is a convenience function to print spaces table
func PrintSpaces(spaces []*Space) {
	fmt.Println(RenderSpacesTable(spaces))
}

// PrintUnits is a convenience function to print units table
func PrintUnits(units []*Unit, showUpstream bool) {
	fmt.Println(RenderUnitsTable(units, showUpstream))
}

// PrintActivity is a convenience function to print activity table
func PrintActivity(events []ActivityEvent) {
	fmt.Println(RenderActivityTable(events))
}

// PrintStateComparison is a convenience function to print state comparison
func PrintStateComparison(resources []ResourceState) {
	fmt.Println(RenderStateComparisonTable(resources))
}
