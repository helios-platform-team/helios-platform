---
name: use-modern-cue
description: Apply modern CUE language syntax guidelines based on project's CUE version. Use when user asks for modern CUE code guidelines.
---

# Modern CUE Guidelines

## Detected CUE Version

!`grep -rh '"language"' --include="module.cue" cue.mod/ 2>/dev/null | grep -oP 'version:\s*"v?\K[^"]+' | head -1 | grep . || cue version 2>/dev/null | head -1 | grep -oP 'v\K[0-9]+\.[0-9]+\.[0-9]+' | head -1 | grep . || echo unknown`

## How to Use This Skill

DO NOT search for module.cue files or try to detect the version yourself. Use ONLY the version shown above.

**If version detected (not "unknown"):**

- Say: "This project is using CUE X.XX.X, so I'll stick to modern CUE best practices and freely use language features up to and including this version. If you'd prefer a different target version, just let me know."
- Do NOT list features, do NOT ask for confirmation

**If version is "unknown":**

- Say: "Could not detect CUE version in this repository"
- Use AskUserQuestion: "Which CUE version should I target?" → [0.14] / [0.15] / [0.16]

**When writing CUE code**, use ALL features from this document up to the target version:

- Prefer modern patterns over deprecated/legacy ones
- Never use features from newer CUE versions than the target
- Always follow the best practices and avoid anti-patterns listed below

---

## Core CUE Principles

### Types Are Values

CUE unifies types and values into a single lattice. A field can hold a type constraint, a concrete value, or something in between.

```cue
port: int           // type constraint
port: >0 & <65536   // narrower constraint
port: 8080          // concrete value
```

### Unification (`&`)

Fundamental, commutative, associative, and idempotent — order never matters.

```cue
a: {name: string, port: int}
a: {name: "web", replicas: 1}
// Result: a: {name: "web", port: int, replicas: 1}
```

### Definitions (`#`) and Hidden Definitions (`_#`)

`#` defines schemas — closed structs, NOT emitted in output. `_#` for package-internal schemas.

```cue
#Service: {
    name:     string
    port:     int & >0 & <=65535
    replicas: int | *1
}

myService: #Service & {name: "api", port: 8080}

_#internalConfig: {debug: bool | *false}
```

### Hidden Fields (`_`)

Not exported, exempt from closed struct rules. Use for internal values.

```cue
_basePort: 8000
services: {
    api:  {port: _basePort}
    grpc: {port: _basePort + 1000}
}
```

### Disjunctions (`|`), Defaults (`*`), Optional (`?`), Required (`!`)

```cue
#Deployment: {
    name!:      string                        // MUST be specified
    protocol:   "HTTP" | "HTTPS" | *"HTTPS"   // enum with default
    replicas?:  int & >0                      // MAY be specified
    annotations?: {[string]: string}
}
```

### Closed vs Open Structs

Definitions are closed by default. Add `...` to allow additional fields.

```cue
#Strict:   {name: string, port: int}       // no extra fields
#Flexible: {name: string, port: int, ...}  // extra fields allowed
```

### Pattern Constraints, Comprehensions, Interpolation, `let`

```cue
// Pattern constraint: all string-keyed fields must satisfy #Service
[string]: #Service

// List comprehension
ports: [for s in services {s.port}]

// Field comprehension with conditional
rendered: {
    for name, svc in services {
        (name): {image: "reg/\(name):latest", port: svc.port}
    }
}

// Conditional field inclusion
if env == "prod" {replicas: 3}

// let for intermediate computation (not emitted)
let _fullName = "app-\(name)"
metadata: labels: app: _fullName
```

### Validators and Constraints

```cue
import "strings"

#Label: {
    key:   string & =~"^[a-z][a-z0-9-]*$"
    value: strings.MinRunes(1) & strings.MaxRunes(63)
}
#Port: int & >0 & <=65535
```

### Builtin Functions

```cue
len("hello")         // 5
close({a: 1})        // closes an open struct
and([int, >0, <100]) // unifies: int & >0 & <100
or(["a", "b", "c"])  // disjunction: "a" | "b" | "c"
```

---

## Standard Library (Key Packages)

```cue
import ("strings"; "list"; "regexp"; "math"; "encoding/json"; "encoding/yaml"; "struct"; "net"; "strconv")

// strings
strings.Join(["a","b"], ",")  strings.Split("a,b", ",")  strings.Contains("ab", "a")
strings.HasPrefix/HasSuffix   strings.Replace  strings.ToUpper/ToLower  strings.TrimSpace
strings.MinRunes(1)  strings.MaxRunes(63)

// list
list.Concat([[1,2],[3,4]])  list.Repeat([0],3)  list.Contains([1,2],2)
list.Sort([3,1,2], list.Ascending)  list.FlattenN  list.UniqueItems
list.MinItems(1)  list.MaxItems(10)

// regexp
regexp.Match("^[a-z]+$", "hello")  regexp.Find  regexp.FindAll  regexp.ReplaceAll

// math
math.Floor  math.Ceil  math.Abs  math.Pow

// encoding
json.Marshal({a:1})  json.Unmarshal(str)  json.Validate(str, schema)
yaml.Marshal  yaml.Validate

// struct
struct.MinFields(1)  struct.MaxFields(10)

// net
net.IPv4  net.IP  net.FQDN
net.InCIDR  net.ParseCIDR  net.CompareIP  // v0.16+

// strconv
strconv.ParseNumber  // v0.16+: parse CUE numbers like "1Ki"
```

---

## Breaking Changes by Version

| Version | Breaking Change |
|---------|----------------|
| **v0.9** | `language.version` REQUIRED in `module.cue` |
| **v0.11** | List arithmetic (`+`, `*`) removed → use `list.Concat`, `list.Repeat` |
| **v0.12** | `cue.Value.Decode` returns `int64` (not `int`) for CUE integers |
| **v0.12** | `@embed` stable — requires `@extern(embed)` file attribute |
| **v0.13** | Removed deprecated `cue.Runtime` methods |
| **v0.14** | Multiline strings require trailing newline |
| **v0.14** | Custom errors: `error("msg")` builtin |
| **v0.14** | JSON Schema generation: `cue def --out jsonschema` |
| **v0.16** | Multiline trailing newline **strictly enforced** |
| **v0.16** | `cue` commands inside `cue.mod/` directory now fail |
| **v0.16** | `cue mod publish` no longer ignores sub-dirs with `go.mod` |
| **v0.16** | `#"""#` accepted as string literal for `"` |
| **v0.16** | `cmdreferencepkg` experiment stabilized (always on) |
| **v0.16** | Removed deprecated: `cue/ast.Node.Comments`, `cue/parser.FromVersion`, `cue.Instance.Eval` |

## Key Features by Version

| Version | Feature |
|---------|---------|
| **v0.10** | `@if(tag)` conditional file inclusion, `@tag(name)` value injection, `@ignore()` |
| **v0.12** | File embedding: `@extern(embed)` + `@embed(file="x.json")` |
| **v0.13** | `cue refactor imports`, `cue mod mirror`, keywords as field labels |
| **v0.14** | `error()` builtin, `cue def --out jsonschema`, `cue fix --exp`, `cue help experiments` |
| **v0.15** | Full LSP support (go-to-def, find-refs, rename, completion, hover) |
| **v0.16** | `net.InCIDR/ParseCIDR/CompareIP`, `strconv.ParseNumber`, `tool/file.Symlink` |
| **v0.16** | Up to 80% faster evaluation, 60% less memory, embedded JSON/YAML LSP support |

---

## Module & Package Best Practices

### Module Declaration (v0.9+, REQUIRED)

```cue
// cue.mod/module.cue
module: "github.com/myorg/myproject@v0"
language: version: "v0.16.0"

deps: {
    "github.com/some/dep@v0": v: "v0.1.0"
}
```

**ALWAYS** declare `language.version`. Never omit it.

### Import Best Practices

```cue
import (
    "strings"                          // built-in (no domain)
    "list"
    "github.com/myorg/project/schema"  // user-defined (fully-qualified)
    k8s "k8s.io/api/apps/v1"          // named import for conflicts
)
```

- ALWAYS use absolute import paths (no relative imports)
- Group built-in and user-defined imports separately

### Package Organization

- One package per directory
- `package` declaration at top of every file
- Definitions in same package accessible across files without imports
- Hidden fields (`_`) scoped to package — not externally accessible
- Do NOT place CUE code inside `cue.mod/` (v0.16+: this errors)

### File Embedding (v0.12+)

```cue
@extern(embed)
package config

data: _ @embed(file="config.json")
readme: string @embed(file="README.md", type=text)
cert: bytes @embed(file="cert.pem", type=binary)
templates: _ @embed(glob="templates/*.json")
optional: _ @embed(glob="overrides/*.yaml", allowEmptyGlob)
```

ALWAYS use `@extern(embed)` at file level. Only files within the same CUE module can be embedded.

---

## CLI Quick Reference

```sh
cue mod init github.com/myorg/project@v0   # init module
cue mod tidy                                # clean deps
cue fmt ./...                               # format
cue eval ./...                              # evaluate
cue eval -c ./...                           # require concreteness
cue vet data.yaml schema.cue -d '#Schema'   # validate
cue export ./... --out json                 # export JSON
cue export ./... --out yaml                 # export YAML
cue def --out jsonschema ./...              # JSON Schema (v0.14+)
cue fix ./...                               # fix deprecated syntax
cue get go k8s.io/api/apps/v1              # generate CUE from Go
cue refactor imports old/path new/path      # refactor imports (v0.13+)
```

---

## Anti-Patterns

### ❌ List arithmetic (removed in v0.11)

```cue
// BAD                              // GOOD
combined: [1,2] + [3,4]            import "list"
repeated: [0] * 3                  combined: list.Concat([[1,2],[3,4]])
                                   repeated: list.Repeat([0], 3)
```

### ❌ Missing `language.version`

```cue
// BAD                              // GOOD
module: "example.com/foo@v0"       module: "example.com/foo@v0"
                                   language: version: "v0.16.0"
```

### ❌ Overly permissive types

```cue
// BAD — defeats CUE's purpose      // GOOD — constrain narrowly
config: _                           config: #AppConfig
```

### ❌ Concrete values in definitions

```cue
// BAD — mixes data and schema      // GOOD — definitions are constraints
#Service: {name: "my-svc"}         #Service: {name: string & strings.MinRunes(1)}
```

### ❌ Repeated constraints

```cue
// BAD                              // GOOD
svcA: port: int & >0 & <=65535    #Port: int & >0 & <=65535
svcB: port: int & >0 & <=65535    svcA: port: #Port
                                   svcB: port: #Port
```

### ❌ `@embed` without `@extern(embed)` (v0.12+)

```cue
// BAD                              // GOOD
package config                     @extern(embed)
data: _ @embed(file="d.json")     package config
                                   data: _ @embed(file="d.json")
```

### ❌ Relative imports

```cue
// BAD                              // GOOD
import "./utils"                   import "github.com/myorg/project/utils"
```

### ❌ Multiline strings without trailing newline (v0.14+)

```cue
// BAD                              // GOOD
msg: """                           msg: """
    Hello"""                           Hello
                                       """
```
