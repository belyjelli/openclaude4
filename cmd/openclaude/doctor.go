package main

import (
	"os"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Print environment and provider diagnostics",
	Run:   runDoctor,
}

func runDoctor(_ *cobra.Command, _ []string) {
	PrintDoctorReport(os.Stdout, version, commit)
}
