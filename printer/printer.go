package printer

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/CodedMasonry/cc-printer/common"
)

func PrintFile(file string) {
	args := common.GlobalConfig.PrintFlags
	if common.GlobalConfig.Printer != "default" {
		args = append(args, "-d", common.GlobalConfig.Printer)
	}

	args = append(args, file)

	slog.Debug("Printing Details", "args", args)
	out, err := exec.Command("lp", args...).Output()
	if err != nil {
		fmt.Println("Out: ", out)
		if common.GlobalConfig.Printer == "default" {
			fmt.Println("Make sure you set a default printer or\nedit the 'Printer' in the config to the printer you wish to use")
		} else {
			fmt.Println("Failed to print; Make sure the printer in the config is correct & is online")
		}
		log.Fatal("Failed to print file, ", err)
	}

	slog.Info("Added file to print queue", "printer", common.GlobalConfig.Printer)
}

func Rasterize(orig *os.File) string {
	out := strings.TrimSuffix(orig.Name(), ".pdf")
	out = out + ".png"
	_, err := exec.Command("convert", "-density 600", orig.Name(), out).Output()
	if err != nil {
		slog.Error("Failed to rasterize PDF", "error", err)
	}
	os.Remove(orig.Name())
	return out
}
