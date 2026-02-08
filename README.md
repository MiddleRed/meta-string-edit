# Meta String Editor

A simple editor for Unity IL2CPP `global-metadata.dat` files, allowing you to inspect and modify string literals.

Available as both a Go library/CLI and Python binding package, making it easy to cross platform and integrate with Python scripts.

## Quick Start

### Use CLI

You can download the pre-built executable in [GitHub Release](https://github.com/MiddleRed/meta-string-edit/releases), or use the wrapped library distributed along with Python package.

```bash
pip install metastringedit
```

Usage

```bash
> ./metastringedit
Usage:
  metastringedit <file_path> [flags]

Flags:
  -d, --dump               Dump all strings to JSON
  -e, --edit stringArray   Edit strings (format: nth=value, can input multiple pairs)
  -h, --help               help for metastringedit
  -i, --info               Show metadata file information (default)
  -l, --list string        List strings:
                           - [page:num] for pagination, e.g. 0:20
                           - [nth] for single string, e.g. 3
                           - [start-end] for range, e.g. 1-20
  -o, --output string      Output file path
  -r, --regex string       Search for strings using regex
  -s, --search string      Search for strings (case-insensitive substring)
```

### As Library

See [examples](./examples/).

## Development

**Requirements:**
- Python 3.9+
- Go 1.17+

### Building
Go CLI
```bash
cd cli && go build .
```

Python wheel
```bash
uv build
```

### Running Tests
Go
```bash
go test -v
```

Python
```bash
# Build shared library first. Here is Linux example
cd binding && go build -o metastringedit.so -buildmode=c-shared . && cd ..

uv sync --extra dev --no-install-project
pytest
```

## Acknowledgments

https://github.com/Perfare/Il2CppDumper  
https://github.com/JeremieCHN/MetaDataStringEditor  
GitHub Copilot  

# License

MIT
