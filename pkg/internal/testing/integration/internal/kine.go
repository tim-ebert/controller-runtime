package internal

// KineDefaultArgs allow tests to run offline, by preventing API server from attempting to
// use default route to determine its urls.
var KineDefaultArgs = []string{
	"--listen-address={{ if .URL }}{{ .URL.String }}{{ end }}",
	"--endpoint={{ .DSN }}",
}

// DoKineArgDefaulting will set default values to allow tests to run offline when the args are not informed. Otherwise,
// it will return the same []string arg passed as param.
func DoKineArgDefaulting(args []string) []string {
	if len(args) != 0 {
		return args
	}

	return KineDefaultArgs
}

// GetKineStartMessage returns an start message to inform if the client is or not insecure.
// It will return true when the URL informed has the scheme == "https" || scheme == "unixs"
func GetKineStartMessage() string {
	return "Kine listening on "
}
