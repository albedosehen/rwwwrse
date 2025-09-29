#!/bin/bash

# CI/CD Workflow Validation Script
# This script validates the entire CI/CD pipeline structure

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
GITHUB_DIR="$PROJECT_ROOT/.github"
WORKFLOWS_DIR="$GITHUB_DIR/workflows"
ACTIONS_DIR="$GITHUB_DIR/actions"
CONFIG_DIR="$GITHUB_DIR/config"

# Validation results
VALIDATION_ERRORS=0
VALIDATION_WARNINGS=0

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
    ((VALIDATION_WARNINGS++))
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    ((VALIDATION_ERRORS++))
}

# Check if required tools are available
check_dependencies() {
    log_info "Checking dependencies..."
    
    local missing_deps=()
    
    if ! command -v yq &> /dev/null; then
        missing_deps+=("yq")
    fi
    
    if ! command -v docker &> /dev/null; then
        missing_deps+=("docker")
    fi
    
    if ! command -v go &> /dev/null; then
        missing_deps+=("go")
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        log_info "Please install the missing dependencies and try again"
        exit 1
    fi
    
    log_success "All dependencies are available"
}

# Validate YAML syntax
validate_yaml_syntax() {
    local file="$1"
    local file_type="$2"
    
    log_info "Validating YAML syntax: $file"
    
    if ! yq eval '.' "$file" > /dev/null 2>&1; then
        log_error "Invalid YAML syntax in $file"
        return 1
    fi
    
    # Check for required fields based on file type
    case "$file_type" in
        "workflow")
            if ! yq eval '.name' "$file" > /dev/null 2>&1; then
                log_error "Missing 'name' field in workflow: $file"
                return 1
            fi
            
            if ! yq eval '.on' "$file" > /dev/null 2>&1; then
                log_error "Missing 'on' field in workflow: $file"
                return 1
            fi
            
            if ! yq eval '.jobs' "$file" > /dev/null 2>&1; then
                log_error "Missing 'jobs' field in workflow: $file"
                return 1
            fi
            ;;
        "action")
            if ! yq eval '.name' "$file" > /dev/null 2>&1; then
                log_error "Missing 'name' field in action: $file"
                return 1
            fi
            
            if ! yq eval '.description' "$file" > /dev/null 2>&1; then
                log_error "Missing 'description' field in action: $file"
                return 1
            fi
            
            if ! yq eval '.runs' "$file" > /dev/null 2>&1; then
                log_error "Missing 'runs' field in action: $file"
                return 1
            fi
            ;;
    esac
    
    log_success "YAML syntax validation passed: $file"
    return 0
}

# Validate workflow files
validate_workflows() {
    log_info "Validating workflow files..."
    
    if [ ! -d "$WORKFLOWS_DIR" ]; then
        log_error "Workflows directory not found: $WORKFLOWS_DIR"
        return 1
    fi
    
    local workflow_count=0
    while IFS= read -r -d '' file; do
        if [[ "$file" == *.yml ]] || [[ "$file" == *.yaml ]]; then
            validate_yaml_syntax "$file" "workflow"
            ((workflow_count++))
        fi
    done < <(find "$WORKFLOWS_DIR" -type f \( -name "*.yml" -o -name "*.yaml" \) -print0)
    
    if [ $workflow_count -eq 0 ]; then
        log_warning "No workflow files found in $WORKFLOWS_DIR"
    else
        log_success "Validated $workflow_count workflow files"
    fi
}

# Validate action files
validate_actions() {
    log_info "Validating action files..."
    
    if [ ! -d "$ACTIONS_DIR" ]; then
        log_warning "Actions directory not found: $ACTIONS_DIR"
        return 0
    fi
    
    local action_count=0
    while IFS= read -r -d '' file; do
        if [[ "$(basename "$file")" == "action.yml" ]] || [[ "$(basename "$file")" == "action.yaml" ]]; then
            validate_yaml_syntax "$file" "action"
            ((action_count++))
        fi
    done < <(find "$ACTIONS_DIR" -type f \( -name "action.yml" -o -name "action.yaml" \) -print0)
    
    if [ $action_count -eq 0 ]; then
        log_warning "No action files found in $ACTIONS_DIR"
    else
        log_success "Validated $action_count action files"
    fi
}

# Validate configuration files
validate_configs() {
    log_info "Validating configuration files..."
    
    if [ ! -d "$CONFIG_DIR" ]; then
        log_warning "Config directory not found: $CONFIG_DIR"
        return 0
    fi
    
    local config_files=(
        "$CONFIG_DIR/environments.yml"
        "$CONFIG_DIR/notifications.yml"
    )
    
    for config_file in "${config_files[@]}"; do
        if [ -f "$config_file" ]; then
            validate_yaml_syntax "$config_file" "config"
        else
            log_warning "Configuration file not found: $config_file"
        fi
    done
}

# Validate workflow dependencies
validate_workflow_dependencies() {
    log_info "Validating workflow dependencies..."
    
    # Check for reusable workflow references
    local reusable_workflows=()
    while IFS= read -r -d '' file; do
        if [[ "$file" == *"_reusable"* ]]; then
            local rel_path
            rel_path=$(realpath --relative-to="$WORKFLOWS_DIR" "$file")
            reusable_workflows+=("$rel_path")
        fi
    done < <(find "$WORKFLOWS_DIR" -type f \( -name "*.yml" -o -name "*.yaml" \) -print0)
    
    # Check if referenced workflows exist
    while IFS= read -r -d '' file; do
        if [[ "$file" == *.yml ]] || [[ "$file" == *.yaml ]]; then
            # Extract uses references
            local uses_refs
            uses_refs=$(yq eval '.jobs[].uses // empty' "$file" 2>/dev/null | grep "^\./" || true)
            
            while IFS= read -r ref; do
                if [ -n "$ref" ]; then
                    local referenced_file="${ref#./}"
                    local found=false
                    
                    for reusable in "${reusable_workflows[@]}"; do
                        if [ "$reusable" = "$referenced_file" ]; then
                            found=true
                            break
                        fi
                    done
                    
                    if [ "$found" = false ]; then
                        log_error "Referenced workflow not found: $ref in $file"
                    fi
                fi
            done <<< "$uses_refs"
        fi
    done < <(find "$WORKFLOWS_DIR" -type f \( -name "*.yml" -o -name "*.yaml" \) -print0)
}

# Validate secrets usage
validate_secrets() {
    log_info "Validating secrets usage..."
    
    local secret_refs=()
    
    # Extract secret references from all workflow files
    while IFS= read -r -d '' file; do
        if [[ "$file" == *.yml ]] || [[ "$file" == *.yaml ]]; then
            local file_secrets
            file_secrets=$(grep -o 'secrets\.[A-Z_][A-Z0-9_]*' "$file" || true)
            
            while IFS= read -r secret; do
                if [ -n "$secret" ]; then
                    secret_refs+=("$secret")
                fi
            done <<< "$file_secrets"
        fi
    done < <(find "$WORKFLOWS_DIR" -type f \( -name "*.yml" -o -name "*.yaml" \) -print0)
    
    # Remove duplicates and sort
    local unique_secrets
    mapfile -t unique_secrets < <(printf '%s\n' "${secret_refs[@]}" | sort -u)
    
    if [ ${#unique_secrets[@]} -gt 0 ]; then
        log_info "Found ${#unique_secrets[@]} unique secret references:"
        for secret in "${unique_secrets[@]}"; do
            log_info "  - $secret"
        done
        log_warning "Ensure all secrets are properly configured in your repository settings"
    else
        log_info "No secret references found"
    fi
}

# Run Go-based validation
run_go_validation() {
    log_info "Running Go-based validation..."
    
    local go_validator="$SCRIPT_DIR/validate-workflows.go"
    
    if [ ! -f "$go_validator" ]; then
        log_warning "Go validator not found: $go_validator"
        return 0
    fi
    
    # Change to the project root to run validation
    cd "$PROJECT_ROOT"
    
    if go run "$go_validator"; then
        log_success "Go validation passed"
    else
        log_error "Go validation failed"
        return 1
    fi
}

# Validate Docker Compose files
validate_docker_compose() {
    log_info "Validating Docker Compose files..."
    
    local compose_files=(
        "$PROJECT_ROOT/docker-compose.deploy.yml"
        "$PROJECT_ROOT/docker-compose.yml"
    )
    
    for compose_file in "${compose_files[@]}"; do
        if [ -f "$compose_file" ]; then
            log_info "Validating Docker Compose file: $compose_file"
            
            if docker compose -f "$compose_file" config > /dev/null 2>&1; then
                log_success "Docker Compose validation passed: $compose_file"
            else
                log_error "Docker Compose validation failed: $compose_file"
            fi
        else
            log_warning "Docker Compose file not found: $compose_file"
        fi
    done
}

# Validate Dockerfile
validate_dockerfile() {
    log_info "Validating Dockerfile..."
    
    local dockerfile="$PROJECT_ROOT/Dockerfile"
    
    if [ ! -f "$dockerfile" ]; then
        log_warning "Dockerfile not found: $dockerfile"
        return 0
    fi
    
    # Basic Dockerfile validation
    if grep -q "^FROM " "$dockerfile"; then
        log_success "Dockerfile has FROM instruction"
    else
        log_error "Dockerfile missing FROM instruction"
    fi
    
    if grep -q "^WORKDIR " "$dockerfile"; then
        log_success "Dockerfile has WORKDIR instruction"
    else
        log_warning "Dockerfile missing WORKDIR instruction"
    fi
    
    if grep -q "^EXPOSE " "$dockerfile"; then
        log_success "Dockerfile has EXPOSE instruction"
    else
        log_warning "Dockerfile missing EXPOSE instruction"
    fi
}

# Generate validation report
generate_report() {
    log_info "Generating validation report..."
    
    local report_file="$SCRIPT_DIR/validation-report.txt"
    local timestamp
    timestamp=$(date -u +"%Y-%m-%d %H:%M:%S UTC")
    
    cat > "$report_file" << EOF
CI/CD Workflow Validation Report
================================

Generated: $timestamp
Project: rwwwrse CI/CD Pipeline

Summary:
--------
Total Errors: $VALIDATION_ERRORS
Total Warnings: $VALIDATION_WARNINGS
Overall Status: $([ $VALIDATION_ERRORS -eq 0 ] && echo "PASSED" || echo "FAILED")

Validation Scope:
-----------------
- Workflow YAML syntax and structure
- Action YAML syntax and structure
- Configuration file validation
- Workflow dependency validation
- Secret reference validation
- Docker Compose file validation
- Dockerfile validation

Recommendations:
----------------
EOF

    if [ $VALIDATION_ERRORS -eq 0 ]; then
        cat >> "$report_file" << EOF
✅ All critical validations passed
✅ CI/CD pipeline structure is valid
✅ Ready for deployment testing
EOF
    else
        cat >> "$report_file" << EOF
❌ Critical validation errors found
❌ Fix errors before proceeding with deployment
❌ Review error messages above for details
EOF
    fi

    if [ $VALIDATION_WARNINGS -gt 0 ]; then
        cat >> "$report_file" << EOF

⚠️  Warnings found - review for best practices
⚠️  Consider addressing warnings for optimal pipeline performance
EOF
    fi

    cat >> "$report_file" << EOF

Next Steps:
-----------
1. Address any validation errors
2. Review and resolve warnings
3. Run integration tests
4. Perform security validation
5. Execute performance tests

For detailed logs, review the console output above.
EOF

    log_success "Validation report generated: $report_file"
}

# Main validation function
main() {
    echo "🔍 CI/CD Workflow Validation"
    echo "============================"
    echo ""
    
    # Check dependencies
    check_dependencies
    echo ""
    
    # Run validations
    validate_workflows
    echo ""
    
    validate_actions
    echo ""
    
    validate_configs
    echo ""
    
    validate_workflow_dependencies
    echo ""
    
    validate_secrets
    echo ""
    
    validate_docker_compose
    echo ""
    
    validate_dockerfile
    echo ""
    
    # Run Go-based validation if available
    run_go_validation
    echo ""
    
    # Generate report
    generate_report
    echo ""
    
    # Final summary
    if [ $VALIDATION_ERRORS -eq 0 ]; then
        log_success "🎉 All validations completed successfully!"
        log_info "Errors: $VALIDATION_ERRORS, Warnings: $VALIDATION_WARNINGS"
        echo ""
        log_info "✅ CI/CD pipeline is ready for integration testing"
    else
        log_error "❌ Validation failed with $VALIDATION_ERRORS errors and $VALIDATION_WARNINGS warnings"
        echo ""
        log_error "🚨 Please fix the errors before proceeding"
        exit 1
    fi
}

# Run main function
main "$@"