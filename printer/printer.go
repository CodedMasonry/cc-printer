package printer

import (
	"log/slog"
	"os"
	"os/exec"

	"github.com/CodedMasonry/cc-printer/common"
)

func PrintFile(file *os.File) {
	args := common.GlobalConfig.PrintFlags
	if common.GlobalConfig.Printer != "default" {
		args = append(args, "-d", common.GlobalConfig.Printer)
	}

	slog.Debug("Printing Details", "args", args)
	out, err := exec.Command("lp", args...).Output()
	if err != nil {
		slog.Error("Failed to print file", "error", err)
	}
	slog.Debug("Added file to print queue", "printer", common.GlobalConfig.Printer, "output", out)
}
