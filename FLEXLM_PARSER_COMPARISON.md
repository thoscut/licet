# FlexLM Parser Comparison: PHP vs Go

## Overview
This document compares the FlexLM parser logic between the original PHP implementation and the Go rewrite to identify any differences or missing functionality.

---

## 1. Command Execution

### PHP (Line 22)
```php
$fp = popen($lmutil . " lmstat -i -a -c " . $server,"r");
```

### Go (Line 41)
```go
cmd := exec.Command(p.lmutilPath, "lmstat", "-i", "-a", "-c", hostname)
```

**Status:** ✅ **EQUIVALENT** - Both execute the same command with same flags

---

## 2. Server Status Detection

### PHP (Line 30)
```php
if (preg_match("/([^\s]+): license server UP.*v(\d+\.\d+\.\d+).*/", trim($line), $res))
```

### Go (Line 67)
```go
serverUpRe := regexp.MustCompile(`([^\s]+):\s+license server UP.*v(\d+\.\d+\.\d+)`)
```

**Status:** ✅ **EQUIVALENT** - Both capture server name and version

---

## 3. Error Detection

### PHP
- Line 39: `Cannot connect to license server` → "down"
- Line 46: `Cannot read data` → "down"
- Line 53: `Error getting status` → "down"
- Line 61: `vendor daemon is down` → "warning"

### Go
- Line 68: `Cannot connect to license server` → "down"
- Line 69: `Cannot read data` → "down"
- Line 70: `Error getting status` → "down"
- Line 71: `vendor daemon is down` → "warning"

**Status:** ✅ **EQUIVALENT** - All error conditions handled identically

---

## 4. Feature Parsing - **CRITICAL DIFFERENCE**

### PHP (Line 70)
```php
if ( preg_match('/(users of) (.*)(\(total of) (\d+) (.*) (total of) (\d+) /i', $line, $out)
     && !preg_match('/no such feature exists/i', $line) )
```

**Pattern matches:**
- `users of FEATURE (total of NUM ... total of NUM`
- Note: Case insensitive (`/i` flag)
- Explicitly excludes lines with "no such feature exists"

### Go (Line 74)
```go
featureRe := regexp.MustCompile(`users of\s+(.+?):\s+\(Total of (\d+) license[s]? issued;\s+Total of (\d+) license[s]? in use\)`)
```

**Pattern matches:**
- `users of FEATURE: (Total of NUM license[s] issued; Total of NUM license[s] in use)`
- Note: Case sensitive (no `(?i)` flag)
- Does not check for "no such feature exists"

### ⚠️ **ISSUE 1: Case Sensitivity**
The original PHP implementation uses case-insensitive matching, Licet is case-sensitive. This could fail if output has variations like:
- "Users of" vs "users of"
- "total of" vs "Total of"

### ⚠️ **ISSUE 2: Pattern Format**
The patterns expect different output formats:
- PHP: Flexible pattern that works with multiple FlexLM output formats
- Go: Expects specific format with "issued" and "in use" keywords

### ⚠️ **ISSUE 3: Error Checking**
PHP explicitly filters out "no such feature exists" errors, Go does not.

---

## 5. Uncounted Licenses

### PHP (Line 79)
```php
if ( preg_match('/(users of) (.*)(\(uncounted, node-locked)/i', $line, $out) )
```

### Go (Line 75)
```go
uncountedRe := regexp.MustCompile(`users of\s+(.+?):\s+\(uncounted, node-locked\)`)
```

**Status:** ⚠️ **SIMILAR BUT DIFFERENT**
- PHP: Case insensitive, no colon after feature name
- Go: Case sensitive, expects colon after feature name

---

## 6. Expiration Parsing - **MAJOR DIFFERENCE**

### PHP (Lines 84-87)
```php
$expiration_pattern_base = '/(?<feature>\w+)\s+(?<version>\d+|\d+.\d+)\s+(?<count>\d+)\s+';
$expiration_pattern_old = $expiration_pattern_base . '(?<expiration>\d+-\w+-\d+)(\s+(?<vendor>\w+))?$/i';
$expiration_pattern_new = $expiration_pattern_base . '(?<vendor>\w+)\s+(?<expiration>\d+-\w+-\d+)$/i';
$expiration_pattern_permanent = $expiration_pattern_base . '(?<vendor>\w+)\s+(?<expiration>permanent).*$/i';

if ( preg_match($expiration_pattern_old, $line, $out)
     or preg_match($expiration_pattern_new, $line, $out)
     or preg_match($expiration_pattern_permanent, $line, $out) )
```

**Handles THREE formats:**
1. Old format: `FEATURE VERSION COUNT DATE [VENDOR]`
2. New format: `FEATURE VERSION COUNT VENDOR DATE`
3. Permanent: `FEATURE VERSION COUNT VENDOR permanent`

### Go (Line 78)
```go
expirationRe := regexp.MustCompile(`(\w+)\s+(\d+|\d+\.\d+)\s+(\d+)\s+(\d+-\w+-\d+)\s+(\w+)`)
```

**Handles ONE format:**
- Only: `FEATURE VERSION COUNT DATE VENDOR` (new format)

### 🚨 **CRITICAL ISSUE: Missing Permanent License Support**
The Licet **DOES NOT** handle permanent licenses! This is a significant missing feature.

---

## 7. Date Replacement

### PHP (Line 89)
```php
$out['expiration'] = str_replace("-jan-0000", "-jan-2036", $out['expiration']);
$out['expiration'] = str_replace("-jan-0", "-jan-2036", $out['expiration']);
```

### Go (Line 163)
```go
expirationStr := strings.Replace(matches[4], "-jan-0", "-jan-2036", 1)
```

**Status:** ⚠️ **INCOMPLETE**
- PHP replaces both "-jan-0000" AND "-jan-0"
- Go only replaces "-jan-0" (single occurrence)
- Go should also replace "-jan-0000"

---

## 8. User Parsing

### PHP (Line 104)
```php
if ( preg_match('/.*, start \w+\s+(\d+\/\d+\s+\d+:\d+)/', $line, $out ) )
```
- Flexible pattern, captures any line with ", start DATE"

### Go (Line 81)
```go
userRe := regexp.MustCompile(`\s+(.+?)\s+(.+?)\s+(.+?)\s+\(v\d+\.\d+\).*start\s+(\w+\s+\d+/\d+\s+\d+:\d+)`)
```
- More specific pattern, expects version and extracts username, host

**Status:** ⚠️ **DIFFERENT APPROACHES**
- PHP: Simple pattern, only captures date
- Go: Complex pattern, extracts more information (username, host, date)
- Go approach is better but more brittle if format varies

---

## Summary of Issues

### 🚨 Critical Issues

1. **Missing Permanent License Support**
   - PHP handles "permanent" expiration dates
   - Go does not, will fail to parse permanent licenses
   - **Impact:** High - permanent licenses won't be tracked

2. **Feature Parsing Case Sensitivity**
   - PHP uses case-insensitive matching
   - Go uses case-sensitive matching
   - **Impact:** Medium - could fail with different FlexLM output formats

3. **Incomplete Date Replacement**
   - PHP replaces both "-jan-0000" and "-jan-0"
   - Go only replaces "-jan-0"
   - **Impact:** Low - but could miss some edge cases

### ⚠️ Minor Issues

4. **Missing "no such feature exists" Check**
   - PHP filters out error lines
   - Go does not
   - **Impact:** Low - might capture invalid features

5. **Expiration Pattern Coverage**
   - PHP supports 3 expiration formats (old, new, permanent)
   - Go only supports 1 format (new)
   - **Impact:** High - won't parse older FlexLM output formats

6. **Uncounted License Pattern**
   - PHP: More flexible, no colon requirement
   - Go: Expects colon after feature name
   - **Impact:** Low - most FlexLM outputs include colon

---

## Recommendations

### Priority 1 (Critical)
1. **Add permanent license support** to Go parser
2. **Add old format expiration pattern** support
3. **Make feature regex case-insensitive**

### Priority 2 (Important)
4. Add "-jan-0000" date replacement
5. Add "no such feature exists" filtering
6. Add tests with real FlexLM output samples

### Priority 3 (Nice to have)
7. Document which FlexLM versions are tested
8. Add pattern fallbacks for flexibility
9. Consider making uncounted pattern more flexible

---

## Testing Needed

To verify compatibility, test with:
1. Old FlexLM format (pre-v11)
2. New FlexLM format (v11+)
3. Permanent licenses
4. Uncounted/node-locked licenses
5. Mixed case output (if any versions produce this)
6. Error conditions
7. Multiple vendor daemons

---

## Conclusion

The Go implementation is mostly faithful to the original PHP implementation but has **critical missing functionality**:
- No support for permanent licenses
- No support for old expiration date format
- Case-sensitive matching could cause issues

These should be addressed before considering the Go parser production-ready.
