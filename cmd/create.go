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
	//"github.com/davecgh/go-spew/spew"
	"encoding/json"
	"github.com/cloudevelops/go-foreman"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strconv"
	"strings"
    "os/exec"
    //"bytes"
    "fmt"
    "bufio"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new stackconf host",
	Long:  `Create a new stackconf host.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debugf("Create command: starting")

		//Foreman prototype
		f := foreman.NewForeman(viper.GetString("foreman.config.host"), viper.GetString("foreman.config.username"), viper.GetString("foreman.config.password"))
		// Host
		hostFqdn := viper.GetString("openstackmeta.name")
		hostNameSplit := strings.Split(hostFqdn, ".")
		hostName := hostNameSplit[0]
		domainName := strings.Replace(hostFqdn, hostName+".", "", -1)
		host, err := f.SearchResource("hosts", hostFqdn)
		if err == nil {
			log.Debugf("Host exists, deleting")
			hostId := strconv.FormatFloat(host["id"].(float64), 'f', -1, 64)
			err := f.DeleteHost(hostId)
			if err != nil {
				log.Debugf("Host deletion failed")
			}
		}
		// Domain
		domain, err := f.SearchResource("domains", domainName)
		var domainId string
		if err == nil {
			domainId = strconv.FormatFloat(domain["id"].(float64), 'f', -1, 64)
			log.Debugf("Domain found, name: " + domainName + "; id: " + domainId)
		} else {
			log.Errorf("Domain !")
			return
		}
		//Hostgroup
		hostGroupName := viper.GetString("foreman.host.hostgroup")
		if hostGroupName == "" {
			log.Debugf("Hostgroup not found !")
			return
		}
		hostGroup, err := f.SearchResource("hostgroups", hostGroupName)
		var hostGroupId string
		if err == nil {
			hostGroupId = strconv.FormatFloat(hostGroup["id"].(float64), 'f', -1, 64)
			log.Debugf("Hostgroup found, name: " + hostGroupName + "; id: " + hostGroupId)
		} else {
			log.Errorf("Hostgroup doesnt exist !")
			return
		}
		// Organization
		orgLoc := strings.Split(hostGroupName, "/")
		organizationName := orgLoc[0]
		organization, err := f.SearchResource("organizations", organizationName)
		var organizationId string
		if err == nil {
			organizationId = strconv.FormatFloat(organization["id"].(float64), 'f', -1, 64)
			log.Debugf("Organization found, name: " + organizationName + "; id: " + organizationId)
		} else {
			log.Errorf("Organization doesnt exist !")
			return
		}

		// Location
		locationName := hostNameSplit[len(hostNameSplit)-1]
		location, err := f.SearchResource("locations", locationName)
		var locationId string
		if err == nil {
			locationId = strconv.FormatFloat(location["id"].(float64), 'f', -1, 64)
			log.Debugf("Location found, name: " + locationName + "; id: " + locationId)
		} else {
			log.Errorf("Location doesnt exist !")
			return
		}
		// puppetca
		puppetCaName := viper.GetString("puppet.config.ca")
		if hostGroupName == "" {
			log.Debugf("Puppet CA not found !")
			return
		}
		puppetCa, err := f.SearchResource("smart_proxies", puppetCaName)
		var puppetCaId string
		if err == nil {
			puppetCaId = strconv.FormatFloat(puppetCa["id"].(float64), 'f', -1, 64)
			log.Debugf("Puppet CA found, name: " + puppetCaName + "; id: " + puppetCaId)
		} else {
			log.Errorf("Puppet CA doesnt exist !")
			return
		}
		// environment
		puppetEnvironmentName := viper.GetString("puppet.config.environment")
		if puppetEnvironmentName == "" {
			log.Debugf("Puppet Environment not found !")
			return
		}
		puppetEnvironment, err := f.SearchResource("environments", puppetEnvironmentName)
		var puppetEnvironmentId string
		if err == nil {
			puppetEnvironmentId = strconv.FormatFloat(puppetEnvironment["id"].(float64), 'f', -1, 64)
			log.Debugf("Puppet Environment found, name: " + puppetEnvironmentName + "; id: " + puppetEnvironmentId)
		} else {
			log.Errorf("Puppet Environment doesnt exist !")
			return
		}
		// architecture
		architectureName := viper.GetString("puppetfacter.os.hardware")
		if architectureName == "" {
			log.Debugf("Architecture not found !")
			return
		}
		architecture, err := f.SearchResource("architectures", architectureName)
		var architectureId string
		if err == nil {
			architectureId = strconv.FormatFloat(architecture["id"].(float64), 'f', -1, 64)
			log.Debugf("Architecture found, name: " + architectureName + "; id: " + architectureId)
		} else {
			log.Errorf("Architecture doesnt exist !")
			return
		}
		// operatingsystem
		osName := viper.GetString("puppetfacter.os.name")
		var operatingSystemName string
		if osName == "Ubuntu" {
			operatingSystemName = viper.GetString("puppetfacter.os.distro.description")
		} else {
			operatingSystemName = viper.GetString("puppetfacter.os.distro.id") + " " + viper.GetString("puppetfacter.os.distro.release.full")
		}
		if operatingSystemName == "" {
			log.Debugf("Operating System not found !")
			return
		}
		operatingSystem, err := f.SearchResource("operatingsystems", operatingSystemName)
		var operatingSystemId string
		if err == nil {
			operatingSystemId = strconv.FormatFloat(operatingSystem["id"].(float64), 'f', -1, 64)
			log.Debugf("Operating System found, name: " + operatingSystemName + "; id: " + operatingSystemId)
		} else {
			log.Errorf("Operating System doesnt exist !" + operatingSystemName)
			return
		}
		// ipAddress
		ipAddress := viper.GetString("puppetfacter.networking.ip")
		if ipAddress == "" {
			log.Debugf("IP Address not found !")
			return
		} else {
			log.Debugf("IP Address: " + ipAddress)
		}
		// macAddress
		macAddress := viper.GetString("puppetfacter.networking.mac")
		if macAddress == "" {
			log.Debugf("Mac Address not found !")
			return
		} else {
			log.Debugf("Mac Address: " + macAddress)
		}
		// parameters
		var parameters []map[string]string
		var paramMap map[string]string
		metaparameters, err := metaGetMerge("foreman.host.parameter")
		for metak, metav := range metaparameters {
			paramMap = make(map[string]string)
			paramMap["name"] = metak
			paramMap["value"] = metav
			parameters = append(parameters, paramMap)
		}
		if err != nil {
			log.Debugf("Did not find host parameters")
		}
		// create host
		type HostResource struct {
			HostGroupId         string              `json:"hostgroup_id"`
			PuppetCaId          string              `json:"puppet_ca_proxy_id"`
			LocationId          string              `json:"location_id"`
			OrganizationId      string              `json:"organization_id"`
			PuppetEnvironmentId string              `json:"environment_id"`
			DomainId            string              `json:"domain_id"`
			OperatingSystemId   string              `json:"operatingsystem_id"`
			ArchitectureId      string              `json:"architecture_id"`
			Name                string              `json:"name"`
			Mac                 string              `json:"mac"`
			Ip                  string              `json:"ip"`
			Build               bool                `json:"build"`
			Parameters          []map[string]string `json:"host_parameters_attributes"`
		}
		type HostMap map[string]HostResource

		//var hostMap map[string]HostResource
		hostMap := make(HostMap)
		hostMap["host"] = HostResource{
			HostGroupId:         hostGroupId,
			PuppetCaId:          puppetCaId,
			LocationId:          locationId,
			OrganizationId:      organizationId,
			PuppetEnvironmentId: puppetEnvironmentId,
			DomainId:            domainId,
			OperatingSystemId:   operatingSystemId,
			ArchitectureId:      architectureId,
			Name:                hostName,
			Mac:                 macAddress,
			Ip:                  ipAddress,
			Build:               false,
			Parameters:          parameters,
		}
		jsonText, err := json.Marshal(hostMap)
		data, err := f.Post("hosts", jsonText)
		if err != nil {
			log.Errorf("Error creating host !")
			return
		}
		hostId := strconv.FormatFloat(data["id"].(float64), 'f', 0, 64)
		log.Debugf("Host created, id: " + hostId)
		// Configure Puppet execution
        puppetServer := viper.GetString("puppet.config.server")
        var puppetParam []string
		if puppetServer == "" {
			log.Debugf("Puppet Server not found !")
            puppetSrv := viper.GetString("puppet.config.srv")
            if puppetSrv == "" {
                log.Errorf("Puppet Server or SRV not found, exiting !")
                return
            } else {
                log.Debugf("Puppet will run in SRV mode in domain: "+puppetSrv)
                puppetParam = []string{"agent","-tv","--use_srv_records","--srv_domain",puppetSrv}
            }
        } else {
            log.Debugf("Puppet will run in server mode with server: "+puppetServer)
            puppetParam = []string{"agent","-tv","--no-use_srv_records","--server",puppetServer}
        }
        // Enable Puppet
        log.Debugf("Enabling puppet")
        puppetEnabler := exec.Command("/opt/puppetlabs/bin/puppet", "agent", "--enable")
        c := make(chan struct{})
        go runCommand(puppetEnabler, c)
        c <- struct{}{}
        puppetEnabler.Start()
        <-c
        if err := puppetEnabler.Wait(); err != nil {
            log.Debugf("Error executing puppet !")
        }
        // Run Puppet
        puppetRuns := viper.GetInt("puppet.config.runs")
        for r := 1; r<=puppetRuns; r++ {
            // Run puppet
            runCount := strconv.Itoa(r)
            log.Debugf("Running puppet, run #"+runCount)
            //spew.Dump(puppetParam)
            cmd := exec.Command("/opt/puppetlabs/bin/puppet", puppetParam...)
            c := make(chan struct{})
            go runCommand(cmd, c)
            c <- struct{}{}
            cmd.Start()
            <-c
            if err := cmd.Wait(); err != nil {
                log.Debugf("Error executing puppet !")
            }
        }
	},
}

func runCommand(cmd *exec.Cmd, c chan struct{}) {
    defer func() { c <- struct{}{}  }()
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        panic(err)
    }
    <-c
    scanner := bufio.NewScanner(stdout)
    for scanner.Scan() {
        m := scanner.Text()
        fmt.Println(m)
    }
}

func init() {
	RootCmd.AddCommand(createCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
