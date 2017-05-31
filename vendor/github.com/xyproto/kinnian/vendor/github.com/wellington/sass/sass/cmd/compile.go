package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/wellington/sass/compiler"
)

// compileCmd represents the compile command
var compileCmd = &cobra.Command{
	Use:   "compile",
	Short: "compile takes as input Sass or SCSS and outputs CSS",
	Long: `compile takes as input Sass or SCSS and outputs CSS

Usage: sass compile file.scss
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("files % #v\n", args)
		fmt.Printf("args % #v\n", cmd.Flag("output").Value)
		if len(args) != 1 {
			log.Fatal("must pass a single file ie. compile file.scss")
		}
		for _, file := range args {
			var (
				s   string
				err error
			)
			if len(outFile) > 0 {
				err = compiler.File(file, outFile)
			} else {
				s, err = compiler.Run(file)
			}

			if err != nil {
				log.Fatalf("error compiling %s: %s", file, err)
			}
			fmt.Printf("Compiled %s\n", file)
			fmt.Println(s)
		}
	},
}

var outFile string

func init() {
	RootCmd.AddCommand(compileCmd)

	compileCmd.Flags().StringVarP(&outFile, "output", "o", "", "location of output CSS file")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// compileCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// compileCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
