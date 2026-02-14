package parser

import (
	"regexp"
	"strings"
)

// Assignment operator patterns, ordered by length (longest first to avoid
// partial matches, e.g., ::= before :=).
var assignOps = []string{"::=", "!=", "?=", "+=", ":=", "="}

// Conditional directive keywords.
var conditionalKeywords = map[string]bool{
	"ifeq":   true,
	"ifneq":  true,
	"ifdef":  true,
	"ifndef": true,
	"else":   true,
	"endif":  true,
}

// Include directive keywords.
var includeKeywords = map[string]bool{
	"include":  true,
	"-include": true,
	"sinclude": true,
}

// Directive keywords that start a line (non-conditional, non-include).
var directiveKeywords = map[string]bool{
	".PHONY":        true,
	".DEFAULT_GOAL": true,
	".SUFFIXES":     true,
	".DELETE_ON_ERROR": true,
	".SECONDARY":    true,
	".PRECIOUS":     true,
	".INTERMEDIATE": true,
	".NOTPARALLEL":  true,
	".ONESHELL":     true,
	".POSIX":        true,
	".SILENT":       true,
	".IGNORE":       true,
	".EXPORT_ALL_VARIABLES": true,
	"export":        true,
	"unexport":      true,
	"vpath":         true,
	"override":      true,
}

// bannerRe matches decorative comment lines:
//   - ^#+$                   — line of only # characters
//   - ^#\s*[=\-#]{3,}\s*$   — # followed by repeated =, -, or #
//   - ^#{2,}\s.*\s#{2,}$    — box-style ## Title ##
var bannerRe = regexp.MustCompile(
	`^#+$|^#\s*[=\-#]{3,}\s*$|^#{2,}\s+.*\s+#{2,}$`,
)

// Parse converts Makefile source text into an AST.
func Parse(src string) []*Node {
	p := &state{}
	return p.parse(src)
}

// state tracks parser state across lines.
type state struct {
	inRule   bool // True when we're inside a rule (expecting recipe lines).
	inDefine bool // True when inside define..endef block.
	nodes    []*Node
	lineNum  int
}

func (p *state) parse(src string) []*Node {
	lines := splitLines(src)
	p.nodes = make([]*Node, 0, len(lines))

	for p.lineNum = 0; p.lineNum < len(lines); p.lineNum++ {
		// Handle define/endef blocks.
		if p.inDefine {
			p.handleDefineBlock(lines)
			continue
		}

		// Join continuation lines.
		joined, count := joinContinuations(lines, p.lineNum)
		rawLines := lines[p.lineNum : p.lineNum+count]
		raw := strings.Join(rawLines, "\n")

		node := p.classifyLine(joined, raw)
		node.Line = p.lineNum + 1 // 1-indexed.

		// If we consumed multiple lines via continuation, advance.
		if count > 1 {
			p.lineNum += count - 1
		}

		p.addNode(node)
	}

	return p.nodes
}

// addNode appends a node, managing parent-child relationships.
func (p *state) addNode(node *Node) {
	switch node.Type {
	case NodeRule:
		p.inRule = true
		p.nodes = append(p.nodes, node)

	case NodeRecipe:
		// Attach as child of the most recent rule node.
		if len(p.nodes) > 0 {
			parent := p.findRuleParent()
			if parent != nil {
				parent.Children = append(parent.Children, node)
				return
			}
		}
		// No parent rule found; treat as raw.
		node.Type = NodeRaw
		p.nodes = append(p.nodes, node)

	case NodeBlankLine:
		// A blank line after a rule ends the recipe context.
		p.inRule = false
		p.nodes = append(p.nodes, node)

	case NodeComment, NodeSectionHeader, NodeBannerComment:
		// Comments within a rule context don't end it.
		p.nodes = append(p.nodes, node)

	default:
		// Any non-recipe, non-comment, non-blank line ends recipe context.
		p.inRule = false
		p.nodes = append(p.nodes, node)
	}
}

// findRuleParent returns the most recent NodeRule in the top-level nodes.
func (p *state) findRuleParent() *Node {
	for i := len(p.nodes) - 1; i >= 0; i-- {
		if p.nodes[i].Type == NodeRule {
			return p.nodes[i]
		}
		// Stop searching at non-recipe, non-comment, non-blank nodes.
		switch p.nodes[i].Type {
		case NodeRecipe, NodeComment, NodeBannerComment, NodeSectionHeader, NodeBlankLine:
			continue
		default:
			return nil
		}
	}
	return nil
}

// handleDefineBlock consumes lines until endef.
func (p *state) handleDefineBlock(lines []string) {
	// The define line was already added. Collect body until endef.
	defineNode := p.nodes[len(p.nodes)-1]
	rawParts := []string{defineNode.Raw}

	for p.lineNum < len(lines) {
		line := lines[p.lineNum]
		rawParts = append(rawParts, line)

		trimmed := strings.TrimSpace(line)
		if trimmed == "endef" {
			p.inDefine = false
			defineNode.Raw = strings.Join(rawParts, "\n")
			return
		}
		p.lineNum++
	}

	// If we reach end of file without endef, still record what we have.
	defineNode.Raw = strings.Join(rawParts, "\n")
	p.inDefine = false
}

// classifyLine determines the NodeType for a single (possibly joined) line.
func (p *state) classifyLine(joined, raw string) *Node {
	trimmed := strings.TrimSpace(joined)

	// 1. Blank line.
	if trimmed == "" {
		return &Node{Type: NodeBlankLine, Raw: raw}
	}

	// 2. Check for define blocks.
	if strings.HasPrefix(trimmed, "define ") || trimmed == "define" {
		p.inDefine = true
		return &Node{Type: NodeRaw, Raw: raw}
	}

	// 3. Section header: ##@ ...
	if strings.HasPrefix(trimmed, "##@") {
		text := strings.TrimSpace(trimmed[3:])
		return &Node{
			Type: NodeSectionHeader,
			Raw:  raw,
			Fields: NodeFields{
				Text:   text,
				Prefix: "##@",
			},
		}
	}

	// 4. Banner comment: decorative separator.
	if isBannerComment(trimmed) {
		return &Node{
			Type: NodeBannerComment,
			Raw:  raw,
			Fields: NodeFields{
				Text: trimmed,
			},
		}
	}

	// 5. Comment: starts with #.
	if strings.HasPrefix(trimmed, "#") {
		return parseComment(trimmed, raw)
	}

	// 6. Recipe: starts with tab and we're in a rule context.
	if strings.HasPrefix(joined, "\t") && p.inRule {
		return &Node{
			Type: NodeRecipe,
			Raw:  raw,
			Fields: NodeFields{
				Text: strings.TrimPrefix(joined, "\t"),
			},
		}
	}

	// 7. Conditional: ifeq, ifdef, ifndef, else, endif.
	if node := tryConditional(trimmed, raw); node != nil {
		return node
	}

	// 8. Include: include, -include, sinclude.
	if node := tryInclude(trimmed, raw); node != nil {
		return node
	}

	// 9. Directive: .PHONY, export, etc. (before assignment/rule to prevent
	// ".PHONY: x" being parsed as a rule or ".DEFAULT_GOAL := x" as assignment).
	if node := tryDirective(trimmed, raw); node != nil {
		return node
	}

	// 10. Assignment: contains assignment operator.
	if node := tryAssignment(trimmed, raw); node != nil {
		return node
	}

	// 11. Rule: contains : with target pattern.
	if node := tryRule(trimmed, raw); node != nil {
		return node
	}

	// 12. Raw: anything else.
	return &Node{Type: NodeRaw, Raw: raw}
}

func isBannerComment(trimmed string) bool {
	if !strings.HasPrefix(trimmed, "#") {
		return false
	}
	// A single "#" is just an empty comment, not a banner.
	if trimmed == "#" {
		return false
	}
	return bannerRe.MatchString(trimmed)
}

func parseComment(trimmed, raw string) *Node {
	// Determine prefix: ## or #.
	prefix := "#"
	if strings.HasPrefix(trimmed, "##") && !strings.HasPrefix(trimmed, "##@") {
		prefix = "##"
	}

	text := strings.TrimPrefix(trimmed, prefix)
	text = strings.TrimSpace(text)

	return &Node{
		Type: NodeComment,
		Raw:  raw,
		Fields: NodeFields{
			Text:   text,
			Prefix: prefix,
		},
	}
}

func tryConditional(trimmed, raw string) *Node {
	for keyword := range conditionalKeywords {
		if trimmed == keyword || strings.HasPrefix(trimmed, keyword+" ") || strings.HasPrefix(trimmed, keyword+"\t") {
			condition := ""
			if len(trimmed) > len(keyword) {
				condition = strings.TrimSpace(trimmed[len(keyword):])
			}
			return &Node{
				Type: NodeConditional,
				Raw:  raw,
				Fields: NodeFields{
					Directive: keyword,
					Condition: condition,
				},
			}
		}
	}
	return nil
}

func tryInclude(trimmed, raw string) *Node {
	for keyword := range includeKeywords {
		if trimmed == keyword || strings.HasPrefix(trimmed, keyword+" ") || strings.HasPrefix(trimmed, keyword+"\t") {
			pathStr := ""
			if len(trimmed) > len(keyword) {
				pathStr = strings.TrimSpace(trimmed[len(keyword):])
			}
			var paths []string
			if pathStr != "" {
				paths = strings.Fields(pathStr)
			}
			return &Node{
				Type: NodeInclude,
				Raw:  raw,
				Fields: NodeFields{
					IncludeType: keyword,
					Paths:       paths,
				},
			}
		}
	}
	return nil
}

func tryAssignment(trimmed, raw string) *Node {
	for _, op := range assignOps {
		idx := strings.Index(trimmed, op)
		if idx < 0 {
			continue
		}

		// For plain "=", make sure it's not part of ":=", "::=", "?=", "+=", "!=".
		if op == "=" && idx > 0 {
			prev := trimmed[idx-1]
			if prev == ':' || prev == '?' || prev == '+' || prev == '!' {
				continue
			}
		}

		// For ":=", make sure it's not "::=".
		if op == ":=" && idx > 0 && trimmed[idx-1] == ':' {
			continue
		}

		// The variable name must be before the operator.
		varName := strings.TrimSpace(trimmed[:idx])
		if varName == "" {
			continue
		}

		// Variable names should not contain spaces (they're likely rule targets).
		if strings.ContainsAny(varName, " \t") {
			// Could be "override VAR = val".
			parts := strings.Fields(varName)
			if len(parts) == 2 && parts[0] == "override" {
				varName = "override " + parts[1]
			} else {
				continue
			}
		}

		// Make sure the variable name looks valid (not a target:prereq pattern).
		if strings.Contains(varName, ":") && op == "=" {
			continue
		}

		varValue := ""
		afterOp := idx + len(op)
		if afterOp < len(trimmed) {
			varValue = strings.TrimSpace(trimmed[afterOp:])
		}

		return &Node{
			Type: NodeAssignment,
			Raw:  raw,
			Fields: NodeFields{
				VarName:  varName,
				AssignOp: op,
				VarValue: varValue,
			},
		}
	}
	return nil
}

func tryRule(trimmed, raw string) *Node {
	// Find the colon that separates targets from prerequisites.
	// Must not be part of ::= or := assignment operators.
	colonIdx := findRuleColon(trimmed)
	if colonIdx < 0 {
		return nil
	}

	targetStr := strings.TrimSpace(trimmed[:colonIdx])
	if targetStr == "" {
		return nil
	}

	targets := strings.Fields(targetStr)

	// Parse the rest: prerequisites and optional inline help.
	rest := ""
	if colonIdx+1 < len(trimmed) {
		rest = strings.TrimSpace(trimmed[colonIdx+1:])
	}

	var prerequisites []string
	var orderOnly []string
	var inlineHelp string

	// Check for inline help comment: ## at end of line.
	if helpIdx := strings.Index(rest, "##"); helpIdx >= 0 {
		inlineHelp = strings.TrimSpace(rest[helpIdx+2:])
		rest = strings.TrimSpace(rest[:helpIdx])
	}

	// Split prerequisites at |.
	if pipeIdx := strings.Index(rest, "|"); pipeIdx >= 0 {
		prereqStr := strings.TrimSpace(rest[:pipeIdx])
		orderStr := strings.TrimSpace(rest[pipeIdx+1:])
		if prereqStr != "" {
			prerequisites = strings.Fields(prereqStr)
		}
		if orderStr != "" {
			orderOnly = strings.Fields(orderStr)
		}
	} else if rest != "" {
		prerequisites = strings.Fields(rest)
	}

	return &Node{
		Type: NodeRule,
		Raw:  raw,
		Fields: NodeFields{
			Targets:       targets,
			Prerequisites: prerequisites,
			OrderOnly:     orderOnly,
			InlineHelp:    inlineHelp,
		},
	}
}

// findRuleColon finds the index of the colon that separates targets from
// prerequisites. Returns -1 if not found.
func findRuleColon(line string) int {
	// Skip lines that look like assignments (contain =, :=, etc.).
	for _, op := range assignOps {
		if strings.Contains(line, op) {
			return -1
		}
	}

	for i := 0; i < len(line); i++ {
		if line[i] == ':' {
			return i
		}
	}
	return -1
}

func tryDirective(trimmed, raw string) *Node {
	firstWord := trimmed
	if idx := strings.IndexAny(trimmed, " \t:"); idx >= 0 {
		firstWord = trimmed[:idx]
	}

	if directiveKeywords[firstWord] {
		return &Node{
			Type: NodeDirective,
			Raw:  raw,
			Fields: NodeFields{
				Text: trimmed,
			},
		}
	}
	return nil
}

// splitLines splits source into lines, preserving empty trailing lines.
func splitLines(src string) []string {
	if src == "" {
		return nil
	}
	lines := strings.Split(src, "\n")
	// Remove the trailing empty element that Split produces for a
	// trailing newline (but keep intentional blank lines).
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// joinContinuations joins lines ending in \ and returns the joined
// string and the number of lines consumed.
func joinContinuations(lines []string, start int) (string, int) {
	if !hasContinuation(lines[start]) {
		return lines[start], 1
	}

	var parts []string
	i := start
	for i < len(lines) && hasContinuation(lines[i]) {
		// Strip trailing backslash for the joined representation.
		line := strings.TrimRight(lines[i], " \t")
		line = line[:len(line)-1] // Remove the \.
		parts = append(parts, line)
		i++
	}
	// Add the final line (no continuation).
	if i < len(lines) {
		parts = append(parts, lines[i])
		i++
	}

	return strings.Join(parts, " "), i - start
}

// hasContinuation returns true if the line ends with a backslash.
func hasContinuation(line string) bool {
	trimmed := strings.TrimRight(line, " \t")
	return strings.HasSuffix(trimmed, "\\")
}
