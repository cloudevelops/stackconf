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
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"html/template"

	"github.com/Masterminds/sprig"
	"github.com/davecgh/go-spew/spew"
	"github.com/juju/loggo"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var log = loggo.GetLogger("cmd")
var httpClient = &http.Client{Timeout: time.Second * 10}
var metaData map[string]interface{}
var noop bool
var noopMsg string
var whitelist string
var deleteDomains bool

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
	RootCmd.PersistentFlags().BoolVarP(&noop, "noop", "n", false, "dry run (do not attempt to make any changes)")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	deleteenvCmd.Flags().StringVarP(&whitelist, "whitelist", "w", "", "Whitelisted entries not to be deleted, comma separated")
	deleteenvCmd.Flags().BoolVarP(&deleteDomains, "deletedomains", "d", false, "Domains will be deleted based on input match")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if noop {
		noopMsg = "[NOOP-Skipping]:"
	}
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
	viper.SetDefault("puppet.config.runs", 3)
	viper.SetDefault("puppet.config.runtimeout", 900)
	if _, err := os.Stat("/opt/puppetlabs/bin/puppet"); err == nil {
		viper.SetDefault("puppet.version", 4)
	} else {
		viper.SetDefault("puppet.version", 3)
	}

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

	// Detect and load Openstackmeta source
	if isInArray("openstackmeta", viper.GetStringSlice("stackconf.sources")) {
		log.Debugf("Openstackmeta enabled: starting")
		if err := openstackMeta(); err != nil {
			log.Errorf("Openstackmeta failed: critical error")
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
	puppetVersion := viper.GetInt("puppet.version")
	var facterExecutable string
	if puppetVersion == 4 {
		facterExecutable = "/opt/puppetlabs/bin/facter"
	} else {
		facterExecutable = "/usr/bin/facter"
	}
	cmd := exec.Command(facterExecutable, "-j")
	var outb bytes.Buffer
	cmd.Stdout = &outb
	err = cmd.Run()
	if err != nil {
		log.Debugf("Facter execution failed. Trying to continue...")
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
	viper.MergeConfig(bytes.NewReader(factermash))
	log.Debugf("Facter version: " + viper.GetString("puppetfacter.facterversion"))
	return
}

func openstackMeta() (err error) {
	type openstackMetadata struct {
		Openstackmetadata map[string]interface{} `json:"openstackmeta"`
	}
	var metadata interface{}
	var mv map[string]interface{}
	var envmeta string

	r, err := httpClient.Get("http://169.254.169.254/openstack/latest/meta_data.json")
	if err != nil {
		log.Errorf("HTTP request to Openstack Metadata failed !")
		return
	}
	if r.StatusCode < 200 || r.StatusCode > 299 {
		log.Errorf("HTTP request to Openstack Metadata failed, error: " + r.Status + "!")
		return
	}
	response, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("Error reading body !")
		return
	}
	err = json.Unmarshal(response, &metadata)
	if err != nil {
		log.Errorf("Error while reading JSON !")
		return
	}
	m := metadata.(map[string]interface{})
	// Map JSON and prepend it with openstackmeta key
	metamash, err := json.Marshal(openstackMetadata{Openstackmetadata: m})
	if err != nil {
		log.Debugf("Openstackmeta JSON prepend failed !")
		return
	}
	// Load Openstack metadata into Viper
	viper.SetConfigType("json")
	viper.MergeConfig(bytes.NewReader(metamash))

	log.Debugf("Openstack metadata loaded into config, instance name: " + viper.GetString("openstackmeta.name"))
	// Iterate through raw Opentack metadata and extract host and environment metadata
	for k, v := range m {
		if k == "meta" {
			mv = v.(map[string]interface{})
			for sk, sv := range mv {
				if sk == "metadata" {
					envmeta = sv.(string)
				}
			}
		}
	}
	// Iterate again, append to environment metadata and overwrite if needed
	err = json.Unmarshal([]byte(envmeta), &metaData)
	if err != nil {
		log.Debugf("Failed to Unmarshal env metadata:" + envmeta + " !")
		return
	}
	for k, v := range mv {
		if k != "metadata" {
			// Non string values have to be parsed for array and Unmarshaled to interface again to fix terrible Openstack dual escaping
			vString := v.(string)
			var vMJson interface{}
			if len(vString) != 0 && vString[:1] == "[" {
				vJson := `{"` + k + `":` + vString + `}`
				err := json.Unmarshal([]byte(vJson), &vMJson)
				if err != nil {
					log.Debugf("Openstackmeta Array JSON Unmarshal failed")
				}
				metaData[k] = vMJson.(map[string]interface{})[k]
			} else {
				metaData[k] = vString
			}
		}
	}
	var envStr string
	envStr = viper.GetString("stackenv")
	if envStr != "" {
		log.Debugf("Using stackenv environment defined in config file: " + envStr)
	} else {
		env := metaData["stackenv"]
		if env != nil {
			var ok bool
			envStr, ok = env.(string)
			if !ok {
				log.Debugf("Failed to get stackenv variable from metadata")
			}
		} else {
			log.Debugf("Metadata stackenv variable is not present")
		}
	}
	if envStr != "" {
		log.Debugf("Did get stackenv variable, will set environment specific configuration fore environment: " + envStr)
		envData := viper.Get("env." + envStr)
		envMap, ok := envData.(map[string]interface{})
		if !ok {
			log.Debugf("Failed to read environment specific configuration")
		} else {
			if envData != nil {
				for k, v := range envMap {
					metaData[k] = v
				}
				log.Debugf("Loaded stackenv environment " + envStr)
			} else {
				log.Debugf("Stackenv variable set to " + envStr + " , but did not find environment specific configuration")
			}
		}
	} else {
		log.Debugf("Did not get stackenv variable, will not set environment specific configuration")
	}

	// Marshall metadata
	hostmetadata, err := json.Marshal(metaData)
	if err != nil {
		log.Debugf("Openstackmeta JSON prepend failed !")
		return
	}
	// dump allsettings
	//allsettings := viper.AllSettings()
	//spew.Dump(allsettings)

	// Load host metadata to config
	viper.SetConfigType("json")
	viper.MergeConfig(bytes.NewReader(hostmetadata))
	log.Debugf("Host metadata from openstack loaded into config")

	// Fix puppet runs
	if value, ok := metaData["puppet.config.runs"]; ok {
		viper.Set("puppet.config.runs", value)
	}

	//allsettings = viper.AllSettings()
	//spew.Dump(allsettings)

	return
}

func metaGetMerge(key string) (parameter map[string]string, err error) {
	parameter = make(map[string]string)
	for k, v := range metaData {
		if strings.Contains(k, key) {
			newkey := strings.Replace(k, key+".", "", -1)
			parameter[newkey] = v.(string)
		}
	}
	return
}

func metaTemplate(text string) (parsed string, err error) {
	t := template.Must(template.New("metaTemplate").Funcs(sprig.FuncMap()).Parse(text))
	var tpl bytes.Buffer
	err = t.Execute(&tpl, metaData)
	if err != nil {
		log.Debugf("Error during template execution" + err.Error())
	}
	return tpl.String(), err
}

func unEscape(escaped string) (unescaped string) {

	var escapedBytes []byte = []byte(escaped)
	var resultString string
	log.Debugf("Stringing bytes:")
	for i := 1; i < 4; i++ {
		newchar := string(escapedBytes[i])
		index := strconv.Itoa(i)
		log.Debugf("Index:" + index + ",Char:" + newchar)
	}
	if string(escapedBytes[1]) == "[" {
		for i, char := range escapedBytes {
			if i > 0 && i < len(escapedBytes) {
				resultString = resultString + string(char)
			}
		}
		unescaped = resultString
		log.Debugf("Brackets are matched !")
	} else {
		unescaped = escaped
	}
	spew.Dump(escaped)
	spew.Dump(unescaped)
	//    var resultstring string
	//    for i, char := range val {
	//        index := strconv.Itoa(i)
	//        value := string(char)
	//        fmt.Println("Char:"+index+"Value:"+value)
	//        resultstring = resultstring + value
	//        //spew.Dump(char)
	//    }
	//    s, _ := strconv.Unquote(string(val))
	//    spew.Dump(resultstring)
	//    var err error
	//    unescaped,err = strconv.Unquote(`"`+escaped+`"`)
	//    if err != nil {
	//        spew.Dump(unescaped)
	//        log.Errorf("Failed to unquote: "+err.Error())
	//    }
	//  unattempt := strings.Replace(escaped, "\\", "", -1)
	//    spew.Dump(unattempt)
	//unescaped = strings.Replace(escaped,"\\","", -1)
	//spew.Dump(escaped)
	//spew.Dump(unescaped)
	return unescaped
}
