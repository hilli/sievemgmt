module github.com/hilli/sievemgmt

go 1.26.4

require (
	github.com/spf13/cobra v1.10.2
	go.guido-berhoerster.org/managesieve v0.8.1
	golang.org/x/term v0.44.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	golang.org/x/sys v0.46.0 // indirect
)

// Use a local, patched copy of managesieve that tolerates servers (e.g. mox)
// which send a capabilities list after a successful AUTHENTICATE.
replace go.guido-berhoerster.org/managesieve => ./third_party/managesieve
