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

# Ensure PyYAML is available
if ! python3 -c "import yaml" 2>/dev/null; then
    echo "  Installing PyYAML..."
    python3 -m pip install -q pyyaml || { echo "  ✗ Failed to install PyYAML"; exit 1; }
fi

TEMPLATE_DIR="$TEMPLATE_DIR" python3 << 'PYTHON_EOF'
import yaml
import os
import sys

template_dir = os.environ.get('TEMPLATE_DIR')
if not template_dir:
    template_dir = os.getcwd()

yaml_files = [
    'template.yaml',
    'content/source/catalog-info.yaml',
    'content/gitops/helios-app.yaml',
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

# Check HeliosApp includes database trait
echo ""
echo "Test 4: Verifying HeliosApp CRD structure..."
check_field "content/gitops/helios-app.yaml" "kind: HeliosApp" "HeliosApp kind"
check_field "content/gitops/helios-app.yaml" "type: database" "Database trait"
check_field "content/gitops/helios-app.yaml" "dbType: postgres" "Postgres configuration"



# Check that PostgREST configuration is present
echo ""
echo "Test 5: Verifying PostgREST-specific configuration..."
check_field "content/source/postgrestrc.conf" "db-schema" "PostgREST schema config"
check_field "content/source/postgrestrc.conf" "db-anon-role" "PostgREST anonymous role"
check_field "content/source/postgrestrc.conf" "server-port" "PostgREST server port config"
check_field "content/source/Dockerfile" "postgrest/postgrest" "Official PostgREST image"

# Check database migrations structure
echo ""
echo "Test 6: Verifying database migration structure..."
if [ -d "$TEMPLATE_DIR/content/source/db/migrations" ]; then
    echo "  ✓ db/migrations directory exists"
    if [ -f "$TEMPLATE_DIR/content/source/db/migrations/000001_initial_schema.up.sql" ]; then
        echo "  ✓ Sample migration file present"
    else
        echo "  ⚠ No sample migrations found (this is OK, but users should add them)"
    fi
else
    echo "  ✗ db/migrations directory missing"
    exit 1
fi

# Check migration documentation
if [ -f "$TEMPLATE_DIR/content/source/MIGRATIONS.md" ]; then
    echo "  ✓ Migration documentation present"
else
    echo "  ⚠ MIGRATIONS.md not found"
fi

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
echo "✓ PostgREST configuration present"
echo "✓ Database migrations configured"
echo "✓ PGRST_DB_URI integration configured"
echo ""
echo "✓ PostgREST template is ready for deployment!"
echo ""
echo "Next steps:"
echo "1. Deploy to Helios Platform cluster"
echo "2. Register template in Backstage"
echo "3. Users can scaffold PostgREST services via UI"
echo "4. Users can create database migrations in db/migrations/"
echo ""

