// Copyright © 2017 Zdenek Janda <zdenek.janda@cloudevelops.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/juju/loggo"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"os/exec"
)

var cfgFile string
var log = loggo.GetLogger("cmd")

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "stackconf",
	Short: "Openstack instance config management with ease",
	Long:  `Openstack instance config management with ease`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	loggo.ConfigureLoggers("<root>=TRACE")
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.stackconf.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".stackconf" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath("/etc")
		viper.SetConfigName(".stackconf")
	}

	viper.SetDefault("stackconf.tools", "puppet")
	viper.SetDefault("stackconf.sources", []string{"openstackmeta", "puppetfacter"})

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Debugf("Using config file: " + viper.ConfigFileUsed())
	}
	// Detect and load Facter source
	if isInArray("puppetfacter", viper.GetStringSlice("stackconf.sources")) {
		log.Debugf("Facter enabled: starting")
		if err := facter(); err != nil {
			log.Errorf("Facter failed: critical error")
		}
	}
}

func isInArray(val string, array []string) (ok bool) {
	var i int
	for i = range array {
		if ok = array[i] == val; ok {
			return
		}
	}
	return
}

func facter() (err error) {
	type puppetFacter struct {
		Puppetfacter map[string]interface{} `json:"puppetfacter"`
	}
	var facterdata interface{}
	// Run facter and output JSON
	cmd := exec.Command("facter", "-j")
	var outb bytes.Buffer
	cmd.Stdout = &outb
	err = cmd.Run()
	if err != nil {
		log.Debugf("Facter execution failed !")
		return
	}
	// Unmarshall JSON into plain interface
	err = json.Unmarshal(outb.Bytes(), &facterdata)
	if err != nil {
		log.Debugf("Facter JSON Unmarshall failed !")
		return
	}
	// Map JSON and prepend it with puppetfacter key
	m := facterdata.(map[string]interface{})
	factermash, err := json.Marshal(puppetFacter{Puppetfacter: m})
	if err != nil {
		log.Debugf("Facter JSON prepend failed !")
		return
	}
	// Load final JSON into Viper
	viper.SetConfigType("json")
	viper.ReadConfig(bytes.NewReader(factermash))
	log.Debugf("Facter version: " + viper.GetString("puppetfacter.facterversion"))
	return
}
