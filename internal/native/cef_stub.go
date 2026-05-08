//go:build !linux || !amd64 || !cgo || !electron_go_cef

package native

import "context"

func ExecuteCEFSubprocess(ctx context.Context, args []string) (SubprocessExecutionResult, bool, error) {
	if err := ctx.Err(); err != nil {
		return SubprocessExecutionResult{}, false, err
	}
	if !IsCEFSubprocessArgs(args) {
		return SubprocessExecutionResult{}, false, nil
	}
	return SubprocessExecutionResult{
		ExitCode: CEFSubprocessUnavailableExitCode,
		Status:   StatusUnavailable,
		Error:    ErrBridgeUnavailable.Error(),
	}, true, nil
}

func CheckCEFInitialize(ctx context.Context, appDir string, args []string) error {
	_ = appDir
	_ = args
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrBridgeUnavailable
}

func CheckRuntimeProcessModel(ctx context.Context, appDir string, args []string) (RuntimeProcessModelReport, error) {
	_ = appDir
	_ = args
	if err := ctx.Err(); err != nil {
		return RuntimeProcessModelReport{}, err
	}
	return RuntimeProcessModelReport{}, ErrBridgeUnavailable
}
