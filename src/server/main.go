package server

import "os"

func init() {
	// Skip server startup if running migration commands
	if len(os.Args) > 1 && (os.Args[1] == "migrate:down" || os.Args[1] == "migrate:status" || os.Args[1] == "migrate:down-to") {
		return
	}
	serve()
}
