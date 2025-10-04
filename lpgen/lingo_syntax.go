package lpgen

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/HazelnutParadise/insyra"
)

// LINGO Syntax Parser
//
// This file provides support for parsing native LINGO syntax, allowing users to
// convert LINGO models to LP format without needing to generate the model in LINGO first.
//
// Supported Features:
//   - Comments (! to end of line)
//   - Sections: SETS, DATA, MODEL
//   - @ Functions: @SUM, @FOR, @BIN, @GIN
//   - Set definitions with inline elements
//   - Data value assignments
//   - Variable naming patterns (e.g., X_SETNAME)
//
// Limitations:
//   - Multi-dimensional sets are parsed but may need manual expansion
//   - Nested @SUM within @FOR may require careful structuring
//   - Data arrays are not automatically indexed to variables
//
// Future Enhancements:
//   - Support for @MIN, @MAX functions
//   - Multi-dimensional set indexing
//   - More complex data structures
//   - @IF conditional constraints
//   - @FREE unbounded variables

// LingoModel represents a LINGO model with sets, data, and constraints
type LingoModel struct {
	Sets        map[string][]string // Set definitions (e.g., "PRODUCTS" -> ["P1", "P2"])
	Data        map[string][]float64 // Data values (e.g., "COST" -> [10, 20, 30])
	Variables   map[string]string    // Variable definitions with their indices
	Objective   string               // Objective function
	Constraints []string             // Constraint expressions
}

// ParseLingoSyntax parses LINGO source code with @ functions
func ParseLingoSyntax(source string) (*LPModel, error) {
	lm := &LingoModel{
		Sets:        make(map[string][]string),
		Data:        make(map[string][]float64),
		Variables:   make(map[string]string),
		Constraints: make([]string, 0),
	}

	// Parse the model
	if err := lm.parseModel(source); err != nil {
		return nil, err
	}

	// Expand @ functions and convert to LP model
	return lm.expandToLPModel()
}

// parseModel parses the LINGO source into structured data
func (lm *LingoModel) parseModel(source string) error {
	// Remove comments
	source = lm.removeComments(source)

	// Split into sections and accumulate multi-line expressions
	lines := strings.Split(source, "\n")
	
	currentSection := ""
	var currentExpr strings.Builder
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Detect section boundaries
		upper := strings.ToUpper(line)
		if strings.HasPrefix(upper, "SETS:") {
			currentSection = "SETS"
			continue
		} else if strings.HasPrefix(upper, "DATA:") {
			currentSection = "DATA"
			continue
		} else if strings.HasPrefix(upper, "ENDSETS") || strings.HasPrefix(upper, "ENDDATA") {
			currentSection = ""
			continue
		} else if strings.HasPrefix(upper, "MODEL:") {
			// MODEL: line might have content after it, so process it
			if len(line) > 6 {
				line = strings.TrimSpace(line[6:])
				if line != "" {
					currentExpr.WriteString(line)
				}
			}
			continue
		} else if strings.HasPrefix(upper, "END") {
			// Process any remaining expression
			if currentExpr.Len() > 0 {
				expr := currentExpr.String()
				if err := lm.processExpression(expr, currentSection); err != nil {
					insyra.LogWarning("lpgen", "parseModel", "Failed to process expression: %s", err.Error())
				}
				currentExpr.Reset()
			}
			break
		}

		// Accumulate lines for multi-line expressions
		if currentExpr.Len() > 0 {
			currentExpr.WriteString(" ")
		}
		currentExpr.WriteString(line)
		
		// Check if expression is complete (ends with semicolon)
		if strings.HasSuffix(line, ";") {
			expr := currentExpr.String()
			if err := lm.processExpression(expr, currentSection); err != nil {
				insyra.LogWarning("lpgen", "parseModel", "Failed to process expression: %s", err.Error())
			}
			currentExpr.Reset()
		}
	}
	
	// Process any remaining expression (for incomplete input)
	if currentExpr.Len() > 0 {
		expr := currentExpr.String()
		if err := lm.processExpression(expr, currentSection); err != nil {
			insyra.LogWarning("lpgen", "parseModel", "Failed to process expression: %s", err.Error())
		}
	}

	return nil
}

// processExpression routes an expression to the appropriate parser based on section
func (lm *LingoModel) processExpression(expr string, section string) error {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil
	}
	
	switch section {
	case "SETS":
		return lm.parseSetLine(expr)
	case "DATA":
		return lm.parseDataLine(expr)
	default:
		// No section or MODEL section - could be data or model constraints
		return lm.parseModelLine(expr)
	}
}

// removeComments removes LINGO comments (! to end of line)
func (lm *LingoModel) removeComments(source string) string {
	lines := strings.Split(source, "\n")
	cleaned := make([]string, 0)
	
	for _, line := range lines {
		if idx := strings.Index(line, "!"); idx >= 0 {
			line = line[:idx]
		}
		cleaned = append(cleaned, line)
	}
	
	return strings.Join(cleaned, "\n")
}

// parseSetLine parses a set definition line
func (lm *LingoModel) parseSetLine(line string) error {
	// Examples: 
	// PRODUCTS / P1 P2 P3 /;
	// group /1..5/:group_size, afa, r;
	// formula1(group,group):contact_rate, next_generation;
	
	line = strings.TrimSuffix(line, ";")
	
	// Check for set with attributes (contains colon)
	if strings.Contains(line, ":") {
		parts := strings.SplitN(line, ":", 2)
		setDef := strings.TrimSpace(parts[0])
		// attributes := strings.TrimSpace(parts[1]) // We can parse attributes if needed later
		
		// Extract set name and range/elements
		if strings.Contains(setDef, "/") {
			// Has inline definition
			slashParts := strings.Split(setDef, "/")
			if len(slashParts) >= 2 {
				setName := strings.TrimSpace(slashParts[0])
				rangeOrElements := strings.TrimSpace(slashParts[1])
				
				// Check for range notation (e.g., "1..5")
				if strings.Contains(rangeOrElements, "..") {
					elements := lm.expandRange(rangeOrElements)
					lm.Sets[setName] = elements
				} else {
					// Regular element list
					elements := strings.Fields(rangeOrElements)
					lm.Sets[setName] = elements
				}
			}
		}
	} else if strings.Contains(line, "/") {
		// Simple set definition without attributes
		parts := strings.Split(line, "/")
		if len(parts) >= 2 {
			setName := strings.TrimSpace(parts[0])
			rangeOrElements := strings.TrimSpace(parts[1])
			
			// Check for range notation
			if strings.Contains(rangeOrElements, "..") {
				elements := lm.expandRange(rangeOrElements)
				lm.Sets[setName] = elements
			} else {
				elements := strings.Fields(rangeOrElements)
				lm.Sets[setName] = elements
			}
		}
	}
	
	return nil
}

// expandRange expands a range notation like "1..5" to ["1", "2", "3", "4", "5"]
func (lm *LingoModel) expandRange(rangeStr string) []string {
	parts := strings.Split(rangeStr, "..")
	if len(parts) != 2 {
		return []string{}
	}
	
	start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	
	if err1 != nil || err2 != nil || start > end {
		return []string{}
	}
	
	elements := make([]string, 0, end-start+1)
	for i := start; i <= end; i++ {
		elements = append(elements, strconv.Itoa(i))
	}
	
	return elements
}

// parseDataLine parses a data definition line
func (lm *LingoModel) parseDataLine(line string) error {
	// Example: COST = 10 20 30;
	line = strings.TrimSuffix(line, ";")
	
	if strings.Contains(line, "=") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			dataName := strings.TrimSpace(parts[0])
			valuesStr := strings.Fields(strings.TrimSpace(parts[1]))
			
			values := make([]float64, 0)
			for _, v := range valuesStr {
				if val, err := strconv.ParseFloat(v, 64); err == nil {
					values = append(values, val)
				}
			}
			
			lm.Data[dataName] = values
		}
	}
	
	return nil
}

// parseModelLine parses a model constraint or objective line
func (lm *LingoModel) parseModelLine(line string) error {
	line = strings.TrimSuffix(line, ";")
	line = strings.TrimSpace(line)
	
	if line == "" {
		return nil
	}
	
	// Check for objective function (min= or max=, with or without spaces)
	upper := strings.ToUpper(line)
	// Remove spaces around = for easier matching
	upperNoSpaces := strings.ReplaceAll(upper, " ", "")
	if strings.HasPrefix(upperNoSpaces, "MIN=") || strings.HasPrefix(upperNoSpaces, "MAX=") {
		lm.Objective = line
		return nil
	}
	
	// Check if it's a data assignment (variable = expression, but not starting with @FOR or containing comparison operators)
	if strings.Contains(line, "=") && !strings.HasPrefix(upper, "@FOR") {
		// Check if it contains comparison operators that would make it a constraint
		if strings.Contains(line, "<=") || strings.Contains(line, ">=") || 
		   strings.Contains(line, "<") || strings.Contains(line, ">") {
			// It's a constraint
			lm.Constraints = append(lm.Constraints, line)
		} else {
			// Could be a data assignment or objective definition
			// If it contains @ functions, treat as constraint/objective
			if strings.Contains(line, "@SUM") || strings.Contains(line, "@FOR") {
				lm.Constraints = append(lm.Constraints, line)
			} else {
				// Try to parse as data
				if err := lm.parseDataLine(line + ";"); err == nil {
					return nil
				}
				// If that fails, treat as constraint
				lm.Constraints = append(lm.Constraints, line)
			}
		}
	} else {
		// No equals sign, or starts with @FOR - it's a constraint
		lm.Constraints = append(lm.Constraints, line)
	}
	
	return nil
}

// expandToLPModel expands @ functions and converts to LP model
func (lm *LingoModel) expandToLPModel() (*LPModel, error) {
	model := NewLPModel()

	// Expand objective
	if lm.Objective != "" {
		expanded, err := lm.expandExpression(lm.Objective)
		if err != nil {
			return nil, err
		}
		
		// Extract objective type and expression
		upper := strings.ToUpper(expanded)
		if strings.HasPrefix(upper, "MIN") {
			model.ObjectiveType = "Minimize"
			parts := strings.SplitN(expanded, "=", 2)
			if len(parts) == 2 {
				model.Objective = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(upper, "MAX") {
			model.ObjectiveType = "Maximize"
			parts := strings.SplitN(expanded, "=", 2)
			if len(parts) == 2 {
				model.Objective = strings.TrimSpace(parts[1])
			}
		}
	}

	// Expand constraints
	for _, constraint := range lm.Constraints {
		// Handle @BIN and @GIN declarations
		if strings.Contains(strings.ToUpper(constraint), "@BIN") {
			lm.processBinaryDeclaration(constraint, model)
			continue
		}
		if strings.Contains(strings.ToUpper(constraint), "@GIN") {
			lm.processIntegerDeclaration(constraint, model)
			continue
		}
		
		// Handle @FOR which generates multiple constraints
		if strings.Contains(strings.ToUpper(constraint), "@FOR") {
			constraints := lm.expandFor(constraint)
			for _, c := range constraints {
				// Expand any remaining @ functions in each generated constraint
				expanded, err := lm.expandExpression(c)
				if err != nil {
					insyra.LogWarning("lpgen", "expandToLPModel", "Failed to expand constraint: %s", err.Error())
					continue
				}
				model.AddConstraint(expanded)
			}
		} else {
			// Expand @ functions in single constraint
			expanded, err := lm.expandExpression(constraint)
			if err != nil {
				insyra.LogWarning("lpgen", "expandToLPModel", "Failed to expand constraint: %s", err.Error())
				continue
			}
			model.AddConstraint(expanded)
		}
	}

	return model, nil
}

// processBinaryDeclaration processes @BIN declarations
func (lm *LingoModel) processBinaryDeclaration(expr string, model *LPModel) {
	// Pattern: @BIN(var) or @BIN(var1, var2, ...)
	binPattern := regexp.MustCompile(`@BIN\s*\(([^)]+)\)`)
	matches := binPattern.FindStringSubmatch(expr)
	
	if matches != nil {
		vars := strings.Split(matches[1], ",")
		for _, v := range vars {
			varName := strings.TrimSpace(v)
			if varName != "" {
				model.AddBinaryVar(varName)
			}
		}
	}
}

// processIntegerDeclaration processes @GIN declarations
func (lm *LingoModel) processIntegerDeclaration(expr string, model *LPModel) {
	// Pattern: @GIN(var) or @GIN(var1, var2, ...)
	ginPattern := regexp.MustCompile(`@GIN\s*\(([^)]+)\)`)
	matches := ginPattern.FindStringSubmatch(expr)
	
	if matches != nil {
		vars := strings.Split(matches[1], ",")
		for _, v := range vars {
			varName := strings.TrimSpace(v)
			if varName != "" {
				model.AddIntegerVar(varName)
			}
		}
	}
}

// expandExpression expands @ functions in an expression
func (lm *LingoModel) expandExpression(expr string) (string, error) {
	// Expand @SUM
	expr = lm.expandSum(expr)
	
	// Expand @BIN and @GIN
	expr = lm.expandBinGin(expr)
	
	return expr, nil
}

// expandSum expands @SUM(set: expression) into sum of terms
// Handles nested parentheses properly
func (lm *LingoModel) expandSum(expr string) string {
	maxIterations := 10 // Prevent infinite loops
	
	for i := 0; i < maxIterations; i++ {
		// Find the position of @SUM
		sumIdx := strings.Index(expr, "@SUM")
		if sumIdx == -1 {
			break
		}
		
		// Find the opening parenthesis after @SUM
		openIdx := strings.Index(expr[sumIdx:], "(")
		if openIdx == -1 {
			break
		}
		openIdx += sumIdx
		
		// Find the matching closing parenthesis (handle nested parens)
		closeIdx := lm.findMatchingParen(expr, openIdx)
		if closeIdx == -1 {
			break
		}
		
		// Extract the content inside @SUM(...)
		fullMatch := expr[sumIdx : closeIdx+1]
		content := expr[openIdx+1 : closeIdx]
		
		// Parse the content: setName(index): expression or setName: expression
		colonIdx := strings.Index(content, ":")
		if colonIdx == -1 {
			// Invalid @SUM, skip it
			expr = strings.Replace(expr, fullMatch, "(INVALID_SUM)", 1)
			continue
		}
		
		setDef := strings.TrimSpace(content[:colonIdx])
		innerExpr := strings.TrimSpace(content[colonIdx+1:])
		
		// Parse setDef to extract setName and indexVar
		var setName, indexVar string
		if strings.Contains(setDef, "(") {
			// Format: setName(indexVar)
			parenIdx := strings.Index(setDef, "(")
			setName = strings.TrimSpace(setDef[:parenIdx])
			indexVar = strings.TrimSpace(setDef[parenIdx+1 : len(setDef)-1])
		} else {
			// Format: setName
			setName = setDef
		}
		
		// Get the set elements
		elements, exists := lm.Sets[setName]
		if !exists {
			insyra.LogWarning("lpgen", "expandSum", "Set %s not found", setName)
			expr = strings.Replace(expr, fullMatch, "(UNKNOWN_SET_"+setName+")", 1)
			continue
		}
		
		// Expand the sum
		terms := make([]string, 0)
		for _, elem := range elements {
			term := innerExpr
			
			// If there's an index variable, replace it
			if indexVar != "" {
				// Replace indexed variables: var(indexVar) -> var(elem)
				indexedPattern := regexp.MustCompile(`(\w+)\(` + regexp.QuoteMeta(indexVar) + `\)`)
				term = indexedPattern.ReplaceAllString(term, `${1}(`+elem+`)`)
				
				// Replace standalone index variable
				standalonePattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(indexVar) + `\b`)
				term = standalonePattern.ReplaceAllString(term, elem)
			} else {
				// Replace variable patterns
				varPattern := regexp.MustCompile(`\b(\w+)_` + regexp.QuoteMeta(setName) + `\b`)
				term = varPattern.ReplaceAllString(term, `${1}_`+elem)
				
				standalonePattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(setName) + `\b`)
				term = standalonePattern.ReplaceAllString(term, elem)
			}
			
			terms = append(terms, term)
		}
		
		expanded := "(" + strings.Join(terms, " + ") + ")"
		expr = strings.Replace(expr, fullMatch, expanded, 1)
	}
	
	return expr
}

// findMatchingParen finds the index of the closing parenthesis that matches the opening one at openIdx
func (lm *LingoModel) findMatchingParen(s string, openIdx int) int {
	if openIdx >= len(s) || s[openIdx] != '(' {
		return -1
	}
	
	depth := 1
	for i := openIdx + 1; i < len(s); i++ {
		if s[i] == '(' {
			depth++
		} else if s[i] == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	
	return -1
}

// expandFor expands @FOR(set: constraint) into multiple constraints
func (lm *LingoModel) expandFor(expr string) []string {
	constraints := make([]string, 0)
	
	// Find the position of @FOR
	forIdx := strings.Index(expr, "@FOR")
	if forIdx == -1 {
		return []string{expr}
	}
	
	// Find the opening parenthesis after @FOR
	openIdx := strings.Index(expr[forIdx:], "(")
	if openIdx == -1 {
		return []string{expr}
	}
	openIdx += forIdx
	
	// Find the matching closing parenthesis (handle nested parens)
	closeIdx := lm.findMatchingParen(expr, openIdx)
	if closeIdx == -1 {
		// No matching closing paren found - treat rest of string as the constraint
		content := expr[openIdx+1:]
		colonIdx := strings.Index(content, ":")
		if colonIdx == -1 {
			return []string{expr}
		}
		
		setDef := strings.TrimSpace(content[:colonIdx])
		innerExpr := strings.TrimSpace(content[colonIdx+1:])
		
		// Parse setDef
		var setName, indexVar string
		if strings.Contains(setDef, "(") {
			parenIdx := strings.Index(setDef, "(")
			setName = strings.TrimSpace(setDef[:parenIdx])
			endParenIdx := strings.Index(setDef[parenIdx:], ")")
			if endParenIdx != -1 {
				indexVar = strings.TrimSpace(setDef[parenIdx+1 : parenIdx+endParenIdx])
			}
		} else {
			setName = setDef
		}
		
		// Get the set elements
		elements, exists := lm.Sets[setName]
		if !exists {
			insyra.LogWarning("lpgen", "expandFor", "Set %s not found", setName)
			return []string{expr}
		}
		
		// Expand for each element
		for _, elem := range elements {
			constraint := innerExpr
			
			if indexVar != "" {
				indexedPattern := regexp.MustCompile(`(\w+)\(` + regexp.QuoteMeta(indexVar) + `\)`)
				constraint = indexedPattern.ReplaceAllString(constraint, `${1}(`+elem+`)`)
				
				standalonePattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(indexVar) + `\b`)
				constraint = standalonePattern.ReplaceAllString(constraint, elem)
			} else {
				varPattern := regexp.MustCompile(`\b(\w+)_` + regexp.QuoteMeta(setName) + `\b`)
				constraint = varPattern.ReplaceAllString(constraint, `${1}_`+elem)
				
				standalonePattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(setName) + `\b`)
				constraint = standalonePattern.ReplaceAllString(constraint, elem)
			}
			
			constraints = append(constraints, constraint)
		}
		
		return constraints
	}
	
	// Extract the content inside @FOR(...)
	content := expr[openIdx+1 : closeIdx]
	
	// Parse the content: setName(index): expression or setName: expression
	colonIdx := strings.Index(content, ":")
	if colonIdx == -1 {
		return []string{expr}
	}
	
	setDef := strings.TrimSpace(content[:colonIdx])
	innerExpr := strings.TrimSpace(content[colonIdx+1:])
	
	// Parse setDef to extract setName and indexVar
	var setName, indexVar string
	if strings.Contains(setDef, "(") {
		parenIdx := strings.Index(setDef, "(")
		setName = strings.TrimSpace(setDef[:parenIdx])
		indexVar = strings.TrimSpace(setDef[parenIdx+1 : len(setDef)-1])
	} else {
		setName = setDef
	}
	
	// Get the set elements
	elements, exists := lm.Sets[setName]
	if !exists {
		insyra.LogWarning("lpgen", "expandFor", "Set %s not found", setName)
		return []string{expr}
	}
	
	// Expand for each element
	for _, elem := range elements {
		constraint := innerExpr
		
		// If there's an index variable, replace it
		if indexVar != "" {
			indexedPattern := regexp.MustCompile(`(\w+)\(` + regexp.QuoteMeta(indexVar) + `\)`)
			constraint = indexedPattern.ReplaceAllString(constraint, `${1}(`+elem+`)`)
			
			standalonePattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(indexVar) + `\b`)
			constraint = standalonePattern.ReplaceAllString(constraint, elem)
		} else {
			varPattern := regexp.MustCompile(`\b(\w+)_` + regexp.QuoteMeta(setName) + `\b`)
			constraint = varPattern.ReplaceAllString(constraint, `${1}_`+elem)
			
			standalonePattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(setName) + `\b`)
			constraint = standalonePattern.ReplaceAllString(constraint, elem)
		}
		
		constraints = append(constraints, constraint)
	}
	
	return constraints
}

// expandBinGin handles @BIN and @GIN declarations
func (lm *LingoModel) expandBinGin(expr string) string {
	// These are typically handled separately in variable declarations
	// For now, just return the expression as-is
	return expr
}

// ParseLingoFile reads and parses a LINGO file with native syntax
func ParseLingoFile(filePath string) (*LPModel, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	return ParseLingoSyntax(string(content))
}
