#!/bin/bash
# PostgREST Template E2E Validation Script
# This script validates that the template is correctly structured

set -e

TEMPLATE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=== PostgREST Template E2E Validation ==="
echo ""
echo "Template Directory: $TEMPLATE_DIR"
echo ""

# Test 1: Check file structure
echo "Test 1: Verifying directory structure..."
required_files=(
    "template.yaml"
    "content/source/catalog-info.yaml"
    "content/source/Dockerfile"
    "content/source/postgrestrc.conf"
    "content/source/README.md"
    "content/source/.gitignore"
    "content/gitops/helios-app.yaml"
    "content/gitops/argocd-app.yaml"
    "content/gitops/pipeline.yaml"
    "content/gitops/triggers.yaml"
    "content/gitops/kustomization.yaml"
)

all_exist=true
for file in "${required_files[@]}"; do
    if [ -f "$TEMPLATE_DIR/$file" ]; then
        echo "  ✓ $file"
    else
        echo "  ✗ Missing: $file"
        all_exist=false
    fi
done

if [ "$all_exist" = false ]; then
    echo ""
    echo "✗ Some required files are missing"
    exit 1
fi

# Test 2: Validate YAML syntax of source templates
echo ""
echo "Test 2: Validating YAML syntax..."
python3 << 'PYTHON_EOF'
import yaml
import os
import sys

# Use the directory from which this script is run
template_dir = os.getcwd()
if not os.path.exists(os.path.join(template_dir, 'template.yaml')):
    # If not in template dir, try to find it
    script_dir = os.path.dirname(os.path.realpath(__file__))
    template_dir = script_dir

yaml_files = [
    'template.yaml',
    'content/source/catalog-info.yaml',
    'content/gitops/helios-app.yaml',
    'content/gitops/argocd-app.yaml',
    'content/gitops/pipeline.yaml',
    'content/gitops/triggers.yaml',
    'content/gitops/kustomization.yaml',
]

errors = []
for yaml_file in sorted(yaml_files):
    full_path = os.path.join(template_dir, yaml_file)
    try:
        with open(full_path, 'r') as f:
            list(yaml.safe_load_all(f))
        print(f"  ✓ {yaml_file}")
    except Exception as e:
        errors.append(f"  ✗ {yaml_file}: {e}")

for error in errors:
    print(error)

if errors:
    sys.exit(1)
PYTHON_EOF

# Test 3: Verify template contains required Backstage scaffold fields
echo ""
echo "Test 3: Verifying Backstage scaffolder structure..."

check_field() {
    local file=$1
    local field=$2
    local description=$3
    
    if grep -q "$field" "$TEMPLATE_DIR/$file" 2>/dev/null; then
        echo "  ✓ Found $description"
        return 0
    else
        echo "  ✗ Missing $description in $file"
        return 1
    fi
}

# Check main template
check_field "template.yaml" "kind: Template" "Template kind"
check_field "template.yaml" "parameters:" "Parameters section"
check_field "template.yaml" "steps:" "Steps section"
check_field "template.yaml" "publish:gitea" "Gitea publish action"
check_field "template.yaml" "kubernetes:apply" "Kubernetes apply action"

# Check HeliosApp includes database trait
echo ""
echo "Test 4: Verifying HeliosApp CRD structure..."
check_field "content/gitops/helios-app.yaml" "kind: HeliosApp" "HeliosApp kind"
check_field "content/gitops/helios-app.yaml" "type: database" "Database trait"
check_field "content/gitops/helios-app.yaml" "dbType: postgres" "Postgres configuration"

# Check Tekton resources
echo ""
echo "Test 5: Verifying Tekton configuration..."
check_field "content/gitops/pipeline.yaml" "kind: PipelineRun" "Tekton PipelineRun"
check_field "content/gitops/triggers.yaml" "kind: EventListener" "Tekton EventListener"
check_field "content/gitops/triggers.yaml" "kind: TriggerBinding" "Tekton TriggerBinding"
check_field "content/gitops/triggers.yaml" "kind: TriggerTemplate" "Tekton TriggerTemplate"

# Check that PostgREST configuration is present
echo ""
echo "Test 6: Verifying PostgREST-specific configuration..."
check_field "content/source/postgrestrc.conf" "db-schema" "PostgREST schema config"
check_field "content/source/postgrestrc.conf" "db-anon-role" "PostgREST anonymous role"
check_field "content/source/postgrestrc.conf" "server-port" "PostgREST server port config"
check_field "content/source/Dockerfile" "postgrest/postgrest" "Official PostgREST image"

# Check that PGRST_DB_URI is referenced
echo ""
echo "Test 7: Verifying PGRST_DB_URI integration..."
check_field "content/source/README.md" "PGRST_DB_URI" "PGRST_DB_URI documentation"
check_field "content/gitops/helios-app.yaml" "database" "Database trait for credential injection"

# Summary
echo ""
echo "=== Validation Results ==="
echo "✓ Template structure is valid"
echo "✓ YAML syntax is correct"
echo "✓ HeliosApp CRD correctly configured"
echo "✓ Tekton CI/CD pipeline defined"
echo "✓ PostgREST configuration present"
echo "✓ PGRST_DB_URI integration configured"
echo ""
echo "✓ PostgREST template is ready for deployment!"
echo ""
echo "Next steps:"
echo "1. Deploy to Helios Platform cluster"
echo "2. Register template in Backstage"
echo "3. Users can scaffold PostgREST services via UI"
echo ""

