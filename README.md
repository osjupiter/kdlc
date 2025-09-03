# KDL to JSON Converter

A fast and efficient tool to convert KDL (KDL Document Language) files to JSON format, written in Go.

## Features

- 🚀 **Fast conversion** using kdl-go library
- 📁 **@include support** for including other KDL files
- 🔄 **Circular reference detection** to prevent infinite loops
- 🎯 **Customizable argument names** (arg1, arg2, etc.)
- 📦 **Duplicate node handling** with array grouping
- 🧹 **Clean JSON output** with flattened structure

## Installation

### Pre-built Binaries

Download the latest release from [GitHub Releases](https://github.com/osjupiter/kdlc/releases).

### From Source

```bash
git clone https://github.com/osjupiter/kdlc.git
cd kdlc
go build -o kdlc main.go
```

### Using Go Install

```bash
go install github.com/osjupiter/kdlc@latest
```

## Usage

### Basic Usage

```bash
kdlc <kdl-file>
```

### Custom Argument Names

Customize argument names in the output JSON:

```bash
kdlc -arg1=name -arg2=value input.kdl
```

Available options:
- `-arg1 string`: Name for first argument (default "arg1")
- `-arg2 string`: Name for second argument (default "arg2")
- `-arg3 string`: Name for third argument (default "arg3")
- `-arg4 string`: Name for fourth argument (default "arg4")
- `-arg5 string`: Name for fifth argument (default "arg5")

### @include Support

Include other KDL files:

```kdl
// main.kdl
@include "config.kdl"

scene "MainScene" {
    title "Main Scene"
}
```

```kdl
// config.kdl
config {
    version "1.0"
    theme "dark"
}
```

## Examples

### Input (example.kdl)
```kdl
item "sword" "weapon" damage=10
button "OK" "primary" x=100 y=100
scene "GameScene" {
    player "hero" level=5
}
```

### Default Output
```bash
kdlc example.kdl
```

```json
{
  "item": {
    "arg1": "sword",
    "arg2": "weapon",
    "damage": 10
  },
  "button": {
    "arg1": "OK",
    "arg2": "primary",
    "x": 100,
    "y": 100
  },
  "scene": {
    "arg1": "GameScene",
    "player": {
      "arg1": "hero",
      "level": 5
    }
  }
}
```

### Custom Argument Names
```bash
kdlc -arg1=type -arg2=category example.kdl
```

```json
{
  "item": {
    "type": "sword",
    "category": "weapon",
    "damage": 10
  },
  "button": {
    "type": "OK",
    "category": "primary",
    "x": 100,
    "y": 100
  },
  "scene": {
    "type": "GameScene",
    "player": {
      "type": "hero",
      "level": 5
    }
  }
}
```

## Testing

Run tests:

```bash
go test
go test -v  # verbose output
```

## Dependencies

- [github.com/sblinch/kdl-go](https://github.com/sblinch/kdl-go) - KDL parsing library

## License

MIT License