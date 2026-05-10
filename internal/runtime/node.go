package runtime

type NodeOptions struct {
	ExperimentalTransformTypes bool
}

func ExtractNodeOptions(args []string) NodeOptions {
	var options NodeOptions
	for _, arg := range args {
		if arg == "--experimental-transform-types" {
			options.ExperimentalTransformTypes = true
		}
	}
	return options
}
