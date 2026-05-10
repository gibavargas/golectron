package webcontents

import (
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidPrintOptions = errors.New("invalid print options")

type PageSize struct {
	WidthMicrons  int
	HeightMicrons int
}

type PrintOptions struct {
	Silent                    bool
	PrintBackground           bool
	DeviceName                string
	Copies                    int
	PageSize                  *PageSize
	UsePrinterDefaultPageSize bool
}

type PrintJob struct {
	WebContentsID             int64
	Silent                    bool
	PrintBackground           bool
	DeviceName                string
	Copies                    int
	PageSize                  *PageSize
	UsePrinterDefaultPageSize bool
}

func (wc *WebContents) Print(options PrintOptions) (PrintJob, error) {
	job, err := wc.normalizePrintOptions(options)
	if err != nil {
		return PrintJob{}, err
	}
	wc.lastPrint = &job
	return job, nil
}

func (wc *WebContents) LastPrintJob() (PrintJob, bool) {
	if wc.lastPrint == nil {
		return PrintJob{}, false
	}
	job := *wc.lastPrint
	if job.PageSize != nil {
		size := *job.PageSize
		job.PageSize = &size
	}
	return job, true
}

func (wc *WebContents) normalizePrintOptions(options PrintOptions) (PrintJob, error) {
	copies := options.Copies
	if copies == 0 {
		copies = 1
	}
	if copies < 0 {
		return PrintJob{}, fmt.Errorf("%w: copies must be positive", ErrInvalidPrintOptions)
	}
	deviceName := strings.TrimSpace(options.DeviceName)
	if options.UsePrinterDefaultPageSize && options.PageSize != nil {
		return PrintJob{}, fmt.Errorf("%w: usePrinterDefaultPageSize conflicts with pageSize", ErrInvalidPrintOptions)
	}
	var pageSize *PageSize
	if options.PageSize != nil {
		if options.PageSize.WidthMicrons <= 0 || options.PageSize.HeightMicrons <= 0 {
			return PrintJob{}, fmt.Errorf("%w: pageSize dimensions must be positive", ErrInvalidPrintOptions)
		}
		size := *options.PageSize
		pageSize = &size
	}
	return PrintJob{
		WebContentsID:             wc.id,
		Silent:                    options.Silent,
		PrintBackground:           options.PrintBackground,
		DeviceName:                deviceName,
		Copies:                    copies,
		PageSize:                  pageSize,
		UsePrinterDefaultPageSize: options.UsePrinterDefaultPageSize,
	}, nil
}
