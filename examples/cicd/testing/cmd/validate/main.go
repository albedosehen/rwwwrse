package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	workflowsDir = ".github/workflows"
	actionsDir   = ".github/actions"
	configDir    = ".github/config"
)

// WorkflowValidator validates GitHub Actions workflow files
type WorkflowValidator struct {
	errors   []ValidationError
	warnings []ValidationWarning
}

// ValidationError represents a validation error
type ValidationError struct {
	File    string `json:"file"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	File    string `json:"file"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

// WorkflowFile represents a GitHub Actions workflow
type WorkflowFile struct {
	Name        string                 `yaml:"name"`
	On          interface{}            `yaml:"on"`
	Env         map[string]string      `yaml:"env,omitempty"`
	Concurrency interface{}            `yaml:"concurrency,omitempty"`
	Permissions interface{}            `yaml:"permissions,omitempty"`
	Jobs        map[string]interface{} `yaml:"jobs"`
}

// ActionFile represents a GitHub Actions composite action
type ActionFile struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Author      string                 `yaml:"author,omitempty"`
	Inputs      map[string]interface{} `yaml:"inputs,omitempty"`
	Outputs     map[string]interface{} `yaml:"outputs,omitempty"`
	Runs        interface{}            `yaml:"runs"`
	Branding    interface{}            `yaml:"branding,omitempty"`
}

// NewWorkflowValidator creates a new workflow validator
func NewWorkflowValidator() *WorkflowValidator {
	return &WorkflowValidator{
		errors:   make([]ValidationError, 0),
		warnings: make([]ValidationWarning, 0),
	}
}

// AddError adds a validation error
func (v *WorkflowValidator) AddError(file, message, errorType string) {
	v.errors = append(v.errors, ValidationError{
		File:    file,
		Message: message,
		Type:    errorType,
	})
}

// AddWarning adds a validation warning
func (v *WorkflowValidator) AddWarning(file, message, warningType string) {
	v.warnings = append(v.warnings, ValidationWarning{
		File:    file,
		Message: message,
		Type:    warningType,
	})
}

// ValidateWorkflowSyntax validates YAML syntax and basic structure
func (v *WorkflowValidator) ValidateWorkflowSyntax(ctx context.Context, filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		v.AddError(filePath, fmt.Sprintf("Failed to read file: %v", err), "file_read_error")
		return err
	}

	var workflow WorkflowFile
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		v.AddError(filePath, fmt.Sprintf("Invalid YAML syntax: %v", err), "yaml_syntax_error")
		return err
	}

	// Validate required fields
	if workflow.Name == "" {
		v.AddError(filePath, "Workflow name is required", "missing_required_field")
	}

	if workflow.On == nil {
		v.AddError(filePath, "Workflow trigger (on) is required", "missing_required_field")
	}

	if len(workflow.Jobs) == 0 {
		v.AddError(filePath, "At least one job is required", "missing_required_field")
	}

	// Validate job structure
	for jobName, jobData := range workflow.Jobs {
		if err := v.validateJob(ctx, filePath, jobName, jobData); err != nil {
			v.AddError(filePath, fmt.Sprintf("Invalid job '%s': %v", jobName, err), "job_validation_error")
		}
	}

	return nil
}

// validateJob validates individual job structure
func (v *WorkflowValidator) validateJob(ctx context.Context, filePath, jobName string, jobData interface{}) error {
	jobMap, ok := jobData.(map[string]interface{})
	if !ok {
		return fmt.Errorf("job must be an object")
	}

	// Check for required fields based on job type
	if uses, hasUses := jobMap["uses"]; hasUses {
		// Reusable workflow job
		usesStr, ok := uses.(string)
		if !ok {
			return fmt.Errorf("'uses' must be a string")
		}
		if err := v.validateReusableWorkflowReference(ctx, filePath, usesStr); err != nil {
			v.AddWarning(filePath, fmt.Sprintf("Reusable workflow reference issue: %v", err), "reusable_workflow_warning")
		}
	} else {
		// Regular job
		if _, hasRunsOn := jobMap["runs-on"]; !hasRunsOn {
			return fmt.Errorf("'runs-on' is required for regular jobs")
		}

		if _, hasSteps := jobMap["steps"]; !hasSteps {
			return fmt.Errorf("'steps' is required for regular jobs")
		}
	}

	return nil
}

// validateReusableWorkflowReference validates references to reusable workflows
func (v *WorkflowValidator) validateReusableWorkflowReference(ctx context.Context, filePath, uses string) error {
	if strings.HasPrefix(uses, "./") {
		// Local reusable workflow
		referencedFile := strings.TrimPrefix(uses, "./")
		fullPath := filepath.Join(filepath.Dir(filePath), referencedFile)
		
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return fmt.Errorf("referenced workflow file does not exist: %s", fullPath)
		}
	}
	return nil
}

// ValidateActionSyntax validates composite action files
func (v *WorkflowValidator) ValidateActionSyntax(ctx context.Context, filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		v.AddError(filePath, fmt.Sprintf("Failed to read file: %v", err), "file_read_error")
		return err
	}

	var action ActionFile
	if err := yaml.Unmarshal(content, &action); err != nil {
		v.AddError(filePath, fmt.Sprintf("Invalid YAML syntax: %v", err), "yaml_syntax_error")
		return err
	}

	// Validate required fields
	if action.Name == "" {
		v.AddError(filePath, "Action name is required", "missing_required_field")
	}

	if action.Description == "" {
		v.AddError(filePath, "Action description is required", "missing_required_field")
	}

	if action.Runs == nil {
		v.AddError(filePath, "Action runs configuration is required", "missing_required_field")
	}

	return nil
}

// GenerateReport generates a validation report
func (v *WorkflowValidator) GenerateReport() ValidationReport {
	return ValidationReport{
		Timestamp: time.Now(),
		Summary: ValidationSummary{
			TotalErrors:   len(v.errors),
			TotalWarnings: len(v.warnings),
			Status:        v.getOverallStatus(),
		},
		Errors:   v.errors,
		Warnings: v.warnings,
	}
}

// getOverallStatus determines the overall validation status
func (v *WorkflowValidator) getOverallStatus() string {
	if len(v.errors) > 0 {
		return "FAILED"
	}
	if len(v.warnings) > 0 {
		return "WARNING"
	}
	return "PASSED"
}

// ValidationReport represents the complete validation report
type ValidationReport struct {
	Timestamp time.Time         `json:"timestamp"`
	Summary   ValidationSummary `json:"summary"`
	Errors    []ValidationError `json:"errors"`
	Warnings  []ValidationWarning `json:"warnings"`
}

// ValidationSummary provides a summary of validation results
type ValidationSummary struct {
	TotalErrors   int    `json:"total_errors"`
	TotalWarnings int    `json:"total_warnings"`
	Status        string `json:"status"`
}

func main() {
	ctx := context.Background()
	validator := NewWorkflowValidator()

	fmt.Println("🔍 Starting CI/CD Workflow Validation...")
	fmt.Println("=====================================")

	// Validate workflow files
	workflowsPath := workflowsDir
	if _, err := os.Stat(workflowsPath); !os.IsNotExist(err) {
		fmt.Printf("📁 Validating workflows in %s...\n", workflowsPath)
		
		err := filepath.WalkDir(workflowsPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			if strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml") {
				fmt.Printf("  📄 Validating %s\n", path)
				if err := validator.ValidateWorkflowSyntax(ctx, path); err != nil {
					fmt.Printf("    ❌ Syntax validation failed: %v\n", err)
				} else {
					fmt.Printf("    ✅ Syntax validation passed\n")
				}
			}

			return nil
		})

		if err != nil {
			log.Printf("Error walking workflows directory: %v", err)
		}
	}

	// Validate action files
	actionsPath := actionsDir
	if _, err := os.Stat(actionsPath); !os.IsNotExist(err) {
		fmt.Printf("📁 Validating actions in %s...\n", actionsPath)
		
		err := filepath.WalkDir(actionsPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			if strings.HasSuffix(path, "action.yml") || strings.HasSuffix(path, "action.yaml") {
				fmt.Printf("  📄 Validating %s\n", path)
				if err := validator.ValidateActionSyntax(ctx, path); err != nil {
					fmt.Printf("    ❌ Action validation failed: %v\n", err)
				} else {
					fmt.Printf("    ✅ Action validation passed\n")
				}
			}

			return nil
		})

		if err != nil {
			log.Printf("Error walking actions directory: %v", err)
		}
	}

	// Generate and display report
	fmt.Println("\n📊 Validation Report")
	fmt.Println("====================")
	
	report := validator.GenerateReport()
	fmt.Printf("Status: %s\n", report.Summary.Status)
	fmt.Printf("Errors: %d\n", report.Summary.TotalErrors)
	fmt.Printf("Warnings: %d\n", report.Summary.TotalWarnings)
	
	if len(report.Errors) > 0 {
		fmt.Println("\n❌ Errors:")
		for _, err := range report.Errors {
			fmt.Printf("  - %s: %s (%s)\n", err.File, err.Message, err.Type)
		}
	}
	
	if len(report.Warnings) > 0 {
		fmt.Println("\n⚠️ Warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("  - %s: %s (%s)\n", warning.File, warning.Message, warning.Type)
		}
	}
	
	fmt.Println("\n✅ Workflow validation completed!")
	
	// Exit with appropriate code
	if report.Summary.Status == "FAILED" {
		os.Exit(1)
	}
}