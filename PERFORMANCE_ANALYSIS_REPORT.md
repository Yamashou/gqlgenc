# Performance Analysis Report for gqlgenc

## Overview
This report documents performance inefficiencies identified in the gqlgenc codebase during analysis on June 16, 2025. gqlgenc is a Go library for building GraphQL clients with automatic code generation based on gqlgen.

## Identified Performance Issues

### 1. **Inefficient String Concatenation in Template Generation** (HIGH PRIORITY)
**Location**: `clientgenv2/template.go:100-108`
**Issue**: Using `bytes.Buffer.WriteString()` with string concatenation instead of more efficient methods.
```go
buf.WriteString("func (t *" + name + ") Get" + field.Name() + "() " + returns + "{\n")
buf.WriteString("if t == nil {\n t = &" + name + "{}\n}\n")
buf.WriteString("return " + pointerOrNot + "t." + field.Name() + "\n}\n")
```
**Impact**: Creates multiple temporary strings during concatenation, causing unnecessary memory allocations.
**Solution**: Use `fmt.Fprintf()` or prepare format strings to reduce allocations.

### 2. **Redundant Map Lookups in Fragment Processing** (MEDIUM PRIORITY)
**Location**: `querydocument/query_document.go:95-126`
**Issue**: Multiple map lookups for the same key in `collectInputObjectFieldsWithCycle()`.
```go
if fieldDef, ok := schema.Types[typeName]; ok && fieldDef.IsInputType() {
    collectInputObjectFieldsWithCycle(fieldDef, schema, usedTypes, processedTypes)
}
```
**Impact**: Repeated hash table lookups for the same type names.
**Solution**: Cache the lookup result to avoid redundant map access.

### 3. **Inefficient Slice Growth in Fragment Collection** (MEDIUM PRIORITY)
**Location**: `querydocument/query_document.go:52-70`
**Issue**: Recursive slice appending without pre-allocation in `fragmentsInOperationWalker()`.
```go
fragments = append(fragments, fragmentsInOperationWalker(selectionSet)...)
```
**Impact**: Multiple slice reallocations as fragments are collected recursively.
**Solution**: Pre-calculate capacity or use a more efficient collection strategy.

### 4. **Unnecessary Slice Copying in mergeFieldsRecursively** (MEDIUM PRIORITY)
**Location**: `clientgenv2/source_generator.go:127-162`
**Issue**: Creating new slices and copying data multiple times during field merging.
```go
responseFieldList := make(ResponseFieldList, 0)
// ... multiple append operations
```
**Impact**: O(n²) behavior due to repeated slice growth and copying.
**Solution**: Pre-allocate slices with known capacity or use more efficient merging algorithms.

### 5. **Inefficient Type String Processing** (LOW PRIORITY)
**Location**: `clientgenv2/source_generator.go:31-35`
**Issue**: String splitting and indexing for type name extraction.
```go
parts := strings.Split(fullFieldType, ".")
return parts[len(parts)-1]
```
**Impact**: Unnecessary string allocation and processing.
**Solution**: Use `strings.LastIndex()` or similar for direct extraction.

### 6. **Repeated File System Operations** (LOW PRIORITY)
**Location**: `config/config.go:152-164`
**Issue**: Multiple `filepath.Walk()` calls for schema file discovery.
**Impact**: Redundant file system traversals.
**Solution**: Cache results or combine traversals.

### 7. **Inefficient JSON Marshaling in Client** (MEDIUM PRIORITY)
**Location**: `clientv2/client.go:484-533`
**Issue**: Custom JSON encoder with reflection-heavy operations.
**Impact**: Performance overhead compared to standard library for simple cases.
**Solution**: Use standard `json.Marshal()` for simple types, custom encoder only when needed.

## Recommended Fix Priority

1. **String concatenation in template generation** - Easy fix, high impact
2. **Fragment collection slice growth** - Medium complexity, good impact
3. **Redundant map lookups** - Easy fix, medium impact
4. **Field merging efficiency** - Higher complexity, good impact
5. **Type string processing** - Easy fix, low impact

## Performance Testing Recommendations

1. Create benchmarks for template generation with large schemas
2. Measure memory allocations during fragment processing
3. Test with complex nested GraphQL queries
4. Profile CPU usage during code generation

## Implementation Plan

For this PR, I will implement the fix for **Issue #1 (String Concatenation in Template Generation)** as it provides:
- High performance impact
- Low implementation complexity
- Clear measurable improvement
- No risk of breaking existing functionality

The fix will replace string concatenation with `fmt.Fprintf()` calls to reduce memory allocations and improve performance.
