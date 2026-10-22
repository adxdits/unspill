# Third-Party Notices

The detector embeds Gitleaks v8.24.3 using its historical Go module path
`github.com/zricethezav/gitleaks/v8`. Its engine and embedded configuration are
used without copying agent implementations into this repository.

Upstream: <https://github.com/gitleaks/gitleaks/tree/v8.24.3>

## Gitleaks License

MIT License

Copyright (c) 2019 Zachary Rice

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.

## Binary Distributions

Release archives include a `licenses/` directory collected from the binary's
dependencies using go-licenses v2.0.1. This includes Gitleaks, Viper, Zerolog,
and their linked transitive dependencies. Versioned dependencies are recorded
in go.mod and go.sum.

HashiCorp HCL v1.0.0 is licensed under MPL-2.0. Its unmodified module source and
license are included in `source/hcl/` in each release archive. Other dependency
licenses and copyright notices are preserved in `licenses/`. Unspill's own
source is licensed under the MIT license in LICENSE.