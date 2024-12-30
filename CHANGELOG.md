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
