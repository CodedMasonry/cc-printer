package printer

import (
	"fmt"
	"log"
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

	args = append(args, file.Name())

	slog.Debug("Printing Details", "args", args)
	_, err := exec.Command("lp", args...).Output()
	if err != nil {
		if common.GlobalConfig.Printer == "default" {
			fmt.Println("Make sure you set a default printer or\nedit the 'Printer' in the config to the printer you wish to use")
		} else {
			fmt.Println("Failed to print; Make sure the printer in the config is correct & is online")
		}
        log.Fatal("Failed to print file, ", err)
    }
	
	slog.Info("Added file to print queue", "printer", common.GlobalConfig.Printer)
}
