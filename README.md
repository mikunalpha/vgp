## Usage
*reqire `glide` installed
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

### Using glide to manage packages
```
vgp get some/package/name
```
Below sub-commands will be passed `glide`:
```
"config-wizard", "cw", "get", "update", "up", "remove", "rm", "info", "novendor", "nv", "tree":
```

### Using go command
Other sub-commands will be passed to `go`.
```
vgp build
```
Will build your binary and put it into `dist` directory.