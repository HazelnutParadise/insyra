package lpgen

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/HazelnutParadise/insyra"
)

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

	// Split into sections
	lines := strings.Split(source, "\n")
	
	currentSection := ""
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
			currentSection = "MODEL"
			continue
		} else if strings.HasPrefix(upper, "END") {
			break
		}

		// Parse based on current section
		switch currentSection {
		case "SETS":
			if err := lm.parseSetLine(line); err != nil {
				insyra.LogWarning("lpgen", "parseModel", "Failed to parse set line: %s", err.Error())
			}
		case "DATA":
			if err := lm.parseDataLine(line); err != nil {
				insyra.LogWarning("lpgen", "parseModel", "Failed to parse data line: %s", err.Error())
			}
		case "MODEL":
			if err := lm.parseModelLine(line); err != nil {
				insyra.LogWarning("lpgen", "parseModel", "Failed to parse model line: %s", err.Error())
			}
		}
	}

	return nil
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
	// Example: PRODUCTS / P1 P2 P3 /;
	// Or: PRODUCTS;
	line = strings.TrimSuffix(line, ";")
	
	// Check for inline set definition
	if strings.Contains(line, "/") {
		parts := strings.Split(line, "/")
		if len(parts) >= 2 {
			setName := strings.TrimSpace(parts[0])
			elements := strings.Fields(strings.TrimSpace(parts[1]))
			lm.Sets[setName] = elements
		}
	}
	
	return nil
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
	
	// Check for objective function
	upper := strings.ToUpper(line)
	if strings.HasPrefix(upper, "MIN") || strings.HasPrefix(upper, "MAX") {
		lm.Objective = line
	} else {
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
		
		expanded, err := lm.expandExpression(constraint)
		if err != nil {
			insyra.LogWarning("lpgen", "expandToLPModel", "Failed to expand constraint: %s", err.Error())
			continue
		}
		
		// Handle @FOR which generates multiple constraints
		if strings.Contains(strings.ToUpper(constraint), "@FOR") {
			constraints := lm.expandFor(constraint)
			for _, c := range constraints {
				model.AddConstraint(c)
			}
		} else {
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
func (lm *LingoModel) expandSum(expr string) string {
	// Pattern: @SUM(setName: expression)
	sumPattern := regexp.MustCompile(`@SUM\s*\(\s*(\w+)\s*:\s*([^)]+)\)`)
	
	for {
		matches := sumPattern.FindStringSubmatch(expr)
		if matches == nil {
			break
		}
		
		fullMatch := matches[0]
		setName := matches[1]
		innerExpr := matches[2]
		
		// Get the set elements
		elements, exists := lm.Sets[setName]
		if !exists {
			// If set not found, try to infer from variables
			insyra.LogWarning("lpgen", "expandSum", "Set %s not found", setName)
			break
		}
		
		// Expand the sum
		terms := make([]string, 0)
		for _, elem := range elements {
			// Replace set index with actual element - handle both direct replacement
			// and patterns like X_SETNAME becoming X_elem
			term := innerExpr
			
			// Replace variable patterns: VAR_SETNAME -> VAR_elem
			varPattern := regexp.MustCompile(`\b(\w+)_` + setName + `\b`)
			term = varPattern.ReplaceAllString(term, `${1}_`+elem)
			
			// Also replace standalone SETNAME with elem
			standalonePattern := regexp.MustCompile(`\b` + setName + `\b`)
			term = standalonePattern.ReplaceAllString(term, elem)
			
			terms = append(terms, term)
		}
		
		expanded := strings.Join(terms, " + ")
		expr = strings.ReplaceAll(expr, fullMatch, expanded)
	}
	
	return expr
}

// expandFor expands @FOR(set: constraint) into multiple constraints
func (lm *LingoModel) expandFor(expr string) []string {
	constraints := make([]string, 0)
	
	// Pattern: @FOR(setName: constraint)
	forPattern := regexp.MustCompile(`@FOR\s*\(\s*(\w+)\s*:\s*([^)]+)\)`)
	
	matches := forPattern.FindStringSubmatch(expr)
	if matches == nil {
		return []string{expr}
	}
	
	setName := matches[1]
	innerExpr := matches[2]
	
	// Get the set elements
	elements, exists := lm.Sets[setName]
	if !exists {
		insyra.LogWarning("lpgen", "expandFor", "Set %s not found", setName)
		return []string{expr}
	}
	
	// Expand for each element
	for _, elem := range elements {
		constraint := innerExpr
		
		// Replace variable patterns: VAR_SETNAME -> VAR_elem
		varPattern := regexp.MustCompile(`\b(\w+)_` + setName + `\b`)
		constraint = varPattern.ReplaceAllString(constraint, `${1}_`+elem)
		
		// Also replace standalone SETNAME with elem
		standalonePattern := regexp.MustCompile(`\b` + setName + `\b`)
		constraint = standalonePattern.ReplaceAllString(constraint, elem)
		
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
