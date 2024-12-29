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
    versions:
      dev: "2.0.0"            # Version for development
      stg: "2.0.0"            # Version for staging
      prd: "1.9.0"            # Version for production
```

Note: While examples use "dev", "stg", and "prd" tiers, you can use any tier names that match your infrastructure organization (e.g., "development", "qa", "staging", "production", "sandbox", etc.).

## Version Update Strategies

There are three strategies available for version management:
- `dynamic` (default): Preserves existing version style
- `exact`: Requires exact version specification
- `range`: Forces version ranges

Strategies can be specified:
1. At the module level (applies to all tiers unless overridden)
2. For all tiers using "*"
3. Per specific tier
4. A combination of the above, where more specific settings override general ones

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
