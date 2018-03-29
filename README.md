## Usage
*require `glide` installed
```
go get -u github.com/mikunalpha/vgp
```

### vgp.ini
Create `vgp.ini` file under your project
```
; package name of this go project
package_name=github.com/yourname/yourpackage

; output binary name
out=yourpackage
```

### Init
```
vgp init && vgp up
```

### Project Structure
```
- vgp.ini
- .vscode/
  - settings.json
- dist/
- src/
  - github.com
    - yourname
      - yourpackage
        - vendor/
        - glide.lock
        - glide.yaml
```

### Using glide to manage packages
Below sub-commands will be passed to `glide`:
```
"config-wizard", "cw", "get", "update", "up", "remove", "rm", "info", "novendor", "nv", "tree"
```
Download a package.
```
vgp get some/package/name
```
It's recommended that use `vgp up` after you import a new downloaded package.

### Using go command
Other sub-commands will be passed to `go`.
```
vgp build
```
Will build your binary and put it into `dist` directory.
