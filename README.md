# go-aac

Pure Go AAC (Advanced Audio Coding) decoder. No CGO dependencies.

[![Go Reference](https://pkg.go.dev/badge/github.com/llehouerou/go-aac.svg)](https://pkg.go.dev/github.com/llehouerou/go-aac)
[![License: GPL v2](https://img.shields.io/badge/License-GPL%20v2-blue.svg)](https://www.gnu.org/licenses/old-licenses/gpl-2.0.en.html)

## Overview

go-aac is a pure Go port of [FAAD2](https://github.com/knik0/faad2) (Freeware Advanced Audio Decoder), providing AAC decoding without any C dependencies.

### Supported Formats

- **AAC-LC** (Low Complexity) - Most common AAC profile
- **HE-AAC** (High Efficiency) - AAC-LC + SBR (Spectral Band Replication)
- **HE-AACv2** - HE-AAC + PS (Parametric Stereo)

### Container Support

- Raw AAC (ADTS)
- M4A/MP4 (with external demuxer)

## Installation

```bash
go get github.com/llehouerou/go-aac
```

## Usage

```go
package main

import (
    "os"
    "github.com/llehouerou/go-aac"
)

func main() {
    // Open AAC file
    f, _ := os.Open("audio.aac")
    defer f.Close()

    // Create decoder
    dec, _ := aac.NewDecoder(f)

    // Decode frames
    for {
        samples, err := dec.Decode()
        if err != nil {
            break
        }
        // Process samples ([]int16)
        _ = samples
    }
}
```

## Status

**Work in Progress** - This library is under active development.

See [docs/implementation-plan.md](docs/implementation-plan.md) for the roadmap.

## Building

```bash
# Run tests
make test

# Format and lint
make check

# Generate test data (requires FFmpeg)
make testdata
```

## Attribution

This project is a Go port of FAAD2. The original C implementation is:

> **FAAD2 - Freeware Advanced Audio (AAC) Decoder including SBR decoding**
> Copyright (C) 2003-2005 M. Bakker, Nero AG, http://www.nero.com

Code from FAAD2 is copyright (c) Nero AG, www.nero.com

See [AUTHORS](AUTHORS) for the full list of contributors.

## License

This project is licensed under the GNU General Public License v2.0 - see the [LICENSE](LICENSE) file for details.

As a derivative work of FAAD2, this library must be distributed under the same GPL-2.0 license. Any non-GPL usage is strictly forbidden.

## Related Projects

- [FAAD2](https://github.com/knik0/faad2) - Original C implementation
