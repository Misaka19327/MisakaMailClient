// Command misaka-mail is a third-party email CLI.
package main

import "MisakaMailClient/internal/cmd"

// version is reported via --version. It can be overridden at build time with
// -ldflags "-X main.version=...".
var version = "0.1.0"

func main() {
	cmd.Execute(version)
}
