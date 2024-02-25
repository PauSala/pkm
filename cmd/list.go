package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all packages and apps containing a package.json file in the current directory or the specified directory.",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			entries, err := GetPackageJsonFiles(path, omit)
			if err != nil {
				fmt.Println(err.Error())
			}

			for _, e := range entries {
				fmt.Println(e)
			}
		},
	}
	path string
	omit string
)

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&path, "path", "p", ".", "Path to the directory")
	listCmd.Flags().StringVarP(&omit, "omit", "o", "", "Path to the directory")
}

func GetPackageJsonFiles(path string, omit string) (result []string, err error) {
	result = []string{}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.IsDir() && !strings.Contains(e.Name(), "node_modules") && e.Name() != omit {
			path := path + "/" + e.Name()
			child, err := GetPackageJsonFiles(path, omit)
			if err != nil {
				fmt.Println(err.Error())
			}
			result = append(result, child...)
		}
		if e.Name() == "package.json" {
			path := path + "/" + e.Name()
			result = append(result, path)
		}
	}
	return result, nil
}
