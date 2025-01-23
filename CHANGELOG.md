## [0.1.6] - 2025-01-23

### Added
- Support for single wildcard tier configuration
- Improved handling of wildcard tier inheritance

### Changed
- Removed debug output from module source matching
- Improved module source matching logic for better pattern handling

## [0.1.5] - 2025-01-22

### Added
- Enhanced tier configuration with wildcard support:
  - Added "*" wildcard tier to process all files regardless of location
  - Support for tier-specific overrides when using wildcard
  - Process files in any directory structure or naming pattern
  - Improved tier matching logic for better control over file processing

### Changed
- Modified tier processing behavior:
  - Specific tier settings now take precedence over wildcard
  - Simplified configuration by allowing "*" as default with overrides
  - More intuitive handling of tier-specific settings

## [0.1.4] - 2025-01-22

### Added
- Comprehensive pre-1.0 version handling:
  - Keep pre-1.0 exact versions as exact versions by default
  - Preserve pre-release tags and build metadata
  - Convert pre-1.0 ranges to exact versions
  - Handle complex ranges and mixed version formats
  - Support zero versions (0.0.x) with metadata
- Enhanced version comparison logic for pre-1.0 versions
- Added helper functions for consistent pre-1.0 version handling
- Improved metadata preservation across all strategies

### Bug Fixes
- Fixed metadata handling in complex version ranges
- Improved pre-1.0 version comparison logic
- Enhanced handling of mixed pre-1.0 and post-1.0 versions
- Fixed edge cases with multiple pre-release segments

## [0.1.3] - 2024-12-30

### Bug Fixes
- Fixed version comparison in exact strategy to properly handle backward compatibility
- Improved handling of pre-release versions and build metadata
- Enhanced version range strategy to respect existing ranges when appropriate
- Fixed version 0.x.x handling with proper semver semantics
- Improved version string normalization for complex ranges with OR conditions

### Enhancements
- Added comprehensive handling of complex version ranges
- Improved handling of tilde arrow notation in version ranges
- Enhanced backward compatibility protection in version strategies

## [0.1.2] - 2024-12-30

### Performance Improvements
- Optimized version range operations using binary search instead of linear search
- Improved performance of `findHighestVersionInRange` and `findLowestVersionInRange` functions (O(n³) → O(log n))
- Enhanced version range overlap detection with strategic sampling points
- Optimized version comparison logic in `DecideVersionOrRange`

### Documentation
- Added comprehensive usage examples in README
- Added Sourcegraph integration example with batch changes
- Improved documentation structure and readability

### Bug Fixes
- Fixed version string normalization to handle various spacing formats
- Improved version comparison logic for edge cases
- Added test cases for version string formatting variations

### Code Quality
- Enhanced test coverage for version string normalization
- Improved error handling and edge cases
- Added better code documentation and comments
