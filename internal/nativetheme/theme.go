package nativetheme

import "context"

type Report struct {
	Platform                          string `json:"platform"`
	ShouldDifferentiateWithoutColor   bool   `json:"shouldDifferentiateWithoutColor"`
	SupportsDifferentiateWithoutColor bool   `json:"supportsDifferentiateWithoutColor"`
}

func Snapshot(ctx context.Context) (Report, error) {
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	return platformSnapshot()
}
