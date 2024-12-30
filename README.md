# HCL Version Updater

An automated solution for managing Terraform module versions across different infrastructure tiers with flexible version strategies and patterns.

## Overview

HCL Version Updater automates version management across your infrastructure code. It supports:
- Tier-specific version management (e.g., dev, stg, prd)
- Multiple version update strategies
- Flexible version formats and patterns
- Preservation of existing version styles
- Bulk updates across multiple modules

## Installation

```bash
go install github.com/david1155/hclsemver@latest
```

Alternatively, you can use the Docker image:

```bash
docker pull david1155/hclsemver
# Mount your current directory to /work (default directory)
docker run -v $(pwd):/work david1155/hclsemver [flags]
```

## Configuration

HCL Version Updater uses a YAML or JSON configuration file to specify module updates. The configuration supports:
- Multiple modules
- Tier-specific versions
- Version update strategies
- Custom version patterns

### Basic Configuration Structure

```yaml
modules:
  - source: "module-source"    # Module source pattern to match
    strategy: "dynamic"        # Optional: dynamic (default), exact, or range
    force: false              # Optional: whether to add version if not present (default: false)
    versions:
      dev: "2.0.0"            # Version for development
      stg: "2.0.0"            # Version for staging
      prd: "1.9.0"            # Version for production
```

Note: While examples use "dev", "stg", and "prd" tiers, you can use any tier names that match your infrastructure organization (e.g., "development", "qa", "staging", "production", "sandbox", etc.).

### Module Configuration Options

- `source`: (Required) The module source pattern to match
- `strategy`: (Optional) Default strategy for all tiers unless overridden
- `force`: (Optional) Whether to add version attribute to modules that don't have one (default: false)
- `versions`: (Required) Map of tier-specific version configurations

The `force` flag can be specified at both the module level and tier level:
- Module level: Applies to all tiers unless overridden
- Tier level: Overrides the module-level setting for specific tiers
- Wildcard tier: Applies to all tiers that don't have a specific setting

When a module is found without a version attribute:
- If `force: false` (default): A warning is output and the module is skipped
- If `force: true`: The version attribute is added with the specified version

Example with force flag at different levels:
```yaml
modules:
  - source: "hashicorp/aws/vpc"
    force: true    # Default for all tiers
    versions:
      "*":         # Override for all tiers
        force: false
        version: "2.0.0"
      dev:         # Override for specific tier
        force: true
        version: "2.0.0"
      stg:         # Uses wildcard setting (false)
        version: "2.0.0"
      prd:         # Override for specific tier
        force: false
        version: "2.0.0"

  - source: "custom/module"
    force: false   # Default for all tiers
    versions:
      dev:
        force: true    # Override for dev tier
        version: "1.0.0"
      stg:
        version: "1.0.0"   # Uses module default (false)
```

Force flag precedence (highest to lowest):
1. Tier-specific force setting (e.g., `dev.force`)
2. Wildcard force setting (`"*".force`)
3. Module-level force setting
4. Global default (`false`)

## Version Update Strategies

The tool supports three version update strategies:

1. `dynamic` (default): Intelligently decides between exact versions and ranges
   - Preserves existing version style (exact or range) when possible
   - Prevents backward version changes (keeps higher version if target is lower)
   - Converts between styles only when necessary

2. `exact`: Always uses exact versions (e.g., "1.2.3")
   - Converts any range to an exact version
   - Prevents backward version changes
   - Useful when precise version control is needed

3. `range`: Always uses version ranges (e.g., ">=1.2.3,<2.0.0")
   - Converts any exact version to a range
   - Prevents backward version changes
   - Useful for more flexible version management

### Backward Version Protection

The tool includes built-in protection against backward version changes:

- When updating from a higher version to a lower version, the higher version is preserved
- This applies to both exact versions and version ranges
- Examples:
  - Exact versions: If current version is 2.0.0 and target is 1.0.0, keeps 2.0.0
  - Ranges: If current range is ">=2.0.0,<3.0.0" and target is ">=1.0.0,<2.0.0", keeps current range
  - Mixed: If current version is 2.0.0 and target range is ">=1.0.0,<1.5.0", keeps 2.0.0

This protection ensures that modules don't accidentally downgrade to older versions during updates.

### Strategy Configuration Examples

#### 1. Module-Level Strategy (Default for All Tiers)
```yaml
modules:
  - source: "hashicorp/aws/eks"
    strategy: "exact"  # Default strategy for all tiers
    versions:
      dev: "2.0.0"
      stg: "2.0.0"
      prd: "2.0.0"
```

#### 2. All-Tiers Strategy with Override
```yaml
modules:
  - source: "hashicorp/aws/rds"
    versions:
      "*":  # Applies to all tiers
        strategy: "range"
        version: "2.0.0"  # Will become ">=2.0.0, <3.0.0"
      prd:  # Overrides the "*" setting for production
        strategy: "exact"
        version: "2.0.0"  # Stays as "2.0.0"
```

#### 3. Per-Tier Strategies
```yaml
modules:
  - source: "hashicorp/aws/vpc"
    versions:
      dev:
        strategy: "range"
        version: "1.0.0"  # Becomes ">=1.0.0, <2.0.0"
      stg:
        strategy: "dynamic"
        version: "2.0.0"  # Keeps existing style
      prd:
        strategy: "exact"
        version: "2.0.0"  # Must be exact version
```

#### 4. Mixed Strategy Inheritance
```yaml
modules:
  - source: "terraform-aws-modules/eks/aws"
    strategy: "dynamic"  # Default fallback
    versions:
      "*":  # Base for all tiers
        strategy: "range"
        version: "18.0.0"
      dev:  # Uses "*" strategy (range)
        version: "19.0.0"
      stg:  # Override "*" strategy
        strategy: "dynamic"
        version: "18.0.0"
      prd:  # Override "*" strategy
        strategy: "exact"
        version: "18.0.0"
```

Strategy precedence (highest to lowest):
1. Tier-specific strategy (e.g., `prd.strategy`)
2. Wildcard strategy (`"*".strategy`)
3. Module-level strategy
4. Global default (`dynamic`)

### Strategy Configuration Examples

#### Dynamic Strategy (Default)
Intelligently preserves existing version styles while ensuring version requirements are met:

1. If the existing version is a range:
   - Keeps the range if target version falls within it
   - Updates the range if target version is outside it
   ```yaml
   # Example: existing version is ">=1.0.0, <2.0.0"
   versions:
     dev: "1.5.0"    # Keeps as ">=1.0.0, <2.0.0" (target within range)
     stg: "2.0.0"    # Changes to ">=2.0.0, <3.0.0" (target outside range)
   ```

2. If the existing version is exact:
   - Keeps using exact versions
   ```yaml
   # Example: existing version is "1.0.0"
   versions:
     dev: "2.0.0"    # Updates to exact "2.0.0"
     stg: "2.1.0"    # Updates to exact "2.1.0"
   ```

3. For new files (no existing version):
   - Uses the target version as-is
   ```yaml
   versions:
     dev: "2.0.0"    # Used as exact "2.0.0"
     stg: ">=2.0.0"  # Used as range ">=2.0.0"
   ```

Example configuration:
```yaml
modules:
  - source: "hashicorp/aws/eks"
    strategy: "dynamic"  # Will preserve existing styles
    versions:
      dev: "2.0.0"      # Will adapt to existing format
      stg: ">=2.0.0"    # Will adapt to existing format
      prd: "2.1.0"      # Will adapt to existing format
```

#### Exact Strategy
Requires and enforces exact version pinning. When using this strategy, you must specify exact versions (e.g., "2.0.0") - version ranges or constraints are not allowed:
```yaml
modules:
  - source: "hashicorp/aws/rds"
    strategy: "exact"
    versions:
      dev: "2.0.0"  # Valid - exact version
      stg: "2.1.1"  # Valid - exact version
      prd: "2.1.0"  # Valid - exact version
```

Invalid configurations with exact strategy:
```yaml
modules:
  - source: "hashicorp/aws/rds"
    strategy: "exact"
    versions:
      dev: ">=2.0.0"        # Invalid - range not allowed
      stg: "~>2.1.0"        # Invalid - tilde range not allowed
      prd: "^2.1.0"         # Invalid - caret range not allowed
```

The exact strategy always uses the specified version directly if it's an exact version, or the lowest valid version if given a range. For example:
- "2.0.0" -> "2.0.0" (kept as is)
- ">=2.0.0" -> "2.0.0" (lowest valid version)
- "~>2.1.0" -> "2.1.0" (lowest valid version)

#### Range Strategy
Forces version ranges:
```yaml
modules:
  - source: "hashicorp/aws/vpc"
    strategy: "range"
    versions:
      dev: "1.0.0"  # Becomes ">=1.0.0, <2.0.0"
      stg: "2.0.0"  # Becomes ">=2.0.0, <3.0.0"
      prd: "3.0.0"  # Becomes ">=3.0.0, <4.0.0"
```

## Advanced Use Cases

### 1. Tier-Agnostic Updates
To update all tiers to the same version regardless of tier:
```yaml
modules:
  - source: "terraform-aws-modules/vpc/aws"
    versions:
      "*": "3.0.0"  # Applies to all tiers
```

### 2. Mixed Strategies per Tier
```yaml
modules:
  - source: "terraform-aws-modules/eks/aws"
    versions:
      dev:
        strategy: "range"    # Allow patches in dev
        version: "18.0.0"    # Becomes ">=18.0.0, <19.0.0"
      stg:
        strategy: "dynamic"  # Keep existing style
        version: "18.0.0"
      prd:
        strategy: "exact"    # Pin version in production
        version: "18.0.0"
```

### 3. Multiple Module Sources with Patterns
```yaml
modules:
  - source: "terraform-aws-modules/.*/aws"  # Regex pattern
    strategy: "exact"
    versions:
      "*": "2.0.0"  # Update all tiers

  - source: "hashicorp/google"
    versions:
      dev: "4.0.0"
      "*": "3.5.0"  # All other tiers
```

### 4. Complex Version Specifications
```yaml
modules:
  - source: "custom/module"
    versions:
      dev:
        strategy: "range"
        version: ">=2.0.0, <3.0.0 || >=3.1.0, <4.0.0"
      stg:
        strategy: "exact"
        version: "~>2.1.0"  # Will use lowest valid version
      prd: "2.1.0"         # Simple exact version
```

## Usage Examples

### 1. Basic Update
By default, the tool scans the `work` directory in your current path:
```bash
hclsemver -config versions.yaml
```

### 2. Custom Directory
You can specify a different directory to scan:
```bash
hclsemver -config versions.yaml -dir infrastructure
```

### 3. Dry Run
```bash
hclsemver -config versions.yaml -dry-run
```

### 4. Sourcegraph Integration
HCL Version Updater can be integrated with Sourcegraph batch changes to automate version updates across multiple repositories. Here's an example:

```yaml
name: INFRA-1234
description: >
  Update infrastructure module versions across development environments.
  This batch change targets repositories in the infrastructure organization
  that contain Terraform files in development paths.

on:
  - repositoriesMatchingQuery: >
      context:global
      repo:^github\.com/infrastructure/.*
      file:(development|dev)/.*\.tf$
      fork:no
      archived:no
      (
        content:"registry.terraform.io/hashicorp"
        OR content:"github.com/infrastructure"
        OR content:"terraform.custom-registry.com"
      )
      count:all
      timeout:1200s

steps:
  - run: |
      #!/bin/sh
      cat > /app/versions.yaml << 'EOF'
      modules:
        - source: "registry.terraform.io/hashicorp/aws/eks"
          strategy: "range"    # Use ranges for development
          force: true         # Add version if missing
          versions:
            "*":              # Default for all tiers
              strategy: "range"
              version: "20.0.0"
            dev:             # Override for dev
              strategy: "range"
              version: "21.0.0"
              force: true

        - source: "github.com/infrastructure/networking"
          strategy: "dynamic"  # Preserve existing style
          versions:
            dev:
              strategy: "range"
              version: ">=5.0.0,<6.0.0"
            "*":
              strategy: "exact"
              version: "4.2.1"

        - source: "terraform.custom-registry.com/security/.*"  # Regex pattern
          versions:
            dev:
              strategy: "exact"
              version: "3.1.0"
              force: true     # Add version if missing
            "*":
              strategy: "range"
              version: "2.0.0"
              force: false    # Skip if version missing

        - source: "registry.terraform.io/hashicorp/kubernetes"
          strategy: "exact"   # Always use exact versions
          force: false       # Skip if version missing
          versions:
            dev: "2.23.0"
            "*": "2.22.0"    # For any other tier

        - source: "terraform.custom-registry.com/monitoring/.*"
          strategy: "dynamic"
          versions:
            dev:
              strategy: "range"    # Use ranges in dev
              version: "4.0.0"     # Will become >=4.0.0,<5.0.0
              force: true
            "*":
              strategy: "exact"    # Use exact versions elsewhere
              version: "3.2.1"
              force: false
      EOF

      /app/hclsemver -config /app/versions.yaml

    container: david1155/hclsemver:v0.1.1

changesetTemplate:
  title: 'INFRA-1234: Update Terraform module versions'
  body: |
    This batch change updates Terraform module versions across development environments.
    
    Changes include:
    - EKS module updated to use version ranges (>=20.0.0,<21.0.0)
    - Networking module versions aligned with new architecture
    - Security modules pinned to exact versions in dev
    - Kubernetes provider version bumped to 2.23.0
    - Monitoring modules configured with flexible versioning strategy
    
    Testing:
    - [ ] Terraform plan executed successfully
    - [ ] Integration tests passed
    - [ ] Security scan completed
    
    Related:
    - [INFRA-1234](https://jira.company.com/browse/INFRA-1234)
    - [RFC-789](https://docs.company.com/rfcs/789)
  branch: infra-1234-module-versions
  commit:
    message: 'INFRA-1234: Update Terraform module versions'
```

## Directory Structure Support

HCL Version Updater works with various directory organizations. By default, it looks in the `work` directory, but you can override this with the `-dir` flag.

### 1. Default Structure (work directory)
```
work/
├── dev/
│   └── main.tf
├── stg/
│   └── main.tf
└── prd/
    └── main.tf
```

### 2. Custom Directory Structure
```
infrastructure/  # Specified with -dir flag
├── dev/
│   └── main.tf
├── stg/
│   └── main.tf
└── prd/
    └── main.tf
```

### 3. Tier-Based Files
```
work/  # or custom directory
├── dev.tf
├── stg.tf
└── prd.tf
```

### 4. Mixed Structure
```
work/  # or custom directory
├── shared/
│   ├── dev.tf
│   ├── stg.tf
│   └── prd.tf
└── services/
    ├── dev/
    │   └── main.tf
    ├── stg/
    │   └── main.tf
    └── prd/
        └── main.tf
```

## Version Format Support

Supported version formats include:

- Exact versions: `"1.2.3"`
- Caret ranges: `"^1.2.3"` (equivalent to `>=1.2.3, <2.0.0`)
- Tilde ranges: `"~>1.2.3"` (equivalent to `>=1.2.3, <1.3.0`)
- Complex ranges: `">=1.2.3, <2.0.0 || >=2.1.0, <3.0.0"`
- Wildcards: `"*"` (any version)

## Best Practices

1. **Version Control**: Always commit your configuration file to version control
2. **Tier Progression**: Use increasingly strict version constraints as you move towards production
3. **Documentation**: Comment your configuration file with reasons for version choices
4. **Regular Updates**: Schedule regular version updates as part of your maintenance routine
5. **Testing**: Always test version updates in lower environments first

## Error Handling

HCL Version Updater provides detailed logging and continues processing even if some updates fail:
- Invalid version specifications are logged
- Missing directories are reported
- Parse errors are documented
- Failed updates are listed in the summary

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
