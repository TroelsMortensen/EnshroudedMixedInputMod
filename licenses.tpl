{{ $emiVersion := "VERSION_PLACEHOLDER" }}

# github.com/trmo/EnshroudedMixInputMod

* Name: github.com/trmo/EnshroudedMixInputMod
* Version: {{ $emiVersion }}
* License: [GPL-3.0](https://www.gnu.org/licenses/gpl-3.0.html)

EnshroudedMixInput

Derived from [EnshroudedMixedInput](https://github.com/Alia5/EnshroudedMixedInput) by Peter Repukat (GPL-3.0).

This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public
License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later
version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

{{ range . }}
## {{ .Name }}

* Name: {{ .Name }}
* Version: {{ .Version }}
* License: [{{ .LicenseName }}]({{ .LicenseURL }})
{{ end }}
